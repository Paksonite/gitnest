package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paksonite/gitnest/internal/gitx"
)

func dashApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	a := testApp("/r/api", "/r/web", "/r/quiet")
	for i := range a.rs.all {
		r := &a.rs.all[i]
		r.Name = r.Path[3:]
		r.Upstream = true
	}
	a.rs.all[0].Changes = 2 // api: dirty
	a.rs.all[1].Behind = 1  // web: to pull
	now := time.Now()
	ago := func(d int) int64 { return now.AddDate(0, 0, -d).Unix() }
	a.dash.act = map[string][]gitx.ActivityCommit{
		"/r/api": {{Unix: ago(0), Short: "a1", Subject: "add health check", Author: "Ann Example"},
			{Unix: ago(1), Short: "a2", Subject: "fix", Author: "Ann Example"}, {Unix: ago(20), Short: "a3", Subject: "old", Author: "Bob Example"}},
		"/r/web": {{Unix: ago(3), Short: "w1", Subject: "style", Author: "Bob Example"}},
	}
	a.scr, a.w, a.h = scrDash, 150, 40
	return a
}

func TestDashboard(t *testing.T) {
	a := dashApp(t)
	v := a.View()
	for _, want := range []string{"dashboard", "Activity", "4 commits", "Overview", "Needs attention", "api", "web",
		"Most active", "Recent commits", "AE a1 api · add health check", "Mon", "less", "more", "Streak"} {
		if !strings.Contains(v, want) {
			t.Errorf("dashboard lacks %q", want)
		}
	}
	for i, l := range strings.Split(v, "\n") {
		if w := lipglossWidth(l); w > 150 {
			t.Errorf("line %d is %d wide", i, w)
		}
	}
	if strings.Contains(v, "quiet  ") && strings.Index(v, "quiet") < strings.Index(v, "Most active") {
		t.Error("a clean, synced repository needs no attention")
	}
	// Needs attention comes first: api (dirty) before web (to pull).
	if len(a.dash.items) < 2 || a.dash.items[0] != "/r/api" || a.dash.items[1] != "/r/web" {
		t.Fatalf("items: %v", a.dash.items)
	}
	// j moves, enter opens, esc in the repository comes back here.
	a.onKey(tea.KeyMsg{Type: tea.KeyDown})
	a.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	if a.scr != scrRepo || a.rp.dir != "/r/web" {
		t.Fatalf("enter: screen %d dir %q", a.scr, a.rp.dir)
	}
	a.repoKey(tea.KeyMsg{Type: tea.KeyEsc})
	if a.scr != scrDash {
		t.Fatalf("esc from a repository opened from the dashboard: screen %d", a.scr)
	}
	// esc goes to the list, H back to the dashboard.
	a.onKey(tea.KeyMsg{Type: tea.KeyEsc})
	if a.scr != scrRepos {
		t.Fatalf("esc: screen %d", a.scr)
	}
	a.onKey(key("H"))
	if a.scr != scrDash {
		t.Fatalf("H: screen %d", a.scr)
	}
	// Narrow windows stack the panels and still fit.
	a.w, a.h = 80, 50
	for i, l := range strings.Split(a.View(), "\n") {
		if w := lipglossWidth(l); w > 80 {
			t.Errorf("narrow: line %d is %d wide", i, w)
		}
	}
}

func TestSparklineAndHeatmap(t *testing.T) {
	if got := sparkline([]int{0, 1, 2, 4, 8}); got != "·▁▂▄█" {
		t.Errorf("sparkline: %q", got)
	}
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.Local) // a Sunday
	day := func(d int) time.Time {
		t := now.AddDate(0, 0, -d)
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}
	rows := heatmap(map[time.Time]int{day(0): 4, day(1): 1}, now, 60, false)
	if len(rows) != 9 || !strings.HasPrefix(rows[1], "Mon") || !strings.HasPrefix(rows[7], "Sun") || !strings.Contains(rows[8], "less") {
		t.Errorf("heatmap rows:\n%s", strings.Join(rows, "\n"))
	}
	if weekStart(now).Weekday() != time.Monday {
		t.Error("weeks start on Monday")
	}
}
