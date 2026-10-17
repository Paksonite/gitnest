# Keys

Press `?` in the app for the same list. Panes are described in
[Several repositories side by side](#several-repositories-side-by-side).

## The repository list

| Key | |
|---|---|
| `/` | search by name (also git host and folder; `@text` searches branches); `enter` opens, `tab` keeps the filter, `esc` clears |
| `enter` | open the repository |
| `` ` `` | group repositories by folder (tree) or show a flat list; `enter` folds a folder |
| `s` | sort: recently used, name, last commit, needs attention |
| `d` | only repositories with uncommitted changes |
| `space` | mark (on a folder: everything below); `f` fetch / `p` pull the marked ones |
| `R` / `r` | rescan the folders / refresh the status |
| `q` | quit |
| `o` | quit and go to the repository (needs the [shell integration](INSTALL.md#shell-integration-completion-and-gn)) |
| `T` | theme and icons |
| `ctrl+n` | open a second gitnest next to this one ([panes](#several-repositories-side-by-side)) |

The columns are the repository, its current branch, its status — `↑2` commits
to push, `↓3` to pull, `~4` changed files, `∅` no upstream — and how long ago
the last commit was (`5h`, `3d`, `4mo`, `2y`). `✦` marks repositories found
since gitnest started, `●` uncommitted changes, `!` a conflict or an
operation in progress. Folded directories of the tree stay folded the next
time. Clones of the same project from
different hosts are told apart by host (`org-core-dns · forgejo`).

## Inside a repository

Four panels on the left — **Files**, **Branches** (`Local - Remote - Tags`),
**Commits** (`Commits - Reflog`), **Stash** — and the diff on the right.
`1`–`4` or `tab` switch panels, `enter` on the diff scrolls it; only the
active panel highlights its selection. Branches show how they stand against
their upstream: `✓` in sync, `↑1` / `↓2` to push / pull, `✗ gone` when the
upstream was deleted, `∅` without one.

| Panel | Keys |
|---|---|
| Files | `space` stage/unstage, `a` all, `c` commit, `C` commit in `$EDITOR`, `A` amend, `d` discard, `s` / `S` stash all / staged, `e` edit, `` ` `` tree/flat, **`enter` on a file: stage single lines** |
| Line staging | `space` stage the line, `v` range, `a` hunk, `tab` staged side (unstage), `d` discard lines, `esc` back |
| Branches | `space` checkout, `n` new, `r` rename, `d`/`D` delete, `u`/`U` set/unset upstream, `f` fast-forward, `M` merge, `R` rebase onto, `]`/`[` next/previous tab |
| Tags | `n` new at HEAD, `P` push, `d`/`D` delete local/remote, `space` checkout |
| Commits | `s` squash, `f` fixup, `d` drop, `r` reword, `t` revert, `u` undo last commit, `ctrl+j`/`ctrl+k` move, **`i` interactive rebase**, `c` copy / `v` paste (cherry-pick, also into another repository) / `C` forget copied, `]` reflog |
| Interactive rebase | `p` pick, `r` reword, `e` edit, `s` squash, `f` fixup, `d` drop, `ctrl+j`/`ctrl+k` move, `enter` run, `esc` cancel |
| Reflog | `space` check out an entry, `g` reset the branch to it, `[` back to the commits |
| Stash | `space` apply, `g` pop, `b` branch from it, `d` drop |
| Anywhere | `p` pull, `ctrl+p` pull --rebase, `P` push, `F` force push, `f` or `ctrl+f` fetch (`ctrl+f` in Branches, where `f` fast-forwards), `z` undo the last history change, `o` quit and cd here, `enter` scroll the diff, `esc` back to the list, `!` full error, `?` help |

The commit message editor has a **Subject** and an optional **Body**: type the
subject and press `enter` to commit, or `tab` to write a body, then `ctrl+s`.

When a rebase, merge, cherry-pick or revert stops on a conflict, the header
says so: resolve the files, stage them, and press `C` to continue, `S` to skip
or `X` to abort.

Rewriting commits that are already pushed asks twice; afterwards `P` offers a
push with `--force-with-lease` — but only when the remote has nothing you
haven't seen. Every history change records an undo point: `z` goes back,
`z` again redoes.

## Several repositories side by side

Work on related repositories at the same time — say IAM in one, the
resources in another and a shared module in a third — with up to six
gitnests on screen at once in one terminal:

| Key | |
|---|---|
| `ctrl+n` | open another pane next to the current one; pick its repository from the list |
| `ctrl+←` `ctrl+→` `ctrl+↑` `ctrl+↓` | move to the pane in that direction (`alt+arrows` work too) |
| `ctrl+g` | layout: grid (2×2 for four, 3×2 for six…), side by side, stacked |
| `ctrl+w` | close the pane (the last one quits); `q` on a pane's list closes it too |

Each pane is a complete gitnest with its own repository, panels, search and
editor. The active pane has a heavy border in the accent colour and its title
(`● 2 · repo · branch`) on an accent background; the others are thin and
dimmed. When panes would get too small for the window, gitnest falls back to
another layout and finally to tabs above the active pane; from the seventh
pane on it always shows tabs. When the tabs don't fit on one line, the bar
follows the active one and `‹ 3` / `4 ›` count the tabs hidden on each side.
The layout is remembered. Commits copied with `c` in one pane can be pasted with `v` in
another.

> **macOS:** by default `ctrl+←/→/↑/↓` switch Spaces and open Mission
> Control, so the terminal never sees them. Either turn those shortcuts off in
> System Settings → Keyboard → Keyboard Shortcuts → Mission Control, or use
> `alt+arrows` (in iTerm2 set Profiles → Keys → Left/Right Option key to
> "Esc+").

---

[← README](../README.md)
