package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paksonite/gitnest/internal/gitx"
)

func TestInitials(t *testing.T) {
	cases := map[string]string{
		"Michał Wadecki":      "MW",
		"ada lovelace byron":  "AB",
		"dependabot[bot]":     "DB",
		"x":                   "X ",
		"":                    "??",
		"Jean-Luc Picard":     "JP",
		"github-actions[bot]": "GB",
		"Ćwierć Łukasz Żółć":  "ĆŻ",
	}
	for in, want := range cases {
		if got := initials(in); got != want {
			t.Errorf("initials(%q) = %q, want %q", in, got, want)
		}
	}
	if authorColor("Ann") != authorColor("Ann") {
		t.Error("author colour must be stable")
	}
}

func TestCommitAndReflogRows(t *testing.T) {
	c := gitx.Commit{Short: "abc1234", Author: "Michał Wadecki", Subject: "Revert \"x\""}
	r := commitRow(c, hashLocal, []gcell{{'│', 0}, {' ', -1}, {'○', 1}}, false)
	if !strings.HasPrefix(r.plain, " abc1234 MW │ ○ Revert") {
		t.Errorf("commit row: %q / %q", r.plain, r.styled)
	}
	e := gitx.RefEntry{Short: "def5678", Selector: "HEAD@{2}", Action: "commit (amend)", Message: "fix typo"}
	rr := refRow(e)
	if !strings.HasPrefix(rr.plain, "✎ def5678 commit (amend) fix typo") || !strings.Contains(rr.styled, "✎") {
		t.Errorf("reflog row: %q / %q", rr.plain, rr.styled)
	}
}

func TestReflogToggleKeepsCursors(t *testing.T) {
	a := testApp()
	a.scr = scrRepo
	a.rp.focus = pCommits
	for i := 0; i < 5; i++ {
		a.rp.commits = append(a.rp.commits, gitx.Commit{Hash: string(rune('a' + i)), Short: "c"})
		a.rp.refs = append(a.rp.refs, gitx.RefEntry{Hash: string(rune('a' + i)), Selector: "HEAD@{0}"})
	}
	a.rp.cur[pCommits] = 3
	a.repoKey(key("]"))
	if !a.rp.reflog || a.rp.cur[pCommits] != 0 || !strings.Contains(commitsTitle(&a.rp), "Reflog (5)") {
		t.Fatalf("reflog=%v cur=%d title=%q", a.rp.reflog, a.rp.cur[pCommits], commitsTitle(&a.rp))
	}
	a.repoKey(key("j"))
	a.repoKey(key("["))
	if a.rp.reflog || a.rp.cur[pCommits] != 3 {
		t.Errorf("back: reflog=%v cur=%d", a.rp.reflog, a.rp.cur[pCommits])
	}
	a.repoKey(key("]"))
	if a.rp.cur[pCommits] != 1 {
		t.Errorf("reflog cursor not kept: %d", a.rp.cur[pCommits])
	}
	if !strings.Contains(a.repoHints(), "reset here") {
		t.Errorf("hints: %q", a.repoHints())
	}
}

func TestCheckoutPutsBranchOnTop(t *testing.T) {
	a := testApp()
	a.scr = scrRepo
	a.rp.dir = "/r"
	a.rp.focus = pBranches
	a.rp.local = []gitx.Branch{{Name: "main", Current: true}, {Name: "feature"}, {Name: "old"}}
	a.rp.cur[pBranches] = 2
	a.branchesKey(" ") // checkout "old"
	if !a.rp.jumpToCurrent {
		t.Fatal("checkout should ask to jump to the new current branch")
	}
	a.onRepoData(repoDataMsg{dir: "/r", d: repoData{local: []gitx.Branch{{Name: "main"}, {Name: "feature"}, {Name: "old", Current: true}}}})
	if a.rp.local[0].Name != "old" || a.rp.cur[pBranches] != 0 || a.rp.jumpToCurrent {
		t.Errorf("first=%s cur=%d jump=%v", a.rp.local[0].Name, a.rp.cur[pBranches], a.rp.jumpToCurrent)
	}
}

func TestBranchTabsAndTags(t *testing.T) {
	a := testApp()
	a.scr = scrRepo
	a.rp.focus = pBranches
	a.rp.local = []gitx.Branch{{Name: "main", Current: true}}
	a.rp.remote = []gitx.Branch{{Name: "origin/main", Remote: true}, {Name: "origin/x", Remote: true}}
	a.rp.tags = []gitx.Tag{{Name: "v1.0.0", Short: "abc", Subject: "Release"}}
	for want, tab := range []int{tabRemote, tabTags, tabLocal} {
		a.repoKey(key("]"))
		if a.rp.branchTab != tab {
			t.Fatalf("step %d: tab %d, want %d", want, a.rp.branchTab, tab)
		}
	}
	a.repoKey(key("["))
	if a.rp.branchTab != tabTags || a.rp.count(pBranches) != 1 {
		t.Fatalf("tags tab: %d count %d", a.rp.branchTab, a.rp.count(pBranches))
	}
	if !strings.Contains(a.repoHints(), "push") || !strings.Contains(a.panelItems(pBranches)[0].plain, "v1.0.0") {
		t.Errorf("hints %q / rows %+v", a.repoHints(), a.panelItems(pBranches))
	}
	a.repoKey(key("n"))
	if a.p == nil || !strings.Contains(a.p.title, "New tag") {
		t.Fatal("n should ask for a tag name")
	}
	a.p.input = []rune("v2")
	a.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	if a.p == nil || !a.p.allowEmpty {
		t.Fatal("the message prompt should accept an empty answer (lightweight tag)")
	}
}

func TestCdQuit(t *testing.T) {
	a := testApp("/r/one")
	for shell, want := range map[string]string{
		"/bin/zsh":      `eval "$(gitnest shell zsh)"  to ~/.zshrc`,
		"/usr/bin/bash": `eval "$(gitnest shell bash)"  to ~/.bashrc`,
	} {
		t.Setenv("SHELL", shell)
		if cmd := a.cdQuit("/r/one"); cmd != nil || !strings.Contains(a.toast, want) {
			t.Errorf("without gn in %s: cmd=%v toast=%q", shell, cmd != nil, a.toast)
		}
	}
	a.cdFile = filepath.Join(t.TempDir(), "cd")
	a.reposKey(key("o"))
	b, err := os.ReadFile(a.cdFile)
	if err != nil || strings.TrimSpace(string(b)) != "/r/one" {
		t.Errorf("cd file = %q, %v", b, err)
	}
}

func TestCopyAndPasteAcrossRepos(t *testing.T) {
	a := testApp()
	a.scr = scrRepo
	a.rp = repoState{dir: "/r/src", name: "src", focus: pCommits}
	a.rp.commits = []gitx.Commit{{Hash: "h3", Short: "h3"}, {Hash: "h2", Short: "h2"}, {Hash: "h1", Short: "h1"}}
	a.rp.cur[pCommits] = 2
	a.repoKey(key("c")) // h1
	a.rp.cur[pCommits] = 0
	a.repoKey(key("c")) // h3
	if !a.isCopied("h1") || !a.isCopied("h3") || a.isCopied("h2") || !strings.Contains(a.copiedBadge(), "2 copied from src") {
		t.Fatalf("copied: %+v", a.sh.copied)
	}
	a.rp.cur[pCommits] = 0
	a.repoKey(key("c")) // uncopy h3
	if a.isCopied("h3") {
		t.Error("c again should uncopy")
	}
	a.rp.cur[pCommits] = 0
	a.repoKey(key("c"))

	// another repository: the clipboard stays, v asks to apply as patches
	a.rp = repoState{dir: "/r/dst", name: "dst", focus: pCommits, info: gitx.Info{Branch: "main"}}
	a.rp.commits = []gitx.Commit{{Hash: "x", Short: "x"}}
	a.repoKey(key("v"))
	if a.p == nil || !strings.Contains(a.p.title, "Cherry-pick 2 commit(s) onto main (as patches from src)") {
		t.Fatalf("prompt: %+v", a.p)
	}
	a.p = nil
	a.repoKey(key("C"))
	if a.sh.copied != nil {
		t.Error("C should clear the clipboard")
	}
}
