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

## Jump into a repository from the shell (zsh)

```sh
eval "$(gitnest shell zsh)"   # in ~/.zshrc
```

Then `gn` opens gitnest; press `o` on a repository (or inside one) to quit
and land in its directory.

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
icons: unicode    # "nerd" with a Nerd Font (e.g. MesloLGS NF)
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

In a repository, the Commits panel works like lazygit's: the hash (green on
the main line, yellow pushed, red only local), the author's initials in a
per-author colour, a commit graph with every line of development in its own
colour (○ commit, ◎ merge), the subject and refs. Its title has two tabs,
`Commits - Reflog`: `]` / `[` switch to HEAD's reflog, where the diff panel
shows a graph of what each step changed (`<` only before, `>` only after),
`space` checks out an entry and `g` resets the branch to it (`z` undoes).

`enter` on a file in the Files panel opens its changes line by line, like
lazygit's staging view: `space` stages the line under the cursor, a range
(`v`) or the whole hunk (`a`); `tab` switches to the staged side, where
`space` unstages; `d` discards unstaged lines. New files can be staged in
parts too.

`i` in the Commits panel opens an interactive rebase from the selected commit
up to HEAD: set pick / reword / edit / squash / fixup / drop per commit,
reorder with `ctrl+j` / `ctrl+k`, `enter` runs it (`z` undoes). `ctrl+j` /
`ctrl+k` in the panel itself move a single commit. `c` copies commits and
`v` pastes them (cherry-pick), also into another repository.

The Branches panel has three tabs, `Local - Remote - Tags` (`]` / `[`). In
Tags, `n` creates a tag at HEAD (with a message it is annotated), `P` pushes
it, `d` / `D` delete it locally / on the remote, `space` checks it out.

The checked-out branch is always first in the Branches panel, and checking
out another branch moves the selection to it.

In the repository list, `` ` `` groups repositories by the directories they
live in (for example one folder per git host) with collapsible directories;
`space` on a directory marks everything below it.

Set `icons: nerd` in the config if your terminal uses a Nerd Font (for
example MesloLGS NF): files get type icons, the reflog and branches get git
icons. The default `unicode` works with any font.

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
