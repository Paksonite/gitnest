package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/Paksonite/gitnest/internal/gitx"
	"github.com/Paksonite/gitnest/internal/scan"
)

func TestDetailsCard(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	r := scan.Repo{Path: "/r/platform", Name: "platform", Info: gitx.Info{
		Branch: "main", Upstream: true, Ahead: 1, Changes: 1, Stashes: 2,
		RemoteURL: "git@gitlab.com:platform/platform.git",
	}}
	d := &gitx.Details{
		Upstream: "origin/main", Tag: "v1.2.0",
		Description: "Infrastructure and tooling for our platform, with a description long enough to wrap onto a second line here.",
		Changes:     []string{" M src/main.go"},
		Commits: []gitx.DetailsCommit{
			{Short: "a3f9e2", Subject: "Add health check endpoint", Author: "Ann Example", Unix: now.Add(-21 * 24 * time.Hour).Unix(), Refs: "HEAD -> main"},
			{Short: "7c1d4e", Subject: "Refactor server initialization", Author: "Bob Example", Unix: now.Add(-40 * 24 * time.Hour).Unix()},
		},
	}
	card := strings.Join(detailsCard(r, "", d, 70, 40, now), "\n")
	for _, want := range []string{"platform", "main", "gitlab.com/platform/platform", "Path", "/r/platform", "Upstream",
		"origin/main", "Stashes", "Last commit", "3w ago · Ann Example", "Tag", "v1.2.0", "About", "Infrastructure",
		"Changes", "src/main.go", "Recent commits", "a3f9e2 AE", "7c1d4e BE", "(main) Add health check endpoint"} {
		if !strings.Contains(card, want) {
			t.Errorf("card lacks %q:\n%s", want, card)
		}
	}
	for i, l := range strings.Split(card, "\n") {
		if w := lipglossWidth(l); w > 70 {
			t.Errorf("line %d is %d wide", i, w)
		}
	}
	if got := compactRefs("HEAD -> main, origin/main, origin/HEAD, tag: v1.2"); got != "main, origin/main, v1.2" {
		t.Errorf("compactRefs: %q", got)
	}
	if got := detailsCard(r, "", d, 70, 5, now); len(got) != 5 {
		t.Errorf("card not clipped to the height: %d lines", len(got))
	}
	if loading := strings.Join(detailsCard(r, "", nil, 70, 40, now), "\n"); !strings.Contains(loading, "loading") {
		t.Errorf("no loading note: %s", loading)
	}
}

func TestActionsOpenInPanes(t *testing.T) {
	s := shellOf(t, 220, 50, "/r/app", "/r/lib", "/r/web")
	a := s.focused().app
	a.rs.marked["/r/app"], a.rs.marked["/r/web"] = true, true
	press(t, s, "a")
	if a.menu == nil || !strings.Contains(a.menu.note, "2 marked: app, web") {
		t.Fatalf("actions menu: %+v", a.menu)
	}
	press(t, s, "n")
	if len(s.panes) != 3 || s.panes[1].app.rp.dir != "/r/app" || s.panes[2].app.rp.dir != "/r/web" || s.focus != 2 {
		t.Fatalf("panes: %d, focus %d", len(s.panes), s.focus)
	}
	// U from the menu unmarks everything.
	s.focus = 0
	press(t, s, "a", "U")
	if len(a.rs.marked) != 0 || a.menu != nil {
		t.Errorf("unmark: %v", a.rs.marked)
	}
}
