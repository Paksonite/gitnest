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
  fuzzy search, filters, grouping by folder or remote host, favorites.
- **A full git client per repository.** Line-by-line staging, branches and
  tags, a coloured commit graph, the reflog, stashes, a file browser.
- **History editing you can undo.** Squash, fixup, reword, drop, reorder or an
  interactive rebase; `z` undoes any rewrite.
- **Work across repositories.** Several gitnests side by side (`ctrl+n`),
  cherry-pick between repositories, pull everything at once (`ctrl+p`).
- **Fits your terminal.** Twelve themes and your own, icons with any font,
  zsh and bash integration.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/Paksonite/gitnest/main/install.sh | sh
```

On macOS and Linux (Debian, Ubuntu, Fedora…), without root. It checks the
download, puts gitnest on your `PATH` and adds `gn` to your shell. Open a new
terminal and run `gn`; `gitnest update` brings new versions.

Homebrew, `.deb` and `.rpm` packages, other options and uninstalling:
[docs/INSTALL.md](docs/INSTALL.md).

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
- [Development](docs/DEVELOPMENT.md): building, tests, releases, preview builds

License: [MIT](LICENSE).
