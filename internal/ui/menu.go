package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// menu is a small chooser drawn over the screen: sections of options, each
// a radio button (on is set), or an action. The filters (F), the actions on
// marked repositories (a) and the sort & view chooser (v) are menus.
type menu struct {
	title string
	note  string // a dim line under the title
	items []menuItem
	cur   int
}

type menuItem struct {
	label  string
	right  string      // a count or a detail, right-aligned
	key    string      // a direct shortcut ("" for none)
	header bool        // section title, not selectable
	on     func() bool // radio button state; nil for an action
	pick   func() tea.Cmd
	stay   bool // picking keeps the menu open (radio buttons apply live)
}

func menuHeader(title string) menuItem { return menuItem{label: title, header: true} }

// openMenu shows m with the cursor on its first selected option (or the first
// option).
func (a *App) openMenu(m *menu) {
	m.cur = -1
	for i, it := range m.items {
		if it.header {
			continue
		}
		if m.cur < 0 || (it.on != nil && it.on()) {
			m.cur = i
			if it.on != nil && it.on() {
				break
			}
		}
	}
	if m.cur < 0 {
		m.cur = 0
	}
	a.menu = m
}

func (a *App) menuKey(k tea.KeyMsg) tea.Cmd {
	m := a.menu
	move := func(d int) {
		for i := m.cur + d; i >= 0 && i < len(m.items); i += d {
			if !m.items[i].header {
				m.cur = i
				return
			}
		}
	}
	choose := func(i int) tea.Cmd {
		it := m.items[i]
		if !it.stay {
			a.menu = nil
		}
		if it.pick == nil {
			return nil
		}
		return it.pick()
	}
	switch lookup(scMenu, k.String()) {
	case actUp:
		move(-1)
		return nil
	case actDown:
		move(1)
		return nil
	case actSave:
		return choose(m.cur)
	case actCancel:
		a.menu = nil
		return nil
	}
	for i, it := range m.items {
		if it.key != "" && it.key == k.String() && !it.header {
			m.cur = i
			return choose(i)
		}
	}
	return nil
}

func (a *App) viewMenu() string {
	m := a.menu
	w := clamp(a.w-4, 30, 62)
	inner := w - 6
	listH := clamp(a.h-8-2*editorPad(a.h), 1, len(m.items))
	off := scrollTo(m.cur, clamp(m.cur-listH/2, 0, len(m.items)), listH)
	var lines []string
	for i := off; i < len(m.items) && i < off+listH; i++ {
		it := m.items[i]
		if it.header {
			if i > off {
				lines = append(lines, "")
			}
			lines = append(lines, bold(fg(cAccent, it.label)))
			continue
		}
		mark, smark := "  ", "  "
		if it.on != nil {
			mark, smark = "○ ", fg(cDim, "○ ")
			if it.on() {
				mark, smark = "◉ ", fg(cAccent, "◉ ")
			}
		}
		right := it.right
		if it.key != "" {
			right = strings.TrimSpace(right + "  " + keyLabel(it.key))
		}
		label := fit(it.label, max(1, inner-2-lipgloss.Width(right)-1))
		gap := strings.Repeat(" ", max(1, inner-2-lipgloss.Width(label)-lipgloss.Width(right)))
		if i == m.cur {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Background(cSel).Foreground(cText).Render(fit(mark+label+gap+right, inner)))
			continue
		}
		sright := fg(cDim, it.right)
		if it.key != "" {
			sright = strings.TrimSpace(fg(cDim, it.right) + "  " + fg(cAccent, keyLabel(it.key)))
		}
		lines = append(lines, fit(smark+label+gap+sright, inner))
	}
	body := bold(m.title)
	if m.note != "" {
		body += "\n" + fg(cDim, fit(m.note, inner))
	}
	body += "\n\n" + strings.Join(lines, "\n") + "\n\n" + hints(scMenu)
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cAccent).
		Padding(editorPad(a.h), 2).Width(w).Render(body)
	return lipgloss.Place(a.w, a.h, lipgloss.Center, lipgloss.Center, box)
}

// countLabel is "label" with a count for menus, e.g. a repository total.
func countLabel(n int) string { return fmt.Sprint(n) }
