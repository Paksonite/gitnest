package ui

import (
	"context"
	"github.com/charmbracelet/lipgloss"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paksonite/gitnest/v2/internal/gitx"
	"github.com/Paksonite/gitnest/v2/internal/scan"
	"github.com/Paksonite/gitnest/v2/internal/state"
)

func testApp(paths ...string) *App {
	a := &App{ctx: context.Background(), st: &state.State{Recent: map[string]int64{}}, sh: &shared{}}
	a.rs.marked, a.rs.isNew, a.rs.collapsed = map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, p := range paths {
		a.rs.all = append(a.rs.all, scan.Repo{Path: p, Name: p})
		a.rs.known = append(a.rs.known, p)
	}
	a.rebuildView("")
	return a
}

func listed(a *App) []string {
	var out []string
	for _, i := range a.rs.view {
		out = append(out, a.rs.all[i].Path)
	}
	return out
}

func TestOnStatusKeepsOrderAndCursor(t *testing.T) {
	a := testApp("c", "a", "b")
	a.rs.cur = 2 // on "b"
	a.onStatus(statusMsg{
		repos: []scan.Repo{{Path: "a", Name: "a", Info: gitx.Info{Changes: 3}}},
		asked: []string{"a", "c"}, // c is gone
	})
	if got := listed(a); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("list = %v", got)
	}
	if a.rs.all[0].Changes != 3 {
		t.Error("status not updated")
	}
	if a.selectedPath() != "b" {
		t.Errorf("cursor moved to %q", a.selectedPath())
	}
}

func TestOnScanMarksNewAndDropsRemoved(t *testing.T) {
	a := testApp("a", "b")
	a.rs.marked["b"] = true
	cmd := a.onScan(scanMsg{paths: []string{"a", "c"}})
	if cmd == nil {
		t.Fatal("expected a status command for the new repository")
	}
	if got := listed(a); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("list = %v", got)
	}
	if !a.rs.isNew["c"] || a.rs.marked["b"] {
		t.Errorf("isNew=%v marked=%v", a.rs.isNew, a.rs.marked)
	}
	if a.toast != "+1 new: c  ·  −1 removed" {
		t.Errorf("toast = %q", a.toast)
	}
	if a.onScan(scanMsg{paths: []string{"a", "c"}}) != nil {
		t.Error("unchanged scan should do nothing")
	}
}

func TestChangeSummary(t *testing.T) {
	got := changeSummary([]string{"/r/a", "/r/b", "/r/c", "/r/d"}, nil)
	if got != "+4 new: a, b, c, …" {
		t.Errorf("got %q", got)
	}
}

func TestDisambiguate(t *testing.T) {
	repos := []scan.Repo{
		{Path: "/r/gitea.x/org-core-dns", Name: "org-core-dns", Info: gitx.Info{RemoteURL: "git@gitea.devops.ps.oktawave.com:O/org-core-dns.git"}},
		{Path: "/r/forgejo.y/org-core-dns", Name: "org-core-dns", Info: gitx.Info{RemoteURL: "ssh://git@forgejo.int.waveops.pl/O/org-core-dns.git"}},
		{Path: "/r/a/tool", Name: "tool", Info: gitx.Info{RemoteURL: "git@github.com:a/tool.git"}},
		{Path: "/r/b/tool", Name: "tool", Info: gitx.Info{RemoteURL: "git@github.com:b/tool.git"}},
		{Path: "/r/single", Name: "single"},
	}
	got := disambiguate(repos)
	want := map[string]string{
		"/r/gitea.x/org-core-dns":   "gitea",
		"/r/forgejo.y/org-core-dns": "forgejo",
		"/r/a/tool":                 "a", // same host: fall back to the parent directory
		"/r/b/tool":                 "b",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestFilterRanksAndKeepsKeys(t *testing.T) {
	a := testApp("/r/prj-sp-cot-ipsec-sd3i", "/r/org-core-dns", "/r/terraform-google-service-directory")
	for i := range a.rs.all {
		a.rs.all[i].Name = filepath.Base(a.rs.all[i].Path)
	}
	a.rebuildView("")
	a.reposKey(key("/"))
	for _, k := range []string{"d", "n", "s"} {
		a.reposKey(key(k))
	}
	if got := listed(a); len(got) != 1 || got[0] != "/r/org-core-dns" {
		t.Fatalf("dns: %v", got)
	}
	if hl := a.rs.hl["/r/org-core-dns"]; !reflect.DeepEqual(hl, []int{9, 10, 11}) {
		t.Errorf("highlight = %v", hl)
	}
	// "?" and "q" are part of the query, not help/quit.
	a.reposKey(key("?"))
	if a.rs.filter != "dns?" || a.report != nil {
		t.Errorf("filter = %q help open = %v", a.rs.filter, a.report != nil)
	}
	a.reposKey(tea.KeyMsg{Type: tea.KeyBackspace})
	a.reposKey(tea.KeyMsg{Type: tea.KeyTab})
	if a.rs.filtering || a.rs.filter != "dns" {
		t.Errorf("tab: filtering=%v filter=%q", a.rs.filtering, a.rs.filter)
	}
	a.reposKey(key("/"))
	a.reposKey(tea.KeyMsg{Type: tea.KeyEsc})
	if a.rs.filter != "" || len(listed(a)) != 3 {
		t.Errorf("esc: filter=%q list=%v", a.rs.filter, listed(a))
	}
}

func TestFilterEnterOpensRepo(t *testing.T) {
	a := testApp("/r/alpha", "/r/beta")
	a.st = &state.State{Recent: map[string]int64{}}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for i := range a.rs.all {
		a.rs.all[i].Name = filepath.Base(a.rs.all[i].Path)
	}
	a.reposKey(key("/"))
	a.reposKey(key("b"))
	a.reposKey(tea.KeyMsg{Type: tea.KeyEnter})
	if a.scr != scrRepo || a.rp.dir != "/r/beta" || a.rs.filtering {
		t.Fatalf("scr=%v dir=%q filtering=%v", a.scr, a.rp.dir, a.rs.filtering)
	}
}

func TestSortModes(t *testing.T) {
	a := testApp("/r/b", "/r/a", "/r/c")
	a.rs.all[0].Name, a.rs.all[0].LastUnix = "b", 300
	a.rs.all[1].Name, a.rs.all[1].LastUnix = "a", 100
	a.rs.all[2].Name, a.rs.all[2].LastUnix, a.rs.all[2].Changes = "c", 200, 1
	order := func() string {
		s := ""
		for _, p := range listed(a) {
			s += filepath.Base(p)
		}
		return s
	}
	a.rebuildView("")
	if got := order(); got != "bca" {
		t.Errorf("recent: %s", got)
	}
	want := []string{"abc", "bca", "cba"} // name, last commit, status
	for _, w := range want {
		a.rs.sortMode = (a.rs.sortMode + 1) % sortModes
		a.rebuildView("")
		if got := order(); got != w {
			t.Errorf("%s: got %s, want %s", sortNames[a.rs.sortMode], got, w)
		}
	}
}

func TestRelTime(t *testing.T) {
	now := time.Unix(1_000_000_000, 0)
	cases := map[time.Duration]string{
		30 * time.Second:     "30s",
		5 * time.Minute:      "5m",
		3 * time.Hour:        "3h",
		50 * time.Hour:       "2d",
		21 * 24 * time.Hour:  "3w",
		90 * 24 * time.Hour:  "3mo",
		800 * 24 * time.Hour: "2y",
	}
	for d, want := range cases {
		if got := relTime(now.Add(-d).Unix(), now); got != want {
			t.Errorf("%s ago = %q, want %q", d, got, want)
		}
	}
	if relTime(0, now) != "—" {
		t.Error("no commits")
	}
}

func TestPanelHeights(t *testing.T) {
	for _, total := range []int{12, 16, 20, 27, 40, 80} {
		for focus := 0; focus < 4; focus++ {
			for _, counts := range [][4]int{{0, 0, 0, 0}, {50, 50, 300, 5}, {1, 3, 300, 0}} {
				h := panelHeights(total, focus, counts)
				sum := 0
				for i, x := range h {
					sum += x
					if x < 3 && !(x == 0 && i != focus && total < 16) { // tiny: only the focused panel
						t.Fatalf("total=%d focus=%d counts=%v: panel %d height %d", total, focus, counts, i, x)
					}
					if i != focus && x > h[focus] {
						t.Errorf("total=%d focus=%d counts=%v: focused panel not the largest: %v", total, focus, counts, h)
					}
				}
				if sum != total {
					t.Fatalf("total=%d focus=%d counts=%v: heights %v sum to %d", total, focus, counts, h, sum)
				}
			}
		}
	}
	// Files and branches always show rows when there is room (lazygit-like).
	h := panelHeights(40, 2, [4]int{5, 8, 300, 2})
	if h[0] < 3+5 || h[1] < 3+8 {
		t.Errorf("files/branches squeezed: %v", h)
	}
}

func TestBulkReport(t *testing.T) {
	m := bulkMsg{done: "pulled", total: 3, fails: []bulkFail{
		{path: "/r/x", err: &gitx.Error{Args: []string{"pull"}, Output: "fatal: Not possible to fast-forward\nhint: …"}},
	}}
	r := newReport(m, map[string]string{"/r/x": "forgejo"})
	text := strings.Join(r.lines, "\n")
	if r.title != "1 of 3 failed" || !strings.Contains(text, "fast-forward") || !strings.Contains(text, "hint:") || !strings.Contains(text, "forgejo") {
		t.Errorf("report:\n%s\n%s", r.title, text)
	}
}

func key(s string) tea.KeyMsg {
	if s == " " {
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestStatusCell(t *testing.T) {
	cases := []struct {
		info gitx.Info
		want string
	}{
		{gitx.Info{Upstream: true}, "✓"},
		{gitx.Info{Upstream: true, Changes: 3}, "~3"},
		{gitx.Info{Upstream: true, Ahead: 1, Behind: 2}, "↑1 ↓2"},
		{gitx.Info{}, "∅"},
		{gitx.Info{Upstream: true, Conflicts: true, Changes: 1}, "! conflict"},
		{gitx.Info{Upstream: true, State: gitx.StateRebasing}, "! rebasing"},
	}
	for _, c := range cases {
		if got, _ := statusCell(scan.Repo{Info: c.info}); got != c.want {
			t.Errorf("%+v: %q, want %q", c.info, got, c.want)
		}
	}
	// The mark column holds only the mark: no second status dot.
	row := repoRow(scan.Repo{Name: "api", Info: gitx.Info{Upstream: true, Changes: 2}}, rowOpts{nameW: 20})
	if strings.Count(row.plain, "●") != 0 || !strings.HasPrefix(row.plain, "  api") {
		t.Errorf("row: %q", row.plain)
	}
}

func TestBranchColor(t *testing.T) {
	rp := &repoState{mainBranch: "origin/main"}
	cases := []struct {
		b    gitx.Branch
		want lipgloss.TerminalColor
	}{
		{gitx.Branch{Name: "main"}, cYellow},
		{gitx.Branch{Name: "feature/x"}, cCyan},
		{gitx.Branch{Name: "origin/main", Remote: true}, tint(cYellow, 0.6)},
		{gitx.Branch{Name: "origin/feature/x", Remote: true}, cDim},
	}
	for _, c := range cases {
		if got := rp.branchColor(c.b); got != c.want {
			t.Errorf("%s: %v, want %v", c.b.Name, got, c.want)
		}
	}
	// Without origin/HEAD, master counts when there is no main.
	rp = &repoState{local: []gitx.Branch{{Name: "master"}, {Name: "dev"}}}
	if rp.branchColor(gitx.Branch{Name: "master"}) != cYellow || rp.branchColor(gitx.Branch{Name: "dev"}) != cCyan {
		t.Error("master as the main branch")
	}
}
