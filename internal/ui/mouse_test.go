package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

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

func TestMouseSelectAndLink(t *testing.T) {
	a := testApp("/r/platform")
	a.rs.all[0].Name = "platform"
	a.rs.all[0].RemoteURL = "git@gitlab.com:platform/platform.git"
	a.rebuildView("")
	a.w, a.h = 140, 30
	a.rs.previewPath = "/r/platform"
	v := a.View()
	// Find the Path row of the details card and drag across it.
	lines := strings.Split(v, "\n")
	y := -1
	for i, l := range lines {
		if strings.Contains(l, "Path") && strings.Contains(l, "/r/platform") {
			y = i
		}
	}
	if y < 0 {
		t.Fatalf("no Path row:\n%s", v)
	}
	x := strings.Index(lines[y], "Path")
	x = lipgloss.Width(lines[y][:x])
	a.onMouse(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	a.onMouse(tea.MouseMsg{X: x + 30, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion})
	cmd := a.onMouse(tea.MouseMsg{X: x + 30, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
	if !a.sel.on || cmd == nil || !strings.Contains(a.toast, "copied") {
		t.Fatalf("selection: on %v, toast %q", a.sel.on, a.toast)
	}
	if got := a.selText(); !strings.HasPrefix(got, "Path") || !strings.Contains(got, "/r/platform") {
		t.Errorf("selected text: %q", got)
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyDown}) // any key clears it
	if a.sel.on {
		t.Error("a key should clear the selection")
	}
	// A click on the Link row opens the repository's web page.
	if a.rs.link.url != "https://gitlab.com/platform/platform" {
		t.Fatalf("link: %+v", a.rs.link)
	}
	a.View()
	a.onMouse(click(a.rs.listW+20, a.rs.link.y))
	if !strings.Contains(a.toast, "https://gitlab.com/platform/platform") {
		t.Errorf("link click: toast %q", a.toast)
	}
}
