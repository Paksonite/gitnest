#!/bin/sh
# Installs the gitnest binary from a GitHub release, on macOS or Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/Paksonite/gitnest/main/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/Paksonite/gitnest/main/install.sh | sh -s -- v1.0.0
#
# Installs into ~/.local/bin, or /usr/local/bin when run as root; set
# GITNEST_INSTALL_DIR to choose. The download is checked against the
# release's checksums.txt.
set -eu

repo=Paksonite/gitnest
version=${1:-latest}

say() { printf '%s\n' "$*"; }
die() { say "gitnest install: $*" >&2; exit 1; }

command -v curl >/dev/null 2>&1 || die "curl is required"
command -v tar >/dev/null 2>&1 || die "tar is required"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case $os in
darwin) target=darwin_universal ;;
linux)
  case $arch in
  x86_64 | amd64) target=linux_amd64 ;;
  aarch64 | arm64) target=linux_arm64 ;;
  *) die "unsupported CPU: $arch" ;;
  esac
  ;;
*) die "unsupported system: $os" ;;
esac

if [ "$version" = latest ]; then
  version=$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" |
    sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)
  [ -n "$version" ] || die "could not find the latest release"
fi
v=${version#v}
asset="gitnest_${v}_${target}.tar.gz"
base="https://github.com/$repo/releases/download/v$v"

if [ -n "${GITNEST_INSTALL_DIR:-}" ]; then
  dir=$GITNEST_INSTALL_DIR
elif [ "$(id -u)" = 0 ]; then
  dir=/usr/local/bin
else
  dir=$HOME/.local/bin
fi

tmp=$(mktemp -d "${TMPDIR:-/tmp}/gitnest-install.XXXXXX")
trap 'rm -rf "$tmp"' EXIT INT TERM

say "Downloading gitnest $v ($target)…"
curl -fsSL "$base/$asset" -o "$tmp/$asset" || die "download failed: $base/$asset"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt" || die "checksums.txt not found"

want=$(grep " $asset\$" "$tmp/checksums.txt" | cut -d ' ' -f 1)
if command -v sha256sum >/dev/null 2>&1; then
  have=$(sha256sum "$tmp/$asset" | cut -d ' ' -f 1)
else
  have=$(shasum -a 256 "$tmp/$asset" | cut -d ' ' -f 1)
fi
if [ -z "$want" ] || [ "$want" != "$have" ]; then
  die "checksum mismatch for $asset"
fi

tar -xzf "$tmp/$asset" -C "$tmp"
mkdir -p "$dir"
dir=$(cd "$dir" && pwd -P) # always report an absolute path
cp "$tmp/gitnest" "$dir/gitnest.new"
chmod 755 "$dir/gitnest.new"
mv "$dir/gitnest.new" "$dir/gitnest"

bin="$dir/gitnest"
say "Installed $bin"
"$bin" version

# How to run it: by name when $dir is on PATH (and no other gitnest comes
# first), otherwise by its absolute path until PATH is fixed.
run=gitnest
case ":$PATH:" in
*":$dir:"*)
  found=$(command -v gitnest 2>/dev/null || true)
  if [ -n "$found" ] && [ "$found" != "$bin" ]; then
    say "Note: $found comes first on your PATH; remove it or put $dir before it."
    run=$bin
  fi
  ;;
*)
  run=$bin
  # shellcheck disable=SC2088 # shown to the user, not expanded
  case ${SHELL:-} in
  */zsh) rc="~/.zshrc" ;;
  */bash) rc="~/.bashrc" ;;
  *) rc="your shell's startup file" ;;
  esac
  say ""
  say "$dir is not on your PATH. Add this line to $rc and open a new terminal:"
  say "  export PATH=\"$dir:\$PATH\""
  ;;
esac
command -v git >/dev/null 2>&1 || say "Note: gitnest needs git — install it first."
say ""
say "Next:"
say "  $run roots add ~/repos"
say "  $run doctor"
