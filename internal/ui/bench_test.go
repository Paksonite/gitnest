package ui

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paksonite/gitnest/internal/gitx"
)

// Benchmarks for the screens redrawn on every key and tick, with many
// repositories: go test -bench . -run ^$ ./internal/ui

func benchApp(n int) *App {
	paths := make([]string, n)
	for i := range paths {
		paths[i] = fmt.Sprintf("/r/group%d/sub%d/repo-%04d", i%20, i%3, i)
	}
	a := testApp(paths...)
	now := time.Now()
	act := map[string][]gitx.ActivityCommit{}
	for i := range a.rs.all {
		r := &a.rs.all[i]
		r.Name = fmt.Sprintf("repo-%04d", i)
		r.Branch, r.Upstream, r.Changes, r.Ahead = "main", true, i%5, i%7
		r.RemoteURL = fmt.Sprintf("git@host%d.example.com:team%d/%s.git", i%4, i%9, r.Name)
		for c := 0; c < 50; c++ {
			act[r.Path] = append(act[r.Path], gitx.ActivityCommit{
				Unix: now.Add(-time.Duration(c*37) * time.Hour).Unix(), Short: "abc1234", Subject: "a commit subject", Author: fmt.Sprintf("Author %d", c%6)})
		}
	}
	a.onActivity(activityMsg{act: act})
	a.cfg.Roots = []string{"/r"}
	a.w, a.h = 200, 60
	a.rebuildView("")
	return a
}

func BenchmarkListView(b *testing.B) {
	a := benchApp(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.View()
	}
}

func BenchmarkListViewTree(b *testing.B) {
	a := benchApp(1000)
	a.rs.tree = true
	a.rebuildView("")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.View()
	}
}

func BenchmarkListMoveDown(b *testing.B) {
	a := benchApp(1000)
	a.rs.tree = true
	a.rebuildView("")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.onKey(tea.KeyMsg{Type: tea.KeyDown})
		_ = a.View()
	}
}

func BenchmarkDashboardView(b *testing.B) {
	a := benchApp(200)
	a.scr = scrDash
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.View()
	}
}

func BenchmarkShellPaint(b *testing.B) {
	a := benchApp(1000)
	s := newShell(a)
	s.w, s.h = 200, 60
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.View()
	}
}
