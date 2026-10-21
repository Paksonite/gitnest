package ui

import (
	"reflect"
	"sort"
	"testing"
)

func TestPhraseMatch(t *testing.T) {
	cases := []struct {
		s, q string
		ok   bool
		pos  []int
	}{
		{"org-core-dns", "dns", true, []int{9, 10, 11}},
		{"org-core-dns", "ocd", false, nil}, // letters in order are not a phrase
		{"gitnest", "GIT", true, []int{0, 1, 2}},
		{"gitnest", "tg", false, nil},
		{"gco-cot-service", "gco-cot", true, []int{0, 1, 2, 3, 4, 5, 6}},
		{"anything", "", true, nil},
	}
	for _, c := range cases {
		_, pos, ok := phraseMatch(c.s, c.q)
		if ok != c.ok || !reflect.DeepEqual(pos, c.pos) {
			t.Errorf("phraseMatch(%q, %q) = %v %v, want %v %v", c.s, c.q, pos, ok, c.pos, c.ok)
		}
	}
}

func TestPhraseRanking(t *testing.T) {
	names := []string{ // the list from the report: "gco-cot" found all of these
		"org-core-cot-projects-iam",
		"org-core-cot-spokes",
		"org-core-clients-project-factory",
		"gco-cot-service",
		"terraform-google-service-directory",
		"prj-sp-cot-ipsec-sd3i",
	}
	rank := func(q string) []string {
		type hit struct {
			name  string
			score int
		}
		var hits []hit
		for _, n := range names {
			if sc, _, ok := phraseMatch(n, q); ok {
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
	if got := rank("gco-cot"); !reflect.DeepEqual(got, []string{"gco-cot-service"}) {
		t.Errorf("gco-cot: %v", got)
	}
	// Every name with the phrase, the one where it comes first on top.
	if got := rank("cot"); len(got) != 4 || got[0] != "gco-cot-service" {
		t.Errorf("cot: %v", got)
	}
	if got := rank("service"); got[0] != "gco-cot-service" {
		t.Errorf("service: %v", got)
	}
	a, _, _ := phraseMatch("org-core-dns", "org")
	b, _, _ := phraseMatch("org-core-cot-projects-iam", "core")
	c, _, _ := phraseMatch("terraform-google-service-directory", "rect")
	if !(a > b && b > c) {
		t.Errorf("start %d > part %d > inside %d", a, b, c)
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
