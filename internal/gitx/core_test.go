package gitx

import (
	"os"
	"regexp"
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

func TestThemeColorsReachGit(t *testing.T) {
	dir := newRepo(t)
	commit(t, dir, "a.txt", "a\n", "first")
	writeFile(t, dir, "a.txt", "b\n")
	SetColors(map[string]string{"color.diff.new": "#00ff00", "color.diff.old": "", "color.diff.frag": "5"})
	defer SetColors(nil)
	d := DiffFile(bg, dir, File{Path: "a.txt", X: ' ', Y: 'M'})
	if !strings.Contains(d, "\x1b[38;2;0;255;0m") { // 24-bit green for added lines
		t.Errorf("theme colour not applied:\n%q", d)
	}
	if !strings.Contains(d, "\x1b[35m") { // ANSI 5 for the hunk header
		t.Errorf("ANSI colour not applied:\n%q", d)
	}
	SetColors(nil)
	if d := DiffFile(bg, dir, File{Path: "a.txt", X: ' ', Y: 'M'}); strings.Contains(d, "38;2;0;255;0") {
		t.Error("colours not reset")
	}
}

func TestCommitKindsAndGraph(t *testing.T) {
	dir, _ := mergeRepo(t) // base, side s1, main m1, merge, m2
	h := commit(t, dir, "f.txt", "f\n", "fixup! m2")
	git(t, dir, "revert", "--no-edit", h)
	cs, err := Log(bg, dir, 50)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]CommitKind{}
	for _, c := range cs {
		kinds[c.Subject] = c.Kind()
		if c.Graph == "" {
			t.Errorf("%s has no graph", c.Subject)
		}
	}
	want := map[string]CommitKind{"base": KindRoot, "merge side": KindMerge, "fixup! m2": KindFixup, "m2": KindCommit}
	for subj, k := range want {
		if kinds[subj] != k {
			t.Errorf("%s: kind %d, want %d", subj, kinds[subj], k)
		}
	}
	if kinds[`Revert "fixup! m2"`] != KindRevert {
		t.Errorf("revert kind: %v", kinds)
	}
	var sideGraph string
	for _, c := range cs {
		if c.Subject == "s1" {
			sideGraph = c.Graph
		}
	}
	if !strings.Contains(sideGraph, "|") {
		t.Errorf("side branch commit graph %q should show the parallel line", sideGraph)
	}
}

func TestReflogKindsAndChanges(t *testing.T) {
	dir := newRepo(t)
	commit(t, dir, "a.txt", "a\n", "first")
	commit(t, dir, "b.txt", "b\n", "second")
	git(t, dir, "commit", "-q", "--amend", "-m", "second, amended")
	git(t, dir, "checkout", "-q", "-b", "other")
	git(t, dir, "reset", "-q", "--hard", "HEAD~1")

	es, err := Reflog(bg, dir, 20)
	if err != nil {
		t.Fatal(err)
	}
	var kinds []RefKind
	for _, e := range es {
		kinds = append(kinds, e.Kind())
	}
	want := []RefKind{RefReset, RefCheckout, RefAmend, RefCommit, RefInitial}
	if len(kinds) != len(want) {
		t.Fatalf("entries %+v", es)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Errorf("entry %d (%s): kind %d, want %d", i, es[i].Action, kinds[i], want[i])
		}
	}
	if es[2].Message != "second, amended" || es[0].Selector != "HEAD@{0}" {
		t.Errorf("entries: %+v", es[:3])
	}

	// The amend replaced "second" with "second, amended": both sides show.
	ch := ReflogChanges(bg, dir, es[2], es[3].Hash)
	plain := ansiRe.ReplaceAllString(ch, "")
	if !strings.Contains(plain, "< ") || !strings.Contains(plain, "second") || !strings.Contains(plain, "> ") || !strings.Contains(plain, "second, amended") {
		t.Errorf("changes:\n%s", plain)
	}
	if ch := ReflogChanges(bg, dir, es[1], es[2].Hash); !strings.Contains(ch, "same commit") {
		t.Errorf("checkout without moving:\n%s", ch)
	}
}

func TestResetHardIsUndoable(t *testing.T) {
	dir, h := linearRepo(t)
	if err := ResetHard(bg, dir, h["b"], "reset to b"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "rev-parse", "HEAD"); got != h["b"] {
		t.Fatalf("HEAD %s", got)
	}
	if err := Undo(bg, dir); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "rev-parse", "HEAD"); got != h["d"] {
		t.Errorf("after undo HEAD %s, want %s", got, h["d"])
	}
	writeFile(t, dir, "a.txt", "dirty\n")
	if err := ResetHard(bg, dir, h["a"], "reset"); err == nil {
		t.Error("reset with uncommitted changes accepted")
	}
}

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")
