package ui

import (
	"fmt"
	"reflect"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Shell holds up to maxPanes independent gitnests ("panes") side by side in
// one terminal. Every pane is a full App with its own screen, selection,
// open repository, prompts and editor; the config, themes, scanner and the
// commit clipboard are shared. ctrl+n opens a pane, ctrl+w closes one,
// ctrl+o (or alt+→ / alt+←) moves the focus.
type Shell struct {
	panes  []*pane
	focus  int
	nextID int
	w, h   int
}

type pane struct {
	id  int
	app *App
}

const maxPanes = 4

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
)

func newShell(first *App) *Shell {
	return &Shell{panes: []*pane{{id: 0, app: first}}, nextID: 1}
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
		return paneMsg{id: id, msg: msg}
	}
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
	return tea.Batch(wrap(s.panes[0].id, a.startCmds()), shellRescanTick(a), shellStatusTick(a))
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

func (s *Shell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		s.w, s.h = m.Width, m.Height
		s.resize()
		return s, nil
	case tea.KeyMsg:
		return s, s.onKey(m)
	case tea.FocusMsg, tea.BlurMsg:
		return s, s.deliver(s.focused(), msg)
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
		case actNextPane:
			s.focus = (s.focus + 1) % len(s.panes)
			return nil
		case actPrevPane:
			s.focus = (s.focus + len(s.panes) - 1) % len(s.panes)
			return nil
		}
	}
	return s.deliver(f, k)
}

// open adds a pane next to the focused one, on the repository list.
func (s *Shell) open() tea.Cmd {
	src := s.focused().app
	if len(s.panes) >= maxPanes {
		src.setToast(fmt.Sprintf("at most %d panes", maxPanes), true)
		return nil
	}
	a := src.sibling()
	p := &pane{id: s.nextID, app: a}
	s.nextID++
	s.panes = append(s.panes[:s.focus+1], append([]*pane{p}, s.panes[s.focus+1:]...)...)
	s.focus++
	s.resize()
	a.setToast("new pane: pick a repository (ctrl+w closes it, ctrl+o switches)", false)
	return wrap(p.id, tea.Batch(a.previewCmd(), repoTick()))
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
		marked: map[string]bool{}, isNew: map[string]bool{}, collapsed: map[string]bool{},
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

// layout places n panes in a w×h window: side by side when they fit, a 2×2
// grid (the third pane spanning the bottom row) otherwise, and when even
// that would be too small, tabs (only the focused pane, a tab bar above).
func layout(n, w, h int) (rects []rect, tabs bool) {
	fits := func(rs []rect) bool {
		for _, r := range rs {
			if r.w-2 < minWidth || r.h-2 < minHeight {
				return false
			}
		}
		return true
	}
	cols := func(k, y, height int) []rect {
		var out []rect
		x := 0
		for i := 0; i < k; i++ {
			cw := (w - x) / (k - i)
			out = append(out, rect{x, y, cw, height})
			x += cw
		}
		return out
	}
	if n <= 1 {
		return []rect{{0, 0, w, h}}, false
	}
	var tries [][]rect
	tries = append(tries, cols(n, 0, h)) // all side by side
	if n == 2 {
		tries = append(tries, []rect{{0, 0, w, h / 2}, {0, h / 2, w, h - h/2}})
	}
	if n >= 3 {
		top := cols(2, 0, h/2)
		bottom := cols(n-2, h/2, h-h/2)
		tries = append(tries, append(top, bottom...))
	}
	for _, t := range tries {
		if fits(t) {
			return t, false
		}
	}
	return []rect{{0, 1, w, h - 1}}, true
}

func (s *Shell) resize() {
	if s.w == 0 || s.h == 0 {
		return
	}
	rects, tabs := layout(len(s.panes), s.w, s.h)
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
	if a.scr == scrRepo {
		return t + a.rp.name + " · " + a.rp.info.Branch
	}
	return t + "repositories"
}

func (s *Shell) View() string {
	if s.w == 0 || s.h == 0 {
		return ""
	}
	if len(s.panes) == 1 {
		return s.panes[0].app.View()
	}
	rects, tabs := layout(len(s.panes), s.w, s.h)
	if tabs {
		var parts []string
		for i, p := range s.panes {
			t := " " + p.title(i+1) + " "
			if i == s.focus {
				parts = append(parts, lipgloss.NewStyle().Bold(true).Background(cSel).Foreground(cText).Render(t))
			} else {
				parts = append(parts, fg(cDim, t))
			}
		}
		bar := fit(strings.Join(parts, fg(cDim, "│"))+fg(cDim, "  ctrl+o switches"), s.w)
		return bar + "\n" + s.focused().app.View()
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
	col := cBorder
	ts := lipgloss.NewStyle().Foreground(cDim)
	if focused {
		col = cAccent
		ts = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	}
	b := lipgloss.NewStyle().Foreground(col)
	inner := w - 2
	t := fit(" "+title+" ", max(0, min(inner-2, lipgloss.Width(" "+title+" "))))
	top := b.Render("╭─") + ts.Render(t) + b.Render(strings.Repeat("─", max(0, inner-1-lipgloss.Width(t)))+"╮")
	lines := strings.Split(content, "\n")
	out := []string{top}
	for i := 0; i < h-2; i++ {
		l := ""
		if i < len(lines) {
			l = lines[i]
		}
		out = append(out, b.Render("│")+fit(l, inner)+b.Render("│"))
	}
	out = append(out, b.Render("╰"+strings.Repeat("─", max(0, inner))+"╯"))
	return strings.Join(out, "\n")
}
