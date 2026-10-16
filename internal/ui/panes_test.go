package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/gitx"
	"github.com/Paksonite/gitnest/internal/scan"
)

func TestLayout(t *testing.T) {
	cases := []struct {
		n, w, h int
		want    string // "cols", "rows", "grid", "tabs", "full"
	}{
		{1, 80, 24, "full"},
		{2, 200, 50, "cols"},
		{2, 100, 60, "rows"},
		{2, 80, 24, "tabs"},
		{3, 300, 50, "cols"},
		{3, 150, 60, "grid"},
		{4, 300, 40, "cols"},
		{4, 120, 60, "grid"},
		{4, 100, 30, "tabs"},
	}
	for _, c := range cases {
		rects, tabs := layout(c.n, c.w, c.h)
		got := "full"
		switch {
		case tabs:
			got = "tabs"
		case c.n == 1:
		case rects[0].y == rects[len(rects)-1].y:
			got = "cols"
		case rects[0].w == c.w:
			got = "rows"
		default:
			got = "grid"
		}
		if got != c.want {
			t.Errorf("%d panes in %dx%d: %s, want %s (%+v)", c.n, c.w, c.h, got, c.want, rects)
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
	press(t, s, "ctrl+o")
	if s.focus != 0 {
		t.Errorf("ctrl+o: focus %d", s.focus)
	}
	// While typing a search, ctrl+n moves in the results instead.
	press(t, s, "/", "ctrl+n")
	if len(s.panes) != 2 || !s.focused().app.rs.filtering {
		t.Errorf("ctrl+n while searching opened a pane: %d", len(s.panes))
	}
	press(t, s, "esc")
	for i := 0; i < 4; i++ {
		press(t, s, "ctrl+n")
	}
	if len(s.panes) != maxPanes || !strings.Contains(s.focused().app.toast, "at most 4") {
		t.Errorf("max panes: %d, toast %q", len(s.panes), s.focused().app.toast)
	}
	// q on the list closes just that pane while there are several.
	press(t, s, "q")
	if len(s.panes) != 3 {
		t.Errorf("q: %d panes", len(s.panes))
	}
	press(t, s, "ctrl+w", "ctrl+w")
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
	for n := 1; n <= 4; n++ {
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
			if n > 1 && !strings.Contains(view, "2 · repositories") {
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
	press(t, s, "ctrl+o")
	os.WriteFile(filepath.Join(app, "feature.txt"), []byte("feature\n"), 0o644)
	drive(t, s, wrap(s.panes[0].id, s.panes[0].app.onRepoTick()))
	press(t, s, "a", "c", "type:Add feature", "enter")
	if got := sh(t, app, "log", "-1", "--format=%s"); got != "Add feature" {
		t.Fatalf("commit in app: %q", got)
	}
	// Copy it, paste into lib (pane 2).
	press(t, s, "3", "c", "ctrl+o", "3", "v")
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
