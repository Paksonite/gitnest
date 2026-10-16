package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/config"
)

var (
	cAccent = lipgloss.Color("#cba6f7")
	cDim    = lipgloss.Color("#6c7086")
	cGreen  = lipgloss.Color("#a6e3a1")
	cRed    = lipgloss.Color("#f38ba8")
	cYellow = lipgloss.Color("#f9e2af")
	cCyan   = lipgloss.Color("#89dceb")
	cText   = lipgloss.Color("#cdd6f4")
	cSel    = lipgloss.Color("#45475a")
	cSelDim = lipgloss.Color("#313244")
)

func applyTheme(cfg config.Config) {
	if cfg.Theme.Accent != "" {
		cAccent = lipgloss.Color(cfg.Theme.Accent)
	}
}

func fg(c lipgloss.Color, s string) string { return lipgloss.NewStyle().Foreground(c).Render(s) }
func bold(s string) string                 { return lipgloss.NewStyle().Bold(true).Render(s) }

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

func expandTabs(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\t", "    "), "\r", "")
}

type rowItem struct{ plain, styled string }

// renderRows draws list rows. The selected row is highlighted strongly in the
// focused panel and faintly elsewhere, so every panel shows its selection.
func renderRows(items []rowItem, cur, off, h, w int, focused bool) []string {
	var out []string
	for i := off; i < len(items) && i < off+h; i++ {
		it := items[i]
		switch {
		case i == cur && focused:
			out = append(out, lipgloss.NewStyle().Bold(true).Background(cSel).Foreground(cText).Render(fit(it.plain, w)))
		case i == cur:
			out = append(out, lipgloss.NewStyle().Background(cSelDim).Foreground(cText).Render(fit(it.plain, w)))
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
	col := cDim
	ts := lipgloss.NewStyle().Foreground(cDim)
	if focused {
		col = cAccent
		ts = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	}
	rows := make([]string, 0, ih)
	rows = append(rows, fit(ts.Render(title), inner))
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
