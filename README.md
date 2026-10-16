# gitnest

**One terminal UI for all your git repositories.**

gitnest finds every repository under your project folders and shows them in
one list: branch, ahead/behind, uncommitted changes, last commit. Open any of
them to stage (down to single lines), commit, branch, rebase, cherry-pick,
tag and push — all from the keyboard.

- **Every repository at a glance.** Fuzzy search with `/`, sort, group by
  folder (for example by git host), see which ones need attention.
- **New clones appear on their own.** Background scans pick up repositories
  cloned or removed while gitnest runs.
- **A full git client per repository.** Files as a tree with line-by-line
  staging, branches and tags, a coloured commit graph, the reflog, stashes.
- **History editing you can undo.** Squash, fixup, reword, drop, reorder or
  plan a whole interactive rebase; every rewrite can be undone with `z`, and
  rewriting pushed commits warns you first.
- **Work across repositories.** Several gitnests side by side in one
  terminal (`ctrl+n`), copy commits from one repository into another, fetch
  or pull many repositories at once.
- **Fits your terminal.** Ten built-in themes plus your own, light and dark
  terminals, optional Nerd Font icons, zsh and bash integration.

Runs on macOS 13+ and Linux (Debian, Ubuntu and others), as a
single binary. It only needs `git`.

---

## Contents

- [Install on macOS](#install-on-macos)
- [Install on Debian / Ubuntu](#install-on-debian--ubuntu)
- [Other ways to install](#other-ways-to-install)
- [First start](#first-start)
- [Shell integration: completion and `gn`](#shell-integration-completion-and-gn)
- [Icons (Nerd Font)](#icons-nerd-font)
- [Using gitnest](#using-gitnest)
- [Several repositories side by side](#several-repositories-side-by-side)
- [Configuration](#configuration)
- [Themes](#themes)
- [Troubleshooting](#troubleshooting)
- [Update and uninstall](#update-and-uninstall)

---

## Install on macOS

Requires macOS 13 (Ventura) or newer, Apple Silicon or Intel.

1. **Homebrew** — if you don't have it yet, see [brew.sh](https://brew.sh).
2. **git** — macOS asks to install the Command Line Tools the first time you
   run `git`. A newer git from Homebrew works too:
   ```sh
   brew install git
   ```
3. **gitnest:**
   ```sh
   brew install --cask paksonite/tap/gitnest
   ```
   The cask also installs zsh completion.

Check it:

```sh
gitnest doctor
```

## Install on Debian / Ubuntu

Works on Debian 12+ and Ubuntu 22.04+ (amd64 and arm64).

1. **git** (2.35 or newer; Debian 12 has 2.39, Ubuntu 22.04 has 2.34 — see
   the note below):
   ```sh
   sudo apt update && sudo apt install -y git curl
   ```
2. **gitnest** from the `.deb` package of the latest release:
   ```sh
   arch=$(dpkg --print-architecture)   # amd64 or arm64
   version=$(curl -fsSL https://api.github.com/repos/Paksonite/gitnest/releases/latest | sed -n 's/.*"tag_name": *"v\([^"]*\)".*/\1/p')
   curl -fsSLO "https://github.com/Paksonite/gitnest/releases/download/v${version}/gitnest_${version}_${arch}.deb"
   sudo apt install -y "./gitnest_${version}_${arch}.deb"
   ```
   The package installs `/usr/bin/gitnest` and bash and zsh completion, and
   depends on `git`.

   Fedora / RHEL: the same release has `.rpm` packages
   (`sudo dnf install ./gitnest_<version>_amd64.rpm`).

Check it:

```sh
gitnest doctor
```

> **Ubuntu 22.04** ships git 2.34. gitnest works with it, except "stash only
> staged changes" (`S`), which needs git 2.35. For a newer git:
> `sudo add-apt-repository ppa:git-core/ppa && sudo apt install git`.

## Other ways to install

**Install script** (macOS or Linux, no root needed — installs into
`~/.local/bin`, or `/usr/local/bin` as root, and verifies the checksum):

```sh
curl -fsSL https://raw.githubusercontent.com/Paksonite/gitnest/main/install.sh | sh
```

A specific version: `… | sh -s -- v1.0.0`. Make sure `~/.local/bin` is on
your `PATH`.

**From source** (Go 1.24 or newer):

```sh
go install github.com/Paksonite/gitnest/cmd/gitnest@latest
```

The binary lands in `$(go env GOPATH)/bin` (usually `~/go/bin`).

**Release archives** for every platform are on the
[releases page](https://github.com/Paksonite/gitnest/releases).

## First start

Tell gitnest where your repositories live, then start it:

```sh
gitnest roots add ~/repos          # as many folders as you like
gitnest                            # the repository list
```

Without configuration gitnest looks in the usual project folders that exist
(`~/code`, `~/dev`, `~/Developer`, `~/projects`, `~/src`, `~/work`,
`~/workspace`, `~/repos`, `~/git`) — never your whole home directory.

Other entry points:

```sh
gitnest .                          # open the repository you are in
gitnest roots                      # list the scanned folders (roots rm <dir> to remove)
gitnest scan                       # print every repository found
gitnest doctor                     # check git, config, icons, terminal, shell and updates
```

Press `?` in the app for every key binding.

## Shell integration: completion and `gn`

One line in your shell's startup file gives you completion for `gitnest` and
the **`gn`** command: it opens gitnest, and when you press `o` on a
repository (in the list or inside it) gitnest quits and your shell is in that
repository's folder. After it, `gitnest` itself does the same — `gn` is just
shorter.

**zsh** — add to `~/.zshrc` (anywhere: before or after `compinit` or oh-my-zsh):

```sh
eval "$(gitnest shell zsh)"
```

**bash** — add to `~/.bashrc`:

```sh
eval "$(gitnest shell bash)"
```

Open a new terminal and try `gn` (or `gitnest`), then `o`. (Completion alone: `gitnest completion zsh`
or `gitnest completion bash`.)

## Icons (Nerd Font)

By default gitnest uses plain symbols that every font has. With a
[Nerd Font](https://www.nerdfonts.com) in your terminal you get file-type
icons in the Files tree and git icons for branches, tags and the reflog.

1. **Install a Nerd Font**, for example MesloLGS NF:
   - macOS: `brew install --cask font-meslo-lg-nerd-font`
   - Debian / Ubuntu:
     ```sh
     mkdir -p ~/.local/share/fonts && cd ~/.local/share/fonts
     curl -fsSLO https://github.com/ryanoasis/nerd-fonts/releases/latest/download/Meslo.tar.xz
     tar -xf Meslo.tar.xz && rm Meslo.tar.xz && fc-cache -f
     ```
2. **Select it in your terminal** (iTerm2: Settings → Profiles → Text → Font;
   Terminal.app: Settings → Profiles → Text; GNOME Terminal: Preferences →
   your profile → Custom font). Over SSH, the font of the terminal on your own
   computer is what counts.
3. **Turn icons on in gitnest:** press `T`, then `i` (it is saved), or set
   `icons: nerd` in the config.

## Using gitnest

### The repository list

| Key | |
|---|---|
| `/` | search by name (also git host and folder; `@text` searches branches); `enter` opens, `tab` keeps the filter, `esc` clears |
| `enter` | open the repository |
| `` ` `` | group repositories by folder (tree) or show a flat list; `enter` folds a folder |
| `s` | sort: recently used, name, last commit, needs attention |
| `d` | only repositories with uncommitted changes |
| `space` | mark (on a folder: everything below); `f` fetch / `p` pull the marked ones |
| `R` / `r` | rescan the folders / refresh the status |
| `q` | quit |
| `o` | quit and go to the repository (needs the [shell integration](#shell-integration-completion-and-gn)) |
| `T` | theme and icons |
| `ctrl+n` | open a second gitnest next to this one ([panes](#several-repositories-side-by-side)) |

`✦` marks repositories found since gitnest started, `●` uncommitted changes,
`!` a conflict or an operation in progress. Clones of the same project from
different hosts are told apart by host (`org-core-dns · forgejo`).

### Inside a repository

Four panels on the left — **Files**, **Branches** (`Local - Remote - Tags`),
**Commits** (`Commits - Reflog`), **Stash** — and the diff on the right.
`1`–`4` or `tab` switch panels, `enter` on the diff scrolls it.

| Panel | Keys |
|---|---|
| Files | `space` stage/unstage, `a` all, `c` commit, `C` commit in `$EDITOR`, `A` amend, `d` discard, `s` / `S` stash all / staged, `e` edit, `` ` `` tree/flat, **`enter` on a file: stage single lines** |
| Line staging | `space` stage the line, `v` range, `a` hunk, `tab` staged side (unstage), `d` discard lines, `esc` back |
| Branches | `space` checkout, `n` new, `r` rename, `d`/`D` delete, `u`/`U` set/unset upstream, `f` fast-forward, `M` merge, `R` rebase onto, `]`/`[` next/previous tab |
| Tags | `n` new at HEAD, `P` push, `d`/`D` delete local/remote, `space` checkout |
| Commits | `s` squash, `f` fixup, `d` drop, `r` reword, `t` revert, `u` undo last commit, `ctrl+j`/`ctrl+k` move, **`i` interactive rebase**, `c` copy / `v` paste (cherry-pick, also into another repository) / `C` forget copied, `]` reflog |
| Interactive rebase | `p` pick, `r` reword, `e` edit, `s` squash, `f` fixup, `d` drop, `ctrl+j`/`ctrl+k` move, `enter` run, `esc` cancel |
| Reflog | `space` check out an entry, `g` reset the branch to it, `[` back to the commits |
| Stash | `space` apply, `g` pop, `b` branch from it, `d` drop |
| Anywhere | `p` pull, `ctrl+p` pull --rebase, `P` push, `F` force push, `f` or `ctrl+f` fetch (`ctrl+f` in Branches, where `f` fast-forwards), `z` undo the last history change, `o` quit and cd here, `enter` scroll the diff, `esc` back to the list, `!` full error, `?` help |

The commit message editor has a **Subject** and an optional **Body**: type the
subject and press `enter` to commit, or `tab` to write a body, then `ctrl+s`.

When a rebase, merge, cherry-pick or revert stops on a conflict, the header
says so: resolve the files, stage them, and press `C` to continue, `S` to skip
or `X` to abort.

Rewriting commits that are already pushed asks twice; afterwards `P` offers a
push with `--force-with-lease` — but only when the remote has nothing you
haven't seen. Every history change records an undo point: `z` goes back,
`z` again redoes.

## Several repositories side by side

Work on related repositories at the same time — say IAM in one, the
resources in another and a shared module in a third — with up to six
gitnests on screen at once in one terminal:

| Key | |
|---|---|
| `ctrl+n` | open another pane next to the current one; pick its repository from the list |
| `ctrl+←` `ctrl+→` `ctrl+↑` `ctrl+↓` | move to the pane in that direction (`alt+arrows` work too) |
| `ctrl+g` | layout: grid (2×2 for four, 3×2 for six…), side by side, stacked |
| `ctrl+w` | close the pane (the last one quits); `q` on a pane's list closes it too |

Each pane is a complete gitnest with its own repository, panels, search and
editor. The active pane has a heavy border in the accent colour and its title
(`● 2 · repo · branch`) on an accent background; the others are thin and
dimmed. When panes would get too small for the window, gitnest falls back to
another layout and finally to tabs above the active pane; from the seventh
pane on it always shows tabs. When the tabs don't fit on one line, the bar
follows the active one and `‹ 3` / `4 ›` count the tabs hidden on each side.
The layout is remembered. Commits copied with `c` in one pane can be pasted with `v` in
another.

> **macOS:** by default `ctrl+←/→/↑/↓` switch Spaces and open Mission
> Control, so the terminal never sees them. Either turn those shortcuts off in
> System Settings → Keyboard → Keyboard Shortcuts → Mission Control, or use
> `alt+arrows` (in iTerm2 set Profiles → Keys → Left/Right Option key to
> "Esc+").

## Configuration

`~/.config/gitnest/config.yaml` (or `$XDG_CONFIG_HOME/gitnest/config.yaml`).
`gitnest init` writes a commented one; `gitnest roots add` edits it in place.

```yaml
roots:                 # folders scanned for repositories
  - ~/repos
depth: 4               # how deep below each root to look
exclude: [node_modules, vendor, .terraform, .venv, Library, .Trash]
skip_hidden: true
nested: false          # also find repositories inside repositories (not submodules)
follow_symlinks: false
rescan_interval: 60    # seconds between background scans for new clones (0 = off)
status_interval: 15    # seconds between status refreshes of visible rows (0 = off)
jobs: 8                # parallel git processes
editor: ""             # for e; empty = $VISUAL, then $EDITOR, then vi
icons: unicode         # or nerd (needs a Nerd Font)
update_check: true     # look for a new release once a day (false = never)
theme:
  mode: auto           # follows the terminal background; dark or light to force
  dark: catppuccin-mocha
  light: catppuccin-latte
  diff: theme          # "git" keeps your own git diff colours
  accent: ""           # overrides the theme's accent colour
```

`GITNEST_ICONS=nerd` overrides `icons` for one session;
`GITNEST_NO_UPDATE_CHECK=1` turns the release check off.

## Themes

Press `T` to pick a theme: the list previews each one live, `enter` saves it
for dark or light backgrounds, `esc` goes back, `i` switches icons.

Built in: `catppuccin-mocha`, `catppuccin-latte`, `gruvbox-dark`,
`gruvbox-light`, `nord`, `solarized-dark`, `solarized-light`, `tokyo-night`,
`dracula`, and `terminal`, which uses your terminal's own 16 colours (best
for terminals without true colour, such as macOS Terminal.app).

Your own theme is a small file in `~/.config/gitnest/themes/` that extends
another and changes a few colours:

```yaml
# ~/.config/gitnest/themes/tokyo-orange.yaml
extends: tokyo-night
colors:
  accent: "#ff9e64"
```

```sh
gitnest theme                                    # all themes, and which are in use
gitnest theme export nord > ~/.config/gitnest/themes/my-nord.yaml   # a full template
gitnest theme check my-nord                      # unknown roles, bad colours, low contrast
```

## Troubleshooting

Start with `gitnest doctor`: it checks git, the configuration, the scanned
folders, themes, icons, colour support and the shell setup, and says what to
do.

- **"No directories to scan"** — `gitnest roots add ~/your/projects`.
- **Boxes or question marks instead of icons** — the terminal font is not a
  Nerd Font: install one (see [Icons](#icons-nerd-font)) or press `T` then `i`
  to go back to plain symbols.
- **Colours look off** — your terminal may not report its background: set
  `theme.mode: dark` or `light`. Without true colour, try the `terminal`
  theme.
- **`gitnest: command not found` after the install script** — add
  `~/.local/bin` to your `PATH`.
- **Push or fetch fails without asking for a password** — gitnest never
  prompts: use an SSH agent (`ssh-add`) or a credential helper
  (macOS: `osxkeychain`, Linux: `git config --global credential.helper store`
  or `libsecret`). `!` shows git's full error.
- **macOS says the binary "is damaged"** (only when you downloaded the archive
  in a browser): `xattr -d com.apple.quarantine /path/to/gitnest`.

## Update and uninstall

gitnest looks for a new release on GitHub at most once a day (one small
request, nothing else is sent). When there is one, the header shows
`⬆ 1.2.0 · gitnest update`. Turn this off with `update_check: false` or
`GITNEST_NO_UPDATE_CHECK=1`; development builds never check.

```bash
gitnest update --check
```

```bash
gitnest update
```

`gitnest update` works out how gitnest was installed and updates it the same
way:

| Installed with | `gitnest update` runs | Uninstall |
|---|---|---|
| Homebrew | `brew update && brew upgrade --cask gitnest` | `brew uninstall --cask gitnest` |
| `.deb` (Debian / Ubuntu) | downloads the new `.deb`, checks it against `checksums.txt`, `sudo apt-get install` | `sudo apt remove gitnest` |
| `.rpm` | the same with `sudo dnf install` (or `rpm -U`) | `sudo dnf remove gitnest` |
| install script / archive | downloads the release archive, checks it and replaces the binary in place | `rm ~/.local/bin/gitnest` |
| `go install` | `go install …/cmd/gitnest@<new version>` | `rm "$(go env GOPATH)/bin/gitnest"` |

If the binary sits in a directory you can't write to (e.g. `/usr/local/bin`),
run `sudo gitnest update`. `gitnest update --force` reinstalls the latest
release, e.g. over a development build. `gitnest doctor` also says whether an
update is waiting.

Configuration lives in `~/.config/gitnest`, caches in `~/.cache/gitnest` and
state in `~/.local/state/gitnest`; remove them to start from scratch.

---

Development, tests and releasing: [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).

License: [MIT](LICENSE).
