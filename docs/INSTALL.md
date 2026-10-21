# Install, update and uninstall

- [Install](#install): [macOS (Homebrew)](#macos-homebrew), [Linux (one line)](#linux-one-line)
- [Other ways to install](#other-ways-to-install)
- [First start](#first-start)
- [Shell integration: completion and `gn`](#shell-integration-completion-and-gn)
- [Update](#update)
- [Uninstall](#uninstall)

## Install

### macOS: Homebrew

macOS 13 (Ventura) or newer, Apple Silicon or Intel:

```sh
brew install --cask paksonite/tap/gitnest
```

The cask installs zsh completion too; add the
[shell integration](#shell-integration-completion-and-gn) for `gn`. git comes
with the Command Line Tools, which macOS offers the first time you run `git`
(or `brew install git`).

### Linux: one line

Debian 12+, Ubuntu 22.04+, Fedora and others (amd64, arm64), without root —
and on macOS too, if you'd rather not use Homebrew:

```sh
curl -fsSL https://raw.githubusercontent.com/Paksonite/gitnest/main/install.sh | sh
```

It:

1. downloads the latest release for your system and checks it against the
   release's `checksums.txt`;
2. installs `gitnest` into `~/.local/bin` (`/usr/local/bin` when run as
   root);
3. adds that folder to your `PATH` and the [shell integration](#shell-integration-completion-and-gn)
   (`gn`, completion) to `~/.bashrc` or `~/.zshrc` (`~/.bash_profile` with
   bash on macOS) — each line once, marked "added by the gitnest installer".

Open a new terminal (or `exec $SHELL`) and run `gn`. Running the line again
installs the latest version over the old one and adds nothing twice.

Options go after `sh -s --`:

| | |
|---|---|
| `… \| sh -s -- v1.2.0` | that version instead of the latest |
| `… \| sh -s -- --no-modify` | don't touch your shell's startup files (or `GITNEST_NO_MODIFY=1`) |
| `… \| sh -s -- --dev` | the `gitnest-dev` preview build, next to `gitnest` (see [preview builds](#preview-builds-gitnest-dev)) |
| `GITNEST_INSTALL_DIR=/opt/bin` | install somewhere else |

gitnest needs **git** 2.35 or newer for everything (`sudo apt install git`).

> **Ubuntu 22.04** ships git 2.34. gitnest works with it, except "stash only
> staged changes" (`S`), which needs git 2.35. For a newer git:
> `sudo add-apt-repository ppa:git-core/ppa && sudo apt install git`.

## Other ways to install

**Packages**: every release has `.deb` and `.rpm` packages for amd64 and
arm64 on the [releases page](https://github.com/Paksonite/gitnest/releases):
`sudo apt install ./gitnest_<version>_amd64.deb` or
`sudo dnf install ./gitnest_<version>_amd64.rpm`. They install
`/usr/bin/gitnest` and the bash and zsh completion. (Switching to the install
script later? Remove the package first: `sudo apt remove gitnest`.)

**From source** (Go 1.24 or newer):

```sh
go install github.com/Paksonite/gitnest/v2/cmd/gitnest@latest
```

**Release archives** for every platform are on the
[releases page](https://github.com/Paksonite/gitnest/releases) too.

`gitnest update` works for every one of these ([below](#update)).

## First start

```sh
gitnest doctor                     # checks git, config, icons, terminal, shell, updates
gitnest roots add ~/work           # more folders to scan, as many as you like
gn                                 # or gitnest
```

Without configuration gitnest looks in the usual project folders that exist
(`~/code`, `~/dev`, `~/Developer`, `~/projects`, `~/src`, `~/work`,
`~/workspace`, `~/repos`, `~/git`) — never your whole home directory.

Other entry points:

```sh
gitnest .                          # open the repository you are in
gitnest roots                      # list the scanned folders (roots rm <dir> to remove)
gitnest scan                       # print every repository found
```

Press `?` in the app for every key.

## Shell integration: completion and `gn`

The installer adds it for you. By hand, it is one line in your shell's
startup file:

```sh
eval "$(gitnest shell zsh)"        # ~/.zshrc
eval "$(gitnest shell bash)"       # ~/.bashrc
```

It gives you completion and the **`gn`** command: it opens gitnest, and when
you press `o` on a repository (in the list or inside it) gitnest quits and
your shell is in that repository's folder. `gitnest` itself then does the
same — `gn` is just shorter. (Completion alone: `gitnest completion zsh` or
`gitnest completion bash`.)

## Update

gitnest looks for a new release on GitHub at most every 6 hours, also while
it keeps running (one small request, nothing else is sent) and shows `gitnest 1.2.0 available · gitnest update` in the
header when there is one. Turn this off with `update_check: false` or
`GITNEST_NO_UPDATE_CHECK=1`.

```sh
gitnest update                     # install it (--check only looks)
```

`gitnest update` updates gitnest the way it was installed:

| Installed with | `gitnest update` runs |
|---|---|
| Homebrew | `brew update && brew upgrade --cask gitnest` |
| the install script, a release archive | downloads the release, checks it and replaces the binary in place |
| `.deb` | downloads the new `.deb`, checks it, `sudo apt-get install` |
| `.rpm` | the same with `sudo dnf install` (or `rpm -U`) |
| `go install` | `go install …/cmd/gitnest@<new version>` |

If the binary is in a folder you can't write to (e.g. `/usr/local/bin`), run
`sudo gitnest update`. `gitnest update --force` reinstalls the latest
release. `gitnest doctor` also says whether an update is waiting.

### Preview builds: gitnest-dev

Bigger changes are tried out on the `ui-next` branch first. Every push there
builds **`gitnest-dev`** into the rolling `dev` pre-release (never "latest",
so `gitnest update` and Homebrew never offer it). It installs next to
`gitnest`, which stays untouched:

```sh
curl -fsSL https://raw.githubusercontent.com/Paksonite/gitnest/main/install.sh | sh -s -- --dev
```

`gitnest-dev update` fetches the newest one. Both share gitnest's config and
state; to keep them apart, run `XDG_CONFIG_HOME=~/.config/gitnest-dev
gitnest-dev`. Remove it with `rm ~/.local/bin/gitnest-dev`.

## Uninstall

| Installed with | Remove |
|---|---|
| Homebrew | `brew uninstall --cask gitnest` |
| the install script | `rm ~/.local/bin/gitnest`, and the lines marked "added by the gitnest installer" in `~/.zshrc` / `~/.bashrc` |
| `.deb` / `.rpm` | `sudo apt remove gitnest` / `sudo dnf remove gitnest` |
| `go install` | `rm "$(go env GOPATH)/bin/gitnest"` |

Configuration lives in `~/.config/gitnest`, caches in `~/.cache/gitnest` and
state in `~/.local/state/gitnest`; remove them to start from scratch.

---

[← README](../README.md)
