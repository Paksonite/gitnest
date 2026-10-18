package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Paksonite/gitnest/internal/state"
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
