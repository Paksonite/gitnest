#!/bin/sh
# Installs gitnest on macOS or Linux, ready to use:
#
#   curl -fsSL https://raw.githubusercontent.com/Paksonite/gitnest/main/install.sh | sh
#
# It downloads the latest release (checked against the release's
# checksums.txt) into ~/.local/bin — /usr/local/bin when run as root,
# GITNEST_INSTALL_DIR to choose — puts that directory on your PATH and adds
# the shell integration (gn, completion) to ~/.zshrc or ~/.bashrc, each
# only once. Afterwards `gitnest update` keeps it up to date.
#
# Options (after `sh -s --`):
#   v1.2.0       that version instead of the latest
#   --dev        the gitnest-dev preview build (next to gitnest; update it
#                with gitnest-dev update)
#   --no-modify  leave your shell's startup files alone
set -eu

repo=Paksonite/gitnest
version=latest
dev=false
modify=true
for arg in "$@"; do
  case $arg in
  --dev) dev=true ;;
  --no-modify) modify=false ;;
  -*) printf 'gitnest install: unknown option %s\n' "$arg" >&2; exit 1 ;;
  *) version=$arg ;;
  esac
done
[ "${GITNEST_NO_MODIFY:-}" = 1 ] && modify=false

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

if [ -n "${GITNEST_INSTALL_DIR:-}" ]; then
  dir=$GITNEST_INSTALL_DIR
elif [ "$(id -u)" = 0 ]; then
  dir=/usr/local/bin
else
  dir=$HOME/.local/bin
fi

tmp=$(mktemp -d "${TMPDIR:-/tmp}/gitnest-install.XXXXXX")
trap 'rm -rf "$tmp"' EXIT INT TERM

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d ' ' -f 1
  else
    shasum -a 256 "$1" | cut -d ' ' -f 1
  fi
}

# fetch <base url> <asset>: download the asset and check it against the
# release's checksums.txt.
fetch() {
  curl -fsSL "$1/$2" -o "$tmp/$2" || die "download failed: $1/$2"
  curl -fsSL "$1/checksums.txt" -o "$tmp/checksums.txt" || die "checksums.txt not found at $1"
  want=$(grep " $2\$" "$tmp/checksums.txt" | cut -d ' ' -f 1)
  if [ -z "$want" ] || [ "$want" != "$(sha256 "$tmp/$2")" ]; then
    die "checksum mismatch for $2"
  fi
}

if $dev; then
  name=gitnest-dev
  say "Downloading the gitnest-dev preview build ($target)…"
  fetch "https://github.com/$repo/releases/download/dev" "gitnest-dev_$target"
  mv "$tmp/gitnest-dev_$target" "$tmp/$name"
else
  name=gitnest
  if [ "$version" = latest ]; then
    version=$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" |
      sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)
    [ -n "$version" ] || die "could not find the latest release"
  fi
  v=${version#v}
  say "Downloading gitnest $v ($target)…"
  fetch "https://github.com/$repo/releases/download/v$v" "gitnest_${v}_${target}.tar.gz"
  tar -xzf "$tmp/gitnest_${v}_${target}.tar.gz" -C "$tmp"
fi

mkdir -p "$dir"
dir=$(cd "$dir" && pwd -P) # always report an absolute path
cp "$tmp/$name" "$dir/.$name.new"
chmod 755 "$dir/.$name.new"
mv "$dir/.$name.new" "$dir/$name" # a rename: a running gitnest keeps working
bin="$dir/$name"
say "Installed $("$bin" version) at $bin"

# The shell's startup file, for PATH and the shell integration.
shell=$(basename "${SHELL:-sh}")
case $shell in
zsh) rc=$HOME/.zshrc ;;
bash)
  rc=$HOME/.bashrc
  [ "$os" = darwin ] && rc=$HOME/.bash_profile # macOS terminals start login shells
  ;;
*) rc= ;;
esac

# addline <line> <what>: append line to the startup file unless it is there.
changed=false
addline() {
  if [ -f "$rc" ] && grep -Fqs -- "$1" "$rc"; then
    return
  fi
  printf '\n# %s (added by the gitnest installer)\n%s\n' "$2" "$1" >>"$rc"
  say "Added $2 to $rc"
  changed=true
}

onpath=false
case ":$PATH:" in *":$dir:"*) onpath=true ;; esac
pathdir=$dir
case $dir in "$HOME"/*) pathdir="\$HOME${dir#"$HOME"}" ;; esac # stays right if the home folder moves
if $modify && [ -n "$rc" ]; then
  $onpath || addline "export PATH=\"$pathdir:\$PATH\"" "gitnest's folder on PATH"
  $dev || addline "eval \"\$(gitnest shell $shell)\"" "gitnest: gn and completion"
elif ! $onpath; then
  say ""
  say "$dir is not on your PATH; add it to your shell's startup file:"
  say "  export PATH=\"$dir:\$PATH\""
fi

found=$(command -v "$name" 2>/dev/null || true)
if $onpath && [ -n "$found" ] && [ "$found" != "$bin" ]; then
  say "Note: $found comes first on your PATH; remove it or put $dir before it."
fi
if ! $dev && [ "$bin" != /usr/bin/gitnest ] && command -v dpkg >/dev/null 2>&1 && dpkg -S /usr/bin/gitnest >/dev/null 2>&1; then
  say "Note: the .deb package's /usr/bin/gitnest is still installed; remove it with: sudo apt remove gitnest"
fi
command -v git >/dev/null 2>&1 || say "Note: gitnest needs git — install it first."

say ""
if $changed; then
  say "Open a new terminal (or run: exec $shell), then start it:"
else
  say "Start it:"
fi
if $dev; then
  say "  gitnest-dev                (gitnest-dev update fetches newer builds)"
else
  say "  gn                         (or gitnest; gitnest update keeps it current)"
fi
say "It finds repositories in ~/code, ~/projects, ~/repos and the like; add others with: $name roots add <folder>"
