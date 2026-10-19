package ui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Through Bubble Tea's own input reader: the bytes iTerm2 sends for
// option+arrows must reach the panes as alt+arrows.
func TestITermOptionArrows(t *testing.T) {
	for seq, want := range map[string]string{
		"\x1b[1;9D": "alt+left", "\x1b[1;9C": "alt+right", "\x1b[1;9A": "alt+up", "\x1b[1;9B": "alt+down",
	} {
		var got []tea.Msg
		m := recorder{msgs: &got}
		p := tea.NewProgram(m, tea.WithInput(bytes.NewBufferString(seq)), tea.WithOutput(&bytes.Buffer{}), tea.WithoutRenderer())
		if _, err := p.Run(); err != nil {
			t.Fatal(err)
		}
		var keys []string
		for _, msg := range got {
			if k, ok := metaArrow(msg); ok {
				keys = append(keys, k.String())
			}
		}
		if len(keys) != 1 || keys[0] != want {
			t.Errorf("%q: %v, want %s", seq, keys, want)
		}
	}
	for _, seq := range []string{"\x1b[1;9E", "\x1b[1;3D", "\x1b[1;99D", ""} {
		if k, ok := metaArrowSeq(seq); ok {
			t.Errorf("%q: %s", seq, k)
		}
	}
	if _, ok := metaArrow(tea.KeyMsg{Type: tea.KeyLeft}); ok {
		t.Error("a plain key is no meta arrow")
	}
}

// recorder keeps every message for a moment, then quits.
type recorder struct{ msgs *[]tea.Msg }

func (r recorder) Init() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return tea.QuitMsg{} })
}

func (r recorder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.QuitMsg); ok {
		return r, tea.Quit
	}
	*r.msgs = append(*r.msgs, msg)
	return r, nil
}

func (r recorder) View() string { return "" }
