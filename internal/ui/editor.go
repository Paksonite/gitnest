package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/gitx"
)

// Subject length guidance, as in git's own conventions.
const (
	subjectSoft = 50
	subjectHard = 72
)

// editor is a modal multi-line commit message editor: the first line is the
// subject, a blank line, then the body.
type editor struct {
	title    string
	ta       textarea.Model
	initial  string
	onSave   func(msg string) tea.Cmd
	discard  bool // esc was pressed once on a modified message
	errorMsg string
}

func (a *App) openEditor(title, initial string, onSave func(msg string) tea.Cmd) tea.Cmd {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.CharLimit = 0
	ta.MaxHeight = 0
	ta.Placeholder = "Subject line\n\nOptional body: what changed and why"
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(cDim)
	ta.SetValue(initial)
	a.ed = &editor{title: title, ta: ta, initial: initial, onSave: onSave}
	a.sizeEditor()
	return a.ed.ta.Focus()
}

func (a *App) sizeEditor() {
	if a.ed == nil {
		return
	}
	a.ed.ta.SetWidth(clamp(a.w-12, 20, 100))
	a.ed.ta.SetHeight(clamp(a.h-14, 3, 20))
}

func (a *App) editorKey(k tea.KeyMsg) tea.Cmd {
	ed := a.ed
	switch k.String() {
	case "ctrl+s", "alt+enter":
		msg := gitx.CleanMessage(ed.ta.Value())
		if msg == "" {
			ed.errorMsg = "the message is empty"
			return nil
		}
		a.ed = nil
		return ed.onSave(msg)
	case "esc":
		if ed.ta.Value() != ed.initial && !ed.discard {
			ed.discard = true
			return nil
		}
		a.ed = nil
		return nil
	}
	ed.discard, ed.errorMsg = false, ""
	var cmd tea.Cmd
	ed.ta, cmd = ed.ta.Update(k)
	return cmd
}

// subjectInfo describes the first line's length against the conventions.
func subjectInfo(msg string) (string, lipgloss.Color) {
	subject, rest, _ := strings.Cut(msg, "\n")
	subject = strings.TrimRight(subject, " \t") // trailing spaces are dropped on save
	n := len([]rune(subject))
	s := fmt.Sprintf("subject %d/%d", n, subjectHard)
	c := cGreen
	switch {
	case n == 0:
		c = cDim
	case n > subjectHard:
		c = cRed
		s += " — too long"
	case n > subjectSoft:
		c = cYellow
	}
	if rest != "" && !strings.HasPrefix(rest, "\n") {
		s += " · add a blank line after the subject"
		if c == cGreen {
			c = cYellow
		}
	}
	return s, c
}

func (a *App) viewEditor() string {
	ed := a.ed
	info, col := subjectInfo(ed.ta.Value())
	status := fg(col, info)
	switch {
	case ed.errorMsg != "":
		status = fg(cRed, ed.errorMsg)
	case ed.discard:
		status = fg(cYellow, "discard changes? esc again to discard, any other key to keep editing")
	}
	body := bold(ed.title) + "\n\n" + ed.ta.View() + "\n\n" + status + "\n" +
		fg(cDim, "ctrl+s or alt+enter save · enter new line · esc cancel")
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cAccent).
		Padding(1, 2).Render(body)
	return lipgloss.Place(a.w, a.h, lipgloss.Center, lipgloss.Center, box)
}
