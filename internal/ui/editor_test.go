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

func TestEditorEnterSavesSubjectOnly(t *testing.T) {
	a := testApp()
	a.w, a.h = 100, 30
	var saved string
	a.openEditor("Commit", "", func(msg string) tea.Cmd { saved = msg; return nil })
	a.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	if a.ed == nil || a.ed.errorMsg == "" {
		t.Fatal("an empty subject should be refused and keep the editor open")
	}
	typeText(a, "Add feature  ")
	if !strings.Contains(a.viewEditor(), "11/72") {
		t.Errorf("no subject counter:\n%s", a.viewEditor())
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	if a.ed != nil || saved != "Add feature" {
		t.Fatalf("open=%v saved=%q", a.ed != nil, saved)
	}
}

func TestEditorSubjectAndBody(t *testing.T) {
	a := testApp()
	a.w, a.h = 100, 30
	var saved string
	a.openEditor("Commit", "", func(msg string) tea.Cmd { saved = msg; return nil })
	typeText(a, "Add feature")
	a.onKey(tea.KeyMsg{Type: tea.KeyTab})
	if !a.ed.inBody {
		t.Fatal("tab should move to the body")
	}
	typeText(a, "Why it matters.\nMore detail.\n\n") // enter adds lines in the body
	if a.ed == nil {
		t.Fatal("enter in the body must not save")
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	if a.ed.inBody {
		t.Fatal("shift+tab should go back to the subject")
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	if saved != "Add feature\n\nWhy it matters.\nMore detail." {
		t.Errorf("saved %q", saved)
	}
}

func TestEditorPrefillsSubjectAndBody(t *testing.T) {
	a := testApp()
	a.w, a.h = 100, 30
	a.openEditor("Reword", "Old subject\n\nOld body\nline two\n", func(string) tea.Cmd { return nil })
	if a.ed.subject.Value() != "Old subject" || a.ed.body.Value() != "Old body\nline two" {
		t.Errorf("subject %q body %q", a.ed.subject.Value(), a.ed.body.Value())
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyTab})
	a.onKey(tea.KeyMsg{Type: tea.KeyCtrlU}) // textarea: delete to line start
	a.onKey(tea.KeyMsg{Type: tea.KeyEsc})
	if a.ed == nil || !a.ed.discard {
		t.Error("an edited message should ask before discarding")
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
	cases := []struct{ subject, want string }{
		{"short", "5/72"},
		{strings.Repeat("x", 60), "60/72"},
		{strings.Repeat("x", 80), "too long"},
		{"  padded  ", "6/72"},
	}
	for _, c := range cases {
		if got, _ := subjectInfo(c.subject); !strings.Contains(got, c.want) {
			t.Errorf("subjectInfo(%q) = %q, want %q", c.subject, got, c.want)
		}
	}
	if s, b := splitMessage("\nsubj\n\n\nbody\n"); s != "subj" || b != "body" {
		t.Errorf("split: %q %q", s, b)
	}
	if got := joinMessage(" subj ", "\n\n"); got != "subj" {
		t.Errorf("join without body: %q", got)
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
	a.repoKey(tea.KeyMsg{Type: tea.KeyDown})
	a.repoKey(tea.KeyMsg{Type: tea.KeyDown})
	a.repoKey(key("d"))
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
