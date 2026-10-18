package ui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2"
	tea "github.com/charmbracelet/bubbletea"
)

func TestBrowser(t *testing.T) {
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
	// enter on a folder unfolds it.
	for i := 0; i < 4; i++ {
		a.onKey(tea.KeyMsg{Type: tea.KeyUp})
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	names = names[:0]
	for _, r := range b.rows {
		names = append(names, r.name)
	}
	if strings.Join(names, " ") != "docs deep b.txt a.md .gitignore bin.dat main.go" {
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
