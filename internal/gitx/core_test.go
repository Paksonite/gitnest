package gitx

import (
	"os"
	"path/filepath"
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

func TestInfoFromGitDirFiles(t *testing.T) {
	dir, _ := conflictRepo(t)
	git(t, dir, "remote", "add", "upstream", "https://example.com/up.git")
	git(t, dir, "remote", "add", "origin", "git@example.com:me/repo.git")
	wt := filepath.Join(t.TempDir(), "wt")
	git(t, dir, "worktree", "add", "-q", "-b", "wt-branch", wt, "main")
	if err := Merge(bg, wt, "other"); err == nil {
		t.Fatal("merge should conflict")
	}

	gd, common, ok := gitDirs(wt)
	if !ok || !strings.Contains(gd, filepath.Join(".git", "worktrees")) || filepath.Base(common) != ".git" {
		t.Fatalf("gitDirs(worktree) = %q %q %v", gd, common, ok)
	}
	info, err := GetInfo(bg, wt)
	if err != nil {
		t.Fatal(err)
	}
	if info.State != StateMerging || info.RemoteURL != "git@example.com:me/repo.git" {
		t.Errorf("worktree info: state %v remote %q", info.State, info.RemoteURL)
	}
	info, _ = GetInfo(bg, dir)
	if info.State != StateNone || info.RemoteURL != "git@example.com:me/repo.git" {
		t.Errorf("main info: state %v remote %q", info.State, info.RemoteURL)
	}
	if _, _, ok := gitDirs(t.TempDir()); ok {
		t.Error("gitDirs on a plain directory")
	}
}

func TestRemoteFromConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config")
	write := func(s string) {
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("[core]\n\turl = nope\n[remote \"backup\"]\n\turl = /srv/backup.git\n[remote \"origin\"]\n\tURL = \"ssh://git@host/x.git\"\n")
	if got := remoteFromConfig(p); got != "ssh://git@host/x.git" {
		t.Errorf("origin: %q", got)
	}
	write("[remote \"backup\"]\n\turl = /srv/backup.git\n")
	if got := remoteFromConfig(p); got != "/srv/backup.git" {
		t.Errorf("first: %q", got)
	}
	if got := remoteFromConfig(filepath.Join(t.TempDir(), "none")); got != "" {
		t.Errorf("missing: %q", got)
	}
}
