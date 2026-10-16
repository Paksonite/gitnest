package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/config"
	"github.com/Paksonite/gitnest/internal/gitx"
	"github.com/Paksonite/gitnest/internal/scan"
)

type bulkFail struct {
	path string
	err  error
}

// bulkMsg is the outcome of an operation run on many repositories.
type bulkMsg struct {
	done  string // past tense, e.g. "pulled"
	total int
	fails []bulkFail
}

// report is a scrollable overlay listing the repositories an operation
// failed on, with git's full error output.
type report struct {
	title string
	lines []string
	off   int
}

// bulk runs fn on every path in parallel; failures open a report.
func (a *App) bulk(verb, done string, paths []string, fn func(path string) error) tea.Cmd {
	if len(paths) == 0 {
		a.setToast("nothing to "+verb, false)
		return nil
	}
	a.busy = fmt.Sprintf("%s %d repos", verb, len(paths))
	jobs := a.cfg.Jobs
	return func() tea.Msg {
		var mu sync.Mutex
		var fails []bulkFail
		scan.ForEach(len(paths), jobs, func(i int) {
			if err := fn(paths[i]); err != nil {
				mu.Lock()
				fails = append(fails, bulkFail{paths[i], err})
				mu.Unlock()
			}
		})
		sort.Slice(fails, func(i, j int) bool { return fails[i].path < fails[j].path })
		return bulkMsg{done: done, total: len(paths), fails: fails}
	}
}

func (a *App) onBulk(m bulkMsg) tea.Cmd {
	a.busy = ""
	if len(m.fails) == 0 {
		a.setToast(fmt.Sprintf("%s %d repos ✓", m.done, m.total), false)
	} else {
		a.setToast(fmt.Sprintf("%s %d of %d repos, %d failed", m.done, m.total-len(m.fails), m.total, len(m.fails)), true)
		a.report = newReport(m, a.rs.label)
	}
	return a.gather(a.paths())
}

func newReport(m bulkMsg, labels map[string]string) *report {
	r := &report{title: fmt.Sprintf("%d of %d failed", len(m.fails), m.total)}
	for i, f := range m.fails {
		if i > 0 {
			r.lines = append(r.lines, "")
		}
		name := filepath.Base(f.path)
		if l := labels[f.path]; l != "" {
			name += " · " + l
		}
		r.lines = append(r.lines, fg(cRed, "✗ ")+bold(name)+"  "+fg(cDim, config.Abbrev(f.path)))
		out := f.err.Error()
		var ge *gitx.Error
		if errors.As(f.err, &ge) && ge.Output != "" && ge.Timeout == 0 && !ge.Canceled {
			out = ge.Output
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) > 8 {
			lines = append(lines[:8], "…")
		}
		for _, l := range lines {
			r.lines = append(r.lines, "    "+fg(cDim, expandTabs(l)))
		}
	}
	return r
}

// errorReport shows everything git printed for a failed operation.
func errorReport(label string, err error) *report {
	r := &report{title: label + " failed"}
	out := err.Error()
	var ge *gitx.Error
	if errors.As(err, &ge) && ge.Output != "" && ge.Timeout == 0 && !ge.Canceled {
		out = "$ git " + strings.Join(ge.Args, " ") + "\n\n" + ge.Output
	}
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		r.lines = append(r.lines, expandTabs(l))
	}
	return r
}

func (a *App) reportKey(k tea.KeyMsg) tea.Cmd {
	r := a.report
	page := a.reportHeight()
	switch k.String() {
	case "esc", "q", "enter":
		a.report = nil
	case "j", "down":
		r.off++
	case "k", "up":
		r.off--
	case "pgdown", "ctrl+d", " ":
		r.off += page
	case "pgup", "ctrl+u":
		r.off -= page
	}
	if r := a.report; r != nil {
		r.off = clamp(r.off, 0, len(r.lines)-page)
	}
	return nil
}

func (a *App) reportHeight() int { return clamp(a.h-10, 3, 1<<30) }

func (a *App) viewReport() string {
	r := a.report
	w := clamp(a.w-8, 30, 120)
	h := a.reportHeight()
	r.off = clamp(r.off, 0, len(r.lines)-h)
	var body []string
	for i := r.off; i < len(r.lines) && i < r.off+h; i++ {
		body = append(body, fit(r.lines[i], w-4))
	}
	more := ""
	if len(r.lines) > h {
		more = fmt.Sprintf("  ·  %d–%d of %d lines", r.off+1, min(r.off+h, len(r.lines)), len(r.lines))
	}
	text := bold(r.title) + "\n\n" + strings.Join(body, "\n") + "\n\n" +
		fg(cDim, "j/k scroll · esc close"+more)
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cRed).
		Padding(1, 2).Width(w).Render(text)
	return lipgloss.Place(a.w, a.h, lipgloss.Center, lipgloss.Center, box)
}
