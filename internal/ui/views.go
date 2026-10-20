package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// sortLabels name the sort orders in the sort & view chooser.
var sortLabels = [sortModes]string{"Recently used", "Name", "Last commit", "Status (needs attention first)"}

// openViews shows the view menu (v): sort order, flat or grouped list, all
// repositories or favourites only, and the status and remote filters.
// Choices apply at once.
func (a *App) openViews() {
	rs := &a.rs
	rebuild := func() tea.Cmd {
		a.rebuildView(a.selectedKey())
		a.saveListView()
		return a.previewCmd()
	}
	m := &menu{title: "View & filters", note: "choices apply at once; filters combine"}
	m.items = append(m.items, menuHeader("Sort by"))
	for i := 0; i < sortModes; i++ {
		m.items = append(m.items, menuItem{label: sortLabels[i], stay: true,
			on:   func() bool { return rs.sortMode == i },
			pick: func() tea.Cmd { rs.sortMode = i; return rebuild() }})
	}
	views := []struct {
		label string
		tree  bool
		group int
	}{{"Flat list", false, groupFolders}, {"Grouped by folder", true, groupFolders}, {"Grouped by remote", true, groupRemote}}
	m.items = append(m.items, menuHeader("View"))
	for _, v := range views {
		m.items = append(m.items, menuItem{label: v.label, stay: true,
			on:   func() bool { return rs.tree == v.tree && (!v.tree || rs.group == v.group) },
			pick: func() tea.Cmd { a.setListView(v.tree, v.group); return a.previewCmd() }})
	}
	m.items = append(m.items, menuHeader("Show"),
		menuItem{label: "Every repository", stay: true,
			on:   func() bool { return !rs.favOnly },
			pick: func() tea.Cmd { rs.favOnly = false; return rebuild() }},
		menuItem{label: "Favorites only", right: countLabel(len(a.sh.fav)), stay: true,
			on:   func() bool { return rs.favOnly },
			pick: func() tea.Cmd { rs.favOnly = true; return rebuild() }},
	)
	m.items = append(m.items, a.filterItems()...)
	a.openMenu(m)
}

// toggleFavorite stars or unstars the marked repositories (or the selected
// one). Favorites are shared by every pane and remembered.
func (a *App) toggleFavorite() tea.Cmd {
	paths := a.targets(func() []string {
		if p := a.selectedPath(); p != "" {
			return []string{p}
		}
		return nil
	})
	if len(paths) == 0 {
		return nil
	}
	if a.sh.fav == nil {
		a.sh.fav = map[string]bool{}
	}
	on := !a.sh.fav[paths[0]]
	for _, p := range paths {
		if on {
			a.sh.fav[p] = true
		} else {
			delete(a.sh.fav, p)
		}
	}
	list := make([]string, 0, len(a.sh.fav))
	for p := range a.sh.fav {
		list = append(list, p)
	}
	a.st.SetFavorites(list)
	if on {
		a.setToast("★ added to favorites (v shows favorites only)", false)
	} else {
		a.setToast("removed from favorites", false)
	}
	if a.rs.favOnly {
		a.rebuildView(a.selectedKey())
		return a.previewCmd()
	}
	return nil
}
