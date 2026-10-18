package ui

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Selecting text with the mouse in the right-hand panel (the details card in
// the list, the diff in a repository): drag to select, release to copy.
// While the mouse is on, the terminal's own selection needs shift (option
// on macOS) held down; this one works without.
type textSel struct {
	on, dragging   bool
	ax, ay, bx, by int // anchor and the other end, in screen cells
}

// selArea is the right-hand panel's content: columns x0..x1, rows y0..y1
// (inclusive), or ok=false when there is none.
func (a *App) selArea() (x0, y0, x1, y1 int, ok bool) {
	var left int
	switch {
	case a.browse != nil || a.overlayOpen():
		return 0, 0, 0, 0, false
	case a.scr == scrRepos && a.w >= 90:
		left = a.rs.listW
	case a.scr == scrRepo && a.rp.leftW > 0:
		left = a.rp.leftW
	default:
		return 0, 0, 0, 0, false
	}
	// border + padding on each side; header, border and title above, the
	// bottom border and the two footer lines below
	return left + 1 + panelPad, 3, a.w - 2 - panelPad, a.h - 5, true
}

// selMouse handles the left button in the right-hand panel; it reports
// whether it took the event.
func (a *App) selMouse(m tea.MouseMsg) (tea.Cmd, bool) {
	x0, y0, x1, y1, ok := a.selArea()
	s := &a.sel
	switch {
	case m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft:
		if !ok || m.X < x0 || m.X > x1 || m.Y < y0 || m.Y > y1 {
			s.on = false
			return nil, false
		}
		*s = textSel{dragging: true, ax: m.X, ay: m.Y, bx: m.X, by: m.Y}
		return nil, false // a plain click still does what clicks do
	case m.Action == tea.MouseActionMotion && s.dragging:
		s.bx, s.by = clamp(m.X, x0, x1), clamp(m.Y, y0, y1)
		s.on = s.bx != s.ax || s.by != s.ay
		return nil, true
	case m.Action == tea.MouseActionRelease && s.dragging:
		s.dragging = false
		if !s.on {
			return nil, true
		}
		text := a.selText()
		if text == "" {
			s.on = false
			return nil, true
		}
		a.setToast(fmt.Sprintf("copied %d characters", len([]rune(text))), false)
		return copyText(text), true
	}
	return nil, false
}

// selRange orders the selection's ends: start (x, y) before end.
func (s textSel) selRange() (sx, sy, ex, ey int) {
	sx, sy, ex, ey = s.ax, s.ay, s.bx, s.by
	if ey < sy || (ey == sy && ex < sx) {
		sx, sy, ex, ey = ex, ey, sx, sy
	}
	return
}

// selSpan is the columns of screen row y that the selection covers, like a
// terminal's: from the start to the panel's end on the first row, the whole
// panel width in between, up to the end on the last.
func (a *App) selSpan(y int) (from, to int, ok bool) {
	x0, _, x1, _, _ := a.selArea()
	sx, sy, ex, ey := a.sel.selRange()
	if y < sy || y > ey {
		return 0, 0, false
	}
	from, to = x0, x1
	if y == sy {
		from = sx
	}
	if y == ey {
		to = ex
	}
	return from, to, from <= to
}

// selText is the selected text, without colours, trailing spaces trimmed.
func (a *App) selText() string {
	lines := strings.Split(a.screen(), "\n")
	_, sy, _, ey := a.sel.selRange()
	var out []string
	for y := sy; y <= ey && y < len(lines); y++ {
		from, to, ok := a.selSpan(y)
		if !ok {
			continue
		}
		out = append(out, strings.TrimRight(ansi.Strip(ansi.Cut(lines[y], from, to+1)), " "))
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// withSelection draws the selection over a rendered screen.
func (a *App) withSelection(view string) string {
	if !a.sel.on {
		return view
	}
	lines := strings.Split(view, "\n")
	st := lipgloss.NewStyle().Background(cSel).Foreground(cText)
	for y := range lines {
		from, to, ok := a.selSpan(y)
		if !ok {
			continue
		}
		l := lines[y]
		mid := ansi.Strip(ansi.Cut(l, from, to+1))
		lines[y] = ansi.Cut(l, 0, from) + st.Render(mid) + ansi.Cut(l, to+1, ansi.StringWidth(l))
	}
	return strings.Join(lines, "\n")
}

// copyText puts text on the clipboard: pbcopy on macOS, wl-copy or xclip
// with a display, else the terminal's clipboard (OSC 52, which iTerm2,
// kitty, WezTerm, Ghostty, Windows Terminal and tmux understand).
func copyText(text string) tea.Cmd {
	return func() tea.Msg {
		var c *exec.Cmd
		switch {
		case runtime.GOOS == "darwin":
			c = exec.Command("pbcopy")
		case os.Getenv("WAYLAND_DISPLAY") != "":
			c = exec.Command("wl-copy")
		case os.Getenv("DISPLAY") != "":
			c = exec.Command("xclip", "-selection", "clipboard")
		}
		if c != nil {
			c.Stdin = strings.NewReader(text)
			if c.Run() == nil {
				return nil
			}
		}
		fmt.Fprint(os.Stdout, "\x1b]52;c;"+base64.StdEncoding.EncodeToString([]byte(text))+"\x07")
		return nil
	}
}
