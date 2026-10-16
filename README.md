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

Requires Go 1.22+.

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
gitnest init     # write ~/.config/gitnest/config.yaml, then edit `roots`
gitnest          # repository list
gitnest .        # open the repository you are in
gitnest version
```

## Configuration

`~/.config/gitnest/config.yaml` (or `$XDG_CONFIG_HOME/gitnest/config.yaml`):

```yaml
roots:            # directories scanned for repositories
  - ~/repos
depth: 4          # how deep below each root to look
exclude: [node_modules, vendor, .terraform, .venv, Library, .Trash]
skip_hidden: true
jobs: 8           # parallel git processes
editor: ""        # empty = $VISUAL, then $EDITOR, then vi
theme:
  accent: "#cba6f7"
```

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
