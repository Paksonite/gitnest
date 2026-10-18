package ui

import (
	"fmt"
	"hash/fnv"
	"sort"
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

// hashState says how far a commit has travelled:
// on the main line (green), pushed (yellow), only local (red).
type hashState int

const (
	hashMain hashState = iota
	hashPushed
	hashLocal
)

// commitRow is a Commits panel row: hash, author initials, the graph, the
// subject and refs.
func commitRow(c gitx.Commit, st hashState, g []gcell, copied bool) rowItem {
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
	gp, gs := graphText(g, authorColor(c.Author))
	mark := " "
	if copied {
		mark = "▍"
	}
	subject := printable(c.Subject)
	plain := mark + c.Short + " " + ini + " " + gp + " " + subject + refs
	styled := fg(cAccent, mark) + fg(hc, c.Short) + " " +
		lipgloss.NewStyle().Bold(true).Foreground(authorColor(c.Author)).Render(ini) + " " +
		gs + " " + subject + fg(cCyan, refs)
	return rowItem{plain: plain, styled: styled}
}

func refRow(e gitx.RefEntry) rowItem {
	icon, ic := refIcon(e.Kind())
	msg := printable(e.Message)
	plain := icon + " " + e.Short + " " + e.Action + " " + msg
	styled := fg(ic, icon) + " " + fg(cYellow, e.Short) + " " + fg(cCyan, e.Action) + " " + msg
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

// clipboard holds commits copied for cherry-picking; it survives switching
// repositories, so commits can be pasted into another one.
type clipboard struct {
	repo, name string
	commits    []copiedCommit
}

type copiedCommit struct {
	hash, short, subject string
	order                int // position in the source log: higher is older
}

func (a *App) isCopied(hash string) bool {
	if a.sh.copied == nil || a.sh.copied.repo != a.rp.dir {
		return false
	}
	for _, c := range a.sh.copied.commits {
		if c.hash == hash {
			return true
		}
	}
	return false
}

// copyCommit toggles commit i of the open repository in the clipboard.
// Copying in another repository starts a new clipboard.
func (a *App) copyCommit(i int) {
	rp := &a.rp
	c := rp.commits[i]
	if a.sh.copied == nil || a.sh.copied.repo != rp.dir {
		a.sh.copied = &clipboard{repo: rp.dir, name: rp.name}
	}
	for k, x := range a.sh.copied.commits {
		if x.hash == c.Hash {
			a.sh.copied.commits = append(a.sh.copied.commits[:k], a.sh.copied.commits[k+1:]...)
			if len(a.sh.copied.commits) == 0 {
				a.sh.copied = nil
			}
			return
		}
	}
	a.sh.copied.commits = append(a.sh.copied.commits, copiedCommit{hash: c.Hash, short: c.Short, subject: c.Subject, order: i})
}

func (a *App) copiedBadge() string {
	if a.sh.copied == nil {
		return ""
	}
	return "  " + badge(fmt.Sprintf("⎘ %d copied from %s · v pastes", len(a.sh.copied.commits), a.sh.copied.name), cAccent)
}

// paste cherry-picks the copied commits, oldest first, onto the current
// branch of the open repository.
func (a *App) paste() tea.Cmd {
	cb := a.sh.copied
	if cb == nil || len(cb.commits) == 0 {
		a.setToast("nothing copied — c copies the selected commit", false)
		return nil
	}
	rp := &a.rp
	dir, ctx, src := rp.dir, a.ctx, cb.repo
	cs := append([]copiedCommit(nil), cb.commits...)
	sort.Slice(cs, func(i, j int) bool { return cs[i].order > cs[j].order })
	hashes := make([]string, len(cs))
	for i, c := range cs {
		hashes[i] = c.hash
	}
	how := ""
	if src != dir {
		how = " (as patches from " + cb.name + ")"
	}
	a.confirm(fmt.Sprintf("Cherry-pick %d commit(s) onto %s%s?", len(hashes), rp.info.Branch, how), func() tea.Cmd {
		a.sh.copied = nil
		return a.doUndoable(fmt.Sprintf("cherry-pick %d", len(hashes)), func() error {
			return gitx.CherryPick(ctx, dir, src, hashes)
		})
	})
	return nil
}
