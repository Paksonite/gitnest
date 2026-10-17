# gitnest

**One terminal UI for all your git repositories.**

![gitnest: the repository list, pull all, a repository's panels and four panes side by side](docs/demo.gif)

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
  cherry-pick from one repository into another, pull everything with `ctrl+p`.
- **Fits your terminal.** Ten themes plus your own, file and branch icons
  with any font (richer ones with a Nerd Font), zsh and bash integration.

## Install and update

### macOS

Install (Homebrew, macOS 13+):

```sh
brew install --cask paksonite/tap/gitnest
```

Update:

```sh
brew update && brew upgrade --cask gitnest
```

### Debian / Ubuntu

Install the `.deb` from the latest release (Debian 12+, Ubuntu 22.04+):

```sh
arch=$(dpkg --print-architecture)
version=$(curl -fsSL https://api.github.com/repos/Paksonite/gitnest/releases/latest | sed -n 's/.*"tag_name": *"v\([^"]*\)".*/\1/p')
curl -fsSLO "https://github.com/Paksonite/gitnest/releases/download/v${version}/gitnest_${version}_${arch}.deb"
sudo apt install -y "./gitnest_${version}_${arch}.deb"
```

Update (downloads the new `.deb`, verifies its checksum and installs it):

```sh
gitnest update
```

`gitnest update` works for every install method, and the header shows
`⬆ <version> · gitnest update` when a new release is out. The install
script for any macOS or Linux, Fedora `.rpm` packages, `go install` and
uninstalling are in [docs/INSTALL.md](docs/INSTALL.md).

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
- [Themes and icons](docs/THEMES.md): built-in and custom themes, the icon
  sets (Unicode symbols, Nerd Font)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Development](docs/DEVELOPMENT.md): building, tests and releasing

License: [MIT](LICENSE).
