# Developing gitnest

## Build and test

Requires Go 1.24+ and git 2.35+.

```sh
make build        # bin/gitnest with version information
make test         # go test ./...
make vet
make install      # into ~/.local/bin
```

The tests run the real `git` against temporary repositories, and the
end-to-end tests in `internal/ui/e2e*_test.go` drive the whole app with key
presses (`press(t, app, "3", "s", "y")`): the list, every panel of the
repository view, conflicts, panes, the update notice, and edge cases such as
empty repositories, detached HEAD, huge diffs, odd file names, network
timeouts and tiny windows. `go test -short ./...` skips them.
`go test -run '^$' -bench . ./internal/ui` times the screens redrawn on every
key with 1000 repositories (about a millisecond a frame) and the dashboard.
`TestReadmeKeyTablesMatchKeymap` fails when the key tables in
[KEYS.md](KEYS.md) and `keymap.go` disagree.

CI (`.github/workflows/ci.yml`) runs gofmt, vet, staticcheck and the tests
with `-race` on macOS and Ubuntu, the tests again on Debian 12, shellcheck
on the scripts, installs the built `.deb` on a clean Debian, and runs
`scripts/linux-smoke.sh` on Ubuntu and Debian 12: the commands, the terminal
UI driven through tmux (dashboard, list, search, a repository, the file
browser, the view menu, help) and the one-line installer.

## Layout

| Path | What |
|---|---|
| `cmd/gitnest` | CLI: commands, completion, shell integration, doctor |
| `internal/gitx` | everything that runs git: status, history rewriting, partial staging, cherry-pick |
| `internal/scan` | finding repositories (incremental, mtime-based) and collecting their status |
| `internal/ui` | the terminal UI (Bubble Tea); `keymap.go` is the single source of key bindings |
| `internal/theme` | colour themes and user theme files |
| `internal/config`, `internal/state` | config file, remembered state and caches |

## Releasing

Releases are built by GoReleaser from tags:

```sh
git tag v1.0.0 && git push origin v1.0.0
```

The `release` workflow builds a universal macOS binary and Linux
binaries (amd64, arm64), `.deb` and `.rpm` packages, publishes a GitHub
Release and updates the cask in
[Paksonite/homebrew-tap](https://github.com/Paksonite/homebrew-tap). It needs
a `HOMEBREW_TAP_TOKEN` repository secret: a fine-grained token with
*Contents: read and write* on `homebrew-tap`.

`make snapshot` builds the same artifacts locally without publishing.

A tag with a suffix (`v1.1.0-rc.1`) becomes a GitHub **pre-release**: the
Homebrew cask is not updated, and neither `gitnest update`, the update notice
nor `install.sh` (without a version) offers it. Testers install it on purpose:
`install.sh v1.1.0-rc.1`.

## Preview builds: gitnest-dev

Bigger changes (the UI redesign) are developed on the long-lived `ui-next`
branch: pull requests go into it, `main` is merged into it now and then, and
`main` stays ready to release. Every push to `ui-next` runs
`.github/workflows/dev.yml`, which builds **`gitnest-dev`** binaries and
publishes them to the rolling `dev` pre-release (replaced on every push, never
"latest", so gitnest users never see it).

`gitnest-dev` is installed next to `gitnest`, which stays untouched:

```sh
curl -fsSL https://raw.githubusercontent.com/Paksonite/gitnest/main/install.sh | sh -s -- --dev
```

`gitnest-dev update` then fetches the newest dev build (checked against the
release's `checksums.txt`). From a checkout, `make dev` builds and installs
the same thing locally. Both share gitnest's config and state; to keep them
apart, run `XDG_CONFIG_HOME=~/.config/gitnest-dev gitnest-dev`.

`gitnest update` (in `internal/update`) and `install.sh` depend on the asset
names in `.goreleaser.yaml` — `gitnest_<version>_darwin_universal.tar.gz`,
`gitnest_<version>_linux_<arch>.tar.gz`, `gitnest_<version>_<arch>.deb|.rpm`
and `checksums.txt` — so keep them in step when changing it. Pre-release tags
(`v1.1.0-rc1`) are never offered as updates, and `/releases/latest` skips
releases marked as pre-release on GitHub.

## The README demo

`docs/demo.gif` is recorded with [VHS](https://github.com/charmbracelet/vhs)
(`brew install vhs`) from `docs/demo.tape`:

```sh
vhs docs/demo.tape
```

The tape runs `docs/demo-setup.sh`, which builds gitnest and a throwaway
home in `$TMPDIR/gitnest-demo` with fake repositories (example.com remotes
backed by local bare repositories, author "Ann Example"), so nothing of
yours ends up in the GIF. Keep it under about 3 MB.
