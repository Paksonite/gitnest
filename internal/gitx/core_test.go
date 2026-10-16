package gitx

import (
	"strings"
	"testing"
)

// cloneRepo returns a clone of a fresh repo with one commit on main, plus the
// path of the origin it was cloned from.
func cloneRepo(t *testing.T) (dir, origin string) {
	t.Helper()
	origin = newRepo(t)
	commit(t, origin, "a.txt", "a\n", "first")
	git(t, origin, "branch", "feature")
	dir = t.TempDir()
	git(t, dir, "clone", "-q", origin, ".")
	return dir, origin
}

func TestCleanMessage(t *testing.T) {
	cases := map[string]string{
		"\n\nsubject  \n\nbody line \t\n\n": "subject\n\nbody line",
		"one\r\ntwo":                        "one\ntwo",
		"  \n \n":                           "",
	}
	for in, want := range cases {
		if got := CleanMessage(in); got != want {
			t.Errorf("CleanMessage(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCommitMultiline(t *testing.T) {
	dir := newRepo(t)
	writeFile(t, dir, "a.txt", "a\n")
	git(t, dir, "add", "a.txt")
	if err := CommitMsg(bg, dir, "subject\n\nbody one\nbody two\n\n"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "log", "-1", "--format=%B"); got != "subject\n\nbody one\nbody two" {
		t.Errorf("message = %q", got)
	}
	if err := CommitMsg(bg, dir, " \n "); err == nil {
		t.Error("empty message accepted")
	}
}

func TestRewordMessage(t *testing.T) {
	dir, h := linearRepo(t)
	if err := RewordMessage(bg, dir, h["b"], "b2\n\nnew body"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "log", "-1", "--format=%B", "HEAD~2"); got != "b2\n\nnew body" {
		t.Errorf("older commit: %q", got)
	}
	if err := RewordMessage(bg, dir, "HEAD", "d2\n\nhead body"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "log", "-1", "--format=%B"); got != "d2\n\nhead body" {
		t.Errorf("HEAD: %q", got)
	}
	if err := RewordMessage(bg, dir, "HEAD", "\n \n"); err == nil {
		t.Error("empty message accepted")
	}
}

func TestCheckoutRemoteBranch(t *testing.T) {
	dir, _ := cloneRepo(t)
	// No local "feature" yet: creates it tracking origin/feature.
	if err := Checkout(bg, dir, Branch{Name: "origin/feature", Remote: true}); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "rev-parse", "--abbrev-ref", "HEAD@{u}"); got != "origin/feature" {
		t.Errorf("upstream = %q", got)
	}
	git(t, dir, "checkout", "-q", "main")
	// Local branch exists: just switch to it instead of failing.
	if err := Checkout(bg, dir, Branch{Name: "origin/feature", Remote: true}); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "branch", "--show-current"); got != "feature" {
		t.Errorf("current = %q", got)
	}
}

func TestUpstreamAndFastForward(t *testing.T) {
	dir, origin := cloneRepo(t)
	git(t, dir, "branch", "local-feature", "origin/feature")
	if err := SetUpstream(bg, dir, "local-feature", "origin/feature"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "rev-parse", "--abbrev-ref", "local-feature@{u}"); got != "origin/feature" {
		t.Errorf("upstream = %q", got)
	}

	// The remote branch moves ahead; fast-forward the non-checked-out branch.
	git(t, origin, "checkout", "-q", "feature")
	tip := commit(t, origin, "f.txt", "f\n", "feature work")
	git(t, dir, "fetch", "-q")
	local, _, err := Branches(bg, dir)
	if err != nil {
		t.Fatal(err)
	}
	var lf Branch
	for _, b := range local {
		if b.Name == "local-feature" {
			lf = b
		}
	}
	if err := FastForward(bg, dir, lf); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "rev-parse", "local-feature"); got != tip {
		t.Errorf("local-feature = %s, want %s", got, tip)
	}
	if got := git(t, dir, "branch", "--show-current"); got != "main" {
		t.Errorf("checkout changed to %q", got)
	}

	// Diverged: refused, branch untouched.
	git(t, dir, "checkout", "-q", "local-feature")
	mine := commit(t, dir, "mine.txt", "m\n", "local work")
	git(t, dir, "checkout", "-q", "main")
	commit(t, origin, "g.txt", "g\n", "more feature work")
	if err := FastForward(bg, dir, lf); err == nil {
		t.Error("diverged branch was fast-forwarded")
	}
	if got := git(t, dir, "rev-parse", "local-feature"); got != mine {
		t.Errorf("diverged branch moved to %s", got)
	}

	if err := UnsetUpstream(bg, dir, "local-feature"); err != nil {
		t.Fatal(err)
	}
	if err := FastForward(bg, dir, Branch{Name: "local-feature"}); err == nil || !strings.Contains(err.Error(), "no upstream") {
		t.Errorf("no upstream: err = %v", err)
	}
}

func TestStashStagedAndBranch(t *testing.T) {
	dir := newRepo(t)
	commit(t, dir, "a.txt", "a\n", "first")
	writeFile(t, dir, "a.txt", "staged\n")
	git(t, dir, "add", "a.txt")
	writeFile(t, dir, "b.txt", "untracked\n")
	if err := StashStaged(bg, dir, "only staged"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "status", "--porcelain"); got != "?? b.txt" {
		t.Errorf("after stash --staged: %q", got)
	}
	if err := StashBranch(bg, dir, "from-stash", "stash@{0}"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "branch", "--show-current"); got != "from-stash" {
		t.Errorf("current = %q", got)
	}
	if got := git(t, dir, "stash", "list"); got != "" {
		t.Errorf("stash not dropped: %q", got)
	}
	if got := strings.TrimSpace(git(t, dir, "show", ":a.txt")); got != "staged" {
		t.Errorf("staged content = %q", got)
	}
}
