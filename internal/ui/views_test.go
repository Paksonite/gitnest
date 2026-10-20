package ui

import (
	"context"

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

// Sorting and filtering chosen in the view menu (and d) are there again on
// the next start, as the list's layout is.
func TestViewChoicesAreRemembered(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	a := filtersApp()
	a.onKey(key("v"))
	menuPick(t, a, "Name")
	menuPick(t, a, "Favorites only")
	menuPick(t, a, "Behind remote")
	menuPick(t, a, "github.com")
	a.menu = nil

	saved := state.Load().ListView
	want := state.ListView{Sort: "name", FavoritesOnly: true, Status: "behind", Host: "github.com"}
	if saved != want {
		t.Fatalf("saved %+v, want %+v", saved, want)
	}
	b := newApp(context.Background(), config.Config{}, true, "", "") // the next start

	if b.rs.sortMode != sortName || !b.rs.favOnly || b.rs.status != stBehind || b.rs.host != "github.com" {
		t.Errorf("restored: sort %d fav %v status %d host %q", b.rs.sortMode, b.rs.favOnly, b.rs.status, b.rs.host)
	}

	// d (dirty only) is remembered too, and back to the defaults saves
	// nothing but the defaults.
	a.onKey(key("v"))
	menuPick(t, a, "Recently used")
	menuPick(t, a, "Every repository")
	menuPick(t, a, "Any remote")
	a.menu = nil
	a.onKey(key("d"))
	if got := state.Load().ListView; got != (state.ListView{Status: "dirty only"}) {
		t.Errorf("after d: %+v", got)
	}
	a.onKey(key("d"))
	if got := state.Load().ListView; got != (state.ListView{}) {
		t.Errorf("defaults: %+v", got)
	}

	// Names this version doesn't know keep the defaults.
	c := testApp()
	c.loadListView(state.ListView{Sort: "size", Status: "huge"})
	if c.rs.sortMode != sortRecent || c.rs.status != stAll {
		t.Errorf("unknown names: sort %d status %d", c.rs.sortMode, c.rs.status)
	}
}
