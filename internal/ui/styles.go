package ui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/v2/internal/config"
	"github.com/Paksonite/gitnest/v2/internal/gitx"
	"github.com/Paksonite/gitnest/v2/internal/theme"
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
	cSel     lipgloss.TerminalColor = lipgloss.Color("#1d4a44")
	cSelDim  lipgloss.TerminalColor = lipgloss.Color("#15302f")
	cBorder  lipgloss.TerminalColor = lipgloss.Color("#2f5254")
	cSurface lipgloss.TerminalColor = lipgloss.Color("#0f2024")
	cBack    lipgloss.TerminalColor = lipgloss.Color("#0b1417") // the whole screen
	cAuthors []lipgloss.TerminalColor
	palette  = theme.Palette{Background: "#0b1417"} // the active one, for tints
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
	cSurface, cBack, palette = color(p.Surface), color(p.Background), p
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
// selected row, so the highlight always shows where keys go; an accent bar
// marks its left edge when the row starts with a space.
func renderRows(items []rowItem, cur, off, h, w int, focused bool) []string {
	var out []string
	sel := lipgloss.NewStyle().Bold(true).Background(cSel).Foreground(cText)
	for i := off; i < len(items) && i < off+h; i++ {
		it := items[i]
		switch {
		case i == cur && focused && strings.HasPrefix(it.plain, " ") && w > 1:
			bar := lipgloss.NewStyle().Foreground(cAccent).Background(cSel).Render("▌")
			out = append(out, bar+sel.Render(fit(it.plain[1:], w-1)))
		case i == cur && focused:
			out = append(out, sel.Render(fit(it.plain, w)))
		default:
			out = append(out, fit(it.styled, w))
		}
	}
	return out
}

// panelPad is the space between a panel's border and its content.
const panelPad = 1

// panelInner is the width of a w-wide panel's content.
func panelInner(w int) int { return max(1, w-2-2*panelPad) }

// subtitle joins a panel title and a dimmed note shown after it
// ("Repositories" · "55 · 4 dirty").
func subtitle(title, note string) string {
	if note == "" {
		return title
	}
	return title + "\x1f" + note
}

// panel draws a rounded card of exactly w×h cells: the title (and its
// subtitle) as the first row, the lines below, all on the theme's surface
// colour when it has one.
func panel(title string, lines []string, w, h int, focused bool) string {
	if w < 4+2*panelPad {
		w = 4 + 2*panelPad
	}
	if h < 3 {
		h = 3
	}
	inner, ih := panelInner(w), h-2
	col := cBorder
	ts := lipgloss.NewStyle().Foreground(cText).Bold(true)
	if focused {
		col = cAccent
		ts = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	}
	title, note, _ := strings.Cut(title, "\x1f")
	if !strings.Contains(title, "\x1b") { // pre-styled titles (tabs) are kept as they are
		title = ts.Render(title)
	}
	if note != "" {
		title += "  " + fg(cDim, note)
	}
	pad := strings.Repeat(" ", panelPad)
	bg := themeSeq(cSurface)
	row := func(s string) string { return paintBG(pad+fit(s, inner)+pad, bg) }
	rows := make([]string, 0, ih)
	rows = append(rows, row(title))
	for _, l := range lines {
		if len(rows) >= ih {
			break
		}
		rows = append(rows, row(l))
	}
	for len(rows) < ih {
		rows = append(rows, row(""))
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(col).
		BorderBackground(cSurface).Render(strings.Join(rows, "\n"))
}

// themeSeq is the escape sequence that sets background bg and the theme's
// text colour, so text without a colour of its own is readable on bg
// whatever the terminal's colours are ("" when colours are off or bg is
// NoColor).
func themeSeq(bg lipgloss.TerminalColor) string {
	if _, none := bg.(lipgloss.NoColor); none {
		return ""
	}
	s := lipgloss.NewStyle().Background(bg).Foreground(cText).Render("x")
	seq, _, _ := strings.Cut(s, "x")
	return seq
}

// paintScreen gives the whole w×h screen the theme's background and text
// colour (themes without a background keep the terminal's).
func paintScreen(view string, w, h int) string {
	seq := themeSeq(cBack)
	if seq == "" {
		return view
	}
	lines := strings.Split(view, "\n")
	for len(lines) < h {
		lines = append(lines, "")
	}
	for i, l := range lines {
		width := w
		if i == len(lines)-1 {
			width = w - 1 // never write the bottom-right cell (see App.footer)
		}
		lines[i] = paintBG(fit(l, width), seq)
	}
	// The renderer clears what is left of a line shorter than the window
	// (erase to the end of the line), and terminals clear with the current
	// background: end the last line on the theme's, so the bottom-right
	// cell gets it too instead of the terminal's own (a square of another
	// colour in the corner).
	lines[len(lines)-1] += seq
	return strings.Join(lines, "\n")
}

// paintBG puts background seq under a whole styled line: after every reset
// inside it (lipgloss's and git's), the background is set again.
func paintBG(s, seq string) string {
	if seq == "" {
		return s
	}
	s = strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+seq)
	s = strings.ReplaceAll(s, "\x1b[m", "\x1b[m"+seq)
	return seq + s + "\x1b[0m"
}

// badge is a small label on a tint of its colour: " dirty only ".
func badge(s string, c lipgloss.TerminalColor) string {
	return lipgloss.NewStyle().Foreground(c).Background(tint(c, 0.22)).Render(" " + s + " ")
}

// tint mixes colour c into the theme background (amount 0..1); colours
// that aren't #rrggbb fall back to the dim selection colour.
func tint(c lipgloss.TerminalColor, amount float64) lipgloss.TerminalColor {
	hex, ok := c.(lipgloss.Color)
	r1, g1, b1, ok1 := parseHex(string(hex))
	r0, g0, b0, ok0 := parseHex(palette.Background)
	if !ok || !ok1 || !ok0 {
		return cSelDim
	}
	mix := func(a, b int) int { return int(float64(b) + (float64(a)-float64(b))*amount + 0.5) }
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", mix(r1, r0), mix(g1, g0), mix(b1, b0)))
}

func parseHex(s string) (r, g, b int, ok bool) {
	if len(s) != 7 || s[0] != '#' {
		return 0, 0, 0, false
	}
	_, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return r, g, b, err == nil
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
		b += " " + lipgloss.NewStyle().Bold(true).Foreground(cYellow).Background(tint(cYellow, 0.22)).Render(" dev ")
	}
	return b
}
