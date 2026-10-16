package ui

import (
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
	r := commitRow(c, hashLocal, []gcell{{'│', 0}, {' ', -1}, {'○', 1}})
	if !strings.HasPrefix(r.plain, "abc1234 MW │ ○ Revert") {
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
	if !strings.Contains(a.repoHints(), "push") || !strings.Contains(stripStyles(a.panelItems(pBranches)[0].plain), "v1.0.0") {
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

func stripStyles(s string) string { return s }
