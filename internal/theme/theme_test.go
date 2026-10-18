package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltinsAreCompleteAndReadable(t *testing.T) {
	names := BuiltinNames()
	if len(names) != 12 {
		t.Fatalf("%d built-in themes: %v", len(names), names)
	}
	for _, n := range names {
		th, ok := Builtin(n)
		if !ok {
			t.Fatalf("Builtin(%q) missing", n)
		}
		for role, v := range th.Palette.roles() {
			if err := validColor(*v); err != nil {
				t.Errorf("%s.%s: %v", n, role, err)
			}
			if *v == "" && n != "terminal" && role != "text" {
				t.Errorf("%s.%s is empty", n, role)
			}
		}
		if probs := Check(th); len(probs) > 0 {
			t.Errorf("%s: %s", n, strings.Join(probs, "; "))
		}
	}
}

func writeTheme(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUserThemesExtend(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "orange.yaml", "extends: tokyo-night\ncolors:\n  accent: \"#ff9e64\"\n")
	writeTheme(t, dir, "orange-calm.yml", "name: orange-calm\nextends: orange\ncolors:\n  sel: \"#222\"\nauthors: [\"1\", \"2\"]\n")
	writeTheme(t, dir, "broken.yaml", "extends: nope\n")
	writeTheme(t, dir, "badrole.yaml", "colors:\n  backgroud: \"#000\"\n")
	writeTheme(t, dir, "badcolor.yaml", "colors:\n  accent: red\n")
	writeTheme(t, dir, "loop-a.yaml", "extends: loop-b\n")
	writeTheme(t, dir, "loop-b.yaml", "extends: loop-a\n")

	set, errs := Load(dir)
	joined := ""
	for _, e := range errs {
		joined += e.Error() + "\n"
	}
	for _, want := range []string{`unknown theme "nope"`, `unknown colour role "backgroud"`, `"red" is not a colour`, "cycle"} {
		if !strings.Contains(joined, want) {
			t.Errorf("errors lack %q:\n%s", want, joined)
		}
	}

	o, ok := set.Get("orange")
	base, _ := Builtin("tokyo-night")
	if !ok || o.Palette.Accent != "#ff9e64" || o.Palette.Text != base.Palette.Text || o.Kind != KindDark || o.Source == "" {
		t.Errorf("orange = %+v", o)
	}
	c, ok := set.Get("orange-calm")
	if !ok || c.Palette.Accent != "#ff9e64" || c.Palette.Sel != "#222" || len(c.Palette.Authors) != 2 {
		t.Errorf("orange-calm = %+v", c)
	}
	names := set.Names()
	if names[0] != "gitnest-dark" || names[len(names)-1] != "orange-calm" {
		t.Errorf("order: %v", names)
	}
	if _, ok := set.Get("broken"); ok {
		t.Error("broken theme loaded")
	}
}

func TestExportRoundTrip(t *testing.T) {
	dir := t.TempDir()
	base, _ := Builtin("nord")
	p := writeTheme(t, dir, "mine.yaml", string(Export(base)))
	got, err := LoadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "my-nord" || got.Palette.Accent != base.Palette.Accent || len(got.Palette.Authors) != len(base.Palette.Authors) {
		t.Errorf("round trip: %+v", got)
	}
}

func TestCheckFlagsLowContrast(t *testing.T) {
	th, _ := Builtin("catppuccin-mocha")
	th.Palette.Sel = "#cdd6f4" // same as text
	probs := Check(th)
	if len(probs) == 0 || !strings.Contains(probs[0], "text #cdd6f4 on sel") {
		t.Errorf("problems = %v", probs)
	}
	if r, _ := Contrast("#000000", "#ffffff"); r < 20.9 || r > 21.1 {
		t.Errorf("black/white contrast %.2f", r)
	}
	if _, ok := Contrast("5", "#fff"); ok {
		t.Error("ANSI colours can't be measured")
	}
}
