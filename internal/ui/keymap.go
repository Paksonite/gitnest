package ui

import (
	"strings"
)

// The keymap is the single source of truth for key bindings: key handlers
// look actions up here, and the help screen and footer hints are generated
// from the same table, so they cannot drift apart.

type scope int

const (
	scGlobal   scope = iota // anywhere (outside text input)
	scList                  // repository list
	scFilter                // typing a search query in the list
	scRepo                  // repository view, any panel
	scState                 // repository view while an operation is in progress
	scFiles                 // Files panel
	scBranches              // Branches panel
	scCommits               // Commits panel
	scStash                 // Stash panel
	scMain                  // the focused diff/log panel
	scOverlay               // report / help overlay
	scEditor                // commit message editor
	scPicker                // theme chooser
	scReflog                // Commits panel showing the reflog
	scopes
)

var scopeTitles = [scopes]string{
	scGlobal:   "Everywhere",
	scList:     "Repository list",
	scFilter:   "Searching (/)",
	scRepo:     "Repository view",
	scState:    "Rebase, merge, cherry-pick, revert or bisect in progress",
	scFiles:    "Files panel",
	scBranches: "Branches panel",
	scCommits:  "Commits panel",
	scStash:    "Stash panel",
	scMain:     "Diff panel (after enter)",
	scOverlay:  "Help and reports",
	scEditor:   "Commit message editor",
	scPicker:   "Theme chooser (T)",
	scReflog:   "Reflog (] in the Commits panel)",
}

type action int

const (
	actNone action = iota

	// global
	actHelp
	actThemes
	actErrorDetails
	actQuit

	// movement (shared by several scopes)
	actUp
	actDown
	actPageUp
	actPageDown
	actTop
	actBottom
	actBack

	// list
	actOpen
	actSearch
	actClearSearch
	actDirtyOnly
	actSort
	actMark
	actRescan
	actRefreshList
	actFetchMany
	actPullMany

	// search query
	actOpenMatch
	actKeepFilter
	actCancelFilter
	actDeleteChar
	actClearQuery
	actDeleteWord

	// repository view
	actFocusMain
	actNextPanel
	actPrevPanel
	actPanel1
	actPanel2
	actPanel3
	actPanel4
	actScrollUp
	actScrollDown
	actFetch
	actPull
	actPullRebase
	actPush
	actForcePush
	actReload
	actUndo
	actLazygit

	// operation in progress
	actContinue
	actSkip
	actAbort

	// files
	actToggleStage
	actStageAll
	actCommit
	actCommitEditor
	actAmend
	actDiscard
	actStash
	actStashStaged
	actEdit
	actToggleDir
	actToggleTree

	// branches
	actToggleRemote
	actCheckout
	actNewBranch
	actDeleteBranch
	actForceDeleteBranch
	actRenameBranch
	actSetUpstream
	actUnsetUpstream
	actFastForward
	actMerge
	actRebase

	// commits
	actSquash
	actFixup
	actDrop
	actReword
	actRevert
	actUndoCommit
	actToggleReflog
	actCheckoutEntry
	actResetHere

	// stash
	actApply
	actPop
	actStashBranch
	actStashDrop

	// editor
	actSave
	actEditorEnter
	actNextField
	actPrevField
	actCancel
)

type binding struct {
	act  action
	keys []string // as reported by tea.KeyMsg.String()
	help string   // help screen
	hint string   // footer label; empty keeps it out of the footer
}

func b(act action, help, hint string, keys ...string) binding {
	return binding{act: act, keys: keys, help: help, hint: hint}
}

var keymap = [scopes][]binding{
	scPicker: {
		b(actUp, "previous theme (previewed at once)", "", "up", "k"),
		b(actDown, "next theme (previewed at once)", "", "down", "j"),
		b(actTop, "first theme", "", "home", "g"),
		b(actBottom, "last theme", "", "end", "G"),
		b(actSave, "use and save the theme", "save", "enter"),
		b(actCancel, "keep the previous theme", "cancel", "esc", "q"),
	},
	scGlobal: {
		b(actHelp, "show all keys", "help", "?"),
		b(actThemes, "choose the colour theme (live preview)", "", "T"),
		b(actErrorDetails, "full output of the last failed git command", "", "!"),
		b(actQuit, "quit", "", "ctrl+c"),
	},
	scList: {
		b(actOpen, "open the repository", "open", "enter"),
		b(actSearch, "search by name (also host, path; @text = branch)", "search", "/"),
		b(actClearSearch, "clear the search", "", "esc"),
		b(actUp, "move up", "", "up", "k"),
		b(actDown, "move down", "", "down", "j"),
		b(actPageUp, "page up", "", "pgup", "ctrl+u"),
		b(actPageDown, "page down", "", "pgdown", "ctrl+d"),
		b(actTop, "first repository", "", "home", "g"),
		b(actBottom, "last repository", "", "end", "G"),
		b(actSort, "sort: recent, name, last commit, status", "sort", "s"),
		b(actDirtyOnly, "only repositories with changes", "dirty only", "d"),
		b(actMark, "mark for fetch/pull", "mark", " "),
		b(actFetchMany, "fetch marked (or all) repositories", "fetch", "f"),
		b(actPullMany, "pull --ff-only marked (or selected) repositories", "pull", "p"),
		b(actRescan, "rescan the roots for new and removed repositories", "rescan", "R"),
		b(actRefreshList, "refresh the status of every repository", "refresh", "r"),
		b(actQuit, "quit", "quit", "q"),
	},
	scFilter: {
		b(actOpenMatch, "open the selected repository", "open", "enter"),
		b(actKeepFilter, "keep the filter, back to the list", "keep filter", "tab"),
		b(actCancelFilter, "clear the search", "clear", "esc"),
		b(actUp, "move up", "", "up", "ctrl+p", "ctrl+k"),
		b(actDown, "move down", "", "down", "ctrl+n", "ctrl+j"),
		b(actDeleteChar, "delete a character", "", "backspace"),
		b(actClearQuery, "clear the query", "", "ctrl+u"),
		b(actDeleteWord, "delete a word", "", "ctrl+w"),
	},
	scRepo: {
		b(actFocusMain, "focus the diff to scroll it", "diff", "enter"),
		b(actBack, "back to the list", "back", "esc", "q"),
		b(actNextPanel, "next panel", "", "tab", "right", "l"),
		b(actPrevPanel, "previous panel", "", "shift+tab", "left", "h"),
		b(actPanel1, "Files panel", "", "1"),
		b(actPanel2, "Branches panel", "", "2"),
		b(actPanel3, "Commits panel", "", "3"),
		b(actPanel4, "Stash panel", "", "4"),
		b(actUp, "move up", "", "up", "k"),
		b(actDown, "move down", "", "down", "j"),
		b(actTop, "first item", "", "home"),
		b(actBottom, "last item", "", "end"),
		b(actScrollUp, "scroll the diff up", "", "pgup", "ctrl+u", "K"),
		b(actScrollDown, "scroll the diff down", "", "pgdown", "ctrl+d", "J"),
		b(actFetch, "fetch all remotes", "fetch", "f", "ctrl+f"),
		b(actPull, "pull --ff-only", "pull", "p"),
		b(actPullRebase, "pull --rebase", "", "ctrl+p"),
		b(actPush, "push (sets the upstream if missing)", "push", "P"),
		b(actForcePush, "force push (--force-with-lease)", "", "F"),
		b(actUndo, "undo the last history change (again to redo)", "undo", "z"),
		b(actReload, "reload", "", "ctrl+r"),
		b(actLazygit, "open in lazygit", "", "L"),
	},
	scState: {
		b(actContinue, "continue after resolving conflicts", "continue", "C"),
		b(actSkip, "skip the current commit", "skip", "S"),
		b(actAbort, "abort and restore the previous state", "abort", "X"),
	},
	scFiles: {
		b(actToggleStage, "stage / unstage the file", "stage", " "),
		b(actStageAll, "stage / unstage everything", "all", "a"),
		b(actCommit, "commit (multi-line editor)", "commit", "c"),
		b(actCommitEditor, "commit in $EDITOR", "", "C"),
		b(actAmend, "amend the last commit with staged changes", "amend", "A"),
		b(actDiscard, "discard changes to the file", "discard", "d"),
		b(actStash, "stash all changes", "stash", "s"),
		b(actStashStaged, "stash staged changes only", "", "S"),
		b(actEdit, "edit the file", "edit", "e"),
		b(actToggleDir, "collapse / expand a directory (on files: focus the diff)", "", "enter"),
		b(actToggleTree, "show files as a tree or a flat list", "tree", "`"),
	},
	scBranches: {
		b(actCheckout, "check out (remote: local tracking branch)", "checkout", " "),
		b(actNewBranch, "new branch from the selected one", "new", "n"),
		b(actRenameBranch, "rename", "", "r"),
		b(actDeleteBranch, "delete", "delete", "d"),
		b(actForceDeleteBranch, "force delete", "", "D"),
		b(actSetUpstream, "set upstream", "upstream", "u"),
		b(actUnsetUpstream, "unset upstream", "", "U"),
		b(actFastForward, "fast-forward to upstream without checkout", "fast-forward", "f"),
		b(actMerge, "merge into the current branch", "merge", "M"),
		b(actRebase, "rebase the current branch onto it", "rebase", "R"),
		b(actToggleRemote, "show local / remote branches", "local/remote", "t", "[", "]"),
	},
	scCommits: {
		b(actSquash, "squash into the parent (both messages)", "squash", "s"),
		b(actFixup, "fixup into the parent (parent's message)", "fixup", "f"),
		b(actDrop, "drop the commit", "drop", "d"),
		b(actReword, "reword (full message editor)", "reword", "r"),
		b(actRevert, "revert the commit", "revert", "t"),
		b(actUndoCommit, "undo the last commit, keep changes staged", "undo last", "u"),
		b(actToggleReflog, "show HEAD's reflog (every move, with a graph of what changed)", "reflog", "]", "["),
	},
	scReflog: {
		b(actToggleReflog, "back to the commits", "commits", "[", "]"),
		b(actCheckoutEntry, "check out the entry (detached HEAD)", "checkout", " "),
		b(actResetHere, "reset the current branch to the entry (z undoes)", "reset here", "g"),
	},
	scStash: {
		b(actApply, "apply", "apply", " "),
		b(actPop, "pop", "pop", "g"),
		b(actStashBranch, "new branch from the stash", "branch", "b"),
		b(actStashDrop, "drop", "drop", "d"),
	},
	scMain: {
		b(actBack, "back to the panels", "back", "esc", "q", "enter", "h", "left", "tab"),
		b(actDown, "scroll down a line", "", "j", "down"),
		b(actUp, "scroll up a line", "", "k", "up"),
		b(actPageDown, "half page down", "", "d", "ctrl+d", "pgdown", " ", "J"),
		b(actPageUp, "half page up", "", "u", "ctrl+u", "pgup", "K"),
		b(actTop, "top", "", "g", "home"),
		b(actBottom, "bottom", "", "G", "end"),
	},
	scOverlay: {
		b(actBack, "close", "close", "esc", "q", "enter"),
		b(actDown, "scroll down", "", "j", "down"),
		b(actUp, "scroll up", "", "k", "up"),
		b(actPageDown, "page down", "", "pgdown", "ctrl+d", " "),
		b(actPageUp, "page up", "", "pgup", "ctrl+u"),
	},
	scEditor: {
		b(actEditorEnter, "in the subject: save (the body is optional)", "", "enter"),
		b(actNextField, "move to the body", "", "tab"),
		b(actPrevField, "back to the subject", "", "shift+tab"),
		b(actSave, "save from either field", "save", "ctrl+s", "alt+enter"),
		b(actCancel, "cancel (twice when edited)", "cancel", "esc"),
	},
}

// lookup returns the action bound to key in scope, or actNone.
func lookup(sc scope, key string) action {
	for _, bd := range keymap[sc] {
		for _, k := range bd.keys {
			if k == key {
				return bd.act
			}
		}
	}
	return actNone
}

// keyLabel renders a key the way the help and footer show it.
func keyLabel(k string) string {
	if k == " " {
		return "space"
	}
	return k
}

// hints builds the footer for the given scopes: "key label" pairs for the
// bindings that have a footer label, in table order.
func hints(scs ...scope) string {
	var parts []string
	for _, sc := range scs {
		for _, bd := range keymap[sc] {
			if bd.hint != "" {
				parts = append(parts, keyLabel(bd.keys[0])+" "+bd.hint)
			}
		}
	}
	return strings.Join(parts, "  ")
}

// helpLines renders every scope's bindings for the help overlay.
func helpLines() []string {
	order := []scope{scList, scFilter, scRepo, scFiles, scBranches, scCommits, scStash, scReflog, scState, scMain, scEditor, scPicker, scOverlay, scGlobal}
	var out []string
	for _, sc := range order {
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, bold(scopeTitles[sc]))
		for _, bd := range keymap[sc] {
			labels := make([]string, len(bd.keys))
			for i, k := range bd.keys {
				labels[i] = keyLabel(k)
			}
			out = append(out, "  "+fg(cAccent, padRight(strings.Join(labels, " "), 22))+" "+bd.help)
		}
	}
	return out
}

func padRight(s string, n int) string {
	if w := len([]rune(s)); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s
}
