package ui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paksonite/gitnest/internal/gitx"
)

func TestPlannerBuildsSteps(t *testing.T) {
	a := testApp()
	a.w, a.h = 100, 30
	a.scr = scrRepo
	a.rp = repoState{dir: "/r", focus: pCommits}
	a.rp.commits = []gitx.Commit{{Hash: "d", Short: "d"}, {Hash: "c", Short: "c"}, {Hash: "b", Short: "b"}, {Hash: "a", Short: "a"}}
	a.rp.cur[pCommits] = 2 // plan b..d
	a.repoKey(key("i"))
	if a.plan == nil || len(a.plan.rows) != 3 || a.plan.oldest != "b" {
		t.Fatalf("plan: %+v", a.plan)
	}
	a.onKey(key("d"))                       // drop d
	a.onKey(tea.KeyMsg{Type: tea.KeyDown})  // to c
	a.onKey(tea.KeyMsg{Type: tea.KeyCtrlJ}) // move c below b
	a.onKey(tea.KeyMsg{Type: tea.KeyUp})    // to b (now second)
	a.onKey(key("s"))                       // squash b into c (below it)
	got := a.plan.steps()
	want := []gitx.PlanStep{{Hash: "c", Action: "pick"}, {Hash: "b", Action: "squash"}, {Hash: "d", Action: "drop"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("steps = %+v\nwant %+v", got, want)
	}
	if v := a.View(); !strings.Contains(v, "Interactive rebase") || !strings.Contains(v, "squash") {
		t.Errorf("view:\n%s", v)
	}

	// The oldest row can't be squashed into nothing.
	a.plan.cur = 2
	a.onKey(key("f"))
	a.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	if a.plan == nil || !strings.Contains(a.toast, "nothing to fixup") {
		t.Errorf("toast %q, plan open %v", a.toast, a.plan != nil)
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyEsc})
	if a.plan != nil {
		t.Error("esc should close the planner")
	}
}
