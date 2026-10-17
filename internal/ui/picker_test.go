package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/config"
	"github.com/Paksonite/gitnest/internal/theme"
)

func themedApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := config.Init(false); err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := config.Load()
	a := testApp()
	a.w, a.h = 120, 40
	a.cfg = cfg
	a.cfg.Theme.Mode = "dark"
	a.loadThemes()
	t.Cleanup(func() { mocha, _ := theme.Builtin("catppuccin-mocha"); usePalette(mocha.Palette, "", true) })
	return a
}

func TestPickerPreviewCancelAndSave(t *testing.T) {
	a := themedApp(t)
	if a.theme.Name != "catppuccin-mocha" {
		t.Fatalf("start theme %s", a.theme.Name)
	}
	a.onKey(key("T"))
	if a.picker == nil {
		t.Fatal("T should open the theme chooser")
	}
	a.onKey(key("j")) // catppuccin-latte, previewed at once
	if cAccent != lipgloss.Color("#8839ef") {
		t.Errorf("preview not applied: accent %v", cAccent)
	}
	if v := a.View(); !strings.Contains(v, "catppuccin-latte") || !strings.Contains(v, "Preview") {
		t.Error("chooser view lacks the list or the preview")
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyEsc})
	if a.picker != nil || cAccent != lipgloss.Color("#cba6f7") {
		t.Errorf("esc should restore the previous theme, accent %v", cAccent)
	}

	// Save tokyo-night (dark): written to theme.dark, mode unchanged.
	a.onKey(key("T"))
	for a.picker.names[a.picker.cur] != "tokyo-night" {
		a.onKey(key("j"))
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	cfg, _, _ := config.Load()
	if cfg.Theme.Dark != "tokyo-night" || cfg.Theme.Mode != "auto" || a.theme.Name != "tokyo-night" {
		t.Errorf("saved: %+v, active %s", cfg.Theme, a.theme.Name)
	}

	// Picking a light theme on a dark background also switches the mode.
	a.onKey(key("T"))
	for a.picker.names[a.picker.cur] != "gruvbox-light" {
		a.onKey(key("k"))
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	cfg, _, _ = config.Load()
	if cfg.Theme.Light != "gruvbox-light" || cfg.Theme.Mode != "light" || cfg.Theme.Dark != "tokyo-night" {
		t.Errorf("saved: %+v", cfg.Theme)
	}
	b, _ := os.ReadFile(config.Path())
	if !strings.Contains(string(b), "# auto follows the terminal background") {
		t.Error("config comments were lost")
	}
}

func TestUnknownThemeFallsBack(t *testing.T) {
	a := themedApp(t)
	a.cfg.Theme.Dark = "does-not-exist"
	a.loadThemes()
	if a.theme.Name != "catppuccin-mocha" || !strings.Contains(a.themeWarn, "does-not-exist") {
		t.Errorf("theme %s warn %q", a.theme.Name, a.themeWarn)
	}
}

func TestUserThemeIsOffered(t *testing.T) {
	a := themedApp(t)
	dir := config.ThemesDir()
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "mine.yaml"), []byte("extends: nord\ncolors:\n  accent: \"#ff0000\"\n"), 0o644)
	a.cfg.Theme.Dark = "mine"
	a.loadThemes()
	if a.theme.Name != "mine" || cAccent != lipgloss.Color("#ff0000") {
		t.Errorf("theme %s accent %v warn %q", a.theme.Name, cAccent, a.themeWarn)
	}
}

func TestPickerTogglesIcons(t *testing.T) {
	a := themedApp(t)
	setIcons("symbols")
	defer setIcons("symbols")
	a.onKey(key("T"))
	for _, want := range []struct{ mode, shown string }{{"nerd", "Nerd Font"}, {"plain", "none"}, {"symbols", "Unicode symbols"}} {
		a.onKey(key("i"))
		cfg, _, _ := config.Load()
		if iconModes[iconSet] != want.mode || cfg.Icons != want.mode {
			t.Fatalf("after i: set %s, saved %q, want %s", iconModes[iconSet], cfg.Icons, want.mode)
		}
		if !strings.Contains(a.View(), "icons: "+want.shown) {
			t.Errorf("chooser should show icons: %s", want.shown)
		}
	}
}
