package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/gitx"
	"github.com/Paksonite/gitnest/internal/scan"
)

// shape describes a layout as panes per row, e.g. "2+2" for a 2×2 grid.
func shape(rects []rect) string {
	var rows []string
	count := map[int]int{}
	var ys []int
	for _, r := range rects {
		if count[r.y] == 0 {
			ys = append(ys, r.y)
		}
		count[r.y]++
	}
	for _, y := range ys {
		rows = append(rows, fmt.Sprint(count[y]))
	}
	return strings.Join(rows, "+")
}

func TestLayout(t *testing.T) {
	cases := []struct {
		mode, n, w, h int
		want          string // panes per row, or "tabs"
	}{
		{paneGrid, 1, 80, 24, "1"},
		{paneGrid, 2, 200, 50, "2"},
		{paneGrid, 3, 160, 60, "2+1"},
		{paneGrid, 4, 220, 60, "2+2"},
		{paneGrid, 4, 400, 40, "2+2"},
		{paneGrid, 6, 240, 60, "3+3"},
		{paneGrid, 7, 400, 100, "tabs"}, // more than maxTiled: tabs however big the window
		{paneGrid, 4, 100, 30, "tabs"},
		{paneGrid, 4, 400, 20, "4"}, // too low for two rows: side by side
		{paneColumns, 3, 300, 50, "3"},
		{paneColumns, 4, 200, 60, "2+2"}, // too narrow for four columns: grid
		{paneRows, 2, 100, 60, "1+1"},
		{paneRows, 3, 120, 36, "2+1"}, // too low for three rows: grid
	}
	for _, c := range cases {
		rects, tabs := layout(c.mode, c.n, c.w, c.h)
		got := shape(rects)
		if tabs {
			got = "tabs"
		}
		if got != c.want {
			t.Errorf("%s, %d panes in %dx%d: %s, want %s", paneModeNames[c.mode], c.n, c.w, c.h, got, c.want)
		}
		if tabs || c.n == 1 {
			continue
		}
		area := 0
		for _, r := range rects {
			area += r.w * r.h
			if r.x+r.w > c.w || r.y+r.h > c.h || r.w-2 < minWidth || r.h-2 < minHeight {
				t.Errorf("%d panes in %dx%d: bad rect %+v", c.n, c.w, c.h, r)
			}
		}
		if area != c.w*c.h || len(rects) != c.n {
			t.Errorf("%d panes in %dx%d: %d rects cover %d of %d cells", c.n, c.w, c.h, len(rects), area, c.w*c.h)
		}
	}
}

func TestFocusTowards(t *testing.T) {
	s := shellOf(t, 200, 60, "/r/a")
	for i := 0; i < 3; i++ {
		press(t, s, "ctrl+n")
	}
	// 2×2: 0 1 / 2 3; the new panes went after the focused one each time.
	s.focus = 0
	steps := []struct {
		key  string
		want int
	}{{"ctrl+right", 1}, {"ctrl+down", 3}, {"ctrl+left", 2}, {"ctrl+up", 0}, {"ctrl+up", 0}, {"alt+right", 1}}
	for _, st := range steps {
		press(t, s, st.key)
		if s.focus != st.want {
			t.Fatalf("%s: focus %d, want %d", st.key, s.focus, st.want)
		}
	}
	press(t, s, "ctrl+g") // four side by side need 4×52 columns, 200 is too few: the grid stays
	if s.mode != paneColumns || !strings.Contains(s.focused().app.toast, "doesn't fit this window, showing grid") {
		t.Errorf("ctrl+g: mode %d toast %q", s.mode, s.focused().app.toast)
	}
}

func shellOf(t *testing.T, w, h int, paths ...string) *Shell {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	a := testApp(paths...)
	for i := range a.rs.all {
		a.rs.all[i].Name = filepath.Base(a.rs.all[i].Path)
	}
	a.rebuildView("")
	s := newShell(a)
	s.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return s
}

func TestPanesOpenFocusClose(t *testing.T) {
	s := shellOf(t, 220, 50, "/r/app", "/r/lib")
	press(t, s, "ctrl+n")
	if len(s.panes) != 2 || s.focus != 1 || s.focused().app.scr != scrRepos || len(s.focused().app.rs.all) != 2 {
		t.Fatalf("after ctrl+n: %d panes, focus %d", len(s.panes), s.focus)
	}
	if s.panes[0].app.w != 108 || s.focused().app.w != 108 {
		t.Errorf("pane widths %d / %d", s.panes[0].app.w, s.focused().app.w)
	}
	if s.focused().app.sh != s.panes[0].app.sh {
		t.Error("panes must share the clipboard")
	}
	press(t, s, "ctrl+left")
	if s.focus != 0 {
		t.Errorf("ctrl+left: focus %d", s.focus)
	}
	// While typing a search, ctrl+n moves in the results instead.
	press(t, s, "/", "ctrl+n")
	if len(s.panes) != 2 || !s.focused().app.rs.filtering {
		t.Errorf("ctrl+n while searching opened a pane: %d", len(s.panes))
	}
	press(t, s, "esc")
	for i := 0; i < 5; i++ { // no limit on the number of panes
		press(t, s, "ctrl+n")
	}
	if len(s.panes) != 7 {
		t.Errorf("panes: %d", len(s.panes))
	}
	// q on the list closes just that pane while there are several.
	press(t, s, "q")
	if len(s.panes) != 6 {
		t.Errorf("q: %d panes", len(s.panes))
	}
	for len(s.panes) > 1 {
		press(t, s, "ctrl+w")
	}
	if len(s.panes) != 1 {
		t.Errorf("ctrl+w: %d panes", len(s.panes))
	}
	if _, cmd := s.Update(tea.KeyMsg{Type: tea.KeyCtrlW}); cmd == nil {
		t.Error("ctrl+w on the last pane should quit")
	}
}

func TestPanesRouting(t *testing.T) {
	s := shellOf(t, 220, 50, "/r/app", "/r/lib")
	press(t, s, "ctrl+n")
	first, second := s.panes[0], s.panes[1]
	// A status update asked for by one pane refreshes the list in both.
	s.Update(paneMsg{id: first.id, msg: statusMsg{repos: []scan.Repo{{Path: "/r/lib", Name: "lib", Info: gitx.Info{Changes: 5}}}, asked: []string{"/r/lib"}}})
	for _, p := range []*pane{first, second} {
		for _, r := range p.app.rs.all {
			if r.Path == "/r/lib" && r.Changes != 5 {
				t.Errorf("pane %d didn't get the status", p.id)
			}
		}
	}
	// Messages for a closed pane are dropped.
	press(t, s, "ctrl+w")
	if _, cmd := s.Update(paneMsg{id: second.id, msg: opMsg{label: "x"}}); cmd != nil || len(s.panes) != 1 {
		t.Error("message for a closed pane was delivered")
	}
	// A pane-local message reaches only its pane.
	press(t, s, "ctrl+n")
	third := s.panes[1]
	s.Update(paneMsg{id: third.id, msg: opMsg{label: "fetch", err: fmt.Errorf("boom")}})
	if !strings.Contains(third.app.toast, "boom") || strings.Contains(first.app.toast, "boom") {
		t.Errorf("toasts: %q / %q", first.app.toast, third.app.toast)
	}
}

func TestPanesFitAnyWindow(t *testing.T) {
	sizes := [][2]int{{40, 10}, {80, 24}, {120, 40}, {160, 50}, {220, 60}, {300, 80}}
	for n := 1; n <= 8; n++ {
		for _, sz := range sizes {
			s := shellOf(t, sz[0], sz[1], "/r/a", "/r/b", "/r/c")
			for len(s.panes) < n {
				press(t, s, "ctrl+n")
			}
			view := s.View()
			lines := strings.Split(view, "\n")
			if len(lines) > sz[1] {
				t.Errorf("%d panes at %dx%d: %d lines", n, sz[0], sz[1], len(lines))
			}
			for i, l := range lines {
				if w := lipglossWidth(l); w > sz[0] {
					t.Errorf("%d panes at %dx%d: line %d is %d wide", n, sz[0], sz[1], i, w)
					break
				}
			}
			if n > 1 && !strings.Contains(view, fmt.Sprintf("%d · repositories", n)) {
				t.Errorf("%d panes at %dx%d: no pane title or tab", n, sz[0], sz[1])
			}
		}
	}
}

// Two panes on two repositories: commit in one, copy the commit, paste it
// into the other.
func TestE2EPanes(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, _ := e2eSetup(t)
	s := newShell(a)
	s.Update(tea.WindowSizeMsg{Width: 240, Height: 50})
	app, lib := filepath.Join(root, "app"), filepath.Join(root, "lib")

	press(t, s, "/", "type:app", "enter")           // pane 1: app
	press(t, s, "ctrl+n", "/", "type:lib", "enter") // pane 2: lib
	if s.panes[0].app.rp.dir != app || s.panes[1].app.rp.dir != lib || s.focus != 1 {
		t.Fatalf("panes: %q / %q, focus %d", s.panes[0].app.rp.dir, s.panes[1].app.rp.dir, s.focus)
	}
	if v := s.View(); !strings.Contains(v, "1 · app · main") || !strings.Contains(v, "2 · lib · main") {
		t.Errorf("titles missing:\n%s", v)
	}

	// Commit in app (pane 1).
	press(t, s, "ctrl+left")
	os.WriteFile(filepath.Join(app, "feature.txt"), []byte("feature\n"), 0o644)
	drive(t, s, wrap(s.panes[0].id, s.panes[0].app.onRepoTick()))
	press(t, s, "a", "c", "type:Add feature", "enter")
	if got := sh(t, app, "log", "-1", "--format=%s"); got != "Add feature" {
		t.Fatalf("commit in app: %q", got)
	}
	// Copy it, paste into lib (pane 2).
	press(t, s, "3", "c", "ctrl+right", "3", "v")
	if p := s.panes[1].app.p; p == nil || !strings.Contains(p.title, "as patches from app") {
		t.Fatalf("paste prompt in pane 2: %+v", p)
	}
	press(t, s, "y")
	if got := sh(t, lib, "log", "-1", "--format=%s"); got != "Add feature" {
		t.Fatalf("lib after paste: %q", got)
	}
	// Pane 1 was not disturbed.
	if s.panes[0].app.scr != scrRepo || s.panes[0].app.rp.dir != app {
		t.Error("pane 1 changed")
	}
}

func lipglossWidth(s string) int { return lipgloss.Width(s) }

func TestUpdateBadge(t *testing.T) {
	s := shellOf(t, 220, 50, "/r/app", "/r/lib")
	press(t, s, "ctrl+n")
	if strings.Contains(s.View(), "⬆") {
		t.Fatal("badge without a newer release")
	}
	s.Update(updateMsg{"9.9.9"})
	for i, p := range s.panes {
		if !strings.Contains(p.app.viewRepos(), "⬆ 9.9.9") {
			t.Errorf("pane %d: no update badge", i)
		}
	}
	if !strings.Contains(s.focused().app.toast, "gitnest update") {
		t.Errorf("toast %q", s.focused().app.toast)
	}
	if (&App{}).updateCheck() != nil {
		t.Error("a development build must not look for updates")
	}
}

func TestTabBar(t *testing.T) {
	var titles []string
	for i := 1; i <= 12; i++ {
		titles = append(titles, fmt.Sprintf("%d · repository-%d · main", i, i))
	}
	titles[4] = "5 · a-repository-with-a-really-long-name-indeed · feature/long"
	for _, w := range []int{40, 80, 160} {
		for focus := range titles {
			bar := tabBar(titles, focus, w)
			if lipglossWidth(bar) > w {
				t.Errorf("w %d focus %d: bar is %d wide", w, focus, lipglossWidth(bar))
			}
			want := fmt.Sprintf(" %d · ", focus+1)
			if !strings.Contains(bar, want) {
				t.Errorf("w %d focus %d: focused tab not shown: %q", w, focus, bar)
			}
			// Markers count exactly the tabs left out on each side.
			shown := 0
			for i := range titles {
				if strings.Contains(bar, fmt.Sprintf(" %d · ", i+1)) {
					shown++
				}
			}
			left, right := 0, 0
			fmt.Sscanf(bar[strings.Index(bar, "‹")+len("‹"):], "%d", &left)
			if i := strings.LastIndex(bar, "›"); i >= 0 {
				head := strings.TrimRight(bar[:i], " ")
				fmt.Sscanf(head[strings.LastIndex(head, " ")+1:], "%d", &right)
			}
			if !strings.Contains(bar, "‹") {
				left = 0
			}
			if shown+left+right != len(titles) {
				t.Errorf("w %d focus %d: %d shown + ‹%d + %d› != %d: %q", w, focus, shown, left, right, len(titles), bar)
			}
		}
	}
	if bar := tabBar(titles[:3], 0, 200); !strings.Contains(bar, "ctrl+←/→") || strings.Contains(bar, "‹") || strings.Contains(bar, "›") {
		t.Errorf("all tabs fit: %q", bar)
	}
}

// Bubble Tea's own requests, such as running the editor, must reach the
// program unwrapped; anything else goes back to its pane.
func TestWrapPassesProgramMessages(t *testing.T) {
	if msg := wrap(1, tea.ExecProcess(exec.Command("true"), nil))(); reflect.TypeOf(msg).Name() != "execMsg" {
		t.Errorf("exec request wrapped: %T", msg)
	}
	if msg := wrap(1, func() tea.Msg { return execDoneMsg{} })(); reflect.TypeOf(msg) != reflect.TypeOf(paneMsg{}) {
		t.Errorf("own message not wrapped: %T", msg)
	}
}

func TestCenterOff(t *testing.T) {
	for _, c := range []struct{ cur, n, h, want int }{
		{5, 8, 10, 0},  // everything fits: never scroll
		{0, 30, 10, 0}, // the top
		{15, 30, 10, 10},
		{29, 30, 10, 20}, // the bottom: no blank rows below
	} {
		if got := centerOff(c.cur, c.n, c.h); got != c.want {
			t.Errorf("centerOff(%d, %d, %d) = %d, want %d", c.cur, c.n, c.h, got, c.want)
		}
	}
}
