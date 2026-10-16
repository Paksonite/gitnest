package ui

import (
	"strings"
	"unicode"
)

// Scoring weights for fuzzyMatch.
const (
	scoreChar        = 1
	bonusConsecutive = 6
	bonusStart       = 10 // query starts matching at the first character
	bonusBoundary    = 8  // after - _ . / space, or a lower→upper case change
	penaltyGap       = 1  // per skipped character between matches, capped
	maxGapPenalty    = 5
)

// fuzzyMatch reports whether every rune of query appears in s in order
// (case-insensitively). It returns a score (higher is better) and the rune
// positions in s that matched, for highlighting. Among all alignments it
// keeps the best-scoring one that starts at each candidate position.
func fuzzyMatch(s, query string) (score int, pos []int, ok bool) {
	q := []rune(strings.ToLower(query))
	if len(q) == 0 {
		return 0, nil, true
	}
	orig := []rune(s)
	r := []rune(strings.ToLower(s))
	best := -1 << 31
	for start := 0; start < len(r); start++ {
		if r[start] != q[0] {
			continue
		}
		sc, p, matched := alignFrom(r, orig, q, start)
		if matched && sc > best {
			best, pos, ok = sc, p, true
		}
	}
	return best, pos, ok
}

// alignFrom matches q greedily from start, preferring for each query rune
// the next occurrence that sits on a word boundary within a short window.
func alignFrom(r, orig, q []rune, start int) (int, []int, bool) {
	pos := make([]int, 0, len(q))
	score := 0
	i := start
	for qi, c := range q {
		j := -1
		for k := i; k < len(r); k++ {
			if r[k] != c {
				continue
			}
			if j < 0 {
				j = k
			}
			// The first hit is fine when it continues the previous match or
			// is a boundary; otherwise look a little ahead for a boundary.
			if qi == 0 || k == i || boundary(orig, k) {
				j = k
				break
			}
			if k-i > 8 {
				break
			}
		}
		if j < 0 {
			return 0, nil, false
		}
		score += scoreChar
		switch {
		case j == 0:
			score += bonusStart
		case boundary(orig, j):
			score += bonusBoundary
		}
		if qi > 0 {
			prev := pos[len(pos)-1]
			if j == prev+1 {
				score += bonusConsecutive
			} else {
				gap := j - prev - 1
				if gap > maxGapPenalty {
					gap = maxGapPenalty
				}
				score -= gap * penaltyGap
			}
		}
		pos = append(pos, j)
		i = j + 1
	}
	return score, pos, true
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
