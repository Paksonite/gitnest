# gitnest

**One terminal UI for all your git repositories.**

![gitnest: the repository list, pull all, a repository's panels and four panes side by side](docs/demo.gif)

gitnest finds every repository under your project folders and shows them on
one screen. Open any of them to stage, commit, branch, rebase and push — from
the keyboard or with the mouse. A single binary for macOS 13+ and Linux; it
only needs `git`.

- **A dashboard to start the day.** Commit activity across every repository,
  what needs attention, the most active repositories, the latest commits.
- **Every repository at a glance.** Branch, ahead/behind, uncommitted changes,
  search by name, filters, grouping by folder or remote host, favorites.
- **A full git client per repository.** Line-by-line staging, branches and
  tags, a coloured commit graph, the reflog, stashes, a file browser.
- **History editing you can undo.** Squash, fixup, reword, drop, reorder or an
  interactive rebase; `z` undoes any rewrite.
- **Work across repositories.** Several gitnests side by side (`ctrl+n`),
  cherry-pick between repositories, pull everything at once (`ctrl+p`).
- **Fits your terminal.** Twelve themes and your own, icons with any font,
  zsh and bash integration.

## Install

**macOS** (13+, Homebrew):

```sh
brew install --cask paksonite/tap/gitnest
```

**Linux** (Debian, Ubuntu, Fedora…; also works on macOS), one line without
root — it checks the download, puts gitnest on your `PATH` and adds `gn` to
your shell:

```sh
curl -fsSL https://raw.githubusercontent.com/Paksonite/gitnest/main/install.sh | sh
```

Then run `gitnest`. For `gn` — gitnest that leaves you in the repository's
folder — the one-line install sets it up; with Homebrew add
`eval "$(gitnest shell zsh)"` to `~/.zshrc`. `gitnest update` brings new
versions, whichever way you installed. Packages (`.deb`, `.rpm`), other
options and uninstalling: [docs/INSTALL.md](docs/INSTALL.md).

## Quick start

gitnest looks in the usual project folders (`~/code`, `~/projects`,
`~/repos`, …). Add yours:

```sh
gitnest roots add ~/work
```

Press `enter` to open a repository, `H` for the dashboard, `?` for every key,
and `o` to quit into a repository's folder.

## Documentation

- [Install, update and uninstall](docs/INSTALL.md)
- [Keys](docs/KEYS.md): the dashboard, the list, the panels, panes, the mouse
- [Configuration](docs/CONFIG.md): `config.yaml` and environment variables
- [Themes and icons](docs/THEMES.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)

License: [MIT](LICENSE).
