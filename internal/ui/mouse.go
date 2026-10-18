package ui

import (
	"os"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// Mouse: the wheel scrolls what is under the pointer, a click selects a row
// (and a click on the selected row opens it), and in panes a click focuses
// the pane. Coordinates reaching an App are relative to its own screen.

const wheelStep = 3

func (a *App) onMouse(m tea.MouseMsg) tea.Cmd {
	wheel := 0
	switch m.Button {
	case tea.MouseButtonWheelUp:
		wheel = -1
	case tea.MouseButtonWheelDown:
		wheel = 1
	}
	if wheel != 0 && m.Action == tea.MouseActionPress {
		a.sel.on = false
		return a.onWheel(m.X, m.Y, wheel)
	}
	if cmd, took := a.selMouse(m); took {
		return cmd
	}
	if m.Button != tea.MouseButtonLeft || m.Action != tea.MouseActionPress {
		return nil
	}
	if a.overlayOpen() {
		return nil // overlays: keys only (the wheel moves them)
	}
	if a.browse != nil {
		return a.clickBrowser(m.X, m.Y)
	}
	switch a.scr {
	case scrRepos:
		return a.clickList(m.X, m.Y)
	case scrDash:
		return a.clickDashboard(m.X, m.Y)
	case scrRepo:
		return a.clickRepo(m.X, m.Y)
	}
	return nil
}

// onWheel moves the selection, or scrolls the diff when the pointer is on it.
func (a *App) onWheel(x, y, dir int) tea.Cmd {
	if a.scr == scrRepo && a.browse == nil && !a.overlayOpen() && x >= a.rp.leftW && a.rp.leftW > 0 {
		rp := &a.rp
		rp.mainOff = clamp(rp.mainOff+dir*wheelStep, 0, max(0, rp.mainLen-rp.mainH))
		return nil
	}
	if b := a.browse; b != nil && !a.overlayOpen() {
		if x >= clamp(a.w*2/5, 24, 48) && b.path != "" { // the file side scrolls the file
			b.scroll = clamp(b.scroll+dir*wheelStep, 0, max(0, len(b.lines)-(a.h-6)))
			return nil
		}
	}
	key := tea.KeyMsg{Type: tea.KeyDown}
	if dir < 0 {
		key = tea.KeyMsg{Type: tea.KeyUp}
	}
	var cmds []tea.Cmd
	steps := wheelStep
	if a.overlayOpen() {
		steps = 1 // menus and choosers: one option per notch
	}
	for i := 0; i < steps; i++ {
		cmds = append(cmds, a.onKey(key))
	}
	return tea.Batch(cmds...)
}

func (a *App) overlayOpen() bool {
	return a.ed != nil || a.p != nil || a.report != nil || a.picker != nil || a.plan != nil || a.pull != nil || a.menu != nil
}

// clickList selects the row under the pointer; clicking the selected row
// opens it (a repository) or folds it (a group).
func (a *App) clickList(x, y int) tea.Cmd {
	rs := &a.rs
	if x >= rs.listW && rs.link.url != "" && y == rs.link.y {
		return a.openURL(rs.link.url)
	}
	const firstRow = 4 // header, border, panel title, column names
	if x >= rs.listW || y < firstRow || y >= firstRow+rs.listH {
		return nil
	}
	i := rs.off + y - firstRow
	if i < 0 || i >= len(rs.rows) {
		return nil
	}
	if i == rs.cur {
		return a.listAction(actOpen)
	}
	rs.cur = i
	return a.previewCmd()
}

func (a *App) clickDashboard(x, y int) tea.Cmd {
	d := &a.dash
	for _, h := range d.hits {
		if h.y == y && x >= h.x0 && x < h.x1 {
			if h.i == d.cur {
				return a.openDir(d.items[h.i])
			}
			d.cur = h.i
			return nil
		}
	}
	return nil
}

// clickRepo focuses the panel under the pointer and selects the row.
func (a *App) clickRepo(x, y int) tea.Cmd {
	rp := &a.rp
	if rp.staging != nil {
		return nil
	}
	if x >= rp.leftW {
		rp.mainFocus = true
		return nil
	}
	for i, top := range rp.panelTop {
		if top < 0 || y < top || y >= top+rp.panelH[i] {
			continue
		}
		rp.mainFocus = false
		changed := rp.focus != i
		rp.focus = i
		row := rp.off[i] + y - top - 2 // border and title
		if row >= 0 && row < rp.count(i) {
			rp.cur[i] = row
			changed = true
		}
		if changed {
			return a.mainCmd()
		}
		return nil
	}
	return nil
}

func (a *App) clickBrowser(x, y int) tea.Cmd {
	b := a.browse
	treeW := clamp(a.w*2/5, 24, 48)
	if a.w < 70 {
		treeW = a.w / 2
	}
	if x >= treeW {
		b.focusFile = b.path != ""
		return nil
	}
	b.focusFile = false
	i := b.off + y - 3 // header, border, title
	if i < 0 || i >= len(b.rows) {
		return nil
	}
	if b.rows[i].dir && i == b.cur {
		b.collapsed[b.rows[i].path] = !b.collapsed[b.rows[i].path]
		b.build()
		return nil
	}
	b.cur = i
	b.show()
	return nil
}

// onMouse routes a mouse event to the pane under the pointer, in that
// pane's own coordinates; a click also focuses the pane.
func (s *Shell) onMouse(m tea.MouseMsg) tea.Cmd {
	if len(s.panes) == 1 {
		return s.deliver(s.panes[0], m)
	}
	rects, tabs := layout(s.mode, len(s.panes), s.w, s.h)
	if tabs {
		if m.Y == 0 {
			return nil // the tab bar
		}
		m.Y--
		return s.deliver(s.focused(), m)
	}
	for i, r := range rects {
		if m.X < r.x || m.X >= r.x+r.w || m.Y < r.y || m.Y >= r.y+r.h {
			continue
		}
		if m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft {
			s.focus = i
		}
		m.X, m.Y = m.X-r.x-1, m.Y-r.y-1 // inside the pane's frame
		return s.deliver(s.panes[i], m)
	}
	return nil
}

// openURL opens url in the browser (open on macOS, xdg-open on Linux).
// Without a display, as over SSH, it says to use the terminal's link instead.
func (a *App) openURL(url string) tea.Cmd {
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	} else if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		a.setToast("no browser here (SSH?): cmd/ctrl+click the link, or open "+url, false)
		return nil
	}
	a.setToast("opening "+url, false)
	return func() tea.Msg {
		c := exec.Command(opener, url)
		if err := c.Start(); err != nil {
			return opMsg{label: "open " + url, err: err}
		}
		go c.Wait() //nolint:errcheck // reap it; the browser outlives it anyway
		return nil
	}
}
