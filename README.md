# gitnest

A terminal UI for managing many git repositories at once: a git-scope style
overview of every repository under your project directories, plus a
lazygit-style view for working inside a single repository.

> Early development. See the keys with `?` inside the app.

## Install

### Homebrew (macOS 13+, Apple Silicon and Intel)

```sh
brew install --cask paksonite/tap/gitnest
```

Upgrade with `brew update && brew upgrade --cask gitnest`.

The binary is not notarized by Apple; the cask removes the quarantine flag on
install. If you download the archive from GitHub Releases by hand, run
`xattr -d com.apple.quarantine gitnest` before the first start.

### From source

Requires Go 1.24+.

```sh
go install github.com/Paksonite/gitnest/cmd/gitnest@latest
```

or, from a checkout:

```sh
make install   # installs to ~/.local/bin
```

`git` must be on your `PATH` (the one from Xcode Command Line Tools is fine).

## Usage

```sh
gitnest roots add ~/repos   # directories to scan (also: gitnest roots, gitnest roots rm <dir>)
gitnest          # repository list
gitnest scan     # print every repository found
gitnest .        # open the repository you are in
gitnest version
```

## Shell completion (zsh)

The Homebrew cask installs it. For a build from source, add to `~/.zshrc`:

```sh
eval "$(gitnest completion zsh)"
```

## Configuration

`~/.config/gitnest/config.yaml` (or `$XDG_CONFIG_HOME/gitnest/config.yaml`):

```yaml
roots:            # directories scanned for repositories
  - ~/repos
depth: 4          # how deep below each root to look
exclude: [node_modules, vendor, .terraform, .venv, Library, .Trash]
skip_hidden: true
nested: false     # also find repositories inside repositories (not submodules)
follow_symlinks: false
rescan_interval: 60   # seconds between background scans for new clones (0 = off)
status_interval: 15   # seconds between status refreshes of visible rows (0 = off)
jobs: 8           # parallel git processes
editor: ""        # empty = $VISUAL, then $EDITOR, then vi
theme:
  mode: auto      # follows the terminal background; dark or light to force
  dark: catppuccin-mocha
  light: catppuccin-latte
  diff: theme     # "git" keeps your own git diff colours
  accent: ""      # overrides the theme's accent
```

Without a config file gitnest scans the usual project directories that exist
(`~/code`, `~/dev`, `~/Developer`, `~/projects`, `~/src`, `~/work`,
`~/workspace`, `~/repos`, `~/git`), never your whole home directory.

New clones are picked up automatically: background scans only re-read
directories whose modification time changed, so they stay cheap. Press `R` for
a full rescan.

Press `?` in the app for every key binding.

In a repository, the Commits panel draws the branch graph and marks each
commit with an icon for its kind (● commit, ◆ merge, ↺ revert, ↯ fixup/squash,
◎ root) and the author's initials in a per-author colour. `]` switches it to
HEAD's reflog (● commit, ✎ amend, ↻ rebase, ↶ reset, → checkout, ◆ merge,
↓ pull, ✚ cherry-pick); the diff panel then shows a graph of what each step
changed (`<` only before, `>` only after). From the reflog, `space` checks out
an entry and `g` resets the branch to it (`z` undoes).

## Themes

Press `T` in the app to pick a theme: moving through the list previews it
live, `enter` saves it for dark or light backgrounds, `esc` goes back.

Built in: `catppuccin-mocha`, `catppuccin-latte`, `gruvbox-dark`,
`gruvbox-light`, `nord`, `solarized-dark`, `solarized-light`, `tokyo-night`,
`dracula`, and `terminal`, which uses your terminal's own 16 ANSI colours
(best for Terminal.app, which has no true colour).

A custom theme is a small YAML file in `~/.config/gitnest/themes/` that
extends another theme and overrides a few colours:

```yaml
name: tokyo-orange
extends: tokyo-night
colors:
  accent: "#ff9e64"
  sel: "#3b4261"
```

```sh
gitnest theme                       # list themes and which ones are in use
gitnest theme export nord > ~/.config/gitnest/themes/my-nord.yaml   # full template
gitnest theme check my-nord         # unknown roles, bad colours, low contrast
```

Roles: `background` (not painted; used by `check`), `text`, `dim`, `accent`,
`added`, `removed`, `warn`, `branch`, `sel`, `sel_dim`, `border`, `diff_add`,
`diff_del`, `diff_hunk`, `diff_meta`, plus `authors` (a list of colours for
commit authors). Colours are `#rrggbb`, `#rgb`, ANSI numbers `0`–`255`, or
`""` for the terminal default.

## Releasing

Releases are built by GoReleaser from tags:

```sh
git tag v0.1.0 && git push origin v0.1.0
```

The `release` workflow builds a universal macOS binary, publishes a GitHub
Release and updates the cask in
[Paksonite/homebrew-tap](https://github.com/Paksonite/homebrew-tap). It needs a
`HOMEBREW_TAP_TOKEN` repository secret: a fine-grained token with
*Contents: read and write* on `homebrew-tap`.

`make snapshot` builds the same artifacts locally without publishing.

## License

MIT
