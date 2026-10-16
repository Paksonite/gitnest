// Command gitnest is a terminal UI for managing many git repositories.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Paksonite/gitnest/internal/config"
	"github.com/Paksonite/gitnest/internal/gitx"
	"github.com/Paksonite/gitnest/internal/ui"
)

// Set at build time via -ldflags "-X main.version=… -X main.commit=… -X main.date=…".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const usage = `gitnest — manage many git repositories from one terminal UI

Usage:
  gitnest                 open the repository list
  gitnest <path>          open the repository containing <path>
  gitnest init [--force]  write a default config to %s
  gitnest roots           list the directories scanned for repositories
  gitnest roots add <dir> scan <dir> (and rm <dir> to stop)
  gitnest scan            print every repository found
  gitnest version         print version information
  gitnest help            show this help
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "gitnest:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "help", "-h", "--help":
			fmt.Printf(usage, config.Path())
			return nil
		case "version", "-v", "--version":
			fmt.Printf("gitnest %s (commit %s, built %s)\n", version, commit, date)
			return nil
		case "init":
			return initConfig(args[1:])
		case "roots":
			return rootsCmd(args[1:])
		case "scan":
			return scanCmd(args[1:])
		}
		if len(args) > 1 {
			return fmt.Errorf("unexpected arguments: %v (see `gitnest help`)", args[1:])
		}
	}

	cfg, existed, err := config.Load()
	if err != nil {
		return err
	}
	startDir := ""
	if len(args) == 1 {
		if startDir, err = repoRoot(args[0]); err != nil {
			return err
		}
	}
	return ui.Run(cfg, !existed, startDir)
}

func initConfig(args []string) error {
	force := false
	for _, a := range args {
		switch a {
		case "-f", "--force":
			force = true
		default:
			return fmt.Errorf("init: unknown flag %q", a)
		}
	}
	path, err := config.Init(force)
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("%s already exists (use `gitnest init --force` to overwrite)", path)
	}
	if err != nil {
		return err
	}
	fmt.Println("wrote", path)
	return nil
}

func repoRoot(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if st, err := os.Stat(abs); err != nil {
		return "", err
	} else if !st.IsDir() {
		abs = filepath.Dir(abs)
	}
	top, err := gitx.Toplevel(context.Background(), abs)
	if err != nil || top == "" {
		return "", fmt.Errorf("%s is not inside a git repository", p)
	}
	return top, nil
}
