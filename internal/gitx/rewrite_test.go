package gitx

import (
	"reflect"
	"strings"
	"testing"
)

// linearRepo creates commits a..d (file x.txt per commit) and returns their hashes.
func linearRepo(t *testing.T) (dir string, h map[string]string) {
	t.Helper()
	dir = newRepo(t)
	h = map[string]string{}
	for _, n := range []string{"a", "b", "c", "d"} {
		h[n] = commit(t, dir, n+".txt", n+"\n", n)
	}
	return dir, h
}

func subjects(t *testing.T, dir string) string {
	t.Helper()
	return strings.ReplaceAll(git(t, dir, "log", "--format=%s"), "\n", " ")
}

// mergeRepo: base, then main gets m1, side gets s1, merged into main, then m2.
func mergeRepo(t *testing.T) (dir string, h map[string]string) {
	t.Helper()
	dir = newRepo(t)
	h = map[string]string{}
	h["base"] = commit(t, dir, "base.txt", "x\n", "base")
	git(t, dir, "checkout", "-q", "-b", "side")
	h["s1"] = commit(t, dir, "s1.txt", "x\n", "s1")
	git(t, dir, "checkout", "-q", "main")
	h["m1"] = commit(t, dir, "m1.txt", "x\n", "m1")
	git(t, dir, "merge", "-q", "--no-ff", "-m", "merge side", "side")
	h["merge"] = git(t, dir, "rev-parse", "HEAD")
	h["m2"] = commit(t, dir, "m2.txt", "x\n", "m2")
	return dir, h
}

func TestRebaseActions(t *testing.T) {
	cases := []struct {
		action, target, want string
	}{
		{ActionDrop, "b", "d c a"},
		{ActionDrop, "a", "d c b"},
		{ActionFixup, "c", "d b a"},
		{ActionSquash, "d", "c b a"}, // combined message starts with the parent's
		{ActionFixup, "b", "d c a"},
	}
	for _, c := range cases {
		t.Run(c.action+"_"+c.target, func(t *testing.T) {
			dir, h := linearRepo(t)
			if err := RebaseAction(bg, dir, h[c.target], c.action, ""); err != nil {
				t.Fatal(err)
			}
			if got := subjects(t, dir); got != c.want {
				t.Errorf("history = %q, want %q", got, c.want)
			}
			mustState(t, dir, StateNone)
		})
	}
}

func TestSquashKeepsBothMessagesAndFiles(t *testing.T) {
	dir, h := linearRepo(t)
	if err := RebaseAction(bg, dir, h["c"], ActionSquash, ""); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "log", "-1", "--format=%B", "HEAD~1"); got != "b\n\nc" {
		t.Errorf("squashed message = %q", got)
	}
	if got := git(t, dir, "show", "--name-only", "--format=", "HEAD~1"); got != "b.txt\nc.txt" {
		t.Errorf("squashed files = %q", got)
	}
}

// The todo editor must pick the line by hash, not by position: here the
// commit to drop is the third line of the todo.
func TestTodoEditorMatchesHash(t *testing.T) {
	dir, h := linearRepo(t)
	env := []string{"GITNEST_HASH=" + h["c"], "GITNEST_ACTION=drop",
		"GITNEST_AWK=" + todoAwk, "GIT_SEQUENCE_EDITOR=" + todoEditor}
	if _, err := run(bg, kWrite, dir, env, "rebase", "-i", h["a"]); err != nil {
		t.Fatal(err)
	}
	if got := subjects(t, dir); got != "d b a" {
		t.Errorf("history = %q", got)
	}
}

func TestTodoEditorAbortsWithoutMatch(t *testing.T) {
	dir, h := linearRepo(t)
	env := []string{"GITNEST_HASH=0123456789abcdef0123456789abcdef01234567", "GITNEST_ACTION=drop",
		"GITNEST_AWK=" + todoAwk, "GIT_SEQUENCE_EDITOR=" + todoEditor}
	if _, err := run(bg, kWrite, dir, env, "rebase", "-i", h["a"]); err == nil {
		t.Fatal("rebase should fail when no todo line matches")
	}
	mustState(t, dir, StateNone)
	if got := git(t, dir, "rev-parse", "HEAD"); got != h["d"] {
		t.Errorf("HEAD moved to %s", got)
	}
}

func TestRewriteRefusesMerges(t *testing.T) {
	dir, h := mergeRepo(t)
	for _, target := range []string{"m1", "s1", "base"} {
		_, err := CheckRewrite(bg, dir, h[target], ActionDrop)
		if err == nil || !strings.Contains(err.Error(), "merge commits") {
			t.Errorf("drop %s: err = %v, want merge refusal", target, err)
		}
		if err := RebaseAction(bg, dir, h[target], ActionDrop, ""); err == nil {
			t.Errorf("RebaseAction drop %s succeeded", target)
		}
	}
	if got := git(t, dir, "rev-parse", "HEAD"); got != h["m2"] {
		t.Fatalf("HEAD moved")
	}
	// After the merge everything is linear again.
	if err := Reword(bg, dir, h["m2"], "m2 reworded"); err != nil {
		t.Fatal(err)
	}
	// Squashing m2 into the merge commit would replay the merge: refused.
	if _, err := CheckRewrite(bg, dir, "HEAD", ActionFixup); err == nil {
		t.Error("fixup into a merge commit should be refused")
	}
}

func TestRewordHeadMergeCommit(t *testing.T) {
	dir, h := mergeRepo(t)
	git(t, dir, "reset", "-q", "--hard", h["merge"])
	if err := Reword(bg, dir, "HEAD", "merged side branch"); err != nil {
		t.Fatal(err)
	}
	if parents := strings.Fields(git(t, dir, "log", "-1", "--format=%P")); len(parents) != 2 {
		t.Errorf("reworded merge lost a parent: %v", parents)
	}
}

func TestRewriteRefusals(t *testing.T) {
	dir, h := linearRepo(t)
	if _, err := CheckRewrite(bg, dir, h["a"], ActionSquash); err == nil || !strings.Contains(err.Error(), "no parent") {
		t.Errorf("squash root: err = %v", err)
	}
	if _, err := CheckRewrite(bg, dir, h["b"], ActionAmend); err == nil {
		t.Error("amend of a non-HEAD commit should be refused")
	}
	if _, err := CheckRewrite(bg, dir, h["b"], "explode"); err == nil {
		t.Error("unknown action accepted")
	}
	git(t, dir, "checkout", "-q", "-b", "other", h["b"])
	if _, err := CheckRewrite(bg, dir, h["d"], ActionDrop); err == nil || !strings.Contains(err.Error(), "not on the current branch") {
		t.Errorf("commit off branch: err = %v", err)
	}

	cdir, _ := conflictRepo(t)
	_ = Merge(bg, cdir, "other")
	if _, err := CheckRewrite(bg, cdir, "HEAD", ActionReword); err == nil || !strings.Contains(err.Error(), "merging in progress") {
		t.Errorf("during merge: err = %v", err)
	}
}

func TestCheckRewriteReportsPushedCommits(t *testing.T) {
	dir, h := linearRepo(t)
	remote := t.TempDir()
	git(t, remote, "init", "-q", "--bare")
	git(t, dir, "remote", "add", "origin", remote)
	git(t, dir, "push", "-q", "-u", "origin", "main")
	local := commit(t, dir, "e.txt", "e\n", "e") // not pushed

	check := func(rev, action string, want []string) {
		t.Helper()
		got, err := CheckRewrite(bg, dir, rev, action)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s %s: remotes = %v, want %v", action, short(rev), got, want)
		}
	}
	check(local, ActionReword, nil)
	check(local, ActionAmend, nil)
	check(local, ActionSquash, []string{"origin/main"}) // its parent d is pushed
	check(h["c"], ActionDrop, []string{"origin/main"})

	git(t, dir, "checkout", "-q", "-b", "feature", h["b"])
	commit(t, dir, "f.txt", "f\n", "f")
	got, err := CheckRebaseOnto(bg, dir, "main")
	if err != nil || got != nil {
		t.Errorf("rebase unpushed feature: %v %v", got, err)
	}
	got, err = CheckRebaseOnto(bg, dir, h["b"])
	if err != nil || got != nil {
		t.Errorf("rebase onto own base: %v %v", got, err)
	}
	git(t, dir, "push", "-q", "-u", "origin", "feature")
	got, _ = CheckRebaseOnto(bg, dir, "main")
	if !reflect.DeepEqual(got, []string{"origin/feature"}) {
		t.Errorf("rebase pushed feature: %v", got)
	}
}

func TestUndoAndRedo(t *testing.T) {
	dir, h := linearRepo(t)
	if p, _ := LastUndo(bg, dir); p != nil {
		t.Fatalf("fresh repo has undo point %+v", p)
	}
	if err := Undo(bg, dir); err == nil {
		t.Error("undo without a point should fail")
	}

	if err := RebaseAction(bg, dir, h["b"], ActionDrop, ""); err != nil {
		t.Fatal(err)
	}
	dropped := git(t, dir, "rev-parse", "HEAD")
	p, err := LastUndo(bg, dir)
	if err != nil || p == nil {
		t.Fatalf("LastUndo = %v, %v", p, err)
	}
	if p.Label != "drop "+short(h["b"]) || p.Head != h["d"] || p.Branch != "main" {
		t.Errorf("undo point = %+v", p)
	}
	if refs := parseLog(git(t, dir, "log", "-1", logFormat, h["d"]))[0].Refs; strings.Contains(refs, "gitnest") {
		t.Errorf("undo ref shows up in log decorations: %q", refs)
	}

	if err := Undo(bg, dir); err != nil {
		t.Fatal(err)
	}
	if got := subjects(t, dir); got != "d c b a" {
		t.Errorf("after undo: %q", got)
	}
	if err := Undo(bg, dir); err != nil { // redo
		t.Fatal(err)
	}
	if got := git(t, dir, "rev-parse", "HEAD"); got != dropped {
		t.Errorf("redo: HEAD = %s, want %s", got, dropped)
	}
}

func TestUndoRefusals(t *testing.T) {
	dir, h := linearRepo(t)
	if err := Reword(bg, dir, h["d"], "d2"); err != nil {
		t.Fatal(err)
	}

	writeFile(t, dir, "a.txt", "changed\n")
	if err := Undo(bg, dir); err == nil || !strings.Contains(err.Error(), "commit or stash") {
		t.Errorf("dirty tree: err = %v", err)
	}
	git(t, dir, "checkout", "-q", "--", "a.txt")

	writeFile(t, dir, "untracked.txt", "ok\n") // untracked files don't block undo
	git(t, dir, "checkout", "-q", "-b", "other")
	if err := Undo(bg, dir); err == nil || !strings.Contains(err.Error(), "undo point is for main") {
		t.Errorf("other branch: err = %v", err)
	}
	git(t, dir, "checkout", "-q", "main")
	if err := Undo(bg, dir); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "log", "-1", "--format=%s"); got != "d" {
		t.Errorf("after undo: %q", got)
	}
}

func TestUndoIgnoresStaleFile(t *testing.T) {
	dir, h := linearRepo(t)
	if err := Reword(bg, dir, h["d"], "d2"); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "update-ref", "-d", undoRef)
	if p, err := LastUndo(bg, dir); p != nil || err != nil {
		t.Errorf("LastUndo with deleted ref = %+v, %v", p, err)
	}
}

func TestOperationsRecordUndoPoints(t *testing.T) {
	dir, _ := conflictRepo(t)
	head := git(t, dir, "rev-parse", "HEAD")
	_ = Merge(bg, dir, "other")
	p, _ := LastUndo(bg, dir)
	if p == nil || p.Label != "merge other" || p.Head != head {
		t.Fatalf("merge undo point = %+v", p)
	}
	if err := Undo(bg, dir); err == nil {
		t.Error("undo during a merge should be refused")
	}
	if err := Abort(bg, dir, StateMerging); err != nil {
		t.Fatal(err)
	}

	writeFile(t, dir, "new.txt", "n\n")
	git(t, dir, "add", "new.txt")
	if err := AmendNoEdit(bg, dir); err != nil {
		t.Fatal(err)
	}
	if p, _ := LastUndo(bg, dir); p == nil || p.Label != "amend" || p.Head != head {
		t.Errorf("amend undo point = %+v", p)
	}
	if err := UndoCommit(bg, dir); err != nil {
		t.Fatal(err)
	}
	if p, _ := LastUndo(bg, dir); p == nil || p.Label != "undo commit" {
		t.Errorf("undo commit point = %+v", p)
	}
}
