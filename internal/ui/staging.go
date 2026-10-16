package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/gitx"
)

// staging is the line-by-line staging view of one file (enter on a file in
// the Files panel), like lazygit's: the unstaged or the staged changes,
// with a cursor on change lines; space moves the selected line, range (v)
// or hunk (a) to the other side.
type staging struct {
	path     string
	staged   bool // showing the staged changes
	diff     gitx.Diff
	rows     []stRow
	cur      int  // row under the cursor (always a change line when there is one)
	anchor   int  // start of a range selection, -1 when none
	hunkMode bool // selection is the whole hunk
	off      int
}

type stRow struct {
	text   string
	ref    gitx.LineRef
	header bool
	change bool
}

type stagingMsg struct {
	path   string
	staged bool
	diff   gitx.Diff
	err    error
}

// openStaging shows f's changes: unstaged ones when there are any, else staged.
func (a *App) openStaging(f gitx.File) tea.Cmd {
	return a.loadStaging(f.Path, !f.Unstaged() && f.Staged(), f.Untracked())
}

func (a *App) loadStaging(path string, staged, untracked bool) tea.Cmd {
	ctx, dir := a.ctx, a.rp.dir
	return func() tea.Msg {
		if untracked {
			if err := gitx.IntentToAdd(ctx, dir, path); err != nil {
				return stagingMsg{path: path, err: err}
			}
		}
		d, err := gitx.FileDiff(ctx, dir, path, staged)
		return stagingMsg{path: path, staged: staged, diff: d, err: err}
	}
}

func (a *App) onStaging(m stagingMsg) tea.Cmd {
	if m.err != nil {
		a.setToast(m.err.Error(), true)
		return nil
	}
	old := a.rp.staging
	st := &staging{path: m.path, staged: m.staged, diff: m.diff, anchor: -1}
	for hi, h := range m.diff.Hunks {
		st.rows = append(st.rows, stRow{text: fmt.Sprintf("@@ -%d,%d +%d,%d @@%s", h.OldStart, h.OldLines, h.NewStart, h.NewLines, h.Section), ref: gitx.LineRef{Hunk: hi, Line: -1}, header: true})
		for li, l := range h.Lines {
			st.rows = append(st.rows, stRow{text: l, ref: gitx.LineRef{Hunk: hi, Line: li}, change: gitx.IsChange(l)})
		}
	}
	if len(st.rows) == 0 {
		// Nothing left on this side: show the other one, or leave.
		if old != nil && old.path == m.path && old.staged == m.staged {
			if f, ok := a.fileByPath(m.path); ok && (f.Staged() || f.Unstaged()) {
				return a.loadStaging(m.path, !m.staged, false)
			}
		}
		a.rp.staging = nil
		a.setToast("no more changes in "+m.path, false)
		return nil
	}
	if old != nil && old.path == m.path && old.staged == m.staged {
		st.cur, st.hunkMode, st.off = old.cur, old.hunkMode, old.off // stay where we were
	}
	st.cur = clamp(st.cur, 0, len(st.rows)-1)
	st.seek(0, 1)
	a.rp.staging = st
	return nil
}

func (a *App) fileByPath(path string) (gitx.File, bool) {
	for _, f := range a.rp.files {
		if f.Path == path {
			return f, true
		}
	}
	return gitx.File{}, false
}

// seek moves the cursor by delta change lines (or hunks in hunk mode); with
// delta 0 it settles on the nearest change line in direction dir.
func (st *staging) seek(delta, dir int) {
	isTarget := func(i int) bool {
		if !st.rows[i].change {
			return false
		}
		if !st.hunkMode {
			return true
		}
		return i == 0 || !st.rows[i-1].change || st.rows[i-1].ref.Hunk != st.rows[i].ref.Hunk ||
			firstChangeOfHunk(st, i)
	}
	if delta == 0 {
		for i := st.cur; i >= 0 && i < len(st.rows); i += dir {
			if isTarget(i) {
				st.cur = i
				return
			}
		}
		for i := st.cur; i >= 0 && i < len(st.rows); i -= dir {
			if isTarget(i) {
				st.cur = i
				return
			}
		}
		return
	}
	step := 1
	if delta < 0 {
		step, delta = -1, -delta
	}
	for i := st.cur + step; i >= 0 && i < len(st.rows) && delta > 0; i += step {
		if isTarget(i) {
			st.cur = i
			delta--
		}
	}
}

func firstChangeOfHunk(st *staging, i int) bool {
	h := st.rows[i].ref.Hunk
	for k := i - 1; k >= 0 && st.rows[k].ref.Hunk == h; k-- {
		if st.rows[k].change {
			return false
		}
	}
	return true
}

// selection is what space / d act on.
func (st *staging) selection() map[gitx.LineRef]bool {
	sel := map[gitx.LineRef]bool{}
	if len(st.rows) == 0 {
		return sel
	}
	lo, hi := st.cur, st.cur
	switch {
	case st.hunkMode:
		h := st.rows[st.cur].ref.Hunk
		lo, hi = len(st.rows), -1
		for i, r := range st.rows {
			if r.ref.Hunk == h {
				lo, hi = min(lo, i), max(hi, i)
			}
		}
	case st.anchor >= 0:
		lo, hi = min(st.anchor, st.cur), max(st.anchor, st.cur)
	}
	for i := lo; i <= hi; i++ {
		if st.rows[i].change {
			sel[st.rows[i].ref] = true
		}
	}
	return sel
}

func (a *App) stagingKey(s string) tea.Cmd {
	st := a.rp.staging
	dir, ctx := a.rp.dir, a.ctx
	half := clamp(a.rp.mainH/2, 1, 1<<30)
	reload := func() tea.Cmd { return tea.Batch(a.loadStaging(st.path, st.staged, false), loadRepoData(ctx, dir)) }
	switch lookup(scStaging, s) {
	case actBack:
		a.rp.staging = nil
		a.rp.mainKey = "" // show the plain diff again
		return a.mainCmd()
	case actDown:
		st.seek(1, 1)
	case actUp:
		st.seek(-1, -1)
	case actPageDown:
		st.cur = clamp(st.cur+half, 0, len(st.rows)-1)
		st.seek(0, 1)
	case actPageUp:
		st.cur = clamp(st.cur-half, 0, len(st.rows)-1)
		st.seek(0, -1)
	case actRange:
		if st.anchor >= 0 {
			st.anchor = -1
		} else {
			st.anchor, st.hunkMode = st.cur, false
		}
	case actHunkMode:
		st.hunkMode, st.anchor = !st.hunkMode, -1
		st.seek(0, -1)
	case actSwitchSide:
		return a.loadStaging(st.path, !st.staged, false)
	case actToggleStage:
		d, sel, staged := st.diff, st.selection(), st.staged
		st.anchor = -1
		label := "stage lines"
		if staged {
			label = "unstage lines"
		}
		return tea.Sequence(a.do(label, func() error {
			if staged {
				return gitx.UnstageLines(ctx, dir, d, sel)
			}
			return gitx.StageLines(ctx, dir, d, sel)
		}), reload())
	case actDiscard:
		if st.staged {
			a.setToast("discard works on unstaged changes — tab switches sides", true)
			return nil
		}
		d, sel := st.diff, st.selection()
		a.confirm(fmt.Sprintf("Discard %d selected line(s) in %s? This can't be undone.", len(sel), st.path), func() tea.Cmd {
			st.anchor = -1
			return tea.Sequence(a.do("discard lines", func() error { return gitx.DiscardLines(ctx, dir, d, sel) }), reload())
		})
	}
	return nil
}

// stagingLines renders the view into the main panel area.
func (a *App) stagingLines(h, w int) (title string, lines []string) {
	st := a.rp.staging
	side := "Unstaged"
	if st.staged {
		side = "Staged"
	}
	mode := "line"
	switch {
	case st.hunkMode:
		mode = "hunk"
	case st.anchor >= 0:
		mode = "range"
	}
	title = fmt.Sprintf("%s · %s · %s", side, st.path, mode)
	sel := st.selection()
	st.off = scrollTo(st.cur, st.off, h)
	selStyle := lipgloss.NewStyle().Background(cSelDim)
	curStyle := lipgloss.NewStyle().Background(cSel).Bold(true)
	for i := st.off; i < len(st.rows) && i < st.off+h; i++ {
		r := st.rows[i]
		text := expandTabs(r.text)
		var styled string
		switch {
		case r.header:
			styled = fg(cCyan, text)
		case strings.HasPrefix(text, "+"):
			styled = fg(cGreen, text)
		case strings.HasPrefix(text, "-"):
			styled = fg(cRed, text)
		default:
			styled = fg(cDim, text)
		}
		switch {
		case i == st.cur:
			styled = curStyle.Render(fit(text, w))
		case sel[r.ref] && r.change:
			styled = selStyle.Render(fit(text, w))
		}
		lines = append(lines, styled)
	}
	return title, lines
}
