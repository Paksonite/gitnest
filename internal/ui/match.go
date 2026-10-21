package ui

import (
	"strings"
	"unicode"
)

// Scoring for phraseMatch: a phrase at the very start of the name beats one
// that starts a word inside it ("cot" in "org-core-cot-spokes"), which
// beats one in the middle of a word; then earlier and in shorter names.
const (
	scorePhrase   = 100
	bonusStart    = 50
	bonusBoundary = 30
)

// phraseMatch reports whether query appears in s as one piece
// (case-insensitively): "gco-cot" finds "gco-cot-service" but not
// "org-core-cot-…", whose letters only happen to come in that order. It
// returns a score (higher is better) and the rune positions in s that
// matched, for highlighting; of several occurrences the best one counts.
func phraseMatch(s, query string) (score int, pos []int, ok bool) {
	q := []rune(strings.ToLower(query))
	if len(q) == 0 {
		return 0, nil, true
	}
	orig := []rune(s)
	r := []rune(strings.ToLower(s))
	at := -1
	for i := 0; i+len(q) <= len(r); i++ {
		if !equalRunes(r[i:i+len(q)], q) {
			continue
		}
		sc := scorePhrase - min(i, 20) - min((len(r)-len(q))/4, 20)
		switch {
		case i == 0:
			sc += bonusStart
		case boundary(orig, i):
			sc += bonusBoundary
		}
		if at < 0 || sc > score {
			at, score = i, sc
		}
	}
	if at < 0 {
		return 0, nil, false
	}
	for k := at; k < at+len(q); k++ {
		pos = append(pos, k)
	}
	return score, pos, true
}

func equalRunes(a, b []rune) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// boundary reports whether rune i starts a word in s.
func boundary(s []rune, i int) bool {
	if i == 0 {
		return true
	}
	switch s[i-1] {
	case '-', '_', '.', '/', ' ', ':':
		return true
	}
	return unicode.IsLower(s[i-1]) && unicode.IsUpper(s[i])
}

// highlight renders s with the runes at pos emphasised.
func highlight(s string, pos []int, base, hl func(string) string) string {
	if len(pos) == 0 {
		return base(s)
	}
	var b strings.Builder
	set := make(map[int]bool, len(pos))
	for _, p := range pos {
		set[p] = true
	}
	var run []rune
	inHL := false
	flush := func() {
		if len(run) == 0 {
			return
		}
		if inHL {
			b.WriteString(hl(string(run)))
		} else {
			b.WriteString(base(string(run)))
		}
		run = run[:0]
	}
	for i, c := range []rune(s) {
		if set[i] != inHL {
			flush()
			inHL = set[i]
		}
		run = append(run, c)
	}
	flush()
	return b.String()
}
