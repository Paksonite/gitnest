// Package config loads ~/.config/gitnest/config.yaml and generates it for `gitnest init`.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Theme struct {
	Accent string `yaml:"accent"`
}

type Config struct {
	Roots      []string `yaml:"roots"`
	Depth      int      `yaml:"depth"`
	Exclude    []string `yaml:"exclude"`
	SkipHidden bool     `yaml:"skip_hidden"`
	Jobs       int      `yaml:"jobs"`
	Editor     string   `yaml:"editor"`
	Theme      Theme    `yaml:"theme"`
}

const template = `# gitnest configuration
# Directories scanned for git repositories.
roots:
%s# How many directory levels below each root to search.
depth: 4
# Directory names that are never descended into.
exclude: [node_modules, vendor, .terraform, .venv, Library, .Trash]
# Skip directories starting with "." (the roots themselves are always scanned).
skip_hidden: true
# Parallel git processes used for scanning, fetch and pull.
jobs: 8
# Editor used by "e" (empty = $VISUAL, then $EDITOR, then vi).
editor: ""
theme:
  accent: "#cba6f7"
`

// Dir returns the gitnest config directory (~/.config/gitnest, or $XDG_CONFIG_HOME/gitnest).
func Dir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "gitnest")
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".config", "gitnest")
}

// Path returns the config file path.
func Path() string { return filepath.Join(Dir(), "config.yaml") }

func detectRoots() []string {
	h, _ := os.UserHomeDir()
	var roots []string
	for _, n := range []string{"code", "dev", "Developer", "projects", "src", "work", "repos", "git"} {
		p := filepath.Join(h, n)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			roots = append(roots, p)
		}
	}
	if len(roots) == 0 {
		roots = []string{h}
	}
	return roots
}

// Default returns the built-in configuration.
func Default() Config {
	return Config{
		Roots:      detectRoots(),
		Depth:      4,
		Exclude:    []string{"node_modules", "vendor", ".terraform", ".venv", "Library", ".Trash"},
		SkipHidden: true,
		Jobs:       8,
		Theme:      Theme{Accent: "#cba6f7"},
	}
}

func expand(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		h, _ := os.UserHomeDir()
		return filepath.Join(h, strings.TrimPrefix(p, "~"))
	}
	return p
}

// Load reads the config file. The bool reports whether a file existed;
// when it did not, defaults are returned.
func Load() (Config, bool, error) {
	cfg := Default()
	b, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, false, nil
		}
		return cfg, false, err
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, true, fmt.Errorf("%s: %w", Path(), err)
	}
	if cfg.Depth <= 0 {
		cfg.Depth = 4
	}
	if cfg.Jobs <= 0 {
		cfg.Jobs = 8
	}
	if len(cfg.Roots) == 0 {
		cfg.Roots = detectRoots()
	}
	for i, r := range cfg.Roots {
		cfg.Roots[i] = expand(r)
	}
	return cfg, true, nil
}

// Init writes a commented config.yaml. It returns os.ErrExist (with the path)
// when the file already exists and force is false.
func Init(force bool) (string, error) {
	path := Path()
	if _, err := os.Stat(path); err == nil && !force {
		return path, os.ErrExist
	}
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return path, err
	}
	var roots strings.Builder
	for _, r := range detectRoots() {
		fmt.Fprintf(&roots, "  - %q\n", r)
	}
	return path, os.WriteFile(path, []byte(fmt.Sprintf(template, roots.String())), 0o644)
}
