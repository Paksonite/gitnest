package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestKeymapNoDuplicateKeysInScope(t *testing.T) {
	for sc, bindings := range keymap {
		seen := map[string]action{}
		for _, bd := range bindings {
			if len(bd.keys) == 0 || bd.help == "" {
				t.Errorf("%s: action %d has no keys or help", scopeTitles[sc], bd.act)
			}
			for _, k := range bd.keys {
				if prev, dup := seen[k]; dup {
					t.Errorf("%s: key %q bound to actions %d and %d", scopeTitles[sc], k, prev, bd.act)
				}
				seen[k] = bd.act
			}
		}
	}
}

func TestEveryActionIsBound(t *testing.T) {
	bound := map[action]bool{}
	for _, bindings := range keymap {
		for _, bd := range bindings {
			bound[bd.act] = true
		}
	}
	for act := actNone + 1; act <= actCancel; act++ {
		if !bound[act] {
			t.Errorf("action %d has no key binding", act)
		}
	}
}

func TestHintsAndHelpComeFromKeymap(t *testing.T) {
	if got := hints(scStash); got != "space apply  g pop  b branch  d drop" {
		t.Errorf("stash hints = %q", got)
	}
	help := strings.Join(helpLines(), "\n")
	for _, want := range []string{"Repository list", "fast-forward to upstream", "next panel", "move to the body"} {
		if !strings.Contains(help, want) {
			t.Errorf("help lacks %q", want)
		}
	}
	if lookup(scBranches, "F") != actFastForward || lookup(scRepo, "f") != actFetch || lookup(scList, "x") != actNone ||
		lookup(scCommits, "F") != actFixup || lookup(scBranches, "f") != actNone {
		t.Error("lookup")
	}
}

func TestHelpOverlay(t *testing.T) {
	a := testApp()
	a.w, a.h = 100, 30
	a.onKey(key("?"))
	if a.report == nil || a.report.title != "Keys" {
		t.Fatal("? should open the key overlay")
	}
	a.onKey(tea.KeyMsg{Type: tea.KeyDown})
	if a.report.off != 1 {
		t.Errorf("overlay should scroll, off = %d", a.report.off)
	}
	a.onKey(key("q"))
	if a.report != nil {
		t.Error("q should close the overlay")
	}
}

// readmeKeys reads the key tables of docs/KEYS.md: for each row, its first
// column and the keys written in backticks anywhere in it.
func readmeKeys(t *testing.T) map[string][]string {
	t.Helper()
	b, err := os.ReadFile("../../docs/KEYS.md")
	if err != nil {
		t.Fatal(err)
	}
	tick := regexp.MustCompile("``\\s?(.+?)\\s?``|`([^`]+)`")
	arrows := strings.NewReplacer("←", "left", "→", "right", "↑", "up", "↓", "down")
	rows := map[string][]string{}
	section := ""
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "#") {
			section = strings.TrimSpace(strings.TrimLeft(l, "#"))
			continue
		}
		if !strings.HasPrefix(l, "| ") || strings.HasPrefix(l, "| Key") || strings.HasPrefix(l, "| Panel") {
			continue
		}
		label := section
		if section == "Inside a repository" {
			label = strings.TrimSpace(strings.Split(l, "|")[1])
		}
		for _, m := range tick.FindAllStringSubmatch(l, -1) {
			k := m[1] + m[2]
			if strings.ContainsAny(k, "@$ ") || strings.HasPrefix(k, "alt+arrows") || k == "gn" {
				continue // @text, $EDITOR, the gn function, prose
			}
			if k == "space" {
				k = " "
			}
			rows[label] = append(rows[label], arrows.Replace(k))
		}
	}
	return rows
}

// The key tables in docs/KEYS.md and the keymap must agree: every key they
// name is bound where they say, and every key the footer shows is in them.
func TestReadmeKeyTablesMatchKeymap(t *testing.T) {
	where := map[string][]scope{
		"The repository list":               {scList, scFilter, scGlobal, scPanes, scPullAll},
		"Several repositories side by side": {scPanes, scList},
		"Files":                             {scFiles},
		"Line staging":                      {scStaging},
		"Branches":                          {scBranches},
		"Tags":                              {scTags},
		"Commits":                           {scCommits},
		"Interactive rebase":                {scPlan},
		"Reflog":                            {scReflog},
		"Stash":                             {scStash},
		"Anywhere":                          {scRepo, scGlobal},
	}
	rows := readmeKeys(t)
	for label, scs := range where {
		keys := rows[label]
		if len(keys) == 0 {
			t.Errorf("docs/KEYS.md has no key table row %q", label)
			continue
		}
		have := map[string]bool{}
		for _, k := range keys {
			have[k] = true
			bound := false
			for _, sc := range scs {
				bound = bound || lookup(sc, k) != actNone
			}
			if !bound {
				t.Errorf("docs/KEYS.md %q: %q is not bound", label, k)
			}
		}
		for _, sc := range scs[:1] {
			for _, bd := range keymap[sc] {
				if bd.hint != "" && !have[bd.keys[0]] {
					t.Errorf("docs/KEYS.md %q lacks %q (%s), shown in the footer", label, keyLabel(bd.keys[0]), bd.help)
				}
			}
		}
	}
}

// A panel's footer doesn't offer a repository-wide key the panel uses for
// something else.
func TestFooterSkipsShadowedKeys(t *testing.T) {
	a := testApp()
	a.scr = scrRepo
	for panel, shadowed := range map[int]string{pFiles: "enter diff", pStash: "space"} {
		a.rp.focus = panel
		if h := a.repoHints(); strings.Count(h, shadowed) > 1 {
			t.Errorf("panel %d footer offers %q twice: %s", panel, shadowed, h)
		}
	}
	// f fetches in every panel now (fast-forward and fixup moved to F).
	for _, panel := range []int{pFiles, pBranches, pCommits, pStash} {
		a.rp.focus = panel
		if h := a.repoHints(); !strings.Contains(h, "f fetch") {
			t.Errorf("panel %d footer lacks fetch: %s", panel, h)
		}
	}
}

// The help screen lists the main keys only; aliases still work.
func TestHelpShowsMainKeys(t *testing.T) {
	help := strings.Join(helpLines(), "\n")
	for _, want := range []string{"tab", "ctrl+←→↑↓", "↑ ↓"} {
		if !strings.Contains(help, want) {
			t.Errorf("help lacks %q", want)
		}
	}
	for _, unwanted := range []string{"tab right l", "pgup ctrl+u K", "ctrl+k K", "esc q enter h", "d ctrl+d pgdown", "home g", "ctrl+p ctrl+k", "alt+left", "focus the pane above"} {
		if strings.Contains(help, unwanted) {
			t.Errorf("help still shows %q", unwanted)
		}
	}
	if lookup(scRepo, "right") != actNextPanel || lookup(scMain, "J") != actPageDown || lookup(scPanes, "alt+up") != actPaneUp {
		t.Error("hidden aliases must keep working")
	}
	// No vim-style j/k anywhere, no backspace in the help.
	for sc := scope(0); sc < scopes; sc++ {
		if lookup(sc, "j") == actDown || lookup(sc, "k") == actUp {
			t.Errorf("%s still moves with j/k", scopeTitles[sc])
		}
	}
	if strings.Contains(help, "backspace") {
		t.Error("the help lists backspace")
	}
}

// A footer too long for the window loses hints from the end, never "? help".
func TestFooterKeepsHelp(t *testing.T) {
	a := testApp("/r/a")
	a.w, a.h = 80, 24
	f := a.footer(hints(scList, scPanes, scGlobal))
	last := strings.Split(f, "\n")[1]
	if lipgloss.Width(last) > 80 || !strings.Contains(last, "? help") || !strings.HasPrefix(last, "enter open") {
		t.Errorf("footer: %q", last)
	}
}
