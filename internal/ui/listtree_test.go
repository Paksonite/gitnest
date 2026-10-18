package ui

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paksonite/gitnest/internal/scan"
	"github.com/Paksonite/gitnest/internal/state"
)

func treeApp(t *testing.T, roots []string, paths ...string) *App {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	a := testApp(paths...)
	for i := range a.rs.all {
		a.rs.all[i].Name = filepath.Base(a.rs.all[i].Path)
	}
	a.cfg.Roots = roots
	a.rs.sortMode = sortName
	a.rs.tree = true
	a.rebuildView("")
	return a
}

func rowsText(a *App) []string {
	var out []string
	for _, r := range a.rs.rows {
		ind := strings.Repeat("  ", r.depth)
		if r.repo < 0 {
			out = append(out, ind+r.name+"/")
		} else {
			out = append(out, ind+a.rs.all[r.repo].Name)
		}
	}
	return out
}

func TestRepoTree(t *testing.T) {
	a := treeApp(t, []string{"/r"},
		"/r/forgejo.int/roles/role-a", "/r/forgejo.int/terraform-x", "/r/gitea.ps/org-core-dns",
		"/r/standalone", "/r/github.com/fabric/cloud-fabric")
	a.rs.all[1].Changes = 2
	a.rebuildView("")
	want := []string{
		"forgejo.int/", "  roles/", "    role-a", "  terraform-x",
		"gitea.ps/", "  org-core-dns",
		"github.com/fabric/", "  cloud-fabric", // single-directory chain merged
		"standalone",
	}
	if got := rowsText(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("tree:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if r := a.rs.rows[0]; r.count != 2 || r.dirty != 1 {
		t.Errorf("forgejo counts: %+v", r)
	}

	// enter on a directory folds it, and the cursor stays on it
	a.reposKey(tea.KeyMsg{Type: tea.KeyEnter})
	if got := rowsText(a); got[1] != "gitea.ps/" || a.selectedKey() != "d:forgejo.int" {
		t.Errorf("collapsed: %v key %q", got, a.selectedKey())
	}
	// the fold is remembered for the next start
	if st := state.Load(); !reflect.DeepEqual(st.Collapsed, []string{"forgejo.int"}) {
		t.Errorf("saved folds: %v", st.Collapsed)
	}
	if b := newApp(context.Background(), a.cfg, false, "", ""); !b.rs.collapsed["forgejo.int"] {
		t.Errorf("a new start lost the fold: %v", b.rs.collapsed)
	}
	// space on a directory marks everything below it
	a.reposKey(key(" "))
	if !a.rs.marked["/r/forgejo.int/roles/role-a"] || !a.rs.marked["/r/forgejo.int/terraform-x"] || len(a.rs.marked) != 2 {
		t.Errorf("marked: %v", a.rs.marked)
	}
	// searching expands everything that matches (and merges the now
	// single-directory chain)
	a.rs.filter = "role"
	a.rebuildView("")
	if got := rowsText(a); !reflect.DeepEqual(got, []string{"forgejo.int/roles/", "  role-a"}) {
		t.Errorf("search: %v", got)
	}
	// ` groups by remote next, then switches back to a flat list
	a.rs.filter = ""
	a.reposKey(key("`"))
	if !a.rs.tree || a.rs.group != groupRemote || a.st.ListGroup != "remote" {
		t.Errorf("remote: tree=%v group=%d saved %q", a.rs.tree, a.rs.group, a.st.ListGroup)
	}
	a.reposKey(key("`"))
	if a.rs.tree || len(a.rs.rows) != 5 || !a.st.ListTree == a.rs.tree {
		t.Errorf("flat: tree=%v rows=%d", a.rs.tree, len(a.rs.rows))
	}
}

func TestRepoTreeByRemote(t *testing.T) {
	a := treeApp(t, []string{"/r"}, "/r/api", "/r/web", "/r/infra", "/r/dns", "/r/scratch")
	urls := map[string]string{
		"api": "git@github.com:ann/api.git", "web": "https://github.com/ann/web",
		"infra": "git@gitlab.com:platform/infra.git", "dns": "https://gitlab.com/platform/net/dns.git",
	}
	for i := range a.rs.all {
		a.rs.all[i].RemoteURL = urls[a.rs.all[i].Name]
	}
	a.rs.group = groupRemote
	a.rebuildView("")
	want := []string{"github.com/ann/", "  api", "  web", "gitlab.com/platform/", "  net/", "    dns", "  infra", "local/", "  scratch"}
	if got := rowsText(a); !reflect.DeepEqual(got, want) {
		t.Errorf("by remote:\n%v\nwant\n%v", got, want)
	}
	// Folding a remote group doesn't fold a folder of the same name.
	a.rs.cur = 0
	a.reposKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !a.rs.collapsed["remote:github.com/ann"] || len(a.rs.rows) != len(want)-2 {
		t.Errorf("fold: %v rows %d", a.rs.collapsed, len(a.rs.rows))
	}
	// Space on a group marks its repositories.
	a.reposKey(key(" "))
	if !a.rs.marked["/r/api"] || !a.rs.marked["/r/web"] || len(a.rs.marked) != 2 {
		t.Errorf("marked: %v", a.rs.marked)
	}
}

func TestRepoTreeSeveralRoots(t *testing.T) {
	a := treeApp(t, []string{"/r", "/w"}, "/r/x/a", "/w/b")
	got := rowsText(a)
	if len(got) != 4 || !strings.HasSuffix(got[0], "r/x/") || !strings.HasSuffix(got[2], "w/") {
		t.Errorf("roots: %v", got)
	}
}

func TestTreeSearchLandsOnRepository(t *testing.T) {
	a := treeApp(t, []string{"/r"}, "/r/forgejo/org-core-dns", "/r/gitea/x/other")
	a.reposKey(key("/"))
	for _, k := range []string{"d", "n", "s"} {
		a.reposKey(key(k))
	}
	if a.selectedPath() != "/r/forgejo/org-core-dns" {
		t.Errorf("cursor on %q (row %d of %v)", a.selectedKey(), a.rs.cur, rowsText(a))
	}
}

func TestSearchRanksAndIgnoresScatteredPaths(t *testing.T) {
	a := treeApp(t, []string{"/r"}, "/r/mwadecki-k8s", "/r/team/client-craftware", "/r/team/org-core-dns", "/r/team/wadec-tools")
	for i := range a.rs.all {
		a.rs.all[i].RemoteURL = "git@gitea.devops.ps.oktawave.com:team/" + a.rs.all[i].Name + ".git"
	}
	a.rs.filter = "wadec"
	a.rebuildView("")
	// Only names with "wadec" in them: the letters scattered over the host
	// (gitea.devops.ps.oktawave.com) don't count.
	if got := rowsText(a); !reflect.DeepEqual(got, []string{"team/", "  wadec-tools", "mwadecki-k8s"}) {
		t.Errorf("tree search: %v", got)
	}
	// A contiguous match in the host still finds repositories.
	a.rs.filter = "oktawave"
	a.rebuildView("")
	if len(a.rs.view) != 4 {
		t.Errorf("host search: %d", len(a.rs.view))
	}
	// The better name match comes first even when it is in the root.
	a.rs.filter = "mwadecki"
	a.rebuildView("")
	if got := rowsText(a); !reflect.DeepEqual(got, []string{"mwadecki-k8s"}) {
		t.Errorf("exact: %v", got)
	}
}

func TestGroupSyncMarks(t *testing.T) {
	a := treeApp(t, []string{"/r"}, "/r/ok/a", "/r/ok/b", "/r/mixed/c", "/r/mixed/d", "/r/none/e")
	for i := range a.rs.all {
		r := &a.rs.all[i]
		switch r.Name {
		case "a", "b", "d":
			r.Upstream = true
		case "c":
			r.Upstream, r.Ahead, r.Behind = true, 1, 2
		}
	}
	a.rebuildView("")
	got := map[string]string{}
	for _, r := range a.rs.rows {
		if r.repo < 0 {
			got[r.name] = dirRow(r, false, 80, 30).plain
		}
	}
	// The counts follow the name; the state sits in the status column (at 2+30+1+16+1).
	if !strings.HasPrefix(got["ok"][strings.Index(got["ok"], "ok/"):], "ok/ 2 ") || !strings.HasSuffix(strings.TrimSpace(got["ok"]), "✓") ||
		[]rune(got["ok"])[50] != '✓' || !strings.Contains(got["mixed"], "↑1 ↓1") || strings.ContainsAny(got["none"], "✓↑↓") {
		t.Errorf("group rows: %q", got)
	}
	card := strings.Join(groupCard("ok", []string{"/r/ok/a", "/r/ok/b"}, map[string]scan.Repo{"/r/ok/a": a.rs.all[0], "/r/ok/b": a.rs.all[1]}, 60, 10), "\n")
	if !strings.Contains(card, "✓ all up to date") {
		t.Errorf("group card: %s", card)
	}
}
