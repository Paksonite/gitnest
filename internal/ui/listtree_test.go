package ui

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	// ` switches back to a flat list
	a.rs.filter = ""
	a.reposKey(key("`"))
	if a.rs.tree || len(a.rs.rows) != 5 || !a.st.ListTree == a.rs.tree {
		t.Errorf("flat: tree=%v rows=%d", a.rs.tree, len(a.rs.rows))
	}
}

func TestRepoTreeSeveralRoots(t *testing.T) {
	a := treeApp(t, []string{"/r", "/w"}, "/r/x/a", "/w/b")
	got := rowsText(a)
	if len(got) != 4 || !strings.HasSuffix(got[0], "r/x/") || !strings.HasSuffix(got[2], "w/") {
		t.Errorf("roots: %v", got)
	}
}
