package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/config"
	"github.com/Paksonite/gitnest/internal/gitx"
	"github.com/Paksonite/gitnest/internal/scan"
	"github.com/Paksonite/gitnest/internal/state"
)

type reposState struct {
	all         []scan.Repo
	view        []int // indexes into all, after filtering
	cur, off    int
	filter      string
	filtering   bool
	dirtyOnly   bool
	loading     bool
	rescanned   bool
	marked      map[string]bool
	preview     string
	previewPath string
}

// loadRepos reads the cached repo list (or scans) and gathers status in the background.
func (a *App) loadRepos(forceScan bool) tea.Cmd {
	ctx, cfg, recent := a.ctx, a.cfg, a.st.Snapshot()
	a.rs.loading = true
	return func() tea.Msg {
		var paths []string
		fromCache := false
		if !forceScan {
			paths = state.LoadCache()
			fromCache = len(paths) > 0
		}
		scanned := false
		if len(paths) == 0 {
			paths = scan.Find(cfg.Roots, cfg.Depth, cfg.Exclude, cfg.SkipHidden)
			_ = state.SaveCache(paths)
			scanned = true
		}
		return reposMsg{repos: scan.Gather(ctx, paths, cfg.Jobs, recent), scanned: scanned, fromCache: fromCache}
	}
}

func (a *App) onRepos(m reposMsg) tea.Cmd {
	rs := &a.rs
	rs.loading = false
	keep := a.selectedPath()
	rs.all = m.repos
	a.rebuildView(keep)
	var cmds []tea.Cmd
	if m.fromCache && !rs.rescanned {
		// show the cached list instantly, then refresh it from disk once
		rs.rescanned = true
		cmds = append(cmds, a.loadRepos(true))
	}
	if m.scanned {
		a.setToast(fmt.Sprintf("%d repositories found", len(m.repos)), false)
	}
	cmds = append(cmds, a.previewCmd())
	return tea.Batch(cmds...)
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
		return a.loadRepos(true)
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

func repoRow(r scan.Repo, marked bool, nameW int) rowItem {
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
	br := fit(r.Branch, 16)
	sync := fit(r.Info.Sync(), 11)
	plain := sel + dirty + " " + name + " " + br + " " + sync + "  " + rel
	styled := fg(cGreen, sel) + fg(cYellow, dirty) + " " + bold(name) + " " + fg(cCyan, br) + " " +
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
		items[i] = repoRow(r, rs.marked[r.Path], nameW)
	}
	lh := bodyH - 3
	rs.off = scrollTo(rs.cur, rs.off, lh)
	lines := renderRows(items, rs.cur, rs.off, lh, inner, true)
	if len(items) == 0 {
		msg := "No repositories match."
		if rs.loading {
			msg = "Scanning…"
		} else if len(rs.all) == 0 {
			msg = "No repositories found — set roots in " + config.Path()
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
