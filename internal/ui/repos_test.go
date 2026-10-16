package ui

import (
	"context"
	"reflect"
	"testing"

	"github.com/Paksonite/gitnest/internal/gitx"
	"github.com/Paksonite/gitnest/internal/scan"
	"github.com/Paksonite/gitnest/internal/state"
)

func testApp(paths ...string) *App {
	a := &App{ctx: context.Background(), st: &state.State{Recent: map[string]int64{}}}
	a.rs.marked, a.rs.isNew = map[string]bool{}, map[string]bool{}
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
