package ui

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

// inDejaVuMono are the symbols checked against DejaVu Sans Mono, the
// default monospace font on Debian and Ubuntu; none has an emoji form.
const inDejaVuMono = "○◇≡¶$±◈▧▤▦⊘⊗⌥⇅⚑"

func TestSymbolIcons(t *testing.T) {
	defer setIcons("symbols")
	kinds := []string{nfFile, nfBranchIcon, nfRemoteBranch, nfTag}
	for _, m := range []map[string]string{nfByExt, nfByName} {
		for _, nf := range m {
			kinds = append(kinds, nf)
		}
	}
	setIcons("symbols")
	for _, nf := range kinds {
		s := symbols[nf]
		if runewidth.StringWidth(s) != 1 || !strings.Contains(inDejaVuMono, s) {
			t.Errorf("glyph %q: symbol %q is missing, wide or not checked against DejaVu Sans Mono", nf, s)
		}
	}
	cases := []struct {
		mode, path string
		dir        bool
		want       string
	}{
		{"symbols", "main.go", false, "◇ "},
		{"symbols", "README.md", false, "¶ "},
		{"symbols", "deploy.sh", false, "$ "},
		{"symbols", "Dockerfile.prod", false, "≡ "},
		{"symbols", "notes.unknown", false, "○ "},
		{"symbols", "src", true, ""}, // ▸/▾ and the slash mark folders
		{"nerd", "src", true, nfFolderOpen + " "},
		{"unicode", "main.go", false, "◇ "}, // the old name
		{"nerd", "main.go", false, nfGo + " "},
		{"plain", "main.go", false, ""},
		{"plain", "src", true, ""},
	}
	for _, c := range cases {
		setIcons(c.mode)
		if got := fileIcon(c.path, c.dir, false); got != c.want {
			t.Errorf("%s %s: %q, want %q", c.mode, c.path, got, c.want)
		}
	}
	setIcons("plain")
	if branchGlyph(false) != "" || tagGlyph() != "⚑ " {
		t.Errorf("plain: branch %q tag %q", branchGlyph(false), tagGlyph())
	}
	setIcons("symbols")
	if branchGlyph(false) != "⌥ " || branchGlyph(true) != "⇅ " {
		t.Errorf("symbols: branches %q %q", branchGlyph(false), branchGlyph(true))
	}
}
