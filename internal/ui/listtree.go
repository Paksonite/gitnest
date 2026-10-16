package ui

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/Paksonite/gitnest/internal/config"
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

type dirNode struct {
	name     string
	path     string
	children map[string]*dirNode
	repos    []int // indexes into rs.all, in list order
}

// buildRows turns the filtered, sorted repositories (rs.view) into rows. In
// tree mode repositories are grouped by the directories they live in below
// their scan root (one top-level entry per root when there are several);
// directories come first, then repositories in the current sort order.
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
	roots := a.cfg.Roots
	root := &dirNode{children: map[string]*dirNode{}}
	for _, idx := range rs.view {
		p := rs.all[idx].Path
		var parts []string
		base := ""
		for _, r := range roots {
			if strings.HasPrefix(p, r+string(filepath.Separator)) && len(r) > len(base) {
				base = r
			}
		}
		if base != "" {
			rel := strings.TrimPrefix(p, base+string(filepath.Separator))
			parts = strings.Split(filepath.ToSlash(filepath.Dir(rel)), "/")
			if parts[0] == "." {
				parts = nil
			}
			if len(roots) > 1 {
				parts = append([]string{config.Abbrev(base)}, parts...)
			}
		}
		n := root
		for i, part := range parts {
			c := n.children[part]
			if c == nil {
				c = &dirNode{name: part, path: strings.Join(parts[:i+1], "/"), children: map[string]*dirNode{}}
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

// rowDir is the directory key a repository falls under in tree mode.
func (a *App) rowDir(idx int) string {
	p := a.rs.all[idx].Path
	base := ""
	for _, r := range a.cfg.Roots {
		if strings.HasPrefix(p, r+string(filepath.Separator)) && len(r) > len(base) {
			base = r
		}
	}
	if base == "" {
		return ""
	}
	d := filepath.ToSlash(filepath.Dir(strings.TrimPrefix(p, base+string(filepath.Separator))))
	if d == "." {
		d = ""
	}
	if len(a.cfg.Roots) > 1 {
		if d == "" {
			return config.Abbrev(base)
		}
		return config.Abbrev(base) + "/" + d
	}
	return d
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
	info := " " + itoa(r.count)
	sinfo := fg(cDim, info)
	if r.dirty > 0 {
		info += " ●" + itoa(r.dirty)
		sinfo += fg(cYellow, " ●"+itoa(r.dirty))
	}
	return rowItem{plain: fit(plain+info, w), styled: styled + sinfo}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
