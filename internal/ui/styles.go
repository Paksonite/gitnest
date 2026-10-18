package ui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/config"
	"github.com/Paksonite/gitnest/internal/gitx"
	"github.com/Paksonite/gitnest/internal/theme"
)

// The active palette. These are set from the colour theme by usePalette;
// the defaults are gitnest-dark. Terminals without true colour get the
// nearest 256/16-colour match from lipgloss.
var (
	cAccent  lipgloss.TerminalColor = lipgloss.Color("#2fd896")
	cDim     lipgloss.TerminalColor = lipgloss.Color("#6f8a89")
	cGreen   lipgloss.TerminalColor = lipgloss.Color("#4ade80")
	cRed     lipgloss.TerminalColor = lipgloss.Color("#f87171")
	cYellow  lipgloss.TerminalColor = lipgloss.Color("#facc15")
	cCyan    lipgloss.TerminalColor = lipgloss.Color("#5fb3ff")
	cText    lipgloss.TerminalColor = lipgloss.Color("#d6e3e1")
	cSel     lipgloss.TerminalColor = lipgloss.Color("#173a37")
	cSelDim  lipgloss.TerminalColor = lipgloss.Color("#11272a")
	cBorder  lipgloss.TerminalColor = lipgloss.Color("#2a4a4c")
	cAuthors []lipgloss.TerminalColor
)

func color(c string) lipgloss.TerminalColor {
	if c == "" {
		return lipgloss.NoColor{}
	}
	return lipgloss.Color(c)
}

// usePalette switches the UI (and git's diff colours, unless diffs keep
// git's own) to p. accent, when set, overrides the palette's accent.
func usePalette(p theme.Palette, accent string, gitDiffs bool) {
	if accent != "" {
		p.Accent = accent
	}
	cAccent, cDim, cText = color(p.Accent), color(p.Dim), color(p.Text)
	cGreen, cRed, cYellow, cCyan = color(p.Added), color(p.Removed), color(p.Warn), color(p.Branch)
	cSel, cSelDim, cBorder = color(p.Sel), color(p.SelDim), color(p.Border)
	cAuthors = cAuthors[:0]
	for _, c := range p.Authors {
		cAuthors = append(cAuthors, color(c))
	}
	if gitDiffs {
		gitx.SetColors(nil)
		return
	}
	gitx.SetColors(map[string]string{
		"color.diff.new": p.DiffAdd, "color.diff.old": p.DiffDel,
		"color.diff.frag": p.DiffHunk, "color.diff.meta": p.DiffMeta,
		"color.status.added": p.Added, "color.status.changed": p.Removed, "color.status.untracked": p.Removed,
		"color.status.branch": p.Branch, "color.decorate.branch": p.Branch, "color.decorate.HEAD": p.Accent,
		"color.decorate.remoteBranch": p.Removed, "color.decorate.tag": p.Warn,
	})
}

// darkBackground decides between the dark and light theme: theme.mode wins,
// otherwise the terminal is asked.
func darkBackground(cfg config.Config) bool {
	switch strings.ToLower(cfg.Theme.Mode) {
	case "dark":
		return true
	case "light":
		return false
	}
	return lipgloss.HasDarkBackground()
}

// pickTheme returns the configured theme for the background, falling back
// to the default built-in when the name is unknown.
func pickTheme(set *theme.Set, cfg config.Config, dark bool) (theme.Theme, error) {
	name, fallback := cfg.Theme.Light, "gitnest-light"
	if dark {
		name, fallback = cfg.Theme.Dark, "gitnest-dark"
	}
	if name == "" {
		name = fallback
	}
	if t, ok := set.Get(name); ok {
		return t, nil
	}
	t, _ := set.Get(fallback)
	return t, fmt.Errorf("unknown theme %q, using %s (see gitnest theme list)", name, fallback)
}

func fg(c lipgloss.TerminalColor, s string) string {
	return lipgloss.NewStyle().Foreground(c).Render(s)
}
func bold(s string) string { return lipgloss.NewStyle().Bold(true).Render(s) }

// fit truncates s to w cells (ANSI-aware) and pads with spaces to exactly w.
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = lipgloss.NewStyle().MaxWidth(w).Render(s)
	if pad := w - lipgloss.Width(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// centerOff is the first of n rows to show in h lines so that row cur sits
// near the middle, without scrolling past either end: when all n fit, 0.
func centerOff(cur, n, h int) int {
	return clamp(cur-h/2, 0, max(0, n-h))
}

func scrollTo(cur, off, h int) int {
	if h <= 0 {
		return 0
	}
	if cur < off {
		off = cur
	}
	if cur >= off+h {
		off = cur - h + 1
	}
	if off < 0 {
		off = 0
	}
	return off
}

// printable shows control characters in names and subjects as escapes
// (a tab as \t), so they can't break the layout of a row.
func printable(s string) string {
	if !strings.ContainsFunc(s, unicode.IsControl) {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\n':
			b.WriteString(`\n`)
		case unicode.IsControl(r):
			fmt.Fprintf(&b, `\x%02x`, r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func expandTabs(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\t", "    "), "\r", "")
}

type rowItem struct{ plain, styled string }

// renderRows draws list rows. Only the focused panel highlights its
// selected row, so the highlight always shows where keys go.
func renderRows(items []rowItem, cur, off, h, w int, focused bool) []string {
	var out []string
	for i := off; i < len(items) && i < off+h; i++ {
		it := items[i]
		switch {
		case i == cur && focused:
			out = append(out, lipgloss.NewStyle().Bold(true).Background(cSel).Foreground(cText).Render(fit(it.plain, w)))
		default:
			out = append(out, fit(it.styled, w))
		}
	}
	return out
}

// panel draws a rounded box of exactly w×h cells with the title as its first row.
func panel(title string, lines []string, w, h int, focused bool) string {
	if w < 4 {
		w = 4
	}
	if h < 3 {
		h = 3
	}
	inner, ih := w-2, h-2
	col := cBorder
	ts := lipgloss.NewStyle().Foreground(cDim)
	if focused {
		col = cAccent
		ts = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	}
	rows := make([]string, 0, ih)
	if !strings.Contains(title, "\x1b") { // pre-styled titles (tabs) are kept as they are
		title = ts.Render(title)
	}
	rows = append(rows, fit(title, inner))
	for _, l := range lines {
		if len(rows) >= ih {
			break
		}
		rows = append(rows, fit(l, inner))
	}
	for len(rows) < ih {
		rows = append(rows, strings.Repeat(" ", inner))
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(col).Render(strings.Join(rows, "\n"))
}

// tabsTitle renders a panel title with tabs: "[3] Commits - Reflog"
// with the active tab highlighted.
func tabsTitle(num string, tabs []string, active int, focused bool) string {
	numStyle := lipgloss.NewStyle().Foreground(cDim)
	if focused {
		numStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	}
	parts := make([]string, len(tabs))
	for i, t := range tabs {
		if i == active {
			parts[i] = lipgloss.NewStyle().Foreground(cGreen).Bold(true).Render(t)
		} else {
			parts[i] = fg(cDim, t)
		}
	}
	return numStyle.Render(num) + " " + strings.Join(parts, fg(cDim, " - "))
}

// brand is the "gitnest" at the start of every header; preview builds
// (gitnest-dev, version dev-<commit>) add a "dev" tag so they can't be
// mistaken for the released one.
func brand() string {
	b := lipgloss.NewStyle().Bold(true).Foreground(cAccent).Render("gitnest")
	if strings.HasPrefix(Version, "dev-") {
		b += " " + lipgloss.NewStyle().Bold(true).Reverse(true).Foreground(cYellow).Render(" dev ")
	}
	return b
}
