package ui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/v2/internal/gitx"
)

// browser shows every file of the repository (B): a folding tree on the
// left, the selected file with line numbers and syntax colours on the right.
type browser struct {
	dir       string
	files     []string // slash paths, from git ls-files
	err       string
	rows      []browseRow
	collapsed map[string]bool
	cur, off  int
	focusFile bool // keys scroll the file instead of moving in the tree

	path   string   // file shown on the right
	lines  []string // its styled lines
	note   string   // instead of lines: binary, too big, unreadable
	scroll int
}

type browseRow struct {
	path  string // slash path; directories without a trailing slash
	name  string
	dir   bool
	depth int
}

type browseFilesMsg struct {
	dir   string
	files []string
	err   error
}

// Files bigger than this aren't shown; longer ones are shown uncoloured
// past maxColoredLines.
const (
	maxBrowseBytes  = 2 << 20
	maxColoredLines = 5000
)

func (a *App) openBrowser(dir string) tea.Cmd {
	a.browse = &browser{dir: dir, collapsed: map[string]bool{}}
	ctx := a.ctx
	return func() tea.Msg {
		files, err := gitx.ListFiles(ctx, dir)
		return browseFilesMsg{dir: dir, files: files, err: err}
	}
}

func (a *App) onBrowseFiles(m browseFilesMsg) {
	b := a.browse
	if b == nil || b.dir != m.dir {
		return
	}
	if m.err != nil {
		b.err = m.err.Error()
		return
	}
	b.files = m.files
	// Start with only the top level open, like a file manager.
	for _, f := range b.files {
		if d := path.Dir(f); d != "." {
			top, _, _ := strings.Cut(d, "/")
			b.collapsed[top] = true
		}
	}
	b.build()
	b.show()
}

// build flattens the file list into tree rows: folders first, then files,
// each sorted by name; folded folders hide what's below them.
func (b *browser) build() {
	type node struct {
		dirs  map[string]*node
		files []string
	}
	root := &node{dirs: map[string]*node{}}
	for _, f := range b.files {
		n := root
		parts := strings.Split(f, "/")
		for _, p := range parts[:len(parts)-1] {
			c := n.dirs[p]
			if c == nil {
				c = &node{dirs: map[string]*node{}}
				n.dirs[p] = c
			}
			n = c
		}
		n.files = append(n.files, parts[len(parts)-1])
	}
	b.rows = b.rows[:0]
	var walk func(n *node, prefix string, depth int)
	walk = func(n *node, prefix string, depth int) {
		names := make([]string, 0, len(n.dirs))
		for d := range n.dirs {
			names = append(names, d)
		}
		sort.Strings(names)
		for _, d := range names {
			p := prefix + d
			b.rows = append(b.rows, browseRow{path: p, name: d, dir: true, depth: depth})
			if !b.collapsed[p] {
				walk(n.dirs[d], p+"/", depth+1)
			}
		}
		sort.Strings(n.files)
		for _, f := range n.files {
			b.rows = append(b.rows, browseRow{path: prefix + f, name: f, depth: depth})
		}
	}
	walk(root, "", 0)
	b.cur = clamp(b.cur, 0, len(b.rows)-1)
}

// show loads the file under the cursor into the right pane.
func (b *browser) show() {
	if b.cur < 0 || b.cur >= len(b.rows) || b.rows[b.cur].dir {
		return
	}
	p := b.rows[b.cur].path
	if p == b.path {
		return
	}
	b.path, b.scroll, b.lines, b.note = p, 0, nil, ""
	full := filepath.Join(b.dir, filepath.FromSlash(p))
	st, err := os.Stat(full)
	switch {
	case err != nil:
		b.note = "can't read: " + err.Error()
		return
	case st.Size() > maxBrowseBytes:
		b.note = fmt.Sprintf("%s — too big to show (%d MB)", p, st.Size()>>20)
		return
	}
	data, err := os.ReadFile(full)
	if err != nil {
		b.note = "can't read: " + err.Error()
		return
	}
	if bytes.IndexByte(data[:min(len(data), 8000)], 0) >= 0 {
		b.note = fmt.Sprintf("%s — binary file, %d bytes", p, len(data))
		return
	}
	b.lines = colorize(p, expandTabs(string(data)))
}

// colorize colours source code with the theme: keywords in the accent,
// strings green, numbers yellow, names of functions and types blue,
// comments dimmed. Unknown file types stay plain.
func colorize(name, src string) []string {
	src = strings.TrimSuffix(src, "\n")
	plain := strings.Split(src, "\n")
	lexer := lexers.Match(filepath.Base(name))
	if lexer == nil || len(plain) > maxColoredLines {
		return plain
	}
	it, err := chroma.Coalesce(lexer).Tokenise(nil, src)
	if err != nil {
		return plain
	}
	styles := map[lipgloss.TerminalColor]lipgloss.Style{}
	paint := func(c lipgloss.TerminalColor, s string) string {
		if c == nil || s == "" {
			return s
		}
		st, ok := styles[c]
		if !ok {
			st = lipgloss.NewStyle().Foreground(c)
			styles[c] = st
		}
		return st.Render(s)
	}
	var out []string
	var line strings.Builder
	for tok := it(); tok != chroma.EOF; tok = it() {
		c := tokenColor(tok.Type)
		parts := strings.Split(tok.Value, "\n")
		for i, part := range parts {
			if i > 0 {
				out = append(out, line.String())
				line.Reset()
			}
			line.WriteString(paint(c, part))
		}
	}
	out = append(out, line.String())
	if len(out) != len(plain) { // a lexer quirk: fall back rather than misalign
		return plain
	}
	return out
}

func tokenColor(t chroma.TokenType) lipgloss.TerminalColor {
	switch {
	case t.InCategory(chroma.Comment):
		return cDim
	case t == chroma.KeywordType || t == chroma.NameBuiltin || t == chroma.NameClass ||
		t == chroma.NameFunction || t == chroma.NameTag:
		return cCyan
	case t.InCategory(chroma.Keyword):
		return cAccent
	case t.InSubCategory(chroma.LiteralString):
		return cGreen
	case t.InSubCategory(chroma.LiteralNumber), t == chroma.NameAttribute, t == chroma.NameDecorator,
		t == chroma.NameConstant:
		return cYellow
	case t == chroma.GenericInserted:
		return cGreen
	case t == chroma.GenericDeleted:
		return cRed
	case t == chroma.GenericHeading, t == chroma.GenericSubheading:
		return cAccent
	}
	return nil
}

func (a *App) browserKey(k tea.KeyMsg) tea.Cmd {
	b := a.browse
	page := max(1, a.h-6)
	move := func(d int) {
		if b.focusFile {
			b.scroll = clamp(b.scroll+d, 0, max(0, len(b.lines)-page))
			return
		}
		b.cur = clamp(b.cur+d, 0, len(b.rows)-1)
		b.show()
	}
	switch lookup(scBrowser, k.String()) {
	case actUp:
		move(-1)
	case actDown:
		move(1)
	case actOpen:
		if b.cur < len(b.rows) && b.rows[b.cur].dir {
			p := b.rows[b.cur].path
			b.collapsed[p] = !b.collapsed[p]
			b.build()
		} else if b.path != "" {
			b.focusFile = true
		}
	case actEdit:
		if b.path == "" {
			return nil
		}
		c := exec.Command("sh", "-c", a.editor()+` "$@"`, "sh", filepath.FromSlash(b.path))
		c.Dir = b.dir
		return a.execCmd(c)
	case actBack:
		if b.focusFile {
			b.focusFile = false
			return nil
		}
		a.browse = nil
	}
	return nil
}

func (a *App) viewBrowser() string {
	b := a.browse
	bodyH := a.h - 3
	head := brand() + fg(cDim, " › ") + bold(filepath.Base(b.dir)) + fg(cDim, " › files")
	if b.path != "" {
		head += fg(cDim, " › ") + b.path
	}
	treeW := clamp(a.w*2/5, 24, 48)
	if a.w < 70 {
		treeW = a.w / 2
	}
	fileW := a.w - treeW
	lh := bodyH - 2

	var tree []string
	switch {
	case b.err != "":
		tree = []string{fg(cRed, b.err)}
	case b.files == nil:
		tree = []string{fg(cDim, "loading…")}
	case len(b.rows) == 0:
		tree = []string{fg(cDim, "no files")}
	default:
		b.off = scrollTo(b.cur, b.off, lh)
		var items []rowItem
		for _, r := range b.rows[b.off:min(len(b.rows), b.off+lh)] {
			indent := strings.Repeat("  ", r.depth)
			if r.dir {
				arrow := "▾"
				if b.collapsed[r.path] {
					arrow = "▸"
				}
				icon, sicon := fileIconStyled(r.path, true, b.collapsed[r.path])
				items = append(items, rowItem{plain: indent + arrow + " " + icon + r.name + "/",
					styled: indent + fg(cDim, arrow) + " " + sicon + fg(cCyan, r.name+"/")})
				continue
			}
			icon, sicon := fileIconStyled(r.path, false, false)
			items = append(items, rowItem{plain: indent + "  " + icon + r.name, styled: indent + "  " + sicon + r.name})
		}
		tree = renderRows(items, b.cur-b.off, 0, lh, panelInner(treeW), !b.focusFile)
	}

	var file []string
	inner := panelInner(fileW)
	switch {
	case b.note != "":
		file = []string{fg(cDim, b.note)}
	case b.path == "":
		file = []string{fg(cDim, "choose a file on the left")}
	default:
		numW := len(fmt.Sprint(len(b.lines)))
		for i := b.scroll; i < len(b.lines) && i < b.scroll+lh; i++ {
			file = append(file, fit(fg(cDim, fmt.Sprintf("%*d ", numW, i+1))+b.lines[i], inner))
		}
	}
	fileTitle := "File"
	if b.path != "" {
		fileTitle = fmt.Sprintf("%s · %d lines", path.Base(b.path), len(b.lines))
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		panel(subtitle("Files", fmt.Sprintf("%d", len(b.files))), tree, treeW, bodyH, !b.focusFile),
		panel(fileTitle, file, fileW, bodyH, b.focusFile))
	return strings.Join([]string{fit(head, a.w), body, a.footer(hints(scBrowser))}, "\n")
}
