package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"testing"

	"github.com/Paksonite/gitnest/internal/gitx"
)

func TestStagingCursorAndSelection(t *testing.T) {
	a := testApp()
	a.w, a.h = 120, 30
	a.scr = scrRepo
	d := gitx.ParseDiff("diff --git a/f b/f\n--- a/f\n+++ b/f\n@@ -1,4 +1,4 @@\n a\n-b\n+B\n c\n@@ -10,3 +10,4 @@\n x\n+y\n+z\n w\n")
	a.onStaging(stagingMsg{path: "f", diff: d})
	st := a.rp.staging
	if st == nil || !st.rows[st.cur].change || st.rows[st.cur].text != "-b" {
		t.Fatalf("cursor should start on the first change: %+v", st)
	}
	a.repoKey(tea.KeyMsg{Type: tea.KeyDown})
	a.repoKey(tea.KeyMsg{Type: tea.KeyDown})
	if st.rows[st.cur].text != "+y" {
		t.Errorf("j skips context and headers: on %q", st.rows[st.cur].text)
	}
	if sel := st.selection(); len(sel) != 1 {
		t.Errorf("line selection: %v", sel)
	}
	a.repoKey(key("v"))
	a.repoKey(tea.KeyMsg{Type: tea.KeyDown})
	if sel := st.selection(); len(sel) != 2 {
		t.Errorf("range selection: %v", sel)
	}
	a.repoKey(key("a")) // hunk mode: whole first... current hunk
	if sel := st.selection(); len(sel) != 2 || st.anchor != -1 {
		t.Errorf("hunk selection: %v", sel)
	}
	a.repoKey(tea.KeyMsg{Type: tea.KeyUp})
	if st.rows[st.cur].ref.Hunk != 0 {
		t.Errorf("k in hunk mode should go to the previous hunk")
	}
	title, lines := a.stagingLines(20, 80)
	if !strings.Contains(title, "Unstaged · f · hunk") || len(lines) == 0 {
		t.Errorf("title %q", title)
	}
	if !strings.Contains(a.repoHints(), "tab staged side") {
		t.Errorf("hints: %q", a.repoHints())
	}
	a.repoKey(key("q"))
	if a.rp.staging != nil {
		t.Error("q should close the staging view")
	}
}
