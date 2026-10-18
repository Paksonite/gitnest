package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStateMergesInsteadOfOverwriting(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	// Two gitnests running at once, each with its own copy in memory.
	a, b := Load(), Load()
	a.SetListTree(true)
	b.SetScreen("list")
	a.SetListGroup("remote")
	b.Touch("/r/one")
	a.Touch("/r/two")
	b.SetFavorites([]string{"/r/two", "/r/one"})

	got := Load()
	if !got.ListTree || got.Screen != "list" || got.ListGroup != "remote" ||
		got.Recent["/r/one"] == 0 || got.Recent["/r/two"] == 0 || len(got.Favorites) != 2 || got.Favorites[0] != "/r/one" {
		t.Fatalf("state: %+v", got)
	}
}

func TestStateKeepsUnknownKeys(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	p := statePath()
	os.MkdirAll(filepath.Dir(p), 0o755)
	// A newer gitnest wrote a key this one doesn't know; an older one kept
	// the screen in recent.json.
	os.WriteFile(p, []byte(`{"recent":{},"from_the_future":42,"screen":"dashboard"}`), 0o644)
	s := Load()
	if s.Screen != "dashboard" {
		t.Errorf("screen from recent.json: %q", s.Screen)
	}
	s.SetPaneLayout("grid")
	var obj map[string]json.RawMessage
	b, _ := os.ReadFile(p)
	if json.Unmarshal(b, &obj) != nil || string(obj["from_the_future"]) != "42" || string(obj["pane_layout"]) != `"grid"` {
		t.Errorf("recent.json: %s", b)
	}
	// settings.json wins once it has the key.
	s.SetScreen("list")
	if Load().Screen != "list" {
		t.Error("settings.json should win")
	}
	if leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(p), ".state-*")); len(leftovers) != 0 {
		t.Errorf("temporary files left: %v", leftovers)
	}
}
