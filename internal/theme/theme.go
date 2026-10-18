// Package theme defines gitnest's colour themes: a palette of named roles,
// the built-in themes, and user themes loaded from YAML files that extend
// another theme and override some roles.
package theme

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Kind says which terminal background a theme is made for.
type Kind string

const (
	KindDark  Kind = "dark"
	KindLight Kind = "light"
	KindAny   Kind = "any" // follows the terminal's own palette
)

// Palette assigns a colour to every role the UI uses. A colour is "#rrggbb",
// "#rgb", an ANSI colour number "0"–"255", or empty for the terminal's
// default. Background is never painted: it is the background the theme is
// designed for, used by the contrast check.
type Palette struct {
	Background string `yaml:"background"`
	Text       string `yaml:"text"`    // normal text, selected rows
	Dim        string `yaml:"dim"`     // secondary text, hints
	Accent     string `yaml:"accent"`  // focused borders, titles, matches
	Added      string `yaml:"added"`   // staged, ahead, success
	Removed    string `yaml:"removed"` // unstaged, behind, errors
	Warn       string `yaml:"warn"`    // changes, warnings, operations in progress
	Branch     string `yaml:"branch"`  // branch names, refs
	Sel        string `yaml:"sel"`     // selected row in the focused panel (background)
	SelDim     string `yaml:"sel_dim"` // selected row elsewhere (background)
	Border     string `yaml:"border"`  // unfocused borders
	Surface    string `yaml:"surface"` // panel background (cards); empty keeps the terminal's
	// Diff colours are passed to git; empty keeps the user's git colours.
	DiffAdd  string `yaml:"diff_add"`
	DiffDel  string `yaml:"diff_del"`
	DiffHunk string `yaml:"diff_hunk"`
	DiffMeta string `yaml:"diff_meta"`
	// Authors colours commit author initials (one per author, by hash).
	Authors []string `yaml:"authors,flow"`
}

type Theme struct {
	Name    string
	Kind    Kind
	Palette Palette
	Source  string // file it came from; empty for built-ins
}

// roles lists the palette fields by YAML name, for merging and checking.
func (p *Palette) roles() map[string]*string {
	return map[string]*string{
		"background": &p.Background, "text": &p.Text, "dim": &p.Dim, "accent": &p.Accent,
		"added": &p.Added, "removed": &p.Removed, "warn": &p.Warn, "branch": &p.Branch,
		"sel": &p.Sel, "sel_dim": &p.SelDim, "border": &p.Border, "surface": &p.Surface,
		"diff_add": &p.DiffAdd, "diff_del": &p.DiffDel, "diff_hunk": &p.DiffHunk, "diff_meta": &p.DiffMeta,
	}
}

// Builtin returns a built-in theme by name.
func Builtin(name string) (Theme, bool) {
	for _, t := range builtins {
		if t.Name == name {
			t.Palette.Authors = append([]string(nil), t.Palette.Authors...)
			return t, true
		}
	}
	return Theme{}, false
}

// BuiltinNames lists the built-in themes in display order.
func BuiltinNames() []string {
	out := make([]string, len(builtins))
	for i, t := range builtins {
		out[i] = t.Name
	}
	return out
}

// file is the YAML format of a user theme.
type file struct {
	Name    string            `yaml:"name"`
	Extends string            `yaml:"extends"`
	Kind    Kind              `yaml:"kind"`
	Colors  map[string]string `yaml:"colors"`
	Authors []string          `yaml:"authors,flow"`
}

// Set holds every theme available: built-ins plus user themes.
type Set struct {
	themes map[string]Theme
	order  []string
}

// Load reads user themes (*.yaml, *.yml) from dir on top of the built-ins.
// Broken files are reported in errs and skipped; the rest still load.
func Load(dir string) (*Set, []error) {
	s := &Set{themes: map[string]Theme{}}
	for _, t := range builtins {
		s.add(t)
	}
	files := map[string]file{}
	sources := map[string]string{}
	var errs []error
	paths, _ := filepath.Glob(filepath.Join(dir, "*.y*ml"))
	sort.Strings(paths)
	for _, p := range paths {
		f, err := readFile(p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if _, dup := files[f.Name]; dup {
			errs = append(errs, fmt.Errorf("%s: theme %q is defined twice", p, f.Name))
			continue
		}
		files[f.Name], sources[f.Name] = f, p
	}
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		t, err := s.resolve(n, files, map[string]bool{})
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", sources[n], err))
			continue
		}
		t.Source = sources[n]
		s.add(t)
	}
	return s, errs
}

func (s *Set) add(t Theme) {
	if _, ok := s.themes[t.Name]; !ok {
		s.order = append(s.order, t.Name)
	}
	s.themes[t.Name] = t
}

// Get returns a theme by name.
func (s *Set) Get(name string) (Theme, bool) {
	t, ok := s.themes[name]
	return t, ok
}

// Names lists built-in themes first, then user themes, as loaded.
func (s *Set) Names() []string { return append([]string(nil), s.order...) }

func readFile(path string) (file, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return file{}, err
	}
	var f file
	if err := yaml.Unmarshal(b, &f); err != nil {
		return file{}, fmt.Errorf("%s: %w", path, err)
	}
	if f.Name == "" {
		f.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return f, nil
}

// resolve builds a user theme: its base (another user theme or a built-in,
// catppuccin-mocha when extends is empty) with its own colours on top.
func (s *Set) resolve(name string, files map[string]file, seen map[string]bool) (Theme, error) {
	if seen[name] {
		return Theme{}, fmt.Errorf("theme %q extends itself (via a cycle)", name)
	}
	seen[name] = true
	f := files[name]
	base := f.Extends
	if base == "" {
		base = "catppuccin-mocha"
	}
	var t Theme
	switch {
	case base == name:
		return Theme{}, fmt.Errorf("theme %q extends itself", name)
	case files[base].Name != "":
		b, err := s.resolve(base, files, seen)
		if err != nil {
			return Theme{}, err
		}
		t = b
	default:
		b, ok := Builtin(base)
		if !ok {
			return Theme{}, fmt.Errorf("unknown theme %q in extends (built-in: %s)", base, strings.Join(BuiltinNames(), ", "))
		}
		t = b
	}
	t.Name, t.Source = name, ""
	if f.Kind != "" {
		t.Kind = f.Kind
	}
	roles := t.Palette.roles()
	for k, v := range f.Colors {
		ptr, ok := roles[k]
		if !ok {
			return Theme{}, fmt.Errorf("unknown colour role %q (roles: %s)", k, strings.Join(RoleNames(), ", "))
		}
		if err := validColor(v); err != nil {
			return Theme{}, fmt.Errorf("%s: %w", k, err)
		}
		*ptr = v
	}
	if len(f.Authors) > 0 {
		for _, c := range f.Authors {
			if err := validColor(c); err != nil {
				return Theme{}, fmt.Errorf("authors: %w", err)
			}
		}
		t.Palette.Authors = f.Authors
	}
	return t, nil
}

// RoleNames lists the colour roles a theme file may set, in display order.
func RoleNames() []string {
	return []string{"background", "text", "dim", "accent", "added", "removed", "warn", "branch",
		"sel", "sel_dim", "border", "surface", "diff_add", "diff_del", "diff_hunk", "diff_meta"}
}

var hexRe = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

func validColor(c string) error {
	if c == "" || hexRe.MatchString(c) {
		return nil
	}
	if n, err := strconv.Atoi(c); err == nil && n >= 0 && n <= 255 {
		return nil
	}
	return fmt.Errorf("%q is not a colour (use #rrggbb, #rgb or an ANSI number 0-255)", c)
}

// Export renders t as a complete theme file, a starting point for a custom theme.
func Export(t Theme) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "# gitnest theme. Save as ~/.config/gitnest/themes/<name>.yaml and pick it with T.\n")
	fmt.Fprintf(&b, "# Colours: \"#rrggbb\", \"#rgb\", an ANSI number \"0\"-\"255\", or \"\" for the terminal default.\n")
	fmt.Fprintf(&b, "# Remove any role to inherit it from `extends`.\n")
	fmt.Fprintf(&b, "name: my-%s\nextends: %s\nkind: %s\ncolors:\n", t.Name, t.Name, t.Kind)
	roles := t.Palette.roles()
	for _, r := range RoleNames() {
		fmt.Fprintf(&b, "  %-10s %q\n", r+":", *roles[r])
	}
	quoted := make([]string, len(t.Palette.Authors))
	for i, c := range t.Palette.Authors {
		quoted[i] = strconv.Quote(c)
	}
	fmt.Fprintf(&b, "authors: [%s]\n", strings.Join(quoted, ", "))
	return []byte(b.String())
}

// LoadFile resolves a single theme file against the built-ins and any user
// themes in the same directory (for `gitnest theme check <file>`).
func LoadFile(path string) (Theme, error) {
	f, err := readFile(path)
	if err != nil {
		return Theme{}, err
	}
	set, _ := Load(filepath.Dir(path))
	files := map[string]file{f.Name: f}
	if f.Extends != "" {
		if base, ok := set.Get(f.Extends); ok && base.Source != "" && base.Source != path {
			bf, err := readFile(base.Source)
			if err == nil {
				files[bf.Name] = bf
			}
		}
	}
	t, err := set.resolve(f.Name, files, map[string]bool{})
	if err != nil {
		return Theme{}, fmt.Errorf("%s: %w", path, err)
	}
	t.Source = path
	return t, nil
}

// Check returns readability problems: text that is hard to read on the
// selection or the intended background. Only #rrggbb colours can be
// measured; ANSI colours depend on the terminal.
func Check(t Theme) []string {
	p := t.Palette
	var out []string
	pairs := []struct {
		fg, bg, fgName, bgName string
		min                    float64
	}{
		{p.Text, p.Sel, "text", "sel", 4.5},
		{p.Text, p.SelDim, "text", "sel_dim", 4.5},
		{p.Text, p.Background, "text", "background", 4.5},
		{p.Accent, p.Background, "accent", "background", 3},
		{p.Added, p.Background, "added", "background", 3},
		{p.Removed, p.Background, "removed", "background", 3},
		{p.Warn, p.Background, "warn", "background", 3},
		{p.Branch, p.Background, "branch", "background", 3},
		{p.Dim, p.Background, "dim", "background", 2},
		{p.Text, p.Surface, "text", "surface", 4.5},
		{p.Dim, p.Surface, "dim", "surface", 2},
	}
	for _, c := range pairs {
		r, ok := Contrast(c.fg, c.bg)
		if ok && r < c.min {
			out = append(out, fmt.Sprintf("%s %s on %s %s: contrast %.1f:1, want at least %.1f:1", c.fgName, c.fg, c.bgName, c.bg, r, c.min))
		}
	}
	if len(p.Authors) == 0 {
		out = append(out, "authors: no colours for commit authors")
	}
	return out
}

// Contrast is the WCAG contrast ratio of two hex colours.
func Contrast(a, b string) (float64, bool) {
	la, ok1 := luminance(a)
	lb, ok2 := luminance(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05), true
}

func luminance(hex string) (float64, bool) {
	r, g, b, ok := RGB(hex)
	if !ok {
		return 0, false
	}
	lin := func(c uint8) float64 {
		v := float64(c) / 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b), true
}

// RGB parses #rgb or #rrggbb.
func RGB(hex string) (r, g, b uint8, ok bool) {
	if !hexRe.MatchString(hex) {
		return 0, 0, 0, false
	}
	h := hex[1:]
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return uint8(v >> 16), uint8(v >> 8), uint8(v), true
}
