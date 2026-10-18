package ui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/v2/internal/gitx"
	"github.com/Paksonite/gitnest/v2/internal/scan"
)

// pullAll is "pull all" (ctrl+p in the list): every repository is fetched,
// then every local branch that is behind its upstream is listed — the
// checked-out one and others like main — to choose which to update. All
// updates are fast-forwards, so nothing local is ever overwritten.
type pullAll struct {
	rows    []pullRow
	cur     int
	fetched int        // repositories fetched
	fails   []bulkFail // repositories that could not be fetched
}

type pullRow struct {
	repo, name string // repository path and the name shown (with its clone label)
	branch     gitx.Branch
	behind     int
	note       string // why it is left out, or something worth knowing
	ok         bool   // can be fast-forwarded
	on         bool   // selected
}

// pullScanMsg carries the branches found behind after fetching everything.
type pullScanMsg struct {
	rows    []pullRow
	fetched int
	fails   []bulkFail
}

// startPullAll fetches every repository and collects the branches that
// are behind, for the chooser.
func (a *App) startPullAll() tea.Cmd {
	paths := a.paths()
	if len(paths) == 0 {
		a.setToast("no repositories", false)
		return nil
	}
	a.busy = fmt.Sprintf("fetching %d repos to see what can be pulled", len(paths))
	ctx, jobs := a.ctx, a.cfg.Jobs
	labels := make(map[string]string, len(a.rs.label))
	for p, l := range a.rs.label {
		labels[p] = l
	}
	return func() tea.Msg {
		var mu sync.Mutex
		var rows []pullRow
		var fails []bulkFail
		scan.ForEach(len(paths), jobs, func(i int) {
			p := paths[i]
			err := gitx.Fetch(ctx, p)
			found := behindRows(ctx, p, labels[p])
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				fails = append(fails, bulkFail{p, err})
			}
			rows = append(rows, found...)
		})
		sort.Slice(rows, func(i, j int) bool {
			x, y := rows[i], rows[j]
			if x.name != y.name {
				return strings.ToLower(x.name) < strings.ToLower(y.name)
			}
			if x.repo != y.repo {
				return x.repo < y.repo
			}
			if x.branch.Current != y.branch.Current {
				return x.branch.Current
			}
			return x.branch.Name < y.branch.Name
		})
		sort.Slice(fails, func(i, j int) bool { return fails[i].path < fails[j].path })
		return pullScanMsg{rows: rows, fetched: len(paths), fails: fails}
	}
}

// behindRows lists the local branches of one repository that are behind
// their upstream, and whether each can be fast-forwarded.
func behindRows(ctx context.Context, dir, label string) []pullRow {
	info, err := gitx.GetInfo(ctx, dir)
	if err != nil {
		return nil
	}
	local, _, err := gitx.Branches(ctx, dir)
	if err != nil {
		return nil
	}
	name := filepath.Base(dir)
	if label != "" {
		name += " · " + label
	}
	var rows []pullRow
	for _, b := range local {
		if b.Upstream == "" {
			continue
		}
		ahead, behind, _ := b.AheadBehind()
		if behind == 0 {
			continue
		}
		r := pullRow{repo: dir, name: name, branch: b, behind: behind, ok: true}
		switch {
		case ahead > 0:
			r.ok, r.note = false, fmt.Sprintf("diverged (↑%d): pull it inside the repository", ahead)
		case b.Current && info.State != gitx.StateNone:
			r.ok, r.note = false, strings.ToLower(info.State.String())+" in progress"
		case b.Current && info.Changes > 0:
			r.note = fmt.Sprintf("checked out, %d changed files", info.Changes)
		case b.Current:
			r.note = "checked out"
		}
		r.on = r.ok
		rows = append(rows, r)
	}
	return rows
}

func (a *App) onPullScan(m pullScanMsg) tea.Cmd {
	a.busy = ""
	if len(m.fails) > 0 {
		a.lastErr = newReport(bulkMsg{done: "fetched", total: m.fetched, fails: m.fails}, a.rs.label)
		a.lastErr.title = fmt.Sprintf("fetch failed in %d of %d repos", len(m.fails), m.fetched)
	}
	if len(m.rows) == 0 {
		msg, isErr := fmt.Sprintf("all %d repos are up to date", m.fetched), false
		if len(m.fails) > 0 {
			msg, isErr = fmt.Sprintf("nothing to pull; %d of %d repos could not be fetched  ·  ! details", len(m.fails), m.fetched), true
		}
		a.setToast(msg, isErr)
		return a.gather(a.paths())
	}
	a.pull = &pullAll{rows: m.rows, fetched: m.fetched, fails: m.fails}
	return a.gather(a.paths()) // the fetch changed ahead/behind everywhere
}

// selected counts the branches and repositories that would be pulled.
func (p *pullAll) selected() (branches, repos int) {
	seen := map[string]bool{}
	for _, r := range p.rows {
		if r.on {
			branches++
			if !seen[r.repo] {
				seen[r.repo] = true
				repos++
			}
		}
	}
	return branches, repos
}

func (a *App) pullAllKey(k tea.KeyMsg) tea.Cmd {
	p := a.pull
	switch lookup(scPullAll, k.String()) {
	case actUp:
		p.cur = clamp(p.cur-1, 0, len(p.rows)-1)
	case actDown:
		p.cur = clamp(p.cur+1, 0, len(p.rows)-1)
	case actTogglePull:
		if r := &p.rows[p.cur]; r.ok {
			r.on = !r.on
		} else {
			a.setToast(r.name+" "+r.branch.Name+": "+r.note, true)
		}
		p.cur = clamp(p.cur+1, 0, len(p.rows)-1)
	case actToggleAllPulls:
		branches, _ := p.selected()
		for i := range p.rows {
			p.rows[i].on = p.rows[i].ok && branches == 0
		}
	case actCancel:
		a.pull = nil
	case actSave:
		branches, _ := p.selected()
		if branches == 0 {
			a.setToast("nothing selected — space picks a branch, a picks all", false)
			return nil
		}
		a.pull = nil
		return a.runPullAll(p.rows)
	}
	return nil
}

// runPullAll fast-forwards the selected branches: the checked-out one with
// pull --ff-only, the others without checking them out. Repositories run
// in parallel, the branches of one repository one after another.
func (a *App) runPullAll(rows []pullRow) tea.Cmd {
	byRepo := map[string][]gitx.Branch{}
	var paths []string
	for _, r := range rows {
		if !r.on {
			continue
		}
		if _, ok := byRepo[r.repo]; !ok {
			paths = append(paths, r.repo)
		}
		byRepo[r.repo] = append(byRepo[r.repo], r.branch)
	}
	ctx := a.ctx
	return a.bulk("pull", "pulled", paths, func(p string) error {
		var errs []error
		for _, b := range byRepo[p] {
			if err := gitx.FastForward(ctx, p, b); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", b.Name, err))
			}
		}
		return errors.Join(errs...)
	})
}

func (a *App) viewPullAll() string {
	p := a.pull
	w := min(100, a.w-2)
	inner := w - 6
	listH := clamp(a.h-11-2*editorPad(a.h), 1, len(p.rows))
	off := centerOff(p.cur, len(p.rows), listH)
	var lines []string
	for i := off; i < len(p.rows) && i < off+listH; i++ {
		r := p.rows[i]
		box := "[ ]"
		if r.on {
			box = "[x]"
		}
		if !r.ok {
			box = " – "
		}
		behind := fmt.Sprintf("↓%d", r.behind)
		plain := fmt.Sprintf("%s %s  %s %s  %s", box, r.name, r.branch.Name, behind, r.note)
		if i == p.cur {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Background(cSel).Foreground(cText).Render(fit(plain, inner)))
			continue
		}
		styled := fg(cGreen, box) + " " + bold(r.name) + "  " + fg(cCyan, r.branch.Name) + " " + fg(cRed, behind) + "  " + fg(cDim, r.note)
		if !r.ok {
			styled = fg(cDim, plain)
		}
		lines = append(lines, fit(styled, inner))
	}
	branches, repos := p.selected()
	title := bold("Pull all") + fg(cDim, fmt.Sprintf("  %d repos fetched, %d branches behind", p.fetched, len(p.rows)))
	what := fmt.Sprintf("enter updates the %d selected branches in %d repos (fast-forward only: nothing local is overwritten)", branches, repos)
	if len(p.fails) > 0 {
		what += fg(cRed, fmt.Sprintf("  ·  %d repos could not be fetched (! details)", len(p.fails)))
	}
	body := title + "\n" + fit(what, inner) + "\n\n" + strings.Join(lines, "\n") + "\n\n" + fg(cDim, hints(scPullAll))
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cAccent).
		Padding(editorPad(a.h), 2).Width(w).Render(body)
	return lipgloss.Place(a.w, a.h, lipgloss.Center, lipgloss.Center, box)
}
