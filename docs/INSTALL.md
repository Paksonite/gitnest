# Install, update and uninstall

- [Install on macOS](#install-on-macos)
- [Install on Debian / Ubuntu](#install-on-debian--ubuntu)
- [Other ways to install](#other-ways-to-install)
- [First start](#first-start)
- [Shell integration: completion and `gn`](#shell-integration-completion-and-gn)
- [Update and uninstall](#update-and-uninstall)

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

[← README](../README.md)
