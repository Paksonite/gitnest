package ui

import (
	"hash/fnv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/gitx"
)

// refIcon is the icon and colour for a reflog action (see refGlyph).

func refIcon(k gitx.RefKind) (glyph string, c lipgloss.TerminalColor) {
	glyph = refGlyph(k)
	switch k {
	case gitx.RefCommit:
		return glyph, cAccent
	case gitx.RefAmend:
		return glyph, cYellow
	case gitx.RefMerge, gitx.RefRebase:
		return glyph, cCyan
	case gitx.RefReset, gitx.RefRevert:
		return glyph, cRed
	case gitx.RefPull, gitx.RefCherryPick, gitx.RefInitial:
		return glyph, cGreen
	}
	return glyph, cDim
}

// initials turns an author name into two letters: "Michał Wadecki" → "MW",
// "dependabot[bot]" → "DB", "x" → "X ".
func initials(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	switch len(words) {
	case 0:
		return "??"
	case 1:
		r := []rune(strings.ToUpper(words[0]))
		if len(r) == 1 {
			return string(r) + " "
		}
		return string(r[:2])
	}
	first := []rune(words[0])[0]
	last := []rune(words[len(words)-1])[0]
	return strings.ToUpper(string([]rune{first, last}))
}

// authorColor gives every author a stable colour from the theme.
func authorColor(name string) lipgloss.TerminalColor {
	if len(cAuthors) == 0 {
		return cCyan
	}
	h := fnv.New32a()
	h.Write([]byte(name))
	return cAuthors[h.Sum32()%uint32(len(cAuthors))]
}

// graph colours git's --graph drawing: commits in the accent, lines dimmed.

// hashState says how far a commit has travelled, coloured like lazygit:
// on the main line (green), pushed (yellow), only local (red).
type hashState int

const (
	hashMain hashState = iota
	hashPushed
	hashLocal
)

// commitRow is a Commits panel row: hash, author initials, the graph, the
// subject and refs.
func commitRow(c gitx.Commit, st hashState, g []gcell) rowItem {
	hc := cGreen
	switch st {
	case hashPushed:
		hc = cYellow
	case hashLocal:
		hc = cRed
	}
	refs := ""
	if c.Refs != "" {
		refs = " (" + c.Refs + ")"
	}
	ini := initials(c.Author)
	gp, gs := graphText(g)
	plain := c.Short + " " + ini + " " + gp + " " + c.Subject + refs
	styled := fg(hc, c.Short) + " " +
		lipgloss.NewStyle().Bold(true).Foreground(authorColor(c.Author)).Render(ini) + " " +
		gs + " " + c.Subject + fg(cCyan, refs)
	return rowItem{plain: plain, styled: styled}
}

func refRow(e gitx.RefEntry) rowItem {
	icon, ic := refIcon(e.Kind())
	plain := icon + " " + e.Short + " " + e.Action + " " + e.Message
	styled := fg(ic, icon) + " " + fg(cYellow, e.Short) + " " + fg(cCyan, e.Action) + " " + e.Message
	return rowItem{plain: plain, styled: styled}
}

// toggleReflog switches the Commits panel between commits and the reflog,
// each keeping its own cursor.
func (a *App) toggleReflog() tea.Cmd {
	rp := &a.rp
	rp.cur[pCommits], rp.otherCur = rp.otherCur, rp.cur[pCommits]
	rp.off[pCommits] = 0
	rp.reflog = !rp.reflog
	rp.cur[pCommits] = clamp(rp.cur[pCommits], 0, rp.count(pCommits)-1)
	return a.mainCmd()
}

// reflogKey handles the Commits panel while it shows the reflog.
func (a *App) reflogKey(s string) (tea.Cmd, bool) {
	rp := &a.rp
	dir, ctx := rp.dir, a.ctx
	act := lookup(scReflog, s)
	if act == actToggleReflog {
		return a.toggleReflog(), true
	}
	if len(rp.refs) == 0 {
		return nil, act != actNone
	}
	e := rp.refs[rp.cur[pCommits]]
	switch act {
	case actCheckoutEntry:
		a.confirm("Check out "+e.Short+" ("+e.Selector+") as a detached HEAD?", func() tea.Cmd {
			return a.do("checkout "+e.Short, func() error { return gitx.CheckoutDetached(ctx, dir, e.Hash) })
		})
		return nil, true
	case actResetHere:
		branch := rp.info.Branch
		a.confirm("Reset "+branch+" to "+e.Short+" ("+e.Selector+": "+e.Action+")? Uncommitted changes must be stashed first; z undoes.", func() tea.Cmd {
			return a.doUndoable("reset to "+e.Short, func() error {
				return gitx.ResetHard(ctx, dir, e.Hash, "reset to "+e.Selector)
			})
		})
		return nil, true
	}
	return nil, false
}
