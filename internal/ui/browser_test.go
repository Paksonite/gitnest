package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/Paksonite/gitnest/v2/internal/state"
)

func TestBrowser(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir := t.TempDir()
	sh(t, dir, "init", "-q")
	src := "package main\n\n// hello\nfunc main() {\n\tprintln(\"hi\", 42)\n}\n"
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644)
	os.WriteFile(filepath.Join(dir, "bin.dat"), []byte("a\x00b"), 0o644)
	os.MkdirAll(filepath.Join(dir, "docs", "deep"), 0o755)
	os.WriteFile(filepath.Join(dir, "docs", "a.md"), []byte("# A\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs", "deep", "b.txt"), []byte("b\n"), 0o644)
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("ignored.log\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "ignored.log"), []byte("x"), 0o644)

	a := testApp()
	a.ctx = context.Background()
	a.w, a.h = 120, 30
	cmd := a.openBrowser(dir)
	a.Update(cmd())
	b := a.browse
	var names []string
	for _, r := range b.rows {
		names = append(names, r.name)
	}
	if strings.Join(names, " ") != "docs .gitignore bin.dat main.go" {
		t.Fatalf("rows: %v", names)
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyDown})
	a.onKey(tea.KeyMsg{Type: tea.KeyDown})
	if !strings.Contains(b.note, "binary") {
		t.Errorf("bin.dat: note %q", b.note)
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyDown})
	if b.path != "main.go" || len(b.lines) != 6 || !strings.Contains(b.lines[0], "package main") {
		t.Fatalf("main.go: %q %d lines", b.path, len(b.lines))
	}
	if v := a.View(); !strings.Contains(v, "main.go · 6 lines") || !strings.Contains(v, "1 package main") || !strings.Contains(v, "Files  5") {
		t.Errorf("view:\n%s", v)
	}
	// enter on a file focuses it, j scrolls, esc goes back to the tree.
	a.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !b.focusFile {
		t.Fatal("enter should focus the file")
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyEsc})
	if b.focusFile || a.browse == nil {
		t.Fatal("esc should go back to the tree")
	}
	// enter on a folder unfolds it; the folders inside stay folded.
	for i := 0; i < 4; i++ {
		a.onKey(tea.KeyMsg{Type: tea.KeyUp})
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	names = names[:0]
	for _, r := range b.rows {
		names = append(names, r.name)
	}
	if strings.Join(names, " ") != "docs deep a.md .gitignore bin.dat main.go" {
		t.Fatalf("unfolded: %v", names)
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyEsc})
	if a.browse != nil {
		t.Error("esc in the tree should close the browser")
	}
}

func TestColorize(t *testing.T) {
	src := "package main\n\n// hello\nfunc main() {\n\tprintln(\"hi\")\n}\n"
	if lines := colorize("main.go", src); len(lines) != 6 {
		t.Errorf("main.go: %d lines", len(lines))
	}
	if lines := colorize("notes.unknownext", "a\nb"); len(lines) != 2 || lines[1] != "b" {
		t.Errorf("plain: %q", lines)
	}
	for tt, want := range map[chroma.TokenType]any{
		chroma.Keyword: cAccent, chroma.KeywordType: cCyan, chroma.LiteralString: cGreen,
		chroma.LiteralNumberInteger: cYellow, chroma.CommentSingle: cDim, chroma.NameFunction: cCyan,
	} {
		if got := tokenColor(tt); got != want {
			t.Errorf("%v: %v, want %v", tt, got, want)
		}
	}
	if tokenColor(chroma.Text) != nil {
		t.Error("plain text keeps the terminal colour")
	}
}

// o in the file browser leaves for the selected folder, or the selected
// file's folder, like o does for the repository.
func TestBrowserCdIntoSelection(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir := t.TempDir()
	sh(t, dir, "init", "-q")
	os.MkdirAll(filepath.Join(dir, "docs", "deep"), 0o755)
	os.WriteFile(filepath.Join(dir, "docs", "deep", "b.txt"), []byte("b\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)
	a := testApp()
	a.ctx = context.Background()
	a.w, a.h = 120, 30
	a.cdFile = filepath.Join(t.TempDir(), "cd")
	a.Update(a.openBrowser(dir)())
	pick := func(name string) {
		for i, r := range a.browse.rows {
			if r.name == name {
				a.browse.cur = i
				return
			}
		}
		t.Fatalf("no row %q in %+v", name, a.browse.rows)
	}
	for _, c := range []struct{ row, want string }{
		{"docs", filepath.Join(dir, "docs")}, // a folder: into it
		{"main.go", dir},                     // a file: its folder
	} {
		pick(c.row)
		a.onKey(key("o"))
		got, _ := os.ReadFile(a.cdFile)
		if strings.TrimSpace(string(got)) != c.want {
			t.Errorf("o on %s: %q, want %q", c.row, got, c.want)
		}
	}
}

// Folders start folded and the ones unfolded open again next time, in
// that repository only; folders that are gone are forgotten.
func TestBrowserRemembersFolds(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir, other := t.TempDir(), t.TempDir()
	for _, d := range []string{dir, other} {
		sh(t, d, "init", "-q")
		os.MkdirAll(filepath.Join(d, "docs", "deep"), 0o755)
		os.MkdirAll(filepath.Join(d, "src"), 0o755)
		os.WriteFile(filepath.Join(d, "docs", "deep", "b.txt"), []byte("b\n"), 0o644)
		os.WriteFile(filepath.Join(d, "src", "c.go"), []byte("package c\n"), 0o644)
	}
	rows := func(a *App) string {
		var names []string
		for _, r := range a.browse.rows {
			names = append(names, r.name)
		}
		return strings.Join(names, " ")
	}
	a := testApp()
	a.w, a.h = 120, 30
	a.Update(a.openBrowser(dir)())
	if got := rows(a); got != "docs src" {
		t.Fatalf("first open: %q", got)
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyEnter}) // docs
	a.onKey(tea.KeyMsg{Type: tea.KeyDown})
	a.onKey(tea.KeyMsg{Type: tea.KeyEnter}) // docs/deep
	if got := rows(a); got != "docs deep b.txt src" {
		t.Fatalf("unfolded: %q", got)
	}

	b := testApp()
	b.st = state.Load()
	b.w, b.h = 120, 30
	b.Update(b.openBrowser(dir)())
	if got := rows(b); got != "docs deep b.txt src" {
		t.Errorf("reopened: %q", got)
	}
	b.Update(b.openBrowser(other)())
	if got := rows(b); got != "docs src" {
		t.Errorf("another repository: %q", got)
	}

	os.RemoveAll(filepath.Join(dir, "docs", "deep"))
	os.WriteFile(filepath.Join(dir, "docs", "a.md"), []byte("a\n"), 0o644)
	sh(t, dir, "add", "-A")
	c := testApp()
	c.st = state.Load()
	c.Update(c.openBrowser(dir)())
	if got := c.st.BrowseOpen[dir]; strings.Join(got, " ") != "docs" {
		t.Errorf("after docs/deep went away: %v", got)
	}
}

// Dragging over the file selects its text, without the line numbers, and
// copies it.
func TestBrowserSelectFileText(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir := t.TempDir()
	sh(t, dir, "init", "-q")
	var src strings.Builder
	for i := 1; i <= 40; i++ {
		fmt.Fprintf(&src, "line number %d\n", i)
	}
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte(src.String()), 0o644)
	a := testApp()
	a.w, a.h = 120, 30
	a.Update(a.openBrowser(dir)())
	if a.browse.path != "notes.txt" {
		t.Fatalf("shown: %q", a.browse.path)
	}
	lines := strings.Split(a.View(), "\n")
	find := func(s string) (int, int) {
		for y, l := range lines {
			plain := ansi.Strip(l)
			if i := strings.Index(plain, s); i >= 0 {
				return ansi.StringWidth(plain[:i]), y
			}
		}
		t.Fatalf("no %q on screen", s)
		return 0, 0
	}
	x0, y0 := find("line number 2")
	_, y1 := find("line number 4")
	last, ylast := find("line number 24") // the last row of the panel
	if ylast != a.h-4 {
		t.Fatalf("last file row at %d, want %d", ylast, a.h-4)
	}
	press := func(act tea.MouseAction, x, y int) tea.Cmd {
		return a.onMouse(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: act})
	}
	press(tea.MouseActionPress, x0, y0)
	press(tea.MouseActionMotion, x0+12, y1)
	if cmd := press(tea.MouseActionRelease, x0+12, y1); cmd == nil || !strings.Contains(a.toast, "copied") {
		t.Fatalf("no copy: toast %q", a.toast)
	}
	if got := a.selText(); got != "line number 2\nline number 3\nline number 4" {
		t.Errorf("selected %q", got)
	}
	// down to the last row, and no further
	press(tea.MouseActionPress, last, ylast)
	press(tea.MouseActionMotion, last+20, ylast+3)
	press(tea.MouseActionRelease, last+20, ylast+3)
	if got := a.selText(); got != "line number 24" {
		t.Errorf("last row: %q", got)
	}
}

// The cursor stays on screen at the bottom of a long tree.
func TestBrowserTreeCursorVisible(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir := t.TempDir()
	sh(t, dir, "init", "-q")
	for i := 0; i < 40; i++ {
		os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%02d.txt", i)), []byte("x\n"), 0o644)
	}
	a := testApp()
	a.w, a.h = 120, 30
	a.Update(a.openBrowser(dir)())
	for i := 0; i < 39; i++ {
		a.onKey(tea.KeyMsg{Type: tea.KeyDown})
		name := a.browse.rows[a.browse.cur].name
		var tree []string
		for _, l := range strings.Split(a.View(), "\n")[1:] {
			tree = append(tree, ansi.Strip(ansi.Cut(l, 0, a.browseTreeW())))
		}
		if !strings.Contains(strings.Join(tree, "\n"), name) {
			t.Fatalf("cursor on %s is off screen", name)
		}
	}
}
