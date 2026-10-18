#!/bin/sh
# Seeds the E2E world inside the client container (run as root there).
# The server's bare repositories were created beforehand (see run.py);
# here a developer's clones are made in ~/repos, and a second developer
# (/root/other) pushes so the clones end up behind, ahead, diverged…
set -eu
cd /root
rm -rf repos other
mkdir -p repos/work repos/labs other

commit() { # commit <dir> <file> <content> <message> [author]
  author=${5:-"Dev Example <dev@example.com>"}
  mkdir -p "$(dirname "$1/$2")"
  printf '%s\n' "$3" >"$1/$2"
  git -C "$1" add -- "$2"
  git -C "$1" commit -q --author="$author" -m "$4"
}
ANN="Ann Example <ann@example.com>"
BOB="Bob Builder <bob@example.com>"

# A repository with history, two authors, branches and tags, pushed to the
# server: <name> becomes git@gitserver:<name>.git.
publish() { # publish <name>
  d=/root/other/$1
  git init -q "$d"
  commit "$d" README.md "# $1

The $1 project, for testing gitnest." "Initial commit" "$ANN"
  commit "$d" src/main.go "package main

func main() {}" "Add main" "$BOB"
  commit "$d" src/util.go "package main

func helper() int { return 1 }" "Add helper" "$ANN"
  commit "$d" docs/notes.md "Notes" "Write notes" "$BOB"
  git -C "$d" tag -a v1.0.0 -m "Release 1.0.0"
  commit "$d" src/main.go "package main

func main() { helper() }" "Call the helper" "$ANN"
  git -C "$d" tag v1.1.0
  git -C "$d" branch feature/login
  git -C "$d" checkout -q feature/login
  commit "$d" src/login.go "package main

func login() {}" "Start the login" "$BOB"
  git -C "$d" checkout -q main
  git -C "$d" remote add origin "git@gitserver:$1.git"
  git -C "$d" push -q -u origin main feature/login --tags 2>/dev/null
}

for r in app lib web diverged conflicted detached unicode; do publish "$r"; done

clone() { git clone -q "git@gitserver:$1.git" "/root/repos/$2" 2>/dev/null; }
clone app work/app
clone lib work/lib
clone web work/web
clone diverged labs/diverged
clone conflicted labs/conflicted
clone detached labs/detached
clone unicode labs/unicode

# app: a modified file, a staged one, an untracked one, and a stash.
a=/root/repos/work/app
printf 'stash me\n' >>"$a/docs/notes.md"
git -C "$a" stash push -q -m "half-done notes"
printf 'package main\n\nfunc helper() int { return 2 }\n' >"$a/src/util.go"
printf 'staged change\n' >>"$a/README.md"
git -C "$a" add README.md
printf 'scratch\n' >"$a/todo.txt"

# lib: two commits behind its upstream.
commit /root/other/lib src/lib.go "package lib" "Add the library" "$ANN"
commit /root/other/lib src/lib2.go "package lib // 2" "Grow the library" "$BOB"
git -C /root/other/lib push -q
git -C /root/repos/work/lib fetch -q

# web: one commit ahead of its upstream.
commit /root/repos/work/web src/page.go "package main // page" "Add a page"

# diverged: one commit each side, different files.
commit /root/other/diverged remote.txt "theirs" "Their change" "$BOB"
git -C /root/other/diverged push -q
commit /root/repos/labs/diverged local.txt "ours" "Our change"
git -C /root/repos/labs/diverged fetch -q

# conflicted: a merge stopped on a conflict in README.md.
c=/root/repos/labs/conflicted
git -C "$c" checkout -q -b topic
commit "$c" README.md "topic version" "Topic edit"
git -C "$c" checkout -q main
commit "$c" README.md "main version" "Main edit"
git -C "$c" merge -q topic >/dev/null 2>&1 || true

# detached: HEAD on a tag.
git -C /root/repos/labs/detached checkout -q v1.0.0

# unicode: odd file names, a binary file and a huge diff.
u=/root/repos/labs/unicode
mkdir -p "$u/dir with space" "$u/日本語"
printf 'zażółć\n' >"$u/zażółć gęślą.txt"
printf 'ファイル\n' >"$u/日本語/ファイル.md"
printf 'spaces\n' >"$u/dir with space/my file.txt"
head -c 3000 /dev/urandom >"$u/blob.bin"
seq 1 20000 >"$u/big.txt"
git -C "$u" add -A
git -C "$u" commit -q -m "Odd names, binary, big"
seq 1 20000 | sed 's/0$/zero/' >"$u/big.txt"

# scratch: no remote at all.
s=/root/repos/labs/scratch
git init -q "$s"
commit "$s" notes.txt "local only" "Local notes"

# gitnest looks here.
mkdir -p /root/.config/gitnest
printf 'roots: [/root/repos]\nupdate_check: false\nicons: symbols\n' >/root/.config/gitnest/config.yaml
rm -rf /root/.local/state/gitnest /root/.cache/gitnest
echo seeded
