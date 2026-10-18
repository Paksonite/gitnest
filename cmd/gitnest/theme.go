package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Paksonite/gitnest/v2/internal/config"
	"github.com/Paksonite/gitnest/v2/internal/theme"
)

// themeCmd implements `gitnest theme [list|export <name>|check <name|file>]`.
func themeCmd(args []string) error {
	sub := "list"
	if len(args) > 0 {
		sub, args = args[0], args[1:]
	}
	switch sub {
	case "list", "ls":
		if len(args) > 0 {
			return errors.New("usage: gitnest theme list")
		}
		return listThemes()
	case "export":
		if len(args) != 1 {
			return errors.New("usage: gitnest theme export <name> > ~/.config/gitnest/themes/mine.yaml")
		}
		set, _ := theme.Load(config.ThemesDir())
		t, ok := set.Get(args[0])
		if !ok {
			return fmt.Errorf("unknown theme %q (see gitnest theme list)", args[0])
		}
		_, err := os.Stdout.Write(theme.Export(t))
		return err
	case "check":
		if len(args) != 1 {
			return errors.New("usage: gitnest theme check <name|file.yaml>")
		}
		return checkTheme(args[0])
	}
	return fmt.Errorf("unknown theme command %q (use list, export or check)", sub)
}

func listThemes() error {
	cfg, _, err := config.Load()
	if err != nil {
		return err
	}
	set, errs := theme.Load(config.ThemesDir())
	for _, n := range set.Names() {
		t, _ := set.Get(n)
		var marks []string
		if n == cfg.Theme.Dark || (cfg.Theme.Dark == "" && n == "gitnest-dark") {
			marks = append(marks, "dark")
		}
		if n == cfg.Theme.Light || (cfg.Theme.Light == "" && n == "gitnest-light") {
			marks = append(marks, "light")
		}
		line := fmt.Sprintf("%-20s %-6s", n, t.Kind)
		if t.Source != "" {
			line += "  " + config.Abbrev(t.Source)
		}
		if len(marks) > 0 {
			line += "  ← in use (" + strings.Join(marks, ", ") + ")"
		}
		fmt.Println(line)
	}
	fmt.Printf("\nmode: %s · custom themes: %s\n", orAuto(cfg.Theme.Mode), config.Abbrev(config.ThemesDir()))
	for _, e := range errs {
		fmt.Fprintln(os.Stderr, "error:", e)
	}
	if len(errs) > 0 {
		return fmt.Errorf("%d theme file(s) could not be loaded", len(errs))
	}
	return nil
}

func orAuto(mode string) string {
	if mode == "" {
		return "auto"
	}
	return mode
}

func checkTheme(arg string) error {
	var t theme.Theme
	if st, err := os.Stat(arg); err == nil && !st.IsDir() {
		if t, err = theme.LoadFile(arg); err != nil {
			return err
		}
	} else {
		set, _ := theme.Load(config.ThemesDir())
		var ok bool
		if t, ok = set.Get(arg); !ok {
			return fmt.Errorf("%s is neither a theme file nor a known theme", arg)
		}
	}
	probs := theme.Check(t)
	if len(probs) == 0 {
		fmt.Printf("%s: ok\n", t.Name)
		return nil
	}
	for _, p := range probs {
		fmt.Printf("%s: %s\n", t.Name, p)
	}
	return fmt.Errorf("%d readability problem(s)", len(probs))
}
