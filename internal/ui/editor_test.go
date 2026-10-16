package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func typeText(a *App, s string) {
	for _, r := range s {
		if r == '\n' {
			a.onKey(tea.KeyMsg{Type: tea.KeyEnter})
			continue
		}
		a.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestEditorSavesCleanMultilineMessage(t *testing.T) {
	a := testApp()
	a.w, a.h = 100, 30
	var saved string
	a.openEditor("Commit", "", func(msg string) tea.Cmd { saved = msg; return nil })

	a.onKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	if a.ed == nil || a.ed.errorMsg == "" {
		t.Fatal("empty message should be refused and keep the editor open")
	}
	typeText(a, "Add feature  \n\nWhy it matters.\nMore detail.\n\n")
	if !strings.Contains(a.viewEditor(), "subject 11/72") {
		t.Errorf("editor view lacks subject counter:\n%s", a.viewEditor())
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	if a.ed != nil {
		t.Fatal("editor still open after save")
	}
	if saved != "Add feature\n\nWhy it matters.\nMore detail." {
		t.Errorf("saved %q", saved)
	}
}

func TestEditorEscAsksBeforeDiscarding(t *testing.T) {
	a := testApp()
	a.w, a.h = 100, 30
	called := false
	a.openEditor("Reword", "old subject", func(string) tea.Cmd { called = true; return nil })
	a.onKey(tea.KeyMsg{Type: tea.KeyEsc}) // unchanged: closes at once
	if a.ed != nil {
		t.Fatal("unchanged editor should close on esc")
	}

	a.openEditor("Reword", "old subject", func(string) tea.Cmd { called = true; return nil })
	typeText(a, "!")
	a.onKey(tea.KeyMsg{Type: tea.KeyEsc})
	if a.ed == nil || !a.ed.discard {
		t.Fatal("first esc on a modified message should ask")
	}
	typeText(a, "?") // any other key keeps editing
	if a.ed.discard {
		t.Error("typing should cancel the discard question")
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyEsc})
	a.onKey(tea.KeyMsg{Type: tea.KeyEsc})
	if a.ed != nil || called {
		t.Errorf("double esc should discard without saving (open=%v saved=%v)", a.ed != nil, called)
	}
}

func TestSubjectInfo(t *testing.T) {
	cases := []struct{ msg, want string }{
		{"short", "subject 5/72"},
		{strings.Repeat("x", 60), "subject 60/72"},
		{strings.Repeat("x", 80), "too long"},
		{"subject\nbody right away", "add a blank line"},
	}
	for _, c := range cases {
		if got, _ := subjectInfo(c.msg); !strings.Contains(got, c.want) {
			t.Errorf("subjectInfo(%q) = %q, want %q", c.msg, got, c.want)
		}
	}
}

func TestMainPanelScroll(t *testing.T) {
	a := testApp()
	a.scr = scrRepo
	a.rp.mainLen, a.rp.mainH = 100, 20
	a.repoKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !a.rp.mainFocus {
		t.Fatal("enter should focus the diff")
	}
	for _, k := range []string{"j", "j", "d"} {
		a.repoKey(key(k))
	}
	if a.rp.mainOff != 12 {
		t.Errorf("offset = %d, want 12", a.rp.mainOff)
	}
	a.repoKey(key("G"))
	if a.rp.mainOff != 80 {
		t.Errorf("G: offset = %d, want 80", a.rp.mainOff)
	}
	a.repoKey(tea.KeyMsg{Type: tea.KeyEsc})
	if a.rp.mainFocus || a.scr != scrRepo {
		t.Error("esc should leave the diff, not the repo")
	}
}

func TestErrorDetails(t *testing.T) {
	a := testApp()
	a.onKey(key("!"))
	if a.report != nil {
		t.Fatal("no error yet")
	}
	a.onOp(opMsg{label: "push", err: errLines(5)})
	if !strings.Contains(a.toast, "! details") {
		t.Errorf("toast = %q", a.toast)
	}
	a.onKey(key("!"))
	if a.report == nil || len(a.report.lines) != 5 {
		t.Fatalf("report = %+v", a.report)
	}
}

type multiErr string

func (e multiErr) Error() string { return string(e) }

func errLines(n int) error {
	var lines []string
	for i := 0; i < n; i++ {
		lines = append(lines, "line")
	}
	return multiErr(strings.Join(lines, "\n"))
}
