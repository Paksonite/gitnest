package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/config"
	"github.com/Paksonite/gitnest/internal/gitx"
	"github.com/Paksonite/gitnest/internal/scan"
	"github.com/Paksonite/gitnest/internal/state"
)

type reposState struct {
	all         []scan.Repo
	known       []string // sorted repository paths from the last scan
	view        []int    // indexes into all, after filtering
	cur, off    int
	listH       int // rows of the list on screen (set while rendering)
	filter      string
	filtering   bool
	dirtyOnly   bool
	loading     bool // initial load or full status refresh running
	scanning    bool // background scan for new/removed repositories running
	marked      map[string]bool
	isNew       map[string]bool // found by a rescan, not opened yet
	preview     string
	previewPath string
}

type (
	scanMsg struct {
		paths []string
		full  bool
	}
	// statusMsg refreshes some repositories in place; asked lists the paths
	// whose status was requested (missing from repos = no longer a repo).
	statusMsg struct {
		repos []scan.Repo
		asked []string
		added bool
	}
	rescanTickMsg struct{}
	statusTickMsg struct{}
)

// loadRepos shows the repositories known from the scan cache right away and
// then rescans in the background; without a cache it scans first.
func (a *App) loadRepos() tea.Cmd {
	ctx, sc, jobs, recent := a.ctx, a.scanner, a.cfg.Jobs, a.st.Snapshot()
	a.rs.loading = true
	return func() tea.Msg {
		cached := sc.Load(state.ScanCachePath())
		paths := sc.Repos()
		if !cached {
			paths, _ = sc.Scan(false)
			_ = sc.Save(state.ScanCachePath())
		}
		return reposMsg{repos: scan.Gather(ctx, paths, jobs, recent), paths: paths, cached: cached}
	}
}

// scanCmd looks for new and removed repositories. Incremental scans only
// list directories whose mtime changed; full scans list everything.
func (a *App) scanCmd(full bool) tea.Cmd {
	if a.rs.scanning {
		return nil
	}
	a.rs.scanning = true
	sc := a.scanner
	return func() tea.Msg {
		paths, _ := sc.Scan(full)
		_ = sc.Save(state.ScanCachePath())
		return scanMsg{paths: paths, full: full}
	}
}

func (a *App) statusCmd(paths []string, added bool) tea.Cmd {
	if len(paths) == 0 {
		return nil
	}
	ctx, jobs, recent := a.ctx, a.cfg.Jobs, a.st.Snapshot()
	return func() tea.Msg {
		return statusMsg{repos: scan.Status(ctx, paths, jobs, recent), asked: paths, added: added}
	}
}

func (a *App) onRepos(m reposMsg) tea.Cmd {
	rs := &a.rs
	rs.loading = false
	keep := a.selectedPath()
	rs.all = m.repos
	if m.paths != nil {
		rs.known = m.paths
	}
	a.rebuildView(keep)
	var cmds []tea.Cmd
	if m.cached {
		cmds = append(cmds, a.scanCmd(false))
	} else if m.paths != nil {
		a.setToast(fmt.Sprintf("%d repositories found", len(m.repos)), false)
	}
	cmds = append(cmds, a.previewCmd())
	return tea.Batch(cmds...)
}

func (a *App) onScan(m scanMsg) tea.Cmd {
	rs := &a.rs
	rs.scanning = false
	added, removed := scan.Diff(rs.known, m.paths)
	rs.known = m.paths
	if len(added) == 0 && len(removed) == 0 {
		if m.full {
			a.setToast(fmt.Sprintf("%d repositories, no changes", len(m.paths)), false)
		}
		return nil
	}
	if len(removed) > 0 {
		gone := map[string]bool{}
		for _, p := range removed {
			gone[p] = true
			delete(rs.marked, p)
			delete(rs.isNew, p)
		}
		keep := rs.all[:0]
		for _, r := range rs.all {
			if !gone[r.Path] {
				keep = append(keep, r)
			}
		}
		rs.all = keep
		a.rebuildView(a.selectedPath())
	}
	for _, p := range added {
		rs.isNew[p] = true
	}
	a.setToast(changeSummary(added, removed), false)
	return tea.Batch(a.statusCmd(added, true), a.previewCmd())
}

func changeSummary(added, removed []string) string {
	var parts []string
	if n := len(added); n > 0 {
		names := make([]string, 0, 3)
		for _, p := range added {
			if len(names) == 3 {
				names = append(names, "…")
				break
			}
			names = append(names, filepath.Base(p))
		}
		parts = append(parts, fmt.Sprintf("+%d new: %s", n, strings.Join(names, ", ")))
	}
	if n := len(removed); n > 0 {
		parts = append(parts, fmt.Sprintf("−%d removed", n))
	}
	return strings.Join(parts, "  ·  ")
}

// onStatus merges refreshed repositories into the list without reordering
// it, so rows don't jump under the cursor; new repositories are sorted in.
func (a *App) onStatus(m statusMsg) tea.Cmd {
	rs := &a.rs
	keep := a.selectedPath()
	got := map[string]scan.Repo{}
	for _, r := range m.repos {
		got[r.Path] = r
	}
	asked := map[string]bool{}
	for _, p := range m.asked {
		asked[p] = true
	}
	out := rs.all[:0]
	for _, r := range rs.all {
		if nr, ok := got[r.Path]; ok {
			out = append(out, nr)
			delete(got, r.Path)
		} else if !asked[r.Path] {
			out = append(out, r)
		}
	}
	for _, p := range m.asked { // newly found repositories, in scan order
		if r, ok := got[p]; ok {
			out = append(out, r)
		}
	}
	rs.all = out
	if m.added {
		scan.Sort(rs.all)
	}
	a.rebuildView(keep)
	return nil
}

// visiblePaths are the repositories currently on screen.
func (a *App) visiblePaths() []string {
	rs := &a.rs
	var ps []string
	for i := rs.off; i < len(rs.view) && i < rs.off+rs.listH; i++ {
		ps = append(ps, rs.all[rs.view[i]].Path)
	}
	return ps
}

func (a *App) rescanTick() tea.Cmd {
	if a.cfg.RescanInterval <= 0 {
		return nil
	}
	return tea.Tick(time.Duration(a.cfg.RescanInterval)*time.Second, func(time.Time) tea.Msg { return rescanTickMsg{} })
}

func (a *App) statusTick() tea.Cmd {
	if a.cfg.StatusInterval <= 0 {
		return nil
	}
	return tea.Tick(time.Duration(a.cfg.StatusInterval)*time.Second, func(time.Time) tea.Msg { return statusTickMsg{} })
}

func (a *App) onRescanTick() tea.Cmd {
	return tea.Batch(a.scanCmd(false), a.rescanTick())
}

// onStatusTick refreshes the visible rows while the list is on screen and idle.
func (a *App) onStatusTick() tea.Cmd {
	var cmd tea.Cmd
	if a.scr == scrRepos && !a.rs.loading && a.busy == "" && a.p == nil {
		cmd = a.statusCmd(a.visiblePaths(), false)
	}
	return tea.Batch(cmd, a.statusTick())
}

func (a *App) rebuildView(keep string) {
	rs := &a.rs
	rs.view = rs.view[:0]
	for i, r := range rs.all {
		if rs.dirtyOnly && r.Changes == 0 {
			continue
		}
		hay := r.Name + " " + r.Branch + " " + filepath.Base(filepath.Dir(r.Path))
		if rs.filter != "" && !fuzzy(hay, rs.filter) {
			continue
		}
		rs.view = append(rs.view, i)
	}
	rs.cur = clamp(rs.cur, 0, len(rs.view)-1)
	if keep != "" {
		for i, idx := range rs.view {
			if rs.all[idx].Path == keep {
				rs.cur = i
				break
			}
		}
	}
}

func (a *App) selectedPath() string {
	rs := &a.rs
	if rs.cur >= 0 && rs.cur < len(rs.view) {
		return rs.all[rs.view[rs.cur]].Path
	}
	return ""
}

func (a *App) previewCmd() tea.Cmd {
	p := a.selectedPath()
	if p == "" {
		a.rs.preview, a.rs.previewPath = "", ""
		return nil
	}
	if p != a.rs.previewPath {
		a.rs.preview = ""
	}
	a.rs.previewPath = p
	return func() tea.Msg { return previewMsg{path: p, text: gitx.Preview(a.ctx, p)} }
}

func (a *App) openDir(path string) tea.Cmd {
	a.st.Touch(path)
	delete(a.rs.isNew, path)
	a.scr = scrRepo
	a.rp = repoState{dir: path, name: filepath.Base(path)}
	return loadRepoData(a.ctx, path)
}

func (a *App) moveRepos(delta int) tea.Cmd {
	rs := &a.rs
	rs.cur = clamp(rs.cur+delta, 0, len(rs.view)-1)
	return a.previewCmd()
}

// targets returns marked repos, or fallback() when nothing is marked.
func (a *App) targets(fallback func() []string) []string {
	var ps []string
	for _, r := range a.rs.all {
		if a.rs.marked[r.Path] {
			ps = append(ps, r.Path)
		}
	}
	if len(ps) == 0 {
		return fallback()
	}
	return ps
}

func (a *App) reposKey(k tea.KeyMsg) tea.Cmd {
	rs := &a.rs
	ctx := a.ctx
	s := k.String()

	if rs.filtering {
		refilter := func() tea.Cmd {
			a.rebuildView("")
			rs.cur = 0
			return a.previewCmd()
		}
		switch k.Type {
		case tea.KeyEsc:
			rs.filtering, rs.filter = false, ""
			return refilter()
		case tea.KeyEnter:
			rs.filtering = false
			return nil
		case tea.KeyBackspace:
			if r := []rune(rs.filter); len(r) > 0 {
				rs.filter = string(r[:len(r)-1])
			}
			return refilter()
		case tea.KeyRunes:
			rs.filter += string(k.Runes)
			return refilter()
		case tea.KeySpace:
			rs.filter += " "
			return refilter()
		}
	}

	switch s {
	case "up":
		return a.moveRepos(-1)
	case "down":
		return a.moveRepos(1)
	case "pgup":
		return a.moveRepos(-10)
	case "pgdown":
		return a.moveRepos(10)
	case "home":
		return a.moveRepos(-len(rs.view))
	case "end":
		return a.moveRepos(len(rs.view))
	}
	if rs.filtering {
		return nil
	}

	switch s {
	case "q":
		return tea.Quit
	case "esc":
		if rs.filter != "" {
			rs.filter = ""
			a.rebuildView(a.selectedPath())
			return a.previewCmd()
		}
	case "k":
		return a.moveRepos(-1)
	case "j":
		return a.moveRepos(1)
	case "/":
		rs.filtering = true
	case "enter":
		if p := a.selectedPath(); p != "" {
			return a.openDir(p)
		}
	case "d":
		rs.dirtyOnly = !rs.dirtyOnly
		a.rebuildView(a.selectedPath())
		return a.previewCmd()
	case " ":
		if p := a.selectedPath(); p != "" {
			if rs.marked[p] {
				delete(rs.marked, p)
			} else {
				rs.marked[p] = true
			}
		}
		return a.moveRepos(1)
	case "R":
		a.setToast("rescanning all roots…", false)
		return a.scanCmd(true)
	case "r":
		return a.gather(a.paths())
	case "f":
		ps := a.targets(a.paths)
		jobs := a.cfg.Jobs
		return a.doMsg(fmt.Sprintf("fetching %d repos", len(ps)), func() (string, error) {
			failed := scan.RunAll(ps, jobs, func(p string) error { return gitx.Fetch(ctx, p) })
			return fmt.Sprintf("fetched %d repos, %d failed", len(ps), failed), nil
		})
	case "p":
		ps := a.targets(func() []string {
			if p := a.selectedPath(); p != "" {
				return []string{p}
			}
			return nil
		})
		jobs := a.cfg.Jobs
		return a.doMsg(fmt.Sprintf("pulling %d repos", len(ps)), func() (string, error) {
			failed := scan.RunAll(ps, jobs, func(p string) error { return gitx.Pull(ctx, p, false) })
			return fmt.Sprintf("pulled %d repos, %d failed", len(ps), failed), nil
		})
	}
	return nil
}

// ---------------------------------------------------------------------- view

func syncStyled(i gitx.Info) string {
	var parts []string
	if i.Upstream {
		if i.Ahead > 0 {
			parts = append(parts, fg(cGreen, fmt.Sprintf("↑%d", i.Ahead)))
		}
		if i.Behind > 0 {
			parts = append(parts, fg(cRed, fmt.Sprintf("↓%d", i.Behind)))
		}
	} else if !i.Detached {
		parts = append(parts, fg(cDim, "∅"))
	}
	if i.Changes > 0 {
		parts = append(parts, fg(cYellow, fmt.Sprintf("~%d", i.Changes)))
	}
	return strings.Join(parts, " ")
}

func repoRow(r scan.Repo, marked, isNew bool, nameW int) rowItem {
	sel, dirty := " ", " "
	if marked {
		sel = "✓"
	}
	if r.Changes > 0 {
		dirty = "●"
	}
	rel := r.LastRel
	if rel == "" {
		rel = "no commits"
	}
	name := fit(r.Name, nameW)
	styledName := bold(name)
	if isNew {
		name = fit(r.Name+" ✦", nameW)
		styledName = fit(bold(r.Name)+fg(cAccent, " ✦"), nameW)
	}
	br := fit(r.Branch, 16)
	sync := fit(r.Info.Sync(), 11)
	plain := sel + dirty + " " + name + " " + br + " " + sync + "  " + rel
	styled := fg(cGreen, sel) + fg(cYellow, dirty) + " " + styledName + " " + fg(cCyan, br) + " " +
		fit(syncStyled(r.Info), 11) + "  " + fg(cDim, rel)
	return rowItem{plain: plain, styled: styled}
}

func (a *App) viewRepos() string {
	rs := &a.rs
	bodyH := a.h - 3
	dirty := 0
	for _, r := range rs.all {
		if r.Changes > 0 {
			dirty++
		}
	}

	head := lipgloss.NewStyle().Bold(true).Foreground(cAccent).Render("gitnest") +
		fg(cDim, fmt.Sprintf("  %d repos · %d dirty", len(rs.all), dirty))
	if rs.dirtyOnly {
		head += "  " + fg(cYellow, "[dirty only]")
	}
	if rs.filtering || rs.filter != "" {
		cursor := ""
		if rs.filtering {
			cursor = "▏"
		}
		head += "  " + fg(cAccent, "/"+rs.filter+cursor)
	}
	if rs.loading {
		head += "  " + fg(cYellow, "⟳ scanning…")
	}

	listW := a.w
	if a.w >= 90 {
		listW = a.w * 11 / 20
	}
	inner := listW - 2
	nameW := clamp(inner-48, 10, 40)
	items := make([]rowItem, len(rs.view))
	for i, idx := range rs.view {
		r := rs.all[idx]
		items[i] = repoRow(r, rs.marked[r.Path], rs.isNew[r.Path], nameW)
	}
	lh := bodyH - 3
	rs.listH = lh
	rs.off = scrollTo(rs.cur, rs.off, lh)
	lines := renderRows(items, rs.cur, rs.off, lh, inner, true)
	if len(items) == 0 {
		msg := "No repositories match."
		if rs.loading {
			msg = "Scanning…"
		} else if len(a.cfg.Roots) == 0 {
			msg = "No directories to scan — run: gitnest roots add ~/repos"
		} else if len(rs.all) == 0 {
			msg = "No repositories found under " + rootsList(a.cfg.Roots)
		}
		lines = []string{fg(cDim, msg)}
	}
	list := panel(fmt.Sprintf("Repositories (%d)", len(rs.view)), lines, listW, bodyH, true)

	body := list
	if a.w >= 90 {
		var pl []string
		if rs.preview != "" {
			pl = strings.Split(strings.TrimRight(expandTabs(rs.preview), "\n"), "\n")
		}
		title := "Preview"
		if p := a.selectedPath(); p != "" {
			title = "Preview · " + filepath.Base(p)
		}
		body = lipgloss.JoinHorizontal(lipgloss.Top, list, panel(title, pl, a.w-listW, bodyH, false))
	}

	hints := "enter open  / search  space mark  d dirty only  f fetch  p pull  R rescan  r refresh  ? help  q quit"
	if rs.filtering {
		hints = "type to filter  ·  ↑/↓ move  ·  enter keep filter  ·  esc clear"
	}
	return strings.Join([]string{fit(head, a.w), body, a.footer(hints)}, "\n")
}

func rootsList(roots []string) string {
	out := make([]string, len(roots))
	for i, r := range roots {
		out[i] = config.Abbrev(r)
	}
	return strings.Join(out, ", ")
}
