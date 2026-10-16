// Command gitnest is a terminal UI for managing many git repositories.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
  gitnest theme           list colour themes (also: theme export <name>, theme check <name|file>)
  gitnest completion zsh  print the completion script (zsh or bash)
  gitnest shell zsh       gn + completion for ~/.zshrc (or: shell bash for ~/.bashrc)
  gitnest version         print version information
  gitnest doctor          check git, config, icons, terminal and shell setup
  gitnest help            show this help
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "gitnest:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cdFile, args, err := cutCdFile(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		switch args[0] {
		case "help", "-h", "--help":
			fmt.Printf(usage, config.Path())
			return nil
		case "version", "-v", "--version":
			v, c, d := buildVersion()
			fmt.Printf("gitnest %s (commit %s, built %s)\n", v, c, d)
			return nil
		case "init":
			return initConfig(args[1:])
		case "roots":
			return rootsCmd(args[1:])
		case "scan":
			return scanCmd(args[1:])
		case "completion":
			return completionCmd(args[1:])
		case "theme", "themes":
			return themeCmd(args[1:])
		case "shell":
			return shellCmd(args[1:])
		case "doctor":
			return doctorCmd(args[1:])
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
	return ui.Run(cfg, !existed, startDir, cdFile)
}

// cutCdFile removes "--cd-file <path>" (used by the gn shell function) from args.
func cutCdFile(args []string) (string, []string, error) {
	var rest []string
	cdFile := ""
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--cd-file":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--cd-file needs a path")
			}
			cdFile = args[i+1]
			i++
		case strings.HasPrefix(a, "--cd-file="):
			cdFile = strings.TrimPrefix(a, "--cd-file=")
		default:
			rest = append(rest, a)
		}
	}
	return cdFile, rest, nil
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
