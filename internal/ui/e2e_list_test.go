package ui

// End-to-end tests of the repository list, panes and the update notice.

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paksonite/gitnest/internal/update"
)

func TestE2EListSearchFilterTree(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, _ := e2eSetup(t)
	for _, p := range []string{"group/one", "group/two"} {
		dir := filepath.Join(root, p)
		writeFile(t, filepath.Join(dir, "f.txt"), "f\n")
		sh(t, dir, "init", "-q", "-b", "main")
		sh(t, dir, "add", ".")
		sh(t, dir, "commit", "-q", "-m", "init")
	}
	sh(t, filepath.Join(root, "group/two"), "checkout", "-q", "-b", "release-x")
	writeFile(t, filepath.Join(root, "lib", "dirty.txt"), "d\n")
	drive(t, a, a.scanCmd(true))
	drive(t, a, a.gather(a.paths()))
	if len(a.rs.all) != 4 {
		t.Fatalf("repositories: %d", len(a.rs.all))
	}

	// d shows only repositories with changes, and again all of them.
	press(t, a, "d")
	if got := listed(a); len(got) != 1 || filepath.Base(got[0]) != "lib" || !strings.Contains(a.View(), "[dirty only]") {
		t.Fatalf("dirty only: %v", got)
	}
	press(t, a, "d")

	// @text searches branches; tab keeps the filter, esc clears it.
	press(t, a, "/", "type:@release")
	if got := listed(a); len(got) != 1 || filepath.Base(got[0]) != "two" {
		t.Fatalf("@branch: %v", got)
	}
	press(t, a, "tab")
	if a.rs.filtering || a.rs.filter != "@release" || len(listed(a)) != 1 {
		t.Fatalf("tab keeps the filter: filtering=%v filter=%q", a.rs.filtering, a.rs.filter)
	}
	press(t, a, "j", "esc")
	if a.rs.filter != "" || len(listed(a)) != 4 {
		t.Fatalf("esc clears: %q %v", a.rs.filter, listed(a))
	}
	// While searching, j, q and ? are text; backspace and ctrl+w edit it.
	press(t, a, "/", "type:q?j", "backspace", "ctrl+w", "type:grp one")
	if a.rs.filter != "grp one" || len(listed(a)) != 1 || a.report != nil {
		t.Fatalf("query editing: %q %v", a.rs.filter, listed(a))
	}
	press(t, a, "esc")

	// The tree folds a directory with enter and marks everything below it
	// with space; f fetches the marked repositories.
	press(t, a, "`")
	var dirRow int = -1
	for i, r := range a.rs.rows {
		if r.repo < 0 && r.name == "group" {
			dirRow = i
		}
	}
	if dirRow < 0 {
		t.Fatalf("no group directory: %+v", a.rs.rows)
	}
	a.rs.cur = dirRow
	press(t, a, "enter")
	if len(a.rs.rows) != 3 {
		t.Fatalf("folded: %d rows", len(a.rs.rows))
	}
	press(t, a, "enter", "space")
	if len(a.rs.marked) != 2 {
		t.Fatalf("marking a directory: %v", a.rs.marked)
	}
	press(t, a, "f")
	if !strings.Contains(a.toast, "fetched 2 repos ✓") {
		t.Errorf("fetch marked: %q", a.toast)
	}

	// s cycles the sort modes, r refreshes, R rescans.
	for _, want := range []string{"name", "last commit", "status", "recent"} {
		press(t, a, "s")
		if a.toast != "sorted by "+want {
			t.Errorf("sort: %q, want %q", a.toast, want)
		}
	}
	press(t, a, "r")
	press(t, a, "R")
	if !strings.Contains(a.toast, "4 repositories, no changes") {
		t.Errorf("rescan: %q", a.toast)
	}
	press(t, a, "?")
	if a.report == nil || a.report.title != "Keys" {
		t.Fatal("? opens the help")
	}
	press(t, a, "q")
}

func TestE2EPanesLayoutAndTabs(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, _ := e2eSetup(t)
	s := newShell(a)
	s.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
	press(t, s, "ctrl+n", "ctrl+n", "ctrl+n")
	if got := s.shownLayout(); got != "grid" {
		t.Fatalf("four panes: %s", got)
	}
	// ctrl+g cycles the layouts; the choice is remembered.
	press(t, s, "ctrl+g")
	if s.mode != paneColumns || a.st.PaneLayout != "side by side" {
		t.Fatalf("layout %d, saved %q", s.mode, a.st.PaneLayout)
	}
	if !strings.Contains(s.focused().app.toast, "doesn't fit") {
		t.Errorf("four columns don't fit in 200 columns: %q", s.focused().app.toast)
	}
	press(t, s, "ctrl+g", "ctrl+g")
	if s.mode != paneGrid {
		t.Fatalf("back to the grid: %d", s.mode)
	}

	// ctrl+arrows move between grid cells.
	s.focus = 0
	press(t, s, "ctrl+right")
	if s.focus != 1 {
		t.Errorf("right: %d", s.focus)
	}
	press(t, s, "ctrl+down")
	if s.focus != 3 {
		t.Errorf("down: %d", s.focus)
	}
	press(t, s, "alt+left")
	if s.focus != 2 {
		t.Errorf("alt+left: %d", s.focus)
	}

	// Seven panes are tabs; the focused pane gets the whole window.
	press(t, s, "ctrl+n", "ctrl+n", "ctrl+n")
	if got := s.shownLayout(); got != "tabs" || s.focused().app.w != 200 || s.focused().app.h != 59 {
		t.Fatalf("seven panes: %s, %dx%d", got, s.focused().app.w, s.focused().app.h)
	}
	press(t, s, "/", "type:lib", "enter")
	if !strings.Contains(strings.SplitN(s.View(), "\n", 2)[0], "lib · main") {
		t.Errorf("tab bar: %q", strings.SplitN(s.View(), "\n", 2)[0])
	}
	// A pane that opened a repository goes back to its list with q, and
	// closes with q there.
	press(t, s, "q", "q")
	if len(s.panes) != 6 {
		t.Errorf("q twice: %d panes", len(s.panes))
	}

	// In a tiny window panes fall back to tabs, and below the minimum the
	// size notice shows.
	s.Update(tea.WindowSizeMsg{Width: 60, Height: 14})
	if s.shownLayout() != "tabs" || !strings.Contains(s.View(), "repositories") {
		t.Errorf("small window: %s", s.shownLayout())
	}
	s.Update(tea.WindowSizeMsg{Width: 45, Height: 10})
	if !strings.Contains(s.View(), "needs at least 50×12") {
		t.Errorf("tiny window:\n%s", s.View())
	}
	_ = root
}

// The update notice comes from a real check against a fake release server.
func TestE2EUpdateNotice(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, _, _ := e2eSetup(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/releases/latest" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"https://example.com/r"}`))
	}))
	defer srv.Close()
	defer func(base, v string) { update.APIBase, Version = base, v }(update.APIBase, Version)
	update.APIBase, Version = srv.URL, "0.9.1"

	s := newShell(a)
	s.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	drive(t, s, s.Init())
	if v := s.View(); !strings.Contains(v, "⬆ 9.9.9 · gitnest update") {
		t.Fatalf("no update badge:\n%s", v)
	}
	press(t, s, "ctrl+n")
	if !strings.Contains(s.View(), "⬆ 9.9.9") {
		t.Error("a new pane shows the badge too")
	}

	// Up to date, or GitHub failing: nothing is shown.
	for _, body := range []string{`{"tag_name":"v0.9.1"}`, ""} {
		srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if body == "" {
				http.Error(w, "rate limited", http.StatusForbidden)
				return
			}
			w.Write([]byte(body))
		})
		a2, _, _ := e2eSetup(t)
		s2 := newShell(a2)
		s2.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
		drive(t, s2, s2.Init())
		if v := s2.View(); strings.Contains(v, "⬆") || a2.toastErr {
			t.Errorf("response %q: badge or error shown (toast %q)", body, a2.toast)
		}
	}

	// update_check: false turns the check off.
	f := false
	a.cfg.UpdateCheck = &f
	if a.updateCheck() != nil {
		t.Error("update_check: false")
	}
}
