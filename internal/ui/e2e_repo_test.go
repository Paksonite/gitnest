package ui

// End-to-end tests of the repository view: every panel's keys on real
// repositories (see e2e_test.go for drive and press).

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Paksonite/gitnest/internal/gitx"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// commitFile writes a file and commits it with message msg.
func commitFile(t *testing.T, dir, name, content, msg string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, name), content)
	sh(t, dir, "add", "--", name)
	sh(t, dir, "commit", "-q", "-m", msg)
}

// openRepo opens dir in the repository view, as enter on the list does.
func openRepo(t *testing.T, a *App, dir string) {
	t.Helper()
	drive(t, a, a.openDir(dir))
	if a.scr != scrRepo || a.rp.dir != dir {
		t.Fatalf("open %s: scr=%v dir=%q", dir, a.scr, a.rp.dir)
	}
}

// refresh reloads the open repository, as the live refresh does, for
// changes made outside the app.
func refresh(t *testing.T, a *App) {
	t.Helper()
	a.rp.mainKey = ""
	drive(t, a, loadRepoData(a.ctx, a.rp.dir))
}

// selectRow moves the cursor of the focused panel with j/k until label
// (the row's plain text) contains want.
func selectRow(t *testing.T, a *App, want string) {
	t.Helper()
	a.rp.cur[a.rp.focus] = 0
	for i := 0; i < 200; i++ {
		items := a.panelItems(a.rp.focus)
		if c := a.rp.cur[a.rp.focus]; c < len(items) && strings.Contains(items[c].plain, want) {
			return
		}
		before := a.rp.cur[a.rp.focus]
		press(t, a, "down")
		if a.rp.cur[a.rp.focus] == before {
			break
		}
	}
	t.Fatalf("no row %q in panel %d: %v", want, a.rp.focus, a.panelItems(a.rp.focus))
}

func subjects(t *testing.T, dir string, n string) string {
	t.Helper()
	return sh(t, dir, "log", "-n", n, "--format=%s")
}

func TestE2EFilesPanel(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, _ := e2eSetup(t)
	lib := filepath.Join(root, "lib")
	writeFile(t, filepath.Join(lib, "lib.txt"), "lib\none\ntwo\n")
	writeFile(t, filepath.Join(lib, "new.txt"), "new\n")
	openRepo(t, a, lib)

	// space stages a file and unstages it again.
	selectRow(t, a, "new.txt")
	press(t, a, "space")
	if got := sh(t, lib, "diff", "--cached", "--name-only"); got != "new.txt" {
		t.Fatalf("staged: %q", got)
	}
	press(t, a, "space")
	if got := sh(t, lib, "diff", "--cached", "--name-only"); got != "" {
		t.Fatalf("after unstage: %q", got)
	}

	// Line staging: stage the first added line, switch to the staged side
	// and unstage it, then stage a range, then the whole hunk.
	selectRow(t, a, "lib.txt")
	press(t, a, "enter", "space")
	if got := sh(t, lib, "show", ":lib.txt"); got != "lib\none" {
		t.Fatalf("index after one line: %q", got)
	}
	press(t, a, "tab")
	if a.rp.staging == nil || !a.rp.staging.staged {
		t.Fatal("tab should show the staged side")
	}
	press(t, a, "space")
	if got := sh(t, lib, "show", ":lib.txt"); got != "lib" {
		t.Fatalf("index after unstaging the line: %q", got)
	}
	if a.rp.staging == nil || a.rp.staging.staged {
		t.Fatal("with nothing staged left the view goes back to the unstaged side")
	}
	press(t, a, "v", "down", "space")
	if got := sh(t, lib, "show", ":lib.txt"); got != "lib\none\ntwo" {
		t.Fatalf("index after a range: %q", got)
	}
	press(t, a, "esc")
	sh(t, lib, "reset", "-q")
	writeFile(t, filepath.Join(lib, "lib.txt"), "lib\none\ntwo\nthree\n")
	refresh(t, a)
	selectRow(t, a, "lib.txt")
	press(t, a, "enter", "a", "space")
	if got := sh(t, lib, "show", ":lib.txt"); got != "lib\none\ntwo\nthree" {
		t.Fatalf("index after the hunk: %q", got)
	}
	if a.rp.staging == nil || !a.rp.staging.staged {
		t.Fatal("after staging everything the staged side shows")
	}

	// Discarding lines works on the unstaged side only.
	press(t, a, "d")
	if !strings.Contains(a.toast, "discard works on unstaged changes") {
		t.Errorf("toast %q", a.toast)
	}
	writeFile(t, filepath.Join(lib, "lib.txt"), "lib\none\ntwo\nthree\nfour\nfive\n")
	press(t, a, "tab", "space", "down", "d", "y")
	if got, _ := os.ReadFile(filepath.Join(lib, "lib.txt")); string(got) != "lib\none\ntwo\nthree\nfour\n" {
		t.Fatalf("worktree after discarding a line: %q", got)
	}
	if got := sh(t, lib, "show", ":lib.txt"); got != "lib\none\ntwo\nthree\nfour" {
		t.Fatalf("index after staging four: %q", got)
	}
	press(t, a, "esc")

	// Commit with a subject and a body.
	press(t, a, "c", "type:Count to four", "tab", "type:One per line.", "ctrl+s")
	if got := sh(t, lib, "log", "-1", "--format=%B"); got != "Count to four\n\nOne per line." {
		t.Fatalf("commit: %q", got)
	}

	// Amend folds staged changes into HEAD and keeps its message.
	writeFile(t, filepath.Join(lib, "lib.txt"), "lib\none\ntwo\nthree\nfour\nfive\n")
	sh(t, lib, "add", "lib.txt")
	refresh(t, a)
	press(t, a, "A", "y")
	if got := sh(t, lib, "log", "-1", "--format=%s"); got != "Count to four" {
		t.Errorf("amend changed the message: %q", got)
	}
	if got := sh(t, lib, "show", "HEAD:lib.txt"); !strings.HasSuffix(got, "five") {
		t.Errorf("amend: %q", got)
	}
	if got := sh(t, lib, "rev-list", "--count", "HEAD"); got != "2" {
		t.Errorf("amend made a new commit: %s commits", got)
	}

	// With nothing staged, c offers to commit everything.
	press(t, a, "c")
	if a.p == nil || !strings.Contains(a.p.title, "Nothing is staged") {
		t.Fatalf("expected the stage-all question, got %+v", a.p)
	}
	press(t, a, "y", "type:Add new.txt", "enter")
	if got := sh(t, lib, "log", "-1", "--format=%s", "--name-only"); got != "Add new.txt\n\nnew.txt" {
		t.Fatalf("commit all: %q", got)
	}
	press(t, a, "c")
	if a.ed != nil || a.toast != "nothing to commit" {
		t.Errorf("clean tree: editor=%v toast=%q", a.ed != nil, a.toast)
	}

	// Discard a modified file and an untracked one; a directory at once.
	writeFile(t, filepath.Join(lib, "lib.txt"), "changed\n")
	writeFile(t, filepath.Join(lib, "junk.txt"), "junk\n")
	writeFile(t, filepath.Join(lib, "tmp", "a.txt"), "a\n")
	writeFile(t, filepath.Join(lib, "tmp", "b.txt"), "b\n")
	refresh(t, a)
	selectRow(t, a, "tmp/")
	press(t, a, "d", "y")
	selectRow(t, a, "junk.txt")
	press(t, a, "d", "y")
	selectRow(t, a, "lib.txt")
	press(t, a, "d", "y")
	if got := sh(t, lib, "status", "--porcelain"); got != "" {
		t.Errorf("after discarding: %q", got)
	}

	// Stage all and unstage all.
	writeFile(t, filepath.Join(lib, "x.txt"), "x\n")
	writeFile(t, filepath.Join(lib, "lib.txt"), "y\n")
	refresh(t, a)
	press(t, a, "a")
	if got := sh(t, lib, "status", "--porcelain"); got != "M  lib.txt\nA  x.txt" {
		t.Errorf("stage all: %q", got)
	}
	press(t, a, "a")
	if got := sh(t, lib, "status", "--porcelain"); got != "M lib.txt\n?? x.txt" {
		t.Errorf("unstage all: %q", got)
	}

	// The file tree folds directories and switches to a flat list.
	writeFile(t, filepath.Join(lib, "src", "deep", "a.go"), "package a\n")
	refresh(t, a)
	selectRow(t, a, "src/deep/")
	press(t, a, "enter")
	for _, n := range a.rp.fileRows {
		if n.Path == "src/deep/a.go" {
			t.Error("enter on a directory should fold it")
		}
	}
	press(t, a, "`")
	if !a.rp.flat || len(a.rp.fileRows) != 3 {
		t.Errorf("flat list: %v %d rows", a.rp.flat, len(a.rp.fileRows))
	}
}

func TestE2EBranches(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, origin := e2eSetup(t)
	app := filepath.Join(root, "app")
	openRepo(t, a, app)
	current := func() string { return sh(t, app, "branch", "--show-current") }

	// n makes a branch from the selected one and checks it out.
	press(t, a, "2", "n", "type:feature", "enter")
	if current() != "feature" || a.rp.local[0].Name != "feature" || a.rp.cur[pBranches] != 0 {
		t.Fatalf("new branch: on %q, list %v", current(), a.rp.local)
	}
	commitFile(t, app, "feature.txt", "f\n", "Add feature")
	refresh(t, a)

	// space checks out, M merges into the current branch.
	selectRow(t, a, "main")
	press(t, a, "space")
	if current() != "main" {
		t.Fatalf("checkout: on %q", current())
	}
	selectRow(t, a, "feature")
	press(t, a, "M", "y")
	if got := subjects(t, app, "1"); got != "Add feature" {
		t.Fatalf("merge: %q", got)
	}

	// r renames, d deletes a merged branch, D an unmerged one.
	selectRow(t, a, "feature")
	press(t, a, "r", "ctrl+u", "type:done", "enter")
	if got := sh(t, app, "branch", "--list", "done"); !strings.Contains(got, "done") {
		t.Fatalf("rename: %q", sh(t, app, "branch"))
	}
	selectRow(t, a, "done")
	press(t, a, "d", "y")
	sh(t, app, "branch", "wip")
	sh(t, app, "checkout", "-q", "wip")
	commitFile(t, app, "wip.txt", "w\n", "wip")
	sh(t, app, "checkout", "-q", "main")
	refresh(t, a)
	selectRow(t, a, "wip")
	press(t, a, "d", "y")
	if !strings.Contains(a.toast, "not fully merged") {
		t.Errorf("deleting an unmerged branch with d: %q", a.toast)
	}
	press(t, a, "D", "y")
	if got := sh(t, app, "branch", "--format=%(refname:short)"); got != "main" {
		t.Fatalf("branches after deleting: %q", got)
	}
	selectRow(t, a, "main")
	press(t, a, "d")
	if !strings.Contains(a.toast, "checked-out branch") {
		t.Errorf("deleting the current branch: %q", a.toast)
	}

	// U stops tracking, u sets the upstream again.
	press(t, a, "U", "y")
	if a.rp.info.Upstream {
		t.Fatal("unset upstream")
	}
	press(t, a, "p")
	if !strings.Contains(a.toast, "no upstream") {
		t.Errorf("pull without upstream: %q", a.toast)
	}
	press(t, a, "u", "enter")
	if got := sh(t, app, "rev-parse", "--abbrev-ref", "main@{u}"); got != "origin/main" {
		t.Fatalf("upstream: %q", got)
	}

	// P pushes the merge; another clone adds a commit; f fast-forwards a
	// branch that is not checked out, p pulls the current one.
	press(t, a, "P")
	other := filepath.Join(t.TempDir(), "other")
	sh(t, root, "clone", "-q", origin, other)
	commitFile(t, other, "other.txt", "o\n", "From elsewhere")
	sh(t, other, "push", "-q", "origin", "main")
	sh(t, app, "branch", "--track", "lagging", "origin/main")
	press(t, a, "f")
	selectRow(t, a, "lagging")
	press(t, a, "F")
	if got := sh(t, app, "log", "-1", "--format=%s", "lagging"); got != "From elsewhere" {
		t.Fatalf("fast-forward: %q (toast %q)", got, a.toast)
	}
	press(t, a, "p")
	if got := subjects(t, app, "1"); got != "From elsewhere" {
		t.Fatalf("pull: %q (toast %q)", got, a.toast)
	}

	// R rebases the current branch onto the selected one.
	sh(t, app, "checkout", "-q", "-b", "topic", "HEAD~1")
	commitFile(t, app, "topic.txt", "t\n", "Topic work")
	refresh(t, a)
	selectRow(t, a, "main")
	press(t, a, "R", "y")
	if got := subjects(t, app, "2"); got != "Topic work\nFrom elsewhere" {
		t.Fatalf("rebase: %q (toast %q)", got, a.toast)
	}

	// The Remote tab: space on a remote branch checks out a tracking branch.
	sh(t, other, "push", "-q", "origin", "main:shared")
	press(t, a, "f", "]")
	if a.rp.branchTab != tabRemote {
		t.Fatalf("tab %d", a.rp.branchTab)
	}
	selectRow(t, a, "origin/shared")
	press(t, a, "space")
	if current() != "shared" || sh(t, app, "rev-parse", "--abbrev-ref", "shared@{u}") != "origin/shared" {
		t.Fatalf("remote checkout: on %q", current())
	}
	if a.rp.branchTab != tabLocal {
		t.Error("after a checkout the Local tab shows the new branch")
	}

	// Tags: a lightweight tag (empty message), checkout, delete.
	press(t, a, "[", "n", "type:light", "enter", "enter")
	if got := sh(t, app, "cat-file", "-t", "light"); got != "commit" {
		t.Fatalf("lightweight tag: %q", got)
	}
	press(t, a, "space", "y")
	if _, err := gitx.Toplevel(a.ctx, app); err != nil || !a.rp.info.Detached {
		t.Errorf("tag checkout should detach HEAD: %+v", a.rp.info)
	}
	press(t, a, "d", "y")
	if got := sh(t, app, "tag"); got != "" {
		t.Errorf("tags after delete: %q", got)
	}
}

func TestE2ECommitsPanel(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, _ := e2eSetup(t)
	lib := filepath.Join(root, "lib")
	for _, n := range []string{"a", "b", "c", "d"} {
		commitFile(t, lib, n+".txt", n+"\n", "add "+n)
	}
	openRepo(t, a, lib)
	press(t, a, "3")

	// s squashes the selected commit into its parent; f fixes it up.
	selectRow(t, a, "add d")
	press(t, a, "s", "y")
	if got := sh(t, lib, "log", "-1", "--format=%B"); got != "add c\n\nadd d" {
		t.Fatalf("squash: %q", got)
	}
	selectRow(t, a, "add b")
	press(t, a, "F", "y")
	if got := subjects(t, lib, "3"); got != "add c\nadd a\nlib initial" {
		t.Fatalf("fixup: %q", got)
	}
	if got := sh(t, lib, "show", "--name-only", "--format=", "HEAD~1"); got != "a.txt\nb.txt" {
		t.Errorf("fixup kept files: %q", got)
	}

	// d drops; z undoes it; z again redoes.
	selectRow(t, a, "add c")
	press(t, a, "d", "y")
	if got := subjects(t, lib, "1"); got != "add a" {
		t.Fatalf("drop: %q", got)
	}
	press(t, a, "z", "y")
	if got := subjects(t, lib, "1"); got != "add c" {
		t.Fatalf("undo: %q", got)
	}
	press(t, a, "z", "y")
	if got := subjects(t, lib, "1"); got != "add a" {
		t.Fatalf("redo: %q", got)
	}
	press(t, a, "z", "y")

	// t reverts, u undoes the last commit and keeps it staged.
	selectRow(t, a, "add a")
	press(t, a, "t", "y")
	if got := subjects(t, lib, "1"); got != `Revert "add a"` {
		t.Fatalf("revert: %q", got)
	}
	press(t, a, "u", "y")
	if got := sh(t, lib, "status", "--porcelain"); got != "D  a.txt\nD  b.txt" {
		t.Fatalf("undo commit: %q", got)
	}
	sh(t, lib, "reset", "-q", "--hard")
	refresh(t, a)

	// ctrl+j moves a commit down (older), ctrl+k back up.
	selectRow(t, a, "add c")
	press(t, a, "ctrl+j")
	if got := subjects(t, lib, "2"); got != "add a\nadd c" {
		t.Fatalf("move down: %q", got)
	}
	if c := a.rp.commits[a.rp.cur[pCommits]]; c.Subject != "add c" {
		t.Errorf("cursor should follow the moved commit, on %q", c.Subject)
	}
	press(t, a, "ctrl+k")
	if got := subjects(t, lib, "2"); got != "add c\nadd a" {
		t.Fatalf("move up: %q", got)
	}

	// The planner: reword one commit, move another, run.
	selectRow(t, a, "add a")
	press(t, a, "i")
	if a.plan == nil || len(a.plan.rows) != 2 {
		t.Fatalf("planner: %+v", a.plan)
	}
	press(t, a, "r", "ctrl+u", "type:Add c", "ctrl+s", "J", "enter", "y")
	if got := subjects(t, lib, "3"); got != "add a\nAdd c\nlib initial" {
		t.Fatalf("plan: %q (toast %q)", got, a.toast)
	}

	// An edit step stops the rebase; C continues it.
	selectRow(t, a, "Add c")
	press(t, a, "i", "e", "enter", "y")
	if a.rp.info.State != gitx.StateRebasing || !strings.Contains(a.toast, "stopped for editing") {
		t.Fatalf("edit stop: state %v toast %q", a.rp.info.State, a.toast)
	}
	if !strings.Contains(a.View(), "REBASING") {
		t.Error("the header should show the rebase")
	}
	press(t, a, "C")
	if a.rp.info.State != gitx.StateNone {
		t.Fatalf("continue: state %v toast %q", a.rp.info.State, a.toast)
	}

	// Copy and paste within one repository is a plain cherry-pick.
	selectRow(t, a, "add a")
	press(t, a, "c")
	if !strings.Contains(a.View(), "1 copied from lib") {
		t.Error("copied badge")
	}
	sh(t, lib, "checkout", "-q", "-b", "other", "HEAD~2")
	refresh(t, a)
	press(t, a, "v")
	if a.p == nil || strings.Contains(a.p.title, "patches") {
		t.Fatalf("paste in the same repository: %+v", a.p)
	}
	press(t, a, "y")
	if got := subjects(t, lib, "1"); got != "add a" {
		t.Fatalf("cherry-pick: %q", got)
	}
	if a.sh.copied != nil {
		t.Error("the clipboard is emptied by a paste")
	}
	sh(t, lib, "checkout", "-q", "main")
	refresh(t, a)

	// The reflog: check out an entry (detached), then reset a branch to one.
	press(t, a, "]")
	if !a.rp.reflog || len(a.rp.refs) == 0 {
		t.Fatal("reflog")
	}
	press(t, a, "down", "space", "y")
	if !a.rp.info.Detached {
		t.Fatalf("reflog checkout: %+v (toast %q)", a.rp.info, a.toast)
	}
	sh(t, lib, "checkout", "-q", "main")
	refresh(t, a)
	before := sh(t, lib, "rev-parse", "HEAD")
	selectRow(t, a, "lib initial")
	press(t, a, "g", "y")
	if got := subjects(t, lib, "2"); got != "lib initial" {
		t.Fatalf("reset here: %q (toast %q)", got, a.toast)
	}
	press(t, a, "z", "y")
	if got := sh(t, lib, "rev-parse", "HEAD"); got != before {
		t.Fatalf("undo of the reset: %s, want %s", got, before)
	}
	press(t, a, "[")
	if a.rp.reflog {
		t.Error("[ goes back to the commits")
	}
}

func TestE2EStash(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, _ := e2eSetup(t)
	lib := filepath.Join(root, "lib")
	writeFile(t, filepath.Join(lib, "lib.txt"), "changed\n")
	writeFile(t, filepath.Join(lib, "new.txt"), "new\n")
	openRepo(t, a, lib)

	// s stashes everything (untracked files too) with a message.
	press(t, a, "s", "ctrl+u", "type:first", "enter")
	if got := sh(t, lib, "status", "--porcelain"); got != "" {
		t.Fatalf("after stash: %q", got)
	}
	press(t, a, "4")
	if len(a.rp.stashes) != 1 || !strings.Contains(a.rp.stashes[0].Msg, "first") {
		t.Fatalf("stashes: %+v", a.rp.stashes)
	}
	// space applies and keeps the stash; g pops it.
	press(t, a, "space")
	if got := sh(t, lib, "status", "--porcelain"); got != "M lib.txt\n?? new.txt" || len(a.rp.stashes) != 1 {
		t.Fatalf("apply: %q, %d stashes", got, len(a.rp.stashes))
	}
	sh(t, lib, "reset", "-q", "--hard")
	sh(t, lib, "clean", "-qfd")
	press(t, a, "g")
	if got := sh(t, lib, "status", "--porcelain"); got != "M lib.txt\n?? new.txt" || len(a.rp.stashes) != 0 {
		t.Fatalf("pop: %q, %d stashes", got, len(a.rp.stashes))
	}

	// S stashes only the staged changes (git 2.35 and newer; older ones say so).
	if v, _ := gitx.GitVersion(context.Background()); !gitx.VersionAtLeast(v, gitx.MinGitVersion) {
		press(t, a, "1", "S")
		if !strings.Contains(a.toast, "needs git") {
			t.Errorf("S on git %s: %q", v, a.toast)
		}
		return
	}
	sh(t, lib, "add", "new.txt")
	press(t, a, "1", "S", "type:staged only", "enter")
	if got := sh(t, lib, "status", "--porcelain"); got != "M lib.txt" {
		t.Fatalf("stash staged: %q (toast %q)", got, a.toast)
	}

	// b makes a branch from a stash (applying it there); d drops one.
	press(t, a, "4", "b", "type:from-stash", "enter")
	if got := sh(t, lib, "branch", "--show-current"); got != "from-stash" || len(a.rp.stashes) != 0 {
		t.Fatalf("stash branch: on %q, %d stashes (toast %q)", got, len(a.rp.stashes), a.toast)
	}
	if got := sh(t, lib, "status", "--porcelain"); got != "M lib.txt\nA  new.txt" {
		t.Errorf("stash branch applied: %q", got)
	}
	sh(t, lib, "stash", "push", "-q", "-m", "one")
	sh(t, lib, "stash", "push", "-q", "-m", "two")
	refresh(t, a)
	n := len(a.rp.stashes)
	press(t, a, "d")
	if a.p == nil || !strings.Contains(a.p.title, "Drop stash@{0}") {
		t.Fatalf("drop question: %+v", a.p)
	}
	press(t, a, "y")
	if len(a.rp.stashes) != n-1 {
		t.Errorf("drop: %d stashes, want %d", len(a.rp.stashes), n-1)
	}
}

// Conflicts leave a merge, rebase, cherry-pick or am in progress; the
// header says so and C, S and X continue, skip and abort.
func TestE2EConflicts(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, _ := e2eSetup(t)
	lib := filepath.Join(root, "lib")
	sh(t, lib, "checkout", "-q", "-b", "theirs")
	commitFile(t, lib, "lib.txt", "theirs\n", "theirs")
	sh(t, lib, "checkout", "-q", "main")
	commitFile(t, lib, "lib.txt", "ours\n", "ours")
	openRepo(t, a, lib)
	state := func() gitx.State { return a.rp.info.State }

	// Merge: X aborts; then resolve, stage and C concludes it.
	press(t, a, "2")
	selectRow(t, a, "theirs")
	press(t, a, "M", "y")
	if state() != gitx.StateMerging || !strings.Contains(a.View(), "MERGING") {
		t.Fatalf("merge conflict: state %v toast %q", state(), a.toast)
	}
	if hint := a.repoHints(); !strings.Contains(hint, "C continue") || strings.Contains(hint, "S skip") {
		t.Errorf("merge hints: %q", hint)
	}
	press(t, a, "X", "y")
	if state() != gitx.StateNone || subjects(t, lib, "1") != "ours" {
		t.Fatalf("merge abort: %v", state())
	}
	press(t, a, "M", "y", "1")
	writeFile(t, filepath.Join(lib, "lib.txt"), "both\n")
	refresh(t, a)
	selectRow(t, a, "lib.txt")
	press(t, a, "space", "C")
	if state() != gitx.StateNone || sh(t, lib, "rev-list", "--count", "--merges", "HEAD") != "1" {
		t.Fatalf("merge continue: state %v toast %q", state(), a.toast)
	}
	press(t, a, "z", "y") // back to before the merge

	// Rebase onto a conflicting branch: S skips the conflicting commit.
	press(t, a, "2")
	selectRow(t, a, "theirs")
	press(t, a, "R", "y")
	if state() != gitx.StateRebasing {
		t.Fatalf("rebase conflict: state %v toast %q", state(), a.toast)
	}
	press(t, a, "S")
	if state() != gitx.StateNone || subjects(t, lib, "1") != "theirs" {
		t.Fatalf("rebase skip: state %v, HEAD %q", state(), subjects(t, lib, "1"))
	}
	press(t, a, "z", "y")

	// Cherry-pick within the repository: X aborts.
	sh(t, lib, "checkout", "-q", "theirs")
	refresh(t, a)
	press(t, a, "3")
	selectRow(t, a, "theirs")
	press(t, a, "c")
	sh(t, lib, "checkout", "-q", "main")
	refresh(t, a)
	press(t, a, "v", "y")
	if state() != gitx.StateCherryPicking {
		t.Fatalf("cherry-pick conflict: state %v toast %q", state(), a.toast)
	}
	press(t, a, "X", "y")
	if state() != gitx.StateNone {
		t.Fatalf("cherry-pick abort: %v", state())
	}

	// Across repositories the commit arrives with am: S skips it.
	app := filepath.Join(root, "app")
	commitFile(t, app, "lib.txt", "app\n", "app lib.txt")
	press(t, a, "esc")
	openRepo(t, a, app)
	press(t, a, "3", "c", "esc")
	openRepo(t, a, lib)
	press(t, a, "3", "v", "y")
	if state() != gitx.StateApplying || !strings.Contains(a.View(), "APPLYING PATCHES") {
		t.Fatalf("am conflict: state %v toast %q", state(), a.toast)
	}
	press(t, a, "S")
	if state() != gitx.StateNone || subjects(t, lib, "1") != "ours" {
		t.Fatalf("am skip: state %v HEAD %q", state(), subjects(t, lib, "1"))
	}

	// Rewriting history refuses while an operation is in progress.
	press(t, a, "2")
	selectRow(t, a, "theirs")
	press(t, a, "M", "y", "3", "s")
	if !strings.Contains(a.toast, "finish or abort the merging in progress") {
		t.Errorf("squash during a merge: %q", a.toast)
	}
	press(t, a, "X", "y")
}
