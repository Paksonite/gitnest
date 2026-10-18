package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func click(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
}

func wheel(x, y int, down bool) tea.MouseMsg {
	b := tea.MouseButtonWheelUp
	if down {
		b = tea.MouseButtonWheelDown
	}
	return tea.MouseMsg{X: x, Y: y, Button: b, Action: tea.MouseActionPress}
}

func TestMouseList(t *testing.T) {
	a := testApp("/r/a", "/r/b", "/r/c", "/r/d", "/r/e", "/r/f")
	a.w, a.h = 120, 30
	a.View() // lays the list out
	a.onMouse(click(10, 4+2))
	if a.rs.cur != 2 || a.scr != scrRepos {
		t.Fatalf("click: cur %d", a.rs.cur)
	}
	a.onMouse(wheel(10, 6, true))
	if a.rs.cur != 5 {
		t.Fatalf("wheel down: cur %d", a.rs.cur)
	}
	a.onMouse(wheel(10, 6, false))
	if a.rs.cur != 2 {
		t.Fatalf("wheel up: cur %d", a.rs.cur)
	}
	a.onMouse(click(10, 4+2)) // the selected row: open it
	if a.scr != scrRepo || a.rp.dir != "/r/c" {
		t.Fatalf("click on the selected row: screen %d dir %q", a.scr, a.rp.dir)
	}
}

func TestMouseDashboardAndPanes(t *testing.T) {
	a := dashApp(t)
	a.View()
	h := a.dash.hits[1]
	a.onMouse(click(h.x0+2, h.y))
	if a.dash.cur != 1 || a.scr != scrDash {
		t.Fatalf("dashboard click: cur %d", a.dash.cur)
	}
	a.onMouse(click(h.x0+2, h.y))
	if a.scr != scrRepo {
		t.Fatalf("dashboard click on the selected row: screen %d", a.scr)
	}

	s := shellOf(t, 200, 40, "/r/x", "/r/y")
	press(t, s, "ctrl+n") // two panes side by side, focus on the right
	s.Update(click(5, 10))
	if s.focus != 0 {
		t.Fatalf("a click should focus the pane under it: %d", s.focus)
	}
	s.Update(click(150, 10))
	if s.focus != 1 {
		t.Fatalf("click on the right pane: %d", s.focus)
	}
}
