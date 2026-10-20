package ui

import (
	"context"
	"github.com/charmbracelet/x/ansi"
	"os"
	"path/filepath"

	"github.com/Paksonite/gitnest/v2/internal/config"
	"reflect"
	"strings"
	"testing"

	"github.com/Paksonite/gitnest/v2/internal/state"
)

func TestViewsAndFavorites(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	a := filtersApp()
	a.w, a.h = 120, 40

	// * stars the selected repository; v → Favorites only keeps it alone.
	a.rs.cur = 1 // "behind" (sorted by name: ahead, behind, clean…)
	a.onKey(key("*"))
	if !a.sh.fav["behind"] || !reflect.DeepEqual(state.Load().Favorites, []string{"behind"}) {
		t.Fatalf("favorites: %v saved %v", a.sh.fav, state.Load().Favorites)
	}
	if !strings.Contains(a.View(), "behind ★") {
		t.Error("starred repository should show ★")
	}
	a.onKey(key("v"))
	if a.menu == nil || !strings.Contains(a.View(), "View & filters") {
		t.Fatal("v should open the sort & view chooser")
	}
	menuPick(t, a, "Favorites only")
	if !reflect.DeepEqual(listed(a), []string{"behind"}) {
		t.Fatalf("favorites only: %v", listed(a))
	}
	menuPick(t, a, "Every repository")
	menuPick(t, a, "Grouped by remote")
	if !a.rs.tree || a.rs.group != groupRemote {
		t.Fatalf("view: tree %v group %d", a.rs.tree, a.rs.group)
	}
	menuPick(t, a, "Last commit")
	if a.rs.sortMode != sortCommit {
		t.Fatalf("sort: %d", a.rs.sortMode)
	}
	menuPick(t, a, "Flat list")
	a.menu = nil

	// Marked repositories are starred together; * on a starred one unstars.
	a.rs.marked["clean"], a.rs.marked["dirty"] = true, true
	a.onKey(key("*"))
	if !a.sh.fav["clean"] || !a.sh.fav["dirty"] || len(a.sh.fav) != 3 {
		t.Fatalf("marked favorites: %v", a.sh.fav)
	}
	a.onKey(key("*"))
	if len(a.sh.fav) != 1 {
		t.Fatalf("unstar: %v", a.sh.fav)
	}
}

// The sort order chosen in the view menu is there again on the next start,
// as the list's layout is; the filters are not (one that hides every
// repository would greet you with an empty list).
func TestViewSortIsRememberedFiltersAreNot(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	a := filtersApp()
	a.onKey(key("v"))
	menuPick(t, a, "Last commit")
	menuPick(t, a, "Favorites only")
	menuPick(t, a, "No remote")
	menuPick(t, a, "github.com")
	a.menu = nil
	if saved := state.Load().ListView; saved != (state.ListView{Sort: "last commit"}) {
		t.Fatalf("saved %+v", saved)
	}
	b := newApp(context.Background(), config.Config{}, true, "", "") // the next start
	if b.rs.sortMode != sortCommit || b.rs.listFiltered() {
		t.Errorf("next start: sort %d, filtered %v", b.rs.sortMode, b.rs.listFiltered())
	}

	// gitnest 2.0.7 saved the filters too: a list hidden by them ("no
	// remote") must not come back.
	dir := filepath.Join(os.Getenv("XDG_STATE_HOME"), "gitnest")
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"list_view":{"sort":"name","status":"no remote","host":"x","favorites_only":true}}`), 0o644)
	c := newApp(context.Background(), config.Config{}, true, "", "")
	if c.rs.sortMode != sortName || c.rs.listFiltered() {
		t.Errorf("2.0.7 settings: sort %d, filtered %v", c.rs.sortMode, c.rs.listFiltered())
	}

	// Names this version doesn't know keep the default.
	d := testApp()
	d.loadListView(state.ListView{Sort: "size"})
	if d.rs.sortMode != sortRecent {
		t.Errorf("unknown sort: %d", d.rs.sortMode)
	}
}

// When the filters hide every repository the list says so, and esc (with
// no search to clear) clears them.
func TestEscClearsFiltersThatHideEverything(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	a := filtersApp()
	a.rs.status, a.rs.host = stNoRemote, "github.com"
	a.rebuildView("")
	if len(a.rs.view) != 0 {
		t.Fatalf("expected an empty list, got %v", listed(a))
	}
	if v := strings.Join(strings.Fields(strings.ReplaceAll(ansi.Strip(a.viewRepos()), "│", " ")), " "); !strings.Contains(v, "The filters hide all 6 repositories (no remote · github.com) — esc clears them") {
		t.Errorf("empty list should explain why:\n%s", v)
	}
	a.onKey(key("esc"))
	if a.rs.listFiltered() || len(a.rs.view) != 6 {
		t.Errorf("esc: filtered %v, %d shown", a.rs.listFiltered(), len(a.rs.view))
	}
}
