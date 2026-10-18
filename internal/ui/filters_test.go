package ui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paksonite/gitnest/v2/internal/gitx"
)

func filtersApp() *App {
	a := testApp("clean", "dirty", "stash", "ahead", "behind", "local")
	infos := map[string]gitx.Info{
		"clean":  {Upstream: true, RemoteURL: "git@github.com:ann/clean.git"},
		"dirty":  {Upstream: true, Changes: 2, RemoteURL: "git@github.com:ann/dirty.git"},
		"stash":  {Upstream: true, Stashes: 1, RemoteURL: "https://gitlab.com/p/stash"},
		"ahead":  {Upstream: true, Ahead: 1, RemoteURL: "https://gitlab.com/p/ahead"},
		"behind": {Upstream: true, Behind: 3, RemoteURL: "git@github.com:ann/behind.git"},
		"local":  {},
	}
	for i := range a.rs.all {
		a.rs.all[i].Info = infos[a.rs.all[i].Path]
	}
	a.rs.sortMode = sortName
	a.w, a.h = 120, 40
	a.rebuildView("")
	return a
}

// menuPick moves the menu cursor to the option labelled label and chooses it.
func menuPick(t *testing.T, a *App, label string) {
	t.Helper()
	for i, it := range a.menu.items {
		if it.label == label && !it.header {
			a.menu.cur = i
			a.onKey(tea.KeyMsg{Type: tea.KeyEnter})
			return
		}
	}
	t.Fatalf("no menu option %q", label)
}

func TestFilters(t *testing.T) {
	a := filtersApp()
	a.onKey(key("v"))
	if a.menu == nil {
		t.Fatal("F should open the filters")
	}
	v := a.View()
	for _, want := range []string{"View & filters", "Status filter", "Behind remote", "Remote filter", "github.com", "gitlab.com", "local"} {
		if !strings.Contains(v, want) {
			t.Errorf("filters menu lacks %q", want)
		}
	}
	menuPick(t, a, "Behind remote")
	if a.menu == nil || !reflect.DeepEqual(listed(a), []string{"behind"}) {
		t.Fatalf("behind: menu %v, listed %v", a.menu != nil, listed(a))
	}
	menuPick(t, a, "Any status")
	menuPick(t, a, "gitlab.com")
	if !reflect.DeepEqual(listed(a), []string{"ahead", "stash"}) {
		t.Fatalf("gitlab.com: %v", listed(a))
	}
	menuPick(t, a, "Stashed changes") // filters combine
	if !reflect.DeepEqual(listed(a), []string{"stash"}) {
		t.Fatalf("gitlab.com + stashed: %v", listed(a))
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyEsc})
	if head := strings.SplitN(a.View(), "\n", 2)[0]; a.menu != nil || !strings.Contains(head, " stashed ") || !strings.Contains(head, " gitlab.com ") {
		t.Errorf("header should name the filters: %q", strings.SplitN(a.View(), "\n", 2)[0])
	}
	// d toggles dirty on top of the host filter, and off again.
	a.rs.host = ""
	a.onKey(key("d"))
	if !reflect.DeepEqual(listed(a), []string{"dirty"}) {
		t.Fatalf("d: %v", listed(a))
	}
	a.onKey(key("d"))
	if len(listed(a)) != 6 {
		t.Fatalf("d again: %v", listed(a))
	}
	a.onKey(key("v"))
	menuPick(t, a, "No remote")
	if !reflect.DeepEqual(listed(a), []string{"local"}) {
		t.Fatalf("no remote: %v", listed(a))
	}
}
