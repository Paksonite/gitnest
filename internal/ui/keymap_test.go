package ui

import (
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
	for _, want := range []string{"Repository list", "fast-forward to upstream", "shift+tab", "ctrl+s alt+enter"} {
		if !strings.Contains(help, want) {
			t.Errorf("help lacks %q", want)
		}
	}
	if lookup(scBranches, "f") != actFastForward || lookup(scRepo, "f") != actFetch || lookup(scList, "x") != actNone {
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
	a.onKey(key("j"))
	if a.report.off != 1 {
		t.Errorf("overlay should scroll, off = %d", a.report.off)
	}
	a.onKey(key("q"))
	if a.report != nil {
		t.Error("q should close the overlay")
	}
}
