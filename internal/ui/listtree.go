package ui

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Paksonite/gitnest/internal/config"
	"github.com/Paksonite/gitnest/internal/gitx"
)

// listRow is one row of the repository list: a repository, or (in tree
// mode) a directory grouping the repositories below it.
type listRow struct {
	repo  int    // index into rs.all; -1 for a directory
	dir   string // directories: identifying path ("root/a/b")
	name  string // directories: shown name, single-child chains merged ("a/b")
	depth int
	count int // directories: repositories below (after filtering)
	dirty int // directories: of which with uncommitted changes
}

func (r listRow) key(a *App) string {
	if r.repo < 0 {
		return "d:" + r.dir
	}
	return a.rs.all[r.repo].Path
}

// Ways the list tree groups repositories.
const (
	groupFolders = iota // by the directories below the scan roots
	groupRemote         // by remote host and owner (github.com/ann/), "local" without one
)

var groupNames = [...]string{groupFolders: "folders", groupRemote: "remote"}

// setListView switches the list between flat and grouped, remembers it and
// rebuilds the rows around the selection.
func (a *App) setListView(tree bool, group int) {
	a.rs.tree, a.rs.group = tree, group
	a.st.SetListTree(tree)
	a.st.SetListGroup(groupNames[group])
	a.rebuildView(a.selectedKey())
}

func (a *App) listViewName() string {
	switch {
	case !a.rs.tree:
		return "flat"
	case a.rs.group == groupRemote:
		return "grouped by remote"
	}
	return "grouped by folder"
}

// groupParts is where repository idx goes in the tree: its directories below
// the scan root, or its remote host and owner.
func (a *App) groupParts(idx int) []string {
	if a.rs.group == groupRemote {
		if parts := gitx.RemoteGroup(a.rs.all[idx].RemoteURL); parts != nil {
			return parts
		}
		return []string{"local"}
	}
	base, rel, ok := a.underRoot(a.rs.all[idx].Path)
	if !ok {
		return nil
	}
	var parts []string
	if rel != "" {
		parts = strings.Split(rel, "/")
	}
	if len(a.cfg.Roots) > 1 {
		parts = append([]string{config.Abbrev(base)}, parts...)
	}
	return parts
}

// groupKey identifies a tree directory; remote groups get their own
// namespace so folding github.com/ann doesn't fold a folder of that name.
func (a *App) groupKey(parts []string) string {
	k := strings.Join(parts, "/")
	if a.rs.group == groupRemote {
		return "remote:" + k
	}
	return k
}

type dirNode struct {
	name     string
	path     string
	children map[string]*dirNode
	repos    []int // indexes into rs.all, in list order
}

// buildRows turns the filtered, sorted repositories (rs.view) into rows. In
// tree mode repositories are grouped by the directories they live in below
// their scan root (one top-level entry per root when there are several), or
// by remote host and owner; groups come first, then repositories in the
// current sort order.
// While searching, every directory is expanded so all matches show.
func (a *App) buildRows() {
	rs := &a.rs
	rs.rows = rs.rows[:0]
	if !rs.tree {
		for _, idx := range rs.view {
			rs.rows = append(rs.rows, listRow{repo: idx})
		}
		return
	}
	root := &dirNode{children: map[string]*dirNode{}}
	for _, idx := range rs.view {
		parts := a.groupParts(idx)
		n := root
		for i, part := range parts {
			c := n.children[part]
			if c == nil {
				c = &dirNode{name: part, path: a.groupKey(parts[:i+1]), children: map[string]*dirNode{}}
				n.children[part] = c
			}
			n = c
		}
		n.repos = append(n.repos, idx)
	}

	searching := rs.filter != ""
	var walk func(n *dirNode, depth int)
	walk = func(n *dirNode, depth int) {
		dirs := make([]*dirNode, 0, len(n.children))
		for _, c := range n.children {
			dirs = append(dirs, c)
		}
		sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
		for _, d := range dirs {
			name := d.name
			for len(d.repos) == 0 && len(d.children) == 1 { // merge a/b when a holds only b
				for _, only := range d.children {
					d = only
				}
				name += "/" + d.name
			}
			count, dirty := a.countBelow(d)
			rs.rows = append(rs.rows, listRow{repo: -1, dir: d.path, name: name, depth: depth, count: count, dirty: dirty})
			if searching || !rs.collapsed[d.path] {
				walk(d, depth+1)
			}
		}
		for _, idx := range n.repos {
			rs.rows = append(rs.rows, listRow{repo: idx, depth: depth})
		}
	}
	walk(root, 0)
}

func (a *App) countBelow(n *dirNode) (count, dirty int) {
	for _, idx := range n.repos {
		count++
		if a.rs.all[idx].Changes > 0 {
			dirty++
		}
	}
	for _, c := range n.children {
		cn, dn := a.countBelow(c)
		count, dirty = count+cn, dirty+dn
	}
	return count, dirty
}

// reposUnder lists the repositories shown below directory row i.
func (a *App) reposUnder(i int) []string {
	rs := &a.rs
	prefix := rs.rows[i].dir + "/"
	var out []string
	for _, idx := range rs.view {
		if r := a.rowDir(idx); r == rs.rows[i].dir || strings.HasPrefix(r+"/", prefix) {
			out = append(out, rs.all[idx].Path)
		}
	}
	return out
}

// underRoot finds the scan root p lives under (the longest that matches) and
// p's directory relative to it, with slashes ("" directly in the root).
func (a *App) underRoot(p string) (base, rel string, ok bool) {
	for _, r := range a.cfg.Roots {
		if strings.HasPrefix(p, r+string(filepath.Separator)) && len(r) > len(base) {
			base = r
		}
	}
	if base == "" {
		return "", "", false
	}
	rel = filepath.ToSlash(filepath.Dir(strings.TrimPrefix(p, base+string(filepath.Separator))))
	if rel == "." {
		rel = ""
	}
	return base, rel, true
}

// rowDir is the directory key a repository falls under in tree mode.
func (a *App) rowDir(idx int) string {
	parts := a.groupParts(idx)
	if parts == nil && a.rs.group == groupFolders {
		return ""
	}
	return a.groupKey(parts)
}

func dirRow(r listRow, collapsed bool, w int) rowItem {
	indent := strings.Repeat("  ", r.depth)
	arrow := "▾"
	if collapsed {
		arrow = "▸"
	}
	icon := fileIcon(r.dir, true, collapsed)
	plain := indent + "  " + arrow + " " + icon + r.name + "/"
	styled := indent + "  " + fg(cDim, arrow) + " " + fg(cCyan, icon) + bold(r.name+"/")
	info := " " + strconv.Itoa(r.count)
	sinfo := fg(cDim, info)
	if r.dirty > 0 {
		info += " ●" + strconv.Itoa(r.dirty)
		sinfo += fg(cYellow, " ●"+strconv.Itoa(r.dirty))
	}
	return rowItem{plain: fit(plain+info, w), styled: styled + sinfo}
}
