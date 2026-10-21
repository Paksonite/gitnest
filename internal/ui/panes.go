package ui

import (
	"fmt"
	"go/ast"
	"maps"
	"os"
	"reflect"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"

	"github.com/Paksonite/gitnest/v2/internal/update"
)

// Shell holds several independent gitnests ("panes") in one terminal. Every
// pane is a full App with its own screen, selection, open repository,
// prompts and editor; the config, themes, scanner and the commit clipboard
// are shared. ctrl+n opens a pane, ctrl+w closes one, alt+arrows move the
// focus and ctrl+g changes the layout. Up to maxTiled panes are tiled; more
// are shown as tabs.
type Shell struct {
	panes  []*pane
	focus  int
	nextID int
	w, h   int
	mode   int // paneGrid, paneColumns or paneRows

	resizeSeq int  // counts size changes, for resizeSettledMsg
	fetching  bool // a background fetch is running
}

// Pane arrangements, cycled with ctrl+g.
const (
	paneGrid    = iota // a near-square grid: 2×2 for four, 3×2 for six…
	paneColumns        // all side by side
	paneRows           // all stacked
	paneModes
)

var paneModeNames = [paneModes]string{"grid", "side by side", "stacked"}

type pane struct {
	id  int
	app *App
}

// paneMsg delivers a message to the pane whose command produced it.
type paneMsg struct {
	id  int
	msg tea.Msg
}

// shellRescanTickMsg / shellStatusTickMsg drive the background work once for
// all panes instead of once per pane.
type (
	shellRescanTickMsg struct{}
	shellStatusTickMsg struct{}
	updateTickMsg      struct{} // look for a new release again (a gitnest left open)
	shellFetchTickMsg  struct{} // fetch every repository in the background
	bgFetchedMsg       struct{ status statusMsg }
)

// firstFetch is how soon after start the first background fetch runs.
var firstFetch = 5 * time.Second

// shellFetchTick schedules the next background fetch in d (fetch_interval
// when d is 0); nothing when it is turned off.
func shellFetchTick(a *App, d time.Duration) tea.Cmd {
	if a.cfg.FetchInterval <= 0 {
		return nil
	}
	if d == 0 {
		d = time.Duration(a.cfg.FetchInterval) * time.Second
	}
	return tea.Tick(d, func(time.Time) tea.Msg { return shellFetchTickMsg{} })
}

// updateTick asks for the release check to run again in update.CheckEvery.
func updateTick() tea.Cmd {
	return tea.Tick(update.CheckEvery, func(time.Time) tea.Msg { return updateTickMsg{} })
}

func newShell(first *App) *Shell {
	mode := paneGrid
	for i, name := range paneModeNames {
		if first.st != nil && first.st.PaneLayout == name {
			mode = i
		}
	}
	return &Shell{panes: []*pane{{id: 0, app: first}}, nextID: 1, mode: mode}
}

func (s *Shell) focused() *pane { return s.panes[s.focus] }

func (s *Shell) byID(id int) *pane {
	for _, p := range s.panes {
		if p.id == id {
			return p
		}
	}
	return nil
}

// wrap tags everything cmd produces with the pane's id, so the answer
// reaches that pane (Batch and Sequence are unpacked and wrapped too).
func wrap(id int, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		switch m := msg.(type) {
		case nil:
			return nil
		case tea.BatchMsg:
			out := make(tea.BatchMsg, len(m))
			for i, c := range m {
				out[i] = wrap(id, c)
			}
			return out
		}
		// tea.Sequence's message is an unexported slice of commands.
		if v := reflect.ValueOf(msg); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeOf(tea.Cmd(nil)) {
			cmds := make([]tea.Cmd, v.Len())
			for i := range cmds {
				cmds[i] = wrap(id, v.Index(i).Interface().(tea.Cmd))
			}
			return tea.Sequence(cmds...)()
		}
		if forProgram(msg) {
			return msg
		}
		return paneMsg{id: id, msg: msg}
	}
}

// forProgram reports whether msg is one of Bubble Tea's own requests to the
// program (run a process such as the editor, clear the screen…): those are
// unexported types the program acts on itself, so they must reach it as
// they are. Their answers (execDoneMsg) come back to the focused pane.
func forProgram(msg tea.Msg) bool {
	t := reflect.TypeOf(msg)
	return t != nil && t.PkgPath() == "github.com/charmbracelet/bubbletea" && !ast.IsExported(t.Name())
}

func shellRescanTick(a *App) tea.Cmd {
	return mapCmd(a.rescanTick(), func(tea.Msg) tea.Msg { return shellRescanTickMsg{} })
}

func shellStatusTick(a *App) tea.Cmd {
	return mapCmd(a.statusTick(), func(tea.Msg) tea.Msg { return shellStatusTickMsg{} })
}

func mapCmd(cmd tea.Cmd, f func(tea.Msg) tea.Msg) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg { return f(cmd()) }
}

func (s *Shell) Init() tea.Cmd {
	a := s.panes[0].app
	return tea.Batch(wrap(s.panes[0].id, a.startCmds()), shellRescanTick(a), shellStatusTick(a), a.updateCheck(), updateTick(), shellFetchTick(a, firstFetch))
}

// broadcast lists the messages about the shared repository list: every
// pane updates its list, only the pane that asked acts on the follow-ups.
func broadcast(msg tea.Msg) bool {
	switch msg.(type) {
	case reposMsg, scanMsg, statusMsg:
		return true
	}
	return false
}

// resizeSettledMsg comes settleDelay after a change of the window's size;
// seq tells whether another change came since.
type resizeSettledMsg struct{ seq int }

// settleDelay is how long the window's size must stay put to count as
// settled.
var settleDelay = 150 * time.Millisecond

// termSize asks the terminal for its size (a variable for tests).
var termSize = func() (w, h int, ok bool) {
	w, h, err := term.GetSize(os.Stdout.Fd())
	return w, h, err == nil && w > 0 && h > 0
}

func (s *Shell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := metaArrow(msg); ok {
		msg = k
	}
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		resized := s.w != 0 && (s.w != m.Width || s.h != m.Height)
		s.w, s.h = m.Width, m.Height
		s.resize()
		if resized {
			// Terminals rewrap what was on screen when the window changes
			// (iTerm2 splitting a pane, say). Bubble Tea redraws every line
			// on a new size (each line covers the whole width), so nothing
			// of the old frame stays; once more when the resizing stops
			// (see resizeSettledMsg). The screen is never cleared (ESC[2J):
			// iTerm2 moves what a clear wipes into its scrollback, and the
			// wheel then scrolls through old frames of gitnest.
			s.resizeSeq++
			seq := s.resizeSeq
			return s, tea.Tick(settleDelay, func(time.Time) tea.Msg { return resizeSettledMsg{seq} })
		}
		return s, nil
	case resizeSettledMsg:
		if m.seq != s.resizeSeq {
			return s, nil // still resizing: a later one settles
		}
		// While a window is dragged, a frame can be drawn for a size the
		// terminal has already left, and terminals such as iTerm2 rewrap
		// the screen after the last size change, so the last frame ends up
		// wider than the window: every line spills onto the next. Ask the
		// terminal for its size again and draw everything once more: a
		// size message, even the same size, makes Bubble Tea redraw every
		// line.
		w, h, ok := termSize()
		if !ok {
			w, h = s.w, s.h
		}
		return s, func() tea.Msg { return tea.WindowSizeMsg{Width: w, Height: h} }
	case tea.KeyMsg:
		return s, s.onKey(m)
	case tea.MouseMsg:
		return s, s.onMouse(m)
	case tea.FocusMsg, tea.BlurMsg:
		return s, s.deliver(s.focused(), msg)
	case shellFetchTickMsg:
		a := s.panes[0].app
		next := shellFetchTick(a, 0)
		if s.fetching || len(a.rs.known) == 0 {
			return s, next
		}
		s.fetching = true
		return s, tea.Batch(a.backgroundFetch(append([]string(nil), a.rs.known...)), next)
	case bgFetchedMsg:
		s.fetching = false
		id := s.panes[0].id // every pane gets the statuses (broadcast)
		return s, func() tea.Msg { return paneMsg{id: id, msg: m.status} }
	case updateTickMsg:
		return s, tea.Batch(s.panes[0].app.updateCheck(), updateTick())
	case updateMsg:
		if s.panes[0].app.sh.update == m.version {
			return s, nil // said already; the header still says it
		}
		s.panes[0].app.sh.update = m.version // every pane shares sh
		s.focused().app.setToast("gitnest "+m.version+" is out — quit and run: "+updateCommand(m.version), false)
		return s, nil
	case shellRescanTickMsg:
		a := s.panes[0].app
		return s, tea.Batch(wrap(s.panes[0].id, a.scanCmd(false)), shellRescanTick(a))
	case shellStatusTickMsg:
		var cmds []tea.Cmd
		for _, p := range s.panes {
			cmds = append(cmds, wrap(p.id, p.app.visibleStatusCmd()))
		}
		return s, tea.Batch(append(cmds, shellStatusTick(s.panes[0].app))...)
	case paneMsg:
		p := s.byID(m.id)
		if p == nil { // the pane was closed meanwhile
			return s, nil
		}
		if _, ok := m.msg.(tea.QuitMsg); ok {
			return s, s.quitFrom(p)
		}
		if o, ok := m.msg.(openPanesMsg); ok {
			return s, s.openRepos(o.paths)
		}
		cmd := s.deliver(p, m.msg)
		if broadcast(m.msg) {
			for _, o := range s.panes {
				if o != p {
					o.app.Update(m.msg) // keep their list current; follow-ups are p's
				}
			}
		}
		return s, cmd
	}
	return s, s.deliver(s.focused(), msg) // e.g. cursor blinks
}

func (s *Shell) deliver(p *pane, msg tea.Msg) tea.Cmd {
	_, cmd := p.app.Update(msg)
	return wrap(p.id, cmd)
}

// quitFrom handles a pane quitting: with several panes, q closes just that
// pane; leaving for the gn shell function (o) or the last pane quits.
func (s *Shell) quitFrom(p *pane) tea.Cmd {
	if len(s.panes) == 1 || p.app.quitAll {
		return tea.Quit
	}
	s.close(p)
	return nil
}

// capturesKeys reports whether the pane is taking text input, where ctrl+n
// and ctrl+w mean "next match" and "delete word" rather than pane keys.
func (a *App) capturesKeys() bool {
	return a.ed != nil || a.p != nil || (a.scr == scrRepos && a.rs.filtering)
}

func (s *Shell) onKey(k tea.KeyMsg) tea.Cmd {
	f := s.focused()
	if k.String() == "ctrl+c" {
		return tea.Quit
	}
	if !f.app.capturesKeys() {
		switch lookup(scPanes, k.String()) {
		case actNewPane:
			return s.open()
		case actClosePane:
			if len(s.panes) == 1 {
				return tea.Quit
			}
			s.close(f)
			return nil
		case actPaneLeft:
			s.focusTowards(-1, 0)
			return nil
		case actPaneRight:
			s.focusTowards(1, 0)
			return nil
		case actPaneUp:
			s.focusTowards(0, -1)
			return nil
		case actPaneDown:
			s.focusTowards(0, 1)
			return nil
		case actPaneLayout:
			s.mode = (s.mode + 1) % paneModes
			if f.app.st != nil {
				f.app.st.SetPaneLayout(paneModeNames[s.mode])
			}
			s.resize()
			msg := "panes: " + paneModeNames[s.mode] + " (ctrl+g changes)"
			if shown := s.shownLayout(); shown != paneModeNames[s.mode] {
				msg = "panes: " + paneModeNames[s.mode] + " doesn't fit this window, showing " + shown
			}
			f.app.setToast(msg, false)
			return nil
		}
	}
	return s.deliver(f, k)
}

// open adds a pane next to the focused one, on the repository list.
func (s *Shell) open() tea.Cmd {
	src := s.focused().app
	a := src.sibling()
	p := &pane{id: s.nextID, app: a}
	s.nextID++
	s.panes = append(s.panes[:s.focus+1], append([]*pane{p}, s.panes[s.focus+1:]...)...)
	s.focus++
	s.resize()
	msg := "new pane: pick a repository (alt+arrows switch, ctrl+w closes, ctrl+g layout)"
	if len(s.panes) == maxTiled+1 {
		msg = fmt.Sprintf("more than %d panes are shown as tabs (alt+←/→ switch, ctrl+w closes)", maxTiled)
	}
	a.setToast(msg, false)
	return wrap(p.id, tea.Batch(a.previewCmd(), repoTick()))
}

// openRepos opens each repository in a new pane next to the focused one.
func (s *Shell) openRepos(paths []string) tea.Cmd {
	var cmds []tea.Cmd
	for _, path := range paths {
		cmds = append(cmds, s.open())
		p := s.focused()
		cmds = append(cmds, wrap(p.id, p.app.openDir(path)))
	}
	return tea.Batch(cmds...)
}

func (s *Shell) close(p *pane) {
	for i, o := range s.panes {
		if o == p {
			s.panes = append(s.panes[:i], s.panes[i+1:]...)
			if s.focus >= len(s.panes) || s.focus > i {
				s.focus = max(0, s.focus-1)
			}
			break
		}
	}
	s.resize()
}

// sibling makes a new pane's App: same configuration, themes, scanner and
// clipboard, a copy of the repository list, everything else fresh.
func (a *App) sibling() *App {
	b := &App{ctx: a.ctx, cfg: a.cfg, st: a.st, scanner: a.scanner, sh: a.sh, cdFile: a.cdFile,
		themes: a.themes, theme: a.theme, dark: a.dark}
	b.rs = reposState{
		all: append(a.rs.all[:0:0], a.rs.all...), known: append(a.rs.known[:0:0], a.rs.known...),
		marked: map[string]bool{}, isNew: map[string]bool{}, collapsed: maps.Clone(a.rs.collapsed),
		sortMode: a.rs.sortMode, tree: a.rs.tree,
	}
	b.rebuildView("")
	return b
}

// visibleStatusCmd refreshes the visible rows while the list is idle.
func (a *App) visibleStatusCmd() tea.Cmd {
	if a.scr == scrRepos && !a.rs.loading && a.busy == "" && a.p == nil {
		return a.statusCmd(a.visiblePaths(), false)
	}
	return nil
}

// ------------------------------------------------------------------ layout

type rect struct{ x, y, w, h int }

// maxTiled is the most panes shown at once; more become tabs, since
// smaller panes stop being readable.
const maxTiled = 6

// layout places n panes in a w×h window in the given arrangement, falling
// back to the others when panes would be too small, and to tabs (only the
// focused pane, a tab bar above it) when nothing fits or there are more
// than maxTiled panes.
func layout(mode, n, w, h int) (rects []rect, tabs bool) {
	if n <= 1 {
		return []rect{{0, 0, w, h}}, false
	}
	if n > maxTiled {
		return []rect{{0, 1, w, h - 1}}, true
	}
	fits := func(rs []rect) bool {
		for _, r := range rs {
			if r.w-2 < minWidth || r.h-2 < minHeight {
				return false
			}
		}
		return true
	}
	// split cuts length into k parts that add up exactly.
	split := func(length, k int) (pos, size []int) {
		at := 0
		for i := 0; i < k; i++ {
			sz := (length - at) / (k - i)
			pos, size = append(pos, at), append(size, sz)
			at += sz
		}
		return pos, size
	}
	// table lays out rows of up to cols panes; a short last row stretches.
	table := func(cols int) []rect {
		nrows := (n + cols - 1) / cols
		ys, hs := split(h, nrows)
		var out []rect
		for r := 0; r < nrows; r++ {
			k := min(cols, n-r*cols)
			xs, ws := split(w, k)
			for c := 0; c < k; c++ {
				out = append(out, rect{xs[c], ys[r], ws[c], hs[r]})
			}
		}
		return out
	}
	gridCols := 1
	for gridCols*gridCols < n {
		gridCols++
	}
	grid, columns, rows := table(gridCols), table(n), table(1)
	tries := [][]rect{grid, columns, rows}
	switch mode {
	case paneColumns:
		tries = [][]rect{columns, grid, rows}
	case paneRows:
		tries = [][]rect{rows, grid, columns}
	}
	for _, t := range tries {
		if fits(t) {
			return t, false
		}
	}
	return []rect{{0, 1, w, h - 1}}, true
}

// shownLayout names the arrangement actually on screen, which can differ
// from the chosen one when panes would not fit.
func (s *Shell) shownLayout() string {
	rects, tabs := layout(s.mode, len(s.panes), s.w, s.h)
	switch {
	case tabs:
		return "tabs"
	case len(rects) < 2:
		return paneModeNames[s.mode]
	}
	rowsAt := map[int]bool{}
	for _, r := range rects {
		rowsAt[r.y] = true
	}
	switch {
	case len(rowsAt) == 1:
		return paneModeNames[paneColumns]
	case len(rowsAt) == len(rects):
		return paneModeNames[paneRows]
	}
	return paneModeNames[paneGrid]
}

// focusTowards moves the focus to the nearest pane in a direction (dx, dy
// in -1, 0, 1). With tabs, left/up and right/down go to the previous and
// next pane.
func (s *Shell) focusTowards(dx, dy int) {
	n := len(s.panes)
	if n < 2 {
		return
	}
	rects, tabs := layout(s.mode, n, s.w, s.h)
	if tabs || len(rects) != n {
		s.focus = (s.focus + n + dx + dy) % n
		return
	}
	cur := rects[s.focus]
	cx, cy := 2*cur.x+cur.w, 2*cur.y+cur.h // doubled centres stay integers
	best, bestDist := -1, 0
	for i, r := range rects {
		if i == s.focus {
			continue
		}
		ddx, ddy := 2*r.x+r.w-cx, 2*r.y+r.h-cy
		if (dx != 0 && ddx*dx <= 0) || (dy != 0 && ddy*dy <= 0) {
			continue // not in that direction
		}
		along, across := abs(ddx), abs(ddy)
		if dy != 0 {
			along, across = abs(ddy), abs(ddx)
		}
		if dist := along + 2*across; best < 0 || dist < bestDist { // prefer panes in line
			best, bestDist = i, dist
		}
	}
	if best >= 0 {
		s.focus = best
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (s *Shell) resize() {
	if s.w == 0 || s.h == 0 {
		return
	}
	rects, tabs := layout(s.mode, len(s.panes), s.w, s.h)
	for i, p := range s.panes {
		r := rects[0]
		if !tabs && i < len(rects) {
			r = rects[i]
		}
		iw, ih := r.w, r.h
		if len(s.panes) > 1 && !tabs {
			iw, ih = r.w-2, r.h-2
		}
		p.app.Update(tea.WindowSizeMsg{Width: iw, Height: ih})
	}
}

// title names what a pane shows: the open repository and branch, or the list.
func (p *pane) title(n int) string {
	a := p.app
	t := fmt.Sprintf("%d · ", n)
	switch a.scr {
	case scrRepo:
		return t + a.rp.name + " · " + a.rp.info.Branch
	case scrDash:
		return t + "dashboard"
	}
	return t + "repositories"
}

func (s *Shell) View() string {
	if s.w == 0 || s.h == 0 {
		return ""
	}
	return paintScreen(s.view(), s.w, s.h)
}

func (s *Shell) view() string {
	if len(s.panes) == 1 {
		return s.panes[0].app.View()
	}
	rects, tabs := layout(s.mode, len(s.panes), s.w, s.h)
	if tabs {
		titles := make([]string, len(s.panes))
		for i, p := range s.panes {
			titles[i] = p.title(i + 1)
		}
		return tabBar(titles, s.focus, s.w) + "\n" + s.focused().app.View()
	}
	// Render each pane framed, then lay the frames out row by row.
	type cell struct {
		r    rect
		view string
	}
	var cells []cell
	for i, p := range s.panes {
		cells = append(cells, cell{rects[i], frame(p.app.View(), rects[i].w, rects[i].h, p.title(i+1), i == s.focus)})
	}
	var rows []string
	for y := 0; y < s.h; {
		var line []string
		next := s.h
		for _, c := range cells {
			if c.r.y == y {
				line = append(line, c.view)
				next = min(next, y+c.r.h)
			}
		}
		if len(line) == 0 {
			break
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, line...))
		y = next
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// frame draws content inside a w×h rounded border with title in the top
// edge; the focused pane's border is in the accent colour.
func frame(content string, w, h int, title string, focused bool) string {
	inner := w - 2
	border := lipgloss.RoundedBorder()
	col := cBorder
	ts := lipgloss.NewStyle().Foreground(cDim)
	marker := " "
	if focused { // a heavy border and the title on the accent colour
		border = lipgloss.ThickBorder()
		col = cAccent
		ts = lipgloss.NewStyle().Bold(true).Foreground(cAccent).Reverse(true)
		marker = "●"
	}
	b := lipgloss.NewStyle().Foreground(col)
	label := " " + marker + " " + title + " "
	t := fit(label, max(0, min(inner-2, lipgloss.Width(label))))
	top := b.Render(border.TopLeft+border.Top) + ts.Render(t) +
		b.Render(strings.Repeat(border.Top, max(0, inner-1-lipgloss.Width(t)))+border.TopRight)
	lines := strings.Split(content, "\n")
	out := []string{top}
	for i := 0; i < h-2; i++ {
		l := ""
		if i < len(lines) {
			l = lines[i]
		}
		out = append(out, b.Render(border.Left)+fit(l, inner)+b.Render(border.Right))
	}
	out = append(out, b.Render(border.BottomLeft+strings.Repeat(border.Bottom, max(0, inner))+border.BottomRight))
	return strings.Join(out, "\n")
}

// maxTabWidth caps one tab's title so several fit on the bar.
const maxTabWidth = 30

// tabBar renders the pane tabs in one line of width w. When they don't all
// fit, it shows a run of tabs around the focused one, with "‹ 3" and "5 ›"
// counting the tabs hidden on each side.
func tabBar(titles []string, focus, w int) string {
	n := len(titles)
	labels := make([]string, n)
	widths := make([]int, n)
	for i, t := range titles {
		if lipgloss.Width(t) > maxTabWidth {
			t = strings.TrimRight(fit(t, maxTabWidth-1), " ") + "…"
		}
		labels[i] = " " + t + " "
		widths[i] = lipgloss.Width(labels[i]) + 1 // and a separator
	}
	hint := "  alt+←/→ switch"
	more := func(k int) int { // width of a "‹ k " / " k ›" marker
		if k == 0 {
			return 0
		}
		return len(fmt.Sprint(k)) + 3
	}
	// Grow the run [lo, hi] from the focused tab, alternating right and left,
	// while it and the markers for what is left out still fit.
	lo, hi := focus, focus
	used := widths[focus]
	fitsWith := func(lo, hi, used int) bool { return used+more(lo)+more(n-1-hi) <= w }
	for grew := true; grew; {
		grew = false
		if hi+1 < n && fitsWith(lo, hi+1, used+widths[hi+1]) {
			hi++
			used += widths[hi]
			grew = true
		}
		if lo > 0 && fitsWith(lo-1, hi, used+widths[lo-1]) {
			lo--
			used += widths[lo]
			grew = true
		}
	}
	var b strings.Builder
	if lo > 0 {
		b.WriteString(fg(cAccent, fmt.Sprintf("‹ %d ", lo)))
	}
	for i := lo; i <= hi; i++ {
		if i > lo {
			b.WriteString(fg(cDim, "│"))
		}
		if i == focus {
			b.WriteString(lipgloss.NewStyle().Bold(true).Background(cSel).Foreground(cText).Render(labels[i]))
		} else {
			b.WriteString(fg(cDim, labels[i]))
		}
	}
	if hi < n-1 {
		b.WriteString(fg(cAccent, fmt.Sprintf(" %d ›", n-1-hi)))
	}
	if lo == 0 && hi == n-1 && used+lipgloss.Width(hint) <= w {
		b.WriteString(fg(cDim, hint))
	}
	return fit(b.String(), w)
}
