package ui

import (
	"reflect"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// metaArrow turns the arrows iTerm2 sends for option+arrow (with the
// Option key set to Normal, its default) into alt+arrows: it marks them
// with the "meta" modifier, ESC[1;9D, where xterm sends ESC[1;3D. Bubble
// Tea knows only the latter and hands the former on as an unknown CSI
// sequence, which nothing would see.
func metaArrow(msg tea.Msg) (tea.KeyMsg, bool) {
	v := reflect.ValueOf(msg)
	if !v.IsValid() || v.Kind() != reflect.Slice || v.Type().Elem().Kind() != reflect.Uint8 ||
		v.Type().Name() != "unknownCSISequenceMsg" {
		return tea.KeyMsg{}, false
	}
	return metaArrowSeq(string(v.Bytes()))
}

func metaArrowSeq(seq string) (tea.KeyMsg, bool) {
	rest, ok := strings.CutPrefix(seq, "\x1b[1;9")
	if !ok || len(rest) != 1 {
		return tea.KeyMsg{}, false
	}
	t, ok := map[byte]tea.KeyType{'A': tea.KeyUp, 'B': tea.KeyDown, 'C': tea.KeyRight, 'D': tea.KeyLeft}[rest[0]]
	if !ok {
		return tea.KeyMsg{}, false
	}
	return tea.KeyMsg{Type: t, Alt: true}, true
}
