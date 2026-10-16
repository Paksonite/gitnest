package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Paksonite/gitnest/internal/config"
	"github.com/Paksonite/gitnest/internal/scan"
)

// rootsCmd implements `gitnest roots [list|add|rm] [dir...]`.
func rootsCmd(args []string) error {
	sub := "list"
	if len(args) > 0 {
		sub, args = args[0], args[1:]
	}
	switch sub {
	case "list", "ls":
		switch {
		case len(args) == 0:
			return listRoots()
		case len(args) == 1 && args[0] == "--plain":
			return listRootsPlain()
		}
		return errors.New("usage: gitnest roots list [--plain]")
	case "add":
		return addRoots(args)
	case "rm", "remove":
		return removeRoots(args)
	}
	return fmt.Errorf("unknown roots command %q (use list, add or rm)", sub)
}

func listRoots() error {
	cfg, existed, err := config.Load()
	if err != nil {
		return err
	}
	if !existed && len(cfg.Roots) > 0 {
		fmt.Printf("No config file yet; using detected directories (not saved to %s):\n", config.Path())
	}
	if len(cfg.Roots) == 0 {
		fmt.Println("No directories to scan. Add one with: gitnest roots add ~/repos")
		return nil
	}
	for _, r := range cfg.Roots {
		note := ""
		if st, err := os.Stat(r); err != nil || !st.IsDir() {
			note = "  (missing)"
		}
		fmt.Println(config.Abbrev(r) + note)
	}
	return nil
}

// listRootsPlain prints one root per line and nothing else (for completion).
func listRootsPlain() error {
	cfg, _, err := config.Load()
	if err != nil {
		return err
	}
	for _, r := range cfg.Roots {
		fmt.Println(config.Abbrev(r))
	}
	return nil
}

// savedRoots are the roots written in the config file (none without a file).
func savedRoots() ([]string, error) {
	cfg, existed, err := config.Load()
	if err != nil || !existed {
		return nil, err
	}
	return cfg.Roots, nil
}

func addRoots(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: gitnest roots add <dir> [<dir>]")
	}
	roots, err := savedRoots()
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	var added []string
	for _, a := range args {
		p, err := filepath.Abs(config.Expand(a))
		if err != nil {
			return err
		}
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			return fmt.Errorf("%s is not a directory", a)
		}
		if contains(roots, p) {
			fmt.Printf("%s is already scanned\n", config.Abbrev(p))
			continue
		}
		if p == home {
			fmt.Fprintln(os.Stderr, "warning: scanning your whole home directory is slow, and macOS will ask for access to Desktop, Documents and Downloads")
		}
		roots = append(roots, p)
		added = append(added, p)
	}
	if len(added) == 0 {
		return nil
	}
	if err := config.SetRoots(roots); err != nil {
		return err
	}
	for _, p := range added {
		fmt.Printf("added %s\n", config.Abbrev(p))
	}
	return nil
}

func removeRoots(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: gitnest roots rm <dir> [<dir>]")
	}
	roots, err := savedRoots()
	if err != nil {
		return err
	}
	var removed []string
	for _, a := range args {
		p, err := filepath.Abs(config.Expand(a))
		if err != nil {
			return err
		}
		i := index(roots, p)
		if i < 0 {
			return fmt.Errorf("%s is not in the scanned directories (see `gitnest roots`)", a)
		}
		roots = append(roots[:i], roots[i+1:]...)
		removed = append(removed, p)
	}
	if err := config.SetRoots(roots); err != nil {
		return err
	}
	for _, p := range removed {
		fmt.Printf("removed %s\n", config.Abbrev(p))
	}
	return nil
}

func index(roots []string, p string) int {
	for i, r := range roots {
		if filepath.Clean(r) == p {
			return i
		}
	}
	return -1
}

func contains(roots []string, p string) bool { return index(roots, p) >= 0 }

// scanCmd implements `gitnest scan`: a full scan printing every repository.
func scanCmd(args []string) error {
	if len(args) > 0 {
		return errors.New("scan takes no arguments")
	}
	cfg, _, err := config.Load()
	if err != nil {
		return err
	}
	if len(cfg.Roots) == 0 {
		return errors.New("no directories to scan; add one with: gitnest roots add ~/repos")
	}
	start := time.Now()
	repos, stats := scan.NewScanner(scan.FromConfig(cfg)).Scan(true)
	for _, r := range repos {
		fmt.Println(config.Abbrev(r))
	}
	fmt.Fprintf(os.Stderr, "%d repositories, %d directories, %s\n",
		len(repos), stats.Visited, time.Since(start).Round(10*time.Microsecond))
	return nil
}
