# gitnest

**One terminal UI for all your git repositories.**

![gitnest: the repository list, staging a line, squash and undo, two panes](docs/demo.gif)

gitnest finds every repository under your project folders and shows them in
one list. Open any of them to stage, commit, branch, rebase and push, all
from the keyboard. It is a single binary for macOS 13+ and Linux, and only
needs `git`.

- **Every repository at a glance.** Branch, ahead/behind, uncommitted
  changes and last commit, with fuzzy search (`/`), sorting and grouping by folder.
- **New clones appear on their own.** Background scans pick up repositories
  cloned or removed while gitnest runs.
- **A full git client per repository.** Files as a tree with line-by-line
  staging, branches and tags, a coloured commit graph, the reflog, stashes.
- **History editing you can undo.** Squash, fixup, reword, drop, reorder or
  run an interactive rebase. `z` undoes any rewrite.
- **Work across repositories.** Up to six gitnests side by side (`ctrl+n`),
  cherry-pick from one repository into another, fetch or pull many at once.
- **Fits your terminal.** Ten themes plus your own, optional Nerd Font
  icons, zsh and bash integration.

## Install

**macOS** (Homebrew):

```sh
brew install --cask paksonite/tap/gitnest
```

**Debian / Ubuntu** (`.deb` from the latest release; Fedora has `.rpm`):

```sh
arch=$(dpkg --print-architecture)
version=$(curl -fsSL https://api.github.com/repos/Paksonite/gitnest/releases/latest | sed -n 's/.*"tag_name": *"v\([^"]*\)".*/\1/p')
curl -fsSLO "https://github.com/Paksonite/gitnest/releases/download/v${version}/gitnest_${version}_${arch}.deb"
sudo apt install -y "./gitnest_${version}_${arch}.deb"
```

**Any macOS or Linux** (no root, into `~/.local/bin`):

```sh
curl -fsSL https://raw.githubusercontent.com/Paksonite/gitnest/main/install.sh | sh
```

Requirements, `go install`, archives and updating are covered in
[docs/INSTALL.md](docs/INSTALL.md).

## Quick start

1. Install gitnest (above) and check it with `gitnest doctor`.
2. Tell it where your repositories live:
   ```sh
   gitnest roots add ~/repos
   ```
3. Add the shell integration to `~/.zshrc` (or `~/.bashrc` with `bash`):
   ```sh
   eval "$(gitnest shell zsh)"
   ```

Open a new terminal and run `gn`. Press `enter` to open a repository, `o` to
quit into its folder, and `?` for every key.

## Documentation

- [Keys](docs/KEYS.md): the repository list, the panels inside a repository,
  panes
- [Install, update and uninstall](docs/INSTALL.md): every platform, first
  start, shell integration
- [Configuration](docs/CONFIG.md): `config.yaml` and environment variables
- [Themes and icons](docs/THEMES.md): built-in and custom themes, Nerd Font
  icons
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Development](docs/DEVELOPMENT.md): building, tests and releasing

License: [MIT](LICENSE).
