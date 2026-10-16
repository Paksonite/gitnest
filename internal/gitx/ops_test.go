package gitx

// Integration tests for the everyday operations: files, branches, stash, remotes.

import (
	"regexp"
	"strings"
	"testing"
)

var ansi = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string { return ansi.ReplaceAllString(s, "") }

func TestBasicFileOps(t *testing.T) {
	dir := newRepo(t)
	commit(t, dir, "a.txt", "a\n", "first")
	writeFile(t, dir, "a.txt", "changed\n")
	writeFile(t, dir, "new.txt", "new\n")
	files, err := Files(bg, dir)
	if err != nil || len(files) != 2 {
		t.Fatalf("Files = %+v %v", files, err)
	}
	if d := DiffFile(bg, dir, files[0]); !strings.Contains(stripANSI(d), "+changed") {
		t.Errorf("diff: %q", d)
	}
	if d := DiffFile(bg, dir, File{Path: "new.txt", X: '?', Y: '?'}); !strings.Contains(stripANSI(d), "+new") {
		t.Errorf("untracked diff: %q", d)
	}
	if err := Stage(bg, dir, "a.txt"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "diff", "--cached", "--name-only"); got != "a.txt" {
		t.Errorf("staged: %q", got)
	}
	if err := Unstage(bg, dir, "a.txt"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "diff", "--cached", "--name-only"); got != "" {
		t.Errorf("after unstage: %q", got)
	}
	if err := StageAll(bg, dir); err != nil {
		t.Fatal(err)
	}
	if err := UnstageAll(bg, dir); err != nil {
		t.Fatal(err)
	}
	if err := Discard(bg, dir, File{Path: "a.txt", X: ' ', Y: 'M'}); err != nil {
		t.Fatal(err)
	}
	if err := Discard(bg, dir, File{Path: "new.txt", X: '?', Y: '?'}); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "status", "--porcelain"); got != "" {
		t.Errorf("after discard: %q", got)
	}
	cs, err := Log(bg, dir, 10)
	if err != nil || len(cs) != 1 || cs[0].Subject != "first" {
		t.Errorf("Log = %+v %v", cs, err)
	}
	if s := Show(bg, dir, cs[0].Hash); !strings.Contains(s, "first") {
		t.Errorf("Show: %q", s)
	}
	if p := Preview(bg, dir); !strings.Contains(p, "first") {
		t.Errorf("Preview: %q", p)
	}
}

func TestBranchOps(t *testing.T) {
	dir := newRepo(t)
	commit(t, dir, "a.txt", "a\n", "first")
	if err := NewBranch(bg, dir, "feat", "main"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "branch", "--show-current"); got != "feat" {
		t.Errorf("current %q", got)
	}
	commit(t, dir, "b.txt", "b\n", "feat work")
	git(t, dir, "checkout", "-q", "main")
	if err := RenameBranch(bg, dir, "feat", "feature"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteBranch(bg, dir, "feature", false); err == nil {
		t.Error("unmerged branch deleted without force")
	}
	if err := DeleteBranch(bg, dir, "feature", true); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "branch", "--list", "feature"); got != "" {
		t.Errorf("still there: %q", got)
	}
}

func TestStashOps(t *testing.T) {
	dir := newRepo(t)
	commit(t, dir, "a.txt", "a\n", "first")
	for _, c := range []string{"one\n", "two\n"} {
		writeFile(t, dir, "a.txt", c)
		if err := StashPush(bg, dir, "wip "+strings.TrimSpace(c)); err != nil {
			t.Fatal(err)
		}
	}
	ss, err := Stashes(bg, dir)
	if err != nil || len(ss) != 2 || !strings.Contains(ss[0].Msg, "wip two") {
		t.Fatalf("Stashes = %+v %v", ss, err)
	}
	if s := StashShow(bg, dir, "stash@{1}"); !strings.Contains(stripANSI(s), "+one") {
		t.Errorf("StashShow: %q", s)
	}
	if err := StashApply(bg, dir, "stash@{1}"); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "checkout", "-q", "--", "a.txt")
	if err := StashPop(bg, dir, "stash@{0}"); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "checkout", "-q", "--", "a.txt")
	if err := StashDrop(bg, dir, "stash@{0}"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "stash", "list"); got != "" {
		t.Errorf("stashes left: %q", got)
	}
}

func TestRemoteOps(t *testing.T) {
	seed := newRepo(t)
	commit(t, seed, "a.txt", "a\n", "first")
	origin := t.TempDir()
	git(t, origin, "clone", "-q", "--bare", seed, ".")
	dir := t.TempDir()
	git(t, dir, "clone", "-q", origin, ".")
	commit(t, dir, "c.txt", "c\n", "local")
	if err := Push(bg, dir, false, true); err != nil {
		t.Fatal(err)
	}
	if got, want := git(t, origin, "rev-parse", "main"), git(t, dir, "rev-parse", "HEAD"); got != want {
		t.Errorf("origin main %s, want %s", got, want)
	}
	git(t, dir, "checkout", "-q", "-b", "topic")
	commit(t, dir, "t.txt", "t\n", "topic")
	if err := Push(bg, dir, false, false); err != nil { // sets the upstream
		t.Fatal(err)
	}
	if got := git(t, dir, "rev-parse", "--abbrev-ref", "topic@{u}"); got != "origin/topic" {
		t.Errorf("upstream %q", got)
	}
	other := t.TempDir()
	git(t, other, "clone", "-q", origin, ".")
	tip := commit(t, other, "o.txt", "o\n", "remote work")
	git(t, other, "push", "-q", "origin", "main")
	git(t, dir, "checkout", "-q", "main")
	if err := Fetch(bg, dir); err != nil {
		t.Fatal(err)
	}
	if err := Pull(bg, dir, false); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "rev-parse", "HEAD"); got != tip {
		t.Errorf("after pull HEAD %s want %s", got, tip)
	}
	if err := DeleteRemoteBranch(bg, dir, "origin/topic"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, origin, "branch", "--list", "topic"); got != "" {
		t.Errorf("remote branch left: %q", got)
	}
}
