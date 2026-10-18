package ui

import (
	"strings"
	"testing"

	"github.com/Paksonite/gitnest/internal/gitx"
)

// cs builds commits from "hash:parent,parent" specs, newest first.
func cs(specs ...string) []gitx.Commit {
	var out []gitx.Commit
	for _, s := range specs {
		h, ps, _ := strings.Cut(s, ":")
		var parents []string
		if ps != "" {
			parents = strings.Split(ps, ",")
		}
		out = append(out, gitx.Commit{Hash: h, Parents: parents, Merge: len(parents) > 1})
	}
	return out
}

func TestGraphLinear(t *testing.T) {
	got := graphString(buildGraph(cs("c:b", "b:a", "a:")))
	if got != "○\n○\n○" {
		t.Errorf("linear:\n%s", got)
	}
}

func TestGraphMergeAndBranch(t *testing.T) {
	// m merges s (side) into x; s and x both come from a.
	got := graphString(buildGraph(cs("m:x,s", "s:a", "x:a", "a:")))
	want := strings.Join([]string{
		"◎─╮",
		"│ ○",
		"○ │",
		"○─╯",
	}, "\n")
	if got != want {
		t.Errorf("merge:\n%s\nwant:\n%s", got, want)
	}
}

func TestGraphTwoTips(t *testing.T) {
	// Two branch tips (b2 and c) sharing a, like a feature branch next to main.
	got := graphString(buildGraph(cs("c:a", "b2:b1", "b1:a", "a:")))
	want := strings.Join([]string{
		"○",
		"│ ○",
		"│ ○",
		"○─╯",
	}, "\n")
	if got != want {
		t.Errorf("tips:\n%s\nwant:\n%s", got, want)
	}
}

func TestGraphCrossingAndColours(t *testing.T) {
	// m2 merges y while lane of m1's side is still open: lines cross.
	rows := buildGraph(cs("m2:m1,y", "m1:a,x", "x:a", "y:a", "a:"))
	got := graphString(rows)
	if !strings.Contains(got, "◎") || strings.Count(got, "\n") != 4 {
		t.Errorf("graph:\n%s", got)
	}
	// Every non-space cell has a lane colour.
	for _, r := range rows {
		for _, c := range r {
			if c.ch != ' ' && c.color < 0 {
				t.Errorf("uncoloured %q in:\n%s", c.ch, got)
			}
		}
	}
}

func TestGraphLineThroughConnectedLane(t *testing.T) {
	// m merges b (already its own lane) while c's lane, further right, ends here.
	got := graphString(buildGraph(cs("x:m", "b:m", "c:m", "m:p,b2", "b2:p", "p:")))
	if !strings.Contains(got, "◎") {
		t.Fatalf("graph:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "┤─") || strings.Contains(line, "─├") {
			t.Errorf("line continues past a connector drawn as an end: %q\n%s", line, got)
		}
	}
}

// graphString is the plain graph, one line per commit.
func graphString(rows [][]gcell) string {
	var out []string
	for _, r := range rows {
		p, _ := graphText(r, cCyan)
		out = append(out, p)
	}
	return strings.Join(out, "\n")
}
