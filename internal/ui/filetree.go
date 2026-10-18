package ui

import (
	"sort"
	"strings"

	"github.com/Paksonite/gitnest/v2/internal/gitx"
)

// fileNode is one row of the Files panel: a changed file, or (in tree mode)
// a directory containing changed files.
type fileNode struct {
	Path  string // repository-relative, without a trailing slash
	Name  string // what is shown: the base name, or a chain like "a/b" for directories
	Dir   bool
	Depth int
	File  gitx.File   // files only
	Files []gitx.File // directories: every changed file below
}

// key identifies the row across refreshes (a file and a directory may share a path prefix).
func (n fileNode) key() string {
	if n.Dir {
		return "d:" + n.Path
	}
	return "f:" + n.Path
}

type trieNode struct {
	name     string
	path     string
	file     *gitx.File
	children map[string]*trieNode
}

// buildFileRows lays out the changed files as a tree: a tree with
// directories first, single-directory chains merged ("src/app"), and the
// children of collapsed directories hidden. In flat mode it is a plain list
// sorted by path. The order never depends on staging state.
func buildFileRows(files []gitx.File, flat bool, collapsed map[string]bool) []fileNode {
	sorted := append([]gitx.File(nil), files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	if flat {
		rows := make([]fileNode, len(sorted))
		for i, f := range sorted {
			rows[i] = fileNode{Path: f.Path, Name: f.Path, File: f}
		}
		return rows
	}

	root := &trieNode{children: map[string]*trieNode{}}
	for i := range sorted {
		f := &sorted[i]
		parts := strings.Split(strings.TrimSuffix(f.Path, "/"), "/")
		n := root
		for k, part := range parts {
			child := n.children[part]
			if child == nil {
				child = &trieNode{name: part, path: strings.Join(parts[:k+1], "/"), children: map[string]*trieNode{}}
				n.children[part] = child
			}
			n = child
		}
		n.file = f
	}

	var rows []fileNode
	var walk func(n *trieNode, depth int)
	walk = func(n *trieNode, depth int) {
		var dirs, leaves []*trieNode
		for _, c := range n.children {
			if c.file == nil {
				dirs = append(dirs, c)
			} else {
				leaves = append(leaves, c)
			}
		}
		sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
		sort.Slice(leaves, func(i, j int) bool { return leaves[i].name < leaves[j].name })
		for _, d := range dirs {
			name := d.name
			for len(d.children) == 1 { // merge a/b/c when each level has one directory
				var only *trieNode
				for _, c := range d.children {
					only = c
				}
				if only.file != nil {
					break
				}
				name += "/" + only.name
				d = only
			}
			rows = append(rows, fileNode{Path: d.path, Name: name, Dir: true, Depth: depth, Files: filesUnder(d)})
			if !collapsed[d.path] {
				walk(d, depth+1)
			}
		}
		for _, l := range leaves {
			rows = append(rows, fileNode{Path: l.path, Name: l.name, Depth: depth, File: *l.file})
		}
	}
	walk(root, 0)
	return rows
}

func filesUnder(n *trieNode) []gitx.File {
	var out []gitx.File
	if n.file != nil {
		out = append(out, *n.file)
	}
	for _, c := range n.children {
		out = append(out, filesUnder(c)...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// dirStage summarises a directory: whether anything below is staged and
// whether anything is still unstaged.
func dirStage(files []gitx.File) (anyStaged, anyUnstaged bool) {
	for _, f := range files {
		if f.Staged() {
			anyStaged = true
		}
		if f.Unstaged() {
			anyUnstaged = true
		}
	}
	return anyStaged, anyUnstaged
}

func fileRow(n fileNode, collapsed bool) rowItem {
	indent := strings.Repeat("  ", n.Depth)
	if n.Dir {
		arrow := "▾"
		if collapsed {
			arrow = "▸"
		}
		staged, unstaged := dirStage(n.Files)
		col := cRed // nothing staged
		switch {
		case staged && !unstaged:
			col = cGreen
		case staged:
			col = cYellow
		}
		icon, sicon := fileIconStyled(n.Path, true, collapsed)
		name := printable(n.Name)
		plain := indent + arrow + " " + icon + name + "/"
		return rowItem{plain: plain, styled: indent + fg(cDim, arrow) + " " + sicon + fg(col, name+"/")}
	}
	f := n.File
	code := string([]byte{f.X, f.Y})
	var st string
	switch {
	case f.Conflict():
		st = fg(cYellow, code)
	case f.Untracked():
		st = fg(cRed, code)
	default:
		x, y := string(f.X), string(f.Y)
		if f.X != ' ' {
			x = fg(cGreen, x)
		}
		if f.Y != ' ' {
			y = fg(cRed, y)
		}
		st = x + y
	}
	icon, sicon := fileIconStyled(n.Path, false, false)
	name := printable(n.Name)
	plain, styled := indent+code+" "+icon+name, indent+st+" "+sicon+name
	if f.Orig != "" { // rename or copy
		plain += " ← " + printable(f.Orig)
		styled += fg(cDim, " ← "+printable(f.Orig))
	}
	return rowItem{plain: plain, styled: styled}
}
