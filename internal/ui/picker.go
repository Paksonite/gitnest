package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/config"
	"github.com/Paksonite/gitnest/internal/theme"
)

// themePicker is the theme chooser (T). Moving the cursor applies the theme
// at once as a live preview; enter saves it, esc restores the previous one.
type themePicker struct {
	names []string
	cur   int
	orig  theme.Theme
}

func (a *App) openPicker() {
	names := a.themes.Names()
	p := &themePicker{names: names, orig: a.theme}
	for i, n := range names {
		if n == a.theme.Name {
			p.cur = i
		}
	}
	a.picker = p
}

func (a *App) previewTheme(t theme.Theme) {
	usePalette(t.Palette, a.cfg.Theme.Accent, a.cfg.Theme.Diff == "git")
}

func (a *App) pickerKey(k tea.KeyMsg) tea.Cmd {
	p := a.picker
	move := func(d int) {
		p.cur = clamp(p.cur+d, 0, len(p.names)-1)
		if t, ok := a.themes.Get(p.names[p.cur]); ok {
			a.previewTheme(t)
		}
	}
	switch lookup(scPicker, k.String()) {
	case actUp:
		move(-1)
	case actDown:
		move(1)
	case actTop:
		move(-len(p.names))
	case actBottom:
		move(len(p.names))
	case actToggleIcons:
		mode := iconModes[(iconSet+1)%len(iconModes)]
		setIcons(mode)
		a.cfg.Icons = mode
		if err := config.SetValue([]string{"icons"}, mode); err != nil {
			a.setToast("saving icons: "+err.Error(), true)
		} else {
			a.setToast("icons: "+iconNames[iconSet]+" (saved)"+iconHints[iconSet], false)
		}
	case actCancel:
		a.previewTheme(p.orig)
		a.picker = nil
	case actSave:
		t, _ := a.themes.Get(p.names[p.cur])
		a.picker = nil
		return a.saveTheme(t)
	}
	return nil
}

// saveTheme makes t the theme for its kind of background in the config.
// Choosing a theme made for the other background also switches theme.mode,
// otherwise auto-detection would put the old theme back on the next start.
func (a *App) saveTheme(t theme.Theme) tea.Cmd {
	a.theme = t
	a.previewTheme(t)
	dark := a.dark
	switch t.Kind {
	case theme.KindDark:
		dark = true
	case theme.KindLight:
		dark = false
	}
	slot := "light"
	if dark {
		slot = "dark"
	}
	msg := fmt.Sprintf("theme %s saved for %s backgrounds", t.Name, slot)
	if err := config.SetValue([]string{"theme", slot}, t.Name); err != nil {
		a.setToast("saving the theme: "+err.Error(), true)
		return nil
	}
	if dark != a.dark {
		if err := config.SetValue([]string{"theme", "mode"}, slot); err != nil {
			a.setToast("saving the theme mode: "+err.Error(), true)
			return nil
		}
		a.cfg.Theme.Mode = slot
		msg += " (mode: " + slot + ")"
	}
	if dark {
		a.cfg.Theme.Dark = t.Name
	} else {
		a.cfg.Theme.Light = t.Name
	}
	a.dark = dark
	a.setToast(msg, false)
	return nil
}

func swatch(c string) string {
	if c == "" {
		return "  "
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render("██")
}

func (a *App) viewPicker() string {
	p := a.picker
	// Fixed rows: border, padding, title, gaps and hints; the preview only
	// when there is room for it.
	showPreview := a.h >= 28
	fixed := 2 + 2*editorPad(a.h) + 2 + 1 + 2
	if showPreview {
		fixed += 8
	}
	listH := clamp(a.h-fixed, 1, len(p.names))
	off := centerOff(p.cur, len(p.names), listH)
	var rows []string
	for i := off; i < len(p.names) && i < off+listH; i++ {
		t, _ := a.themes.Get(p.names[i])
		pal := t.Palette
		marker := "  "
		if t.Name == p.orig.Name {
			marker = fg(cAccent, "● ")
		}
		name := padRight(t.Name, 18)
		if i == p.cur {
			name = lipgloss.NewStyle().Bold(true).Background(cSel).Foreground(cText).Render(name)
		}
		kind := string(t.Kind)
		if t.Source != "" {
			kind += ", custom"
		}
		row := marker + name + " "
		if a.w >= 70 { // narrow windows skip the kind column
			row += fg(cDim, padRight(kind, 14)) + " "
		}
		rows = append(rows, row+swatch(pal.Accent)+swatch(pal.Added)+swatch(pal.Removed)+swatch(pal.Warn)+swatch(pal.Branch)+swatch(pal.Text))
	}

	// A small sample of the UI in the previewed theme.
	sel := lipgloss.NewStyle().Bold(true).Background(cSel).Foreground(cText)
	_, sficon := fileIconStyled("main.go", false, false)
	_, sdicon := fileIconStyled("src", true, false)
	iconsMode := iconNames[iconSet]
	sample := []string{
		fg(cDim, "icons: "+iconsMode+" (i)") + "   " + fg(cDim, "▾ ") + sdicon + fg(cCyan, "src/") + "  " + fg(cRed, " M") + " " + sficon + "main.go",
		sel.Render(fit(" ● gitnest       main      ↑1 ↓2 ~3   2h", 44)),
		" " + fg(cYellow, "●") + " " + bold("org-core-dns") + fg(cDim, " · forgejo") + " " + fg(cCyan, "main") + " " + fg(cGreen, "↑1") + " " + fg(cRed, "↓2"),
		"   " + bold("terraform-modules") + "  " + fg(cCyan, "feature/x") + "  " + fg(cDim, "3d"),
		fg(cAccent, "@@ -1,3 +1,4 @@") + "  " + fg(cGreen, "+ added line") + "  " + fg(cRed, "- removed line"),
	}
	body := bold("Theme") + fg(cDim, fmt.Sprintf("  %d/%d", p.cur+1, len(p.names))) + "\n\n" + strings.Join(rows, "\n")
	if showPreview {
		body += "\n\n" + fg(cDim, "Preview") + "\n" + strings.Join(sample, "\n")
	}
	footer := hints(scPicker)
	if a.w >= 80 {
		footer += "  ·  custom themes: " + config.Abbrev(config.ThemesDir())
	}
	body += "\n\n" + fg(cDim, footer)
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cAccent).
		Padding(editorPad(a.h), 2).Width(min(72, a.w-2)).Render(body)
	return lipgloss.Place(a.w, a.h, lipgloss.Center, lipgloss.Center, box)
}

// iconNames and iconHints describe the icon sets in the theme chooser.
var (
	iconNames = [...]string{iconsSymbols: "Unicode symbols", iconsNerd: "Nerd Font", iconsPlain: "none"}
	iconHints = [...]string{
		iconsSymbols: " — any font",
		iconsNerd:    " — they need a Nerd Font in the terminal",
		iconsPlain:   "",
	}
)
