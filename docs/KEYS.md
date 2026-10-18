# Keys

Press `?` in the app for the same list. Panes are described in
[Several repositories side by side](#several-repositories-side-by-side).

## The dashboard

gitnest opens with the dashboard (`start: list` in the config opens the list
instead; `H` switches between them):

- **Activity**: commits per day across every repository for the last 26
  weeks, darker for busier days, with the months above.
- **Overview**: how many repositories are dirty, have commits to push or
  pull, have no upstream or are in the middle of an operation; commits this
  week and last, the busiest day, and your streak of days with commits.
- **Needs attention**: operations in progress and conflicts first, then
  uncommitted changes, commits to pull, commits to push.
- **Most active**: repositories by commits in the last 12 weeks, each with a
  sparkline of commits per week.
- **Recent commits** across every repository, with author initials.

| Key | |
|---|---|
| `↑` `↓` | move between the repositories listed |
| `enter` | open the repository (`esc` comes back to the dashboard) |
| `/` | search the repositories |
| `esc` / `H` | the repository list |
| `r` | read the activity again |

## The mouse

The wheel scrolls what is under the pointer (the diff, a file, a list); a
click selects a row and a click on the selected row opens it (or folds a
group); in a repository a click focuses the panel, and with panes a click
focuses the pane. While the mouse is on, select text in the terminal with
`shift` (`option` in macOS Terminal and iTerm2) held down; `mouse: false` in
the config turns it off.

## The repository list

| Key | |
|---|---|
| `/` | search by name (also git host and folder; `@text` searches branches); `enter` opens, `tab` keeps the filter, `esc` clears |
| `enter` | open the repository |
| `` ` `` | list view: flat, grouped by folder, grouped by remote host and owner (github.com/ann/; local/ without a remote); `enter` folds a group |
| `s` | sort: recently used, name, last commit, needs attention |
| `H` | the dashboard: activity, what needs attention, the most active repositories |
| `v` | sort & view: the sort order, a flat or grouped list (by folder or remote), all repositories or favorites only |
| `B` | open the repository and browse its files (tree and syntax-coloured view) |
| `*` | star the repository (or the marked ones) as a favorite, shown with a star; again to unstar |
| `d` | only repositories with uncommitted changes |
| `F` | filters, with how many repositories each keeps: dirty, stashed changes, ahead of or behind the remote, no remote; and by remote host |
| `space` | mark (on a folder: everything below); `f` fetch / `p` pull the marked ones |
| `a` | actions on the marked repositories (or the selected one): fetch, pull, open each in a pane of its own, mark all / unmark all |
| `A` / `U` | mark every repository shown / unmark everything |
| `ctrl+p` | pull all: fetches every repository, lists every branch that is behind (also main when it isn't checked out); pick with `space` / `a`, `enter` updates them (fast-forward only) |
| `R` / `r` | rescan the folders / refresh the status |
| `q` | quit |
| `o` | quit and go to the repository (needs the [shell integration](INSTALL.md#shell-integration-completion-and-gn)) |
| `T` | theme and icons |
| `ctrl+n` | open a second gitnest next to this one ([panes](#several-repositories-side-by-side)) |

The columns are the repository, its current branch, its status and how long
ago the last commit was (`5h`, `3d`, `4mo`, `2y`). The status column is the one
place a repository's state is shown: `✓` clean and in sync with its upstream,
`~4` changed files, `↑2` / `↓3` commits to push / pull, `∅` no upstream, `!`
a conflict or an operation in progress (`! rebasing`). A group of the tree
shows how many repositories it holds after its name and, in the status
column, how many are dirty and `✓` when all are up to date (or how many have
commits to push `↑` and pull `↓`). `✓` in the first column marks a repository
(`space`); after the name, `★` is a favorite and `✦` a repository found since
gitnest started. Folded directories of the tree stay folded the next time. Clones of the same project from
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
| Files browser | `B`: every file of the repository in a tree, the selected one with line numbers and syntax colours; `enter` folds a folder or scrolls the file, `tab` tree / file, `e` edit, `esc` back |
| Anywhere | `B` browse the files, `p` pull, `ctrl+p` pull --rebase, `P` push, `F` force push, `f` or `ctrl+f` fetch (`ctrl+f` in Branches, where `f` fast-forwards), `z` undo the last history change, `o` quit and cd here, `enter` scroll the diff, `esc` back to the list, `!` full error, `?` help |

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
