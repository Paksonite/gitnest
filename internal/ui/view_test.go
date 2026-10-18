package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/v2/internal/gitx"
	"github.com/Paksonite/gitnest/v2/internal/theme"
)

// checkFrame fails when a rendered frame is wider or taller than the window.
func checkFrame(t *testing.T, name string, a *App, frame string) {
	t.Helper()
	lines := strings.Split(frame, "\n")
	if len(lines) > a.h {
		t.Errorf("%s at %dx%d: %d lines", name, a.w, a.h, len(lines))
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w > a.w {
			t.Errorf("%s at %dx%d: line %d is %d wide", name, a.w, a.h, i, w)
			return
		}
	}
}

func populated() *App {
	var paths []string
	for i := 0; i < 300; i++ {
		paths = append(paths, fmt.Sprintf("/r/group-%d/repository-with-a-long-name-%03d", i%7, i))
	}
	a := testApp(paths...)
	for i := range a.rs.all {
		r := &a.rs.all[i]
		r.Name = r.Path[strings.LastIndex(r.Path, "/")+1:]
		r.Info = gitx.Info{Branch: "feature/some-long-branch-name", Upstream: true, Ahead: i % 3, Behind: i % 5, Changes: i % 4, LastUnix: int64(1_700_000_000 + i)}
	}
	a.rs.isNew[a.rs.all[0].Path] = true
	a.rebuildView("")
	a.rp = repoState{dir: "/r/x", name: "x", info: gitx.Info{Branch: "main", State: gitx.StateRebasing},
		main: strings.Repeat("diff line with some content\n", 500)}
	for i := 0; i < 50; i++ {
		a.rp.files = append(a.rp.files, gitx.File{Path: fmt.Sprintf("dir/file-%d.go", i), X: 'M', Y: ' '})
		a.rp.local = append(a.rp.local, gitx.Branch{Name: fmt.Sprintf("branch-%d", i)})
		a.rp.commits = append(a.rp.commits, gitx.Commit{Short: "abc1234", Subject: "a commit subject that is fairly long"})
	}
	return a
}

func TestViewsFitAnyWindow(t *testing.T) {
	sizes := [][2]int{{1, 1}, {20, 5}, {49, 30}, {50, 12}, {60, 15}, {80, 24}, {89, 30}, {120, 40}, {250, 70}}
	for _, sz := range sizes {
		a := populated()
		a.w, a.h = sz[0], sz[1]
		a.scr = scrRepos
		checkFrame(t, "list", a, a.View())
		a.rs.filtering, a.rs.filter = true, "long 01"
		a.rebuildView("")
		checkFrame(t, "search", a, a.View())

		a.scr = scrRepo
		for focus := 0; focus < 4; focus++ {
			a.rp.focus = focus
			checkFrame(t, fmt.Sprintf("repo/panel%d", focus+1), a, a.View())
		}
		a.rp.mainFocus = true
		checkFrame(t, "repo/diff", a, a.View())

		a.onKey(key("?"))
		checkFrame(t, "help", a, a.View())
		a.report = nil
		a.openEditor("Commit", "subject\n\nbody", func(string) tea.Cmd { return nil })
		checkFrame(t, "editor", a, a.View())
		a.ed = nil
		a.themes, _ = theme.Load(t.TempDir())
		a.openPicker()
		checkFrame(t, "themes", a, a.View())
	}
}

func TestTooSmallNotice(t *testing.T) {
	a := populated()
	a.w, a.h = 40, 10
	if !strings.Contains(a.View(), "needs at least") {
		t.Error("no notice for a tiny window")
	}
}

func TestListRendersOnlyVisibleRows(t *testing.T) {
	a := populated()
	a.w, a.h = 120, 30
	a.View()
	if a.rs.listH <= 0 || a.rs.listH > 30 {
		t.Fatalf("listH = %d", a.rs.listH)
	}
}

// BenchmarkListFrame measures one frame of the list and one search
// keystroke with 300 repositories.
func BenchmarkListFrame(b *testing.B) {
	a := populated()
	a.w, a.h = 160, 50
	for i := 0; i < b.N; i++ {
		a.View()
	}
}

func BenchmarkSearchKeystroke(b *testing.B) {
	a := populated()
	a.rs.filter = "rwln 1"
	for i := 0; i < b.N; i++ {
		a.rebuildView("")
	}
}
