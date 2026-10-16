package ui

import (
	"reflect"
	"sort"
	"testing"
)

func TestFuzzyMatch(t *testing.T) {
	cases := []struct {
		s, q string
		ok   bool
		pos  []int
	}{
		{"org-core-dns", "dns", true, []int{9, 10, 11}},
		{"org-core-dns", "ocd", true, []int{0, 4, 9}},
		{"role-psBasicVars", "bv", true, []int{7, 12}},
		{"gitnest", "GIT", true, []int{0, 1, 2}},
		{"gitnest", "tg", false, nil},
		{"anything", "", true, nil},
	}
	for _, c := range cases {
		_, pos, ok := fuzzyMatch(c.s, c.q)
		if ok != c.ok || !reflect.DeepEqual(pos, c.pos) {
			t.Errorf("fuzzyMatch(%q, %q) = %v %v, want %v %v", c.s, c.q, pos, ok, c.pos, c.ok)
		}
	}
}

func TestFuzzyRanking(t *testing.T) {
	names := []string{
		"terraform-google-service-directory",
		"org-core-shared-iam",
		"prj-sp-cot-ipsec-sd3i",
		"org-core-cot-spokes",
		"org-core-sandboxes-project-factory",
	}
	rank := func(q string) []string {
		type hit struct {
			name  string
			score int
		}
		var hits []hit
		for _, n := range names {
			if sc, _, ok := fuzzyMatch(n, q); ok {
				hits = append(hits, hit{n, sc})
			}
		}
		sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
		var out []string
		for _, h := range hits {
			out = append(out, h.name)
		}
		return out
	}
	if got := rank("spokes"); got[0] != "org-core-cot-spokes" {
		t.Errorf("spokes: %v", got)
	}
	if got := rank("sd"); got[0] != "terraform-google-service-directory" && got[0] != "prj-sp-cot-ipsec-sd3i" {
		t.Errorf("sd: %v", got)
	}
	if got := rank("shared"); len(got) != 1 || got[0] != "org-core-shared-iam" {
		t.Errorf("shared: %v", got)
	}
	// A prefix match beats a scattered one.
	a, _, _ := fuzzyMatch("org-core-dns", "org")
	b, _, _ := fuzzyMatch("prj-sp-cot-ipsec-sd3i", "org")
	if !(a > b) {
		t.Errorf("prefix %d should beat scattered %d", a, b)
	}
}

func TestHighlight(t *testing.T) {
	up := func(s string) string { return "[" + s + "]" }
	id := func(s string) string { return s }
	if got := highlight("org-core-dns", []int{0, 1, 9}, id, up); got != "[or]g-core-[d]ns" {
		t.Errorf("got %q", got)
	}
	if got := highlight("abc", nil, id, up); got != "abc" {
		t.Errorf("got %q", got)
	}
}
