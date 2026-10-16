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
`TestReadmeKeyTablesMatchKeymap` fails when the README's key tables and
`keymap.go` disagree.

CI (`.github/workflows/ci.yml`) runs gofmt, vet, staticcheck and the tests
with `-race` on macOS and Ubuntu, the tests again on Debian 12, shellcheck
on `install.sh`, and installs the built `.deb` on a clean Debian.

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

`gitnest update` (in `internal/update`) and `install.sh` depend on the asset
names in `.goreleaser.yaml` — `gitnest_<version>_darwin_universal.tar.gz`,
`gitnest_<version>_linux_<arch>.tar.gz`, `gitnest_<version>_<arch>.deb|.rpm`
and `checksums.txt` — so keep them in step when changing it. Pre-release tags
(`v1.1.0-rc1`) are never offered as updates, and `/releases/latest` skips
releases marked as pre-release on GitHub.
