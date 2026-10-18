package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// openPanesMsg asks the shell to open each repository in a pane of its own.
type openPanesMsg struct{ paths []string }

// openActions shows what can be done with the marked repositories (or, with
// none marked, the selected one).
func (a *App) openActions() {
	rs := &a.rs
	var marked []string
	for _, r := range rs.all {
		if rs.marked[r.Path] {
			marked = append(marked, r.Path)
		}
	}
	targets, note := marked, ""
	switch {
	case len(marked) > 0:
		names := make([]string, len(marked))
		for i, p := range marked {
			names[i] = filepath.Base(p)
		}
		note = fmt.Sprintf("%d marked: %s", len(marked), strings.Join(names, ", "))
	case a.selectedPath() != "":
		targets = []string{a.selectedPath()}
		note = "nothing marked (space marks): acting on " + filepath.Base(targets[0])
	default:
		a.setToast("nothing marked and no repository selected", false)
		return
	}
	act := func(x action) func() tea.Cmd { return func() tea.Cmd { return a.listAction(x) } }
	m := &menu{title: "Actions", note: note}
	m.items = append(m.items,
		menuHeader("Repositories"),
		menuItem{label: "Fetch", key: "f", pick: act(actFetchMany)},
		menuItem{label: "Pull (fast-forward only)", key: "p", pick: act(actPullMany)},
		menuItem{label: "Open in panes, one each", key: "n", pick: func() tea.Cmd {
			paths := targets
			return func() tea.Msg { return openPanesMsg{paths: paths} }
		}},
	)
	a.openMenu(m)
}
