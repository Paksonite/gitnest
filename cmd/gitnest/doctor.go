package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/muesli/termenv"

	"github.com/Paksonite/gitnest/v2/internal/config"
	"github.com/Paksonite/gitnest/v2/internal/gitx"
	"github.com/Paksonite/gitnest/v2/internal/scan"
	"github.com/Paksonite/gitnest/v2/internal/state"
	"github.com/Paksonite/gitnest/v2/internal/theme"
	"github.com/Paksonite/gitnest/v2/internal/ui"
	"github.com/Paksonite/gitnest/v2/internal/update"
)

// doctorCmd checks the environment gitnest depends on and says how to fix
// what is missing. It exits non-zero only for problems that break gitnest.
func doctorCmd(args []string) error {
	if len(args) > 0 {
		return errors.New("doctor takes no arguments")
	}
	ok := func(label, detail string) { fmt.Printf("  ✓ %-14s %s\n", label, detail) }
	warn := func(label, detail string) { fmt.Printf("  ! %-14s %s\n", label, detail) }
	bad := 0
	fail := func(label, detail string) { bad++; fmt.Printf("  ✗ %-14s %s\n", label, detail) }

	v, c, _ := buildVersion()
	fmt.Printf("gitnest %s (%s) on %s/%s\n\n", v, c, runtime.GOOS, runtime.GOARCH)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if path, err := exec.LookPath("git"); err != nil {
		fail("git", "not found — install it: brew install git / sudo apt install git")
	} else if gv, err := gitx.GitVersion(ctx); err != nil {
		fail("git", err.Error())
	} else if !gitx.VersionAtLeast(gv, gitx.MinGitVersion) {
		warn("git", fmt.Sprintf("%s at %s is older than %s; most things work, stashing staged changes doesn't", gv, path, gitx.MinGitVersion))
	} else {
		ok("git", gv+" ("+path+")")
	}

	cfg, existed, err := config.Load()
	switch {
	case err != nil:
		fail("config", err.Error())
	case !existed:
		warn("config", "no "+config.Abbrev(config.Path())+" yet — gitnest roots add ~/repos creates it")
	default:
		ok("config", config.Abbrev(config.Path()))
	}
	if len(cfg.Roots) == 0 {
		fail("roots", "nothing to scan — gitnest roots add ~/repos")
	} else {
		var missing []string
		for _, r := range cfg.Roots {
			if st, err := os.Stat(r); err != nil || !st.IsDir() {
				missing = append(missing, config.Abbrev(r))
			}
		}
		repos, _ := scan.NewScanner(scan.FromConfig(cfg)).Scan(true)
		detail := fmt.Sprintf("%d repositories under %d root(s)", len(repos), len(cfg.Roots))
		if len(missing) > 0 {
			warn("roots", detail+"; missing: "+strings.Join(missing, ", "))
		} else {
			ok("roots", detail)
		}
	}

	set, terrs := theme.Load(config.ThemesDir())
	if len(terrs) > 0 {
		warn("themes", terrs[0].Error())
	}
	for _, slot := range []struct{ name, kind string }{{cfg.Theme.Dark, "dark"}, {cfg.Theme.Light, "light"}} {
		if slot.name == "" {
			continue
		}
		if _, found := set.Get(slot.name); !found {
			warn("theme", fmt.Sprintf("%s theme %q not found (gitnest theme list)", slot.kind, slot.name))
		}
	}
	ok("theme", fmt.Sprintf("dark %s, light %s, mode %s", orDefault(cfg.Theme.Dark, "gitnest-dark"),
		orDefault(cfg.Theme.Light, "gitnest-light"), orAuto(cfg.Theme.Mode)))

	mode, src := config.Icons(cfg)
	switch mode {
	case "nerd":
		ok("icons", "Nerd Font icons (from "+src+") — the terminal font must be a Nerd Font")
	case "symbols":
		ok("icons", "Unicode symbols (from "+src+"); with a Nerd Font, icons: nerd draws richer ones (T then i in the app)")
	default:
		ok("icons", "none (from "+src+"); icons: symbols shows icons with any font")
	}

	term := orDefault(os.Getenv("TERM"), "unknown")
	if p, why := ui.ColorProfile(); p == termenv.TrueColor {
		ok("terminal", term+", true colour ("+why+"); GITNEST_COLOR=256 if colours look wrong")
	} else {
		warn("terminal", term+", "+ui.ProfileName(p)+" ("+why+"): colours are approximated; GITNEST_COLOR=truecolor if the terminal has true colour")
	}

	shell := filepath.Base(os.Getenv("SHELL"))
	rc := map[string]string{"zsh": ".zshrc", "bash": ".bashrc"}[shell]
	if rc == "" {
		warn("shell", orDefault(shell, "unknown")+": completion and gn exist for zsh and bash")
	} else {
		home, _ := os.UserHomeDir()
		b, _ := os.ReadFile(filepath.Join(home, rc))
		if strings.Contains(string(b), "gitnest shell") {
			ok("shell", shell+": gn is set up in ~/"+rc)
		} else {
			warn("shell", fmt.Sprintf("%s: add  eval \"$(gitnest shell %s)\"  to ~/%s for gn and completion", shell, shell, rc))
		}
	}

	switch {
	case !config.UpdateCheck(cfg):
		warn("updates", "checks are off (update_check / GITNEST_NO_UPDATE_CHECK); gitnest update --check looks now")
	default:
		exe, _ := update.Executable()
		how := update.Detect(exe, update.PackageOwner, update.GopathBin())
		uctx, ucancel := context.WithTimeout(ctx, 5*time.Second)
		rel, newer, err := update.Check(uctx, v, state.UpdateCachePath(), 24*time.Hour)
		ucancel()
		switch {
		case err != nil:
			warn("updates", "can't reach GitHub: "+err.Error())
		case newer:
			warn("updates", fmt.Sprintf("%s is available — gitnest update (installed with %s)", rel.Version, how))
		default:
			ok("updates", fmt.Sprintf("latest release %s, installed with %s", rel.Version, how))
		}
	}

	fmt.Println()
	if bad > 0 {
		return fmt.Errorf("%d problem(s) to fix", bad)
	}
	fmt.Println("All set.")
	return nil
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
