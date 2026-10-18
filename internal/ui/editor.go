package ui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/cursor"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/gitx"
)

// Subject length guidance, as in git's own conventions.
const (
	subjectSoft = 50
	subjectHard = 72
)

// editor is the modal commit message editor: a one-line subject field and
// an optional multi-line body. It starts in the subject; enter there saves
// (an empty body is fine), tab moves to the body, where enter adds lines.
type editor struct {
	title    string
	subject  textinput.Model
	body     textarea.Model
	inBody   bool
	initial  string
	onSave   func(msg string) tea.Cmd
	discard  bool // esc was pressed once on a modified message
	errorMsg string
}

// splitMessage separates a commit message into subject and body.
func splitMessage(msg string) (subject, body string) {
	msg = gitx.CleanMessage(msg)
	subject, body, _ = strings.Cut(msg, "\n")
	return subject, strings.TrimLeft(body, "\n")
}

// joinMessage is the commit message for a subject and an optional body.
func joinMessage(subject, body string) string {
	subject = strings.TrimSpace(subject)
	body = gitx.CleanMessage(body)
	if body == "" {
		return subject
	}
	return subject + "\n\n" + body
}

func (ed *editor) message() string { return joinMessage(ed.subject.Value(), ed.body.Value()) }

// cursorBlink makes the editor's cursor blink; tests turn it off, as each
// blink is a timer the test harness would wait for.
var cursorBlink = true

func (a *App) openEditor(title, initial string, onSave func(msg string) tea.Cmd) tea.Cmd {
	subj, body := splitMessage(initial)

	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "Summary of the change"
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(cDim)
	ti.SetValue(subj)

	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.CharLimit = 0
	ta.MaxHeight = 0
	ta.Placeholder = "Optional: what changed and why"
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(cDim)
	ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(cDim)
	ta.SetValue(body)
	ta.Blur()

	if !cursorBlink {
		ti.Cursor.SetMode(cursor.CursorStatic)
		ta.Cursor.SetMode(cursor.CursorStatic)
	}
	a.ed = &editor{title: title, subject: ti, body: ta, onSave: onSave}
	a.ed.initial = a.ed.message()
	a.sizeEditor()
	return a.ed.subject.Focus()
}

func (a *App) sizeEditor() {
	if a.ed == nil {
		return
	}
	w := clamp(a.w-10, 10, 100)
	a.ed.subject.Width = w - 1
	a.ed.body.SetWidth(w)
	// Room for borders, padding, title, labels, status and hint lines.
	a.ed.body.SetHeight(clamp(a.h-11-2*editorPad(a.h), 1, 20))
}

func (ed *editor) focusBody(on bool) tea.Cmd {
	ed.inBody = on
	if on {
		ed.subject.Blur()
		return ed.body.Focus()
	}
	ed.body.Blur()
	return ed.subject.Focus()
}

func (a *App) editorKey(k tea.KeyMsg) tea.Cmd {
	ed := a.ed
	act := lookup(scEditor, k.String())
	if act == actEditorEnter && ed.inBody {
		act = actNone // enter in the body is a new line
	}
	switch act {
	case actSave, actEditorEnter:
		if strings.TrimSpace(ed.subject.Value()) == "" {
			ed.errorMsg = "the subject is empty"
			return ed.focusBody(false)
		}
		a.ed = nil
		return ed.onSave(ed.message())
	case actCancel:
		if ed.message() != ed.initial && !ed.discard {
			ed.discard = true
			return nil
		}
		a.ed = nil
		return nil
	case actNextField, actPrevField:
		ed.discard, ed.errorMsg = false, ""
		return ed.focusBody(act == actNextField)
	}
	ed.discard, ed.errorMsg = false, ""
	return ed.update(k)
}

// update passes a message (a key, or the cursor blink) to the focused field.
func (ed *editor) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if ed.inBody {
		ed.body, cmd = ed.body.Update(msg)
	} else {
		ed.subject, cmd = ed.subject.Update(msg)
	}
	return cmd
}

// subjectInfo describes the subject's length against the conventions.
func subjectInfo(subject string) (string, lipgloss.TerminalColor) {
	n := len([]rune(strings.TrimSpace(subject)))
	s := fmt.Sprintf("%d/%d", n, subjectHard)
	var c lipgloss.TerminalColor = cGreen
	switch {
	case n == 0:
		c = cDim
	case n > subjectHard:
		c = cRed
		s += " — too long"
	case n > subjectSoft:
		c = cYellow
	}
	return s, c
}

func (a *App) viewEditor() string {
	ed := a.ed
	w := ed.body.Width()
	field := func(content string, focused bool) string {
		var col lipgloss.TerminalColor = cDim
		if focused {
			col = cAccent
		}
		return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(col).Width(w).Render(content)
	}
	label := func(text string, focused bool) string {
		if focused {
			return lipgloss.NewStyle().Bold(true).Foreground(cAccent).Render(text)
		}
		return fg(cDim, text)
	}
	info, col := subjectInfo(ed.subject.Value())
	parts := []string{
		bold(ed.title) + fg(cDim, " · ") + label("Subject", !ed.inBody) + "  " + fg(col, info),
		field(ed.subject.View(), !ed.inBody),
		label("Body (optional)", ed.inBody),
		field(ed.body.View(), ed.inBody),
	}
	switch {
	case ed.errorMsg != "":
		parts = append(parts, fg(cRed, ed.errorMsg))
	case ed.discard:
		parts = append(parts, fg(cYellow, "discard changes? esc again to discard, any other key to keep editing"))
	}
	hint := "enter save · tab body"
	if ed.inBody {
		hint = "enter new line · shift+tab subject"
	}
	if a.w >= 80 {
		hint += "  ·  " + hints(scEditor)
	}
	parts = append(parts, fg(cDim, hint))
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cAccent).
		Padding(editorPad(a.h), 2).Width(min(w+6, a.w-2)).Render(strings.Join(parts, "\n"))
	return lipgloss.Place(a.w, a.h, lipgloss.Center, lipgloss.Center, box)
}

// editorPad is the vertical padding of the editor box; small windows get none.
func editorPad(h int) int {
	if h < 24 {
		return 0
	}
	return 1
}
