#!/bin/sh
# Smoke test of a gitnest binary on Linux (CI runs it on Ubuntu and in a
# Debian 12 container): the commands, the terminal UI driven through tmux,
# and the one-line installer. Needs git, tmux and curl.
#
#   sh scripts/linux-smoke.sh ./gitnest
set -eu

bin=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
here=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'tmux kill-server 2>/dev/null || true; rm -rf "$work"' EXIT
export HOME="$work/home" TERM=xterm-256color GITNEST_NO_UPDATE_CHECK=1 COLORTERM=truecolor
unset XDG_CONFIG_HOME XDG_STATE_HOME XDG_CACHE_HOME
mkdir -p "$HOME/repos"
git config --global user.name "Ann Example"
git config --global user.email ann@example.com
git config --global init.defaultBranch main

fail() { echo "FAIL: $*" >&2; tmux capture-pane -p -t smoke 2>/dev/null >&2 || true; exit 1; }
step() { echo "--- $*"; }

# Two repositories: alpha in sync with a (local, bare) origin, beta dirty.
git init -q --bare "$work/origin.git"
git init -q "$HOME/repos/alpha"
(cd "$HOME/repos/alpha" && echo a >a.txt && git add a.txt && git commit -qm "alpha start" &&
  git remote add origin "$work/origin.git" && git push -qu origin main)
git init -q "$HOME/repos/beta"
(cd "$HOME/repos/beta" && echo b >b.txt && git add b.txt && git commit -qm "beta start" && echo more >>b.txt)

step "commands"
"$bin" version
"$bin" roots add "$HOME/repos" >/dev/null
"$bin" scan | grep -q alpha || fail "scan doesn't list alpha"
"$bin" doctor || true # warnings (no truecolor, no shell setup) are fine here

step "terminal UI"
# wait_for <text>: until the screen shows text (10 s).
wait_for() {
  i=0
  until tmux capture-pane -p -t smoke | grep -Fq -- "$1"; do
    i=$((i + 1))
    [ $i -le 50 ] || fail "the screen never showed \"$1\""
    sleep 0.2
  done
}
keys() { tmux send-keys -t smoke "$@"; sleep 0.4; }

tmux new-session -d -s smoke -x 140 -y 40 "'$bin'; echo \$? >'$work/exit'; sleep 30"
wait_for "dashboard"
wait_for "Needs attention"
wait_for "beta"
keys H
wait_for "Repositories"
keys / && keys -l alpha && keys Enter
wait_for "[1] Files"
wait_for "alpha"
keys B
wait_for "Files"
wait_for "a.txt"
keys Escape
keys Escape
wait_for "Repositories"
keys v
wait_for "View & filters"
keys Escape
keys "?"
wait_for "Keys"
keys q
keys q
i=0
until [ -f "$work/exit" ]; do
  i=$((i + 1))
  [ $i -le 50 ] || fail "gitnest didn't quit"
  sleep 0.2
done
[ "$(cat "$work/exit")" = 0 ] || fail "gitnest exited with $(cat "$work/exit")"
if tmux capture-pane -p -t smoke | grep -q "panic"; then fail "panic"; fi
grep -q '"screen":"list"' "$HOME/.local/state/gitnest/settings.json" || fail "the list wasn't remembered"

step "installer"
rc="$work/fresh/.bashrc" login=-ic
if [ "$(uname -s)" = Darwin ]; then # when run on a Mac
  rc="$work/fresh/.bash_profile" login=-lic
fi
HOME="$work/fresh" GITNEST_INSTALL_DIR="$work/fresh/.local/bin" SHELL=/bin/bash sh "$here/install.sh"
grep -q "gitnest shell bash" "$rc" || fail "no shell integration in $rc"
HOME="$work/fresh" GITNEST_INSTALL_DIR="$work/fresh/.local/bin" SHELL=/bin/bash sh "$here/install.sh" >/dev/null
[ "$(grep -c "gitnest installer" "$rc")" = 2 ] || fail "the installer added its lines twice"
HOME="$work/fresh" bash "$login" 'type gn >/dev/null && gitnest version' || fail "a new bash doesn't find gn and gitnest"
HOME="$work/fresh" GITNEST_NO_UPDATE_CHECK='' "$work/fresh/.local/bin/gitnest" update --check

echo "OK"
