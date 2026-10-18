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
	scTags                  // Branches panel, Tags tab
	scPlan                  // interactive rebase planner
	scStaging               // line-by-line staging of a file
	scPanes                 // several gitnests side by side
	scPullAll               // choosing what "pull all" updates
	scMenu                  // filters, actions, sort & view
	scBrowser               // every file of the repository (B)
	scDash                  // the dashboard (start screen, H)
	scopes
)

var scopeTitles = [scopes]string{
	scGlobal:   "Everywhere",
	scList:     "Repository list",
	scFilter:   "Searching (/)",
	scRepo:     "Repository view",
	scState:    "Rebase, merge, cherry-pick, revert, am or bisect in progress",
	scFiles:    "Files panel",
	scBranches: "Branches panel",
	scCommits:  "Commits panel",
	scStash:    "Stash panel",
	scMain:     "Diff panel (after enter)",
	scOverlay:  "Help and reports",
	scEditor:   "Commit message editor",
	scPicker:   "Theme chooser (T)",
	scReflog:   "Reflog (] in the Commits panel)",
	scTags:     "Tags (third tab of the Branches panel)",
	scPlan:     "Interactive rebase (i in the Commits panel)",
	scStaging:  "Staging lines (enter on a file)",
	scPanes:    "Panes (several repositories side by side)",
	scPullAll:  "Pull all (ctrl+p in the list)",
	scMenu:     "Menus: view & filters (v), actions (a)",
	scBrowser:  "Files of the repository (B)",
	scDash:     "Dashboard (H in the list; the start screen)",
}

type action int

const (
	actNone action = iota

	// global
	actHelp
	actNewPane
	actClosePane
	actPaneLeft
	actPaneRight
	actPaneUp
	actPaneDown
	actPaneLayout
	actThemes
	actErrorDetails
	actQuit
	actCdQuit

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
	actActions
	actViews
	actDashboard
	actFavorite
	actBrowse
	actMarkAll
	actUnmarkAll
	actMark
	actRescan
	actRefresh
	actFetchMany
	actPullMany
	actPullAll

	// pull all
	actTogglePull
	actToggleAllPulls

	// search query
	actOpenMatch
	actKeepFilter
	actCancelFilter
	actDeleteChar

	// repository view
	actFocusMain
	actNextPanel
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
	actUndo

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
	actRange
	actHunkMode
	actSwitchSide
	actToggleTree
	actToggleIcons

	// branches
	actToggleRemote
	actPrevTab
	actNewTag
	actDeleteTag
	actPushTag
	actDeleteRemoteTag
	actCheckoutTag
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
	actCopy
	actRebasePlan
	actMoveUp
	actMoveDown
	actPick
	actEditStep
	actPaste
	actClearCopied
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
	act   action
	keys  []string // as reported by tea.KeyMsg.String()
	help  string   // help screen
	hint  string   // footer label; empty keeps it out of the footer
	shown int      // keys listed on the help screen: 0 all, n the first n, -1 none
	label string   // replaces the key list on the help screen
}

// show lists only the first n keys on the help screen; the others still
// work (vim-style and ctrl aliases).
func (bd binding) show(n int) binding { bd.shown = n; return bd }

// hidden keeps a binding off the help screen; it still works.
func (bd binding) hidden() binding { bd.shown = -1; return bd }

// labeled shows label instead of the keys on the help screen.
func (bd binding) labeled(label string) binding { bd.label = label; return bd }

func b(act action, help, hint string, keys ...string) binding {
	return binding{act: act, keys: keys, help: help, hint: hint}
}

var keymap = [scopes][]binding{
	scPicker: {
		b(actUp, "previous theme (previewed at once)", "", "up"),
		b(actDown, "next theme (previewed at once)", "", "down"),
		b(actTop, "first theme", "", "home", "g").hidden(),
		b(actBottom, "last theme", "", "end", "G").hidden(),
		b(actSave, "use and save the theme", "save", "enter"),
		b(actToggleIcons, "icons: Unicode symbols, Nerd Font or none (saved)", "icons", "i"),
		b(actCancel, "keep the previous theme", "cancel", "esc", "q").show(1),
	},
	scGlobal: {
		b(actHelp, "show all keys", "help", "?"),
		b(actThemes, "choose the colour theme (live preview)", "", "T"),
		b(actErrorDetails, "full output of the last failed git command", "", "!"),
		b(actQuit, "quit", "", "ctrl+c"),
	},
	scList: {
		b(actOpen, "open the repository", "open", "enter"),
		b(actSearch, "new search by name (also host, path; @text = branch)", "search", "/"),
		b(actClearSearch, "clear the search", "", "esc"),
		b(actUp, "move up", "", "up"),
		b(actDown, "move down", "", "down"),
		b(actPageUp, "page up", "", "pgup", "ctrl+u").show(1),
		b(actPageDown, "page down", "", "pgdown", "ctrl+d").show(1),
		b(actTop, "first repository", "", "home", "g").show(1),
		b(actBottom, "last repository", "", "end", "G").show(1),
		b(actViews, "view: sort order, flat or grouped list, favorites, filters (dirty, stashed, ahead, behind, no remote, by host)", "view", "v"),
		b(actDashboard, "the dashboard: activity, what needs attention", "dashboard", "H"),
		b(actFavorite, "star / unstar the repository (or the marked ones) as a favorite", "", "*"),
		b(actBrowse, "browse every file of the repository, with syntax colours", "", "B"),
		b(actDirtyOnly, "only repositories with changes", "dirty only", "d"),
		b(actMark, "mark for fetch/pull (on a directory: everything below)", "mark", " "),
		b(actMarkAll, "mark every repository shown", "mark all", "A"),
		b(actUnmarkAll, "unmark everything", "unmark all", "U"),
		b(actActions, "actions on the marked repositories (or the selected one): fetch, pull, open each in a pane", "actions", "a"),
		b(actToggleTree, "list: flat, grouped by folder, grouped by remote host; enter folds a group", "tree", "`"),
		b(actFetchMany, "fetch marked (or all) repositories", "fetch", "f"),
		b(actPullMany, "pull --ff-only marked (or selected) repositories", "pull", "p"),
		b(actPullAll, "pull all: fetch everything, choose which branches to update", "pull all", "ctrl+p"),
		b(actRescan, "rescan the roots now and refresh every status (both also happen on their own)", "", "R"),
		b(actCdQuit, "quit and cd into the repository (needs the shell integration)", "", "o"),
		b(actQuit, "quit", "quit", "q"),
	},
	scFilter: {
		b(actOpenMatch, "open the selected repository", "open", "enter"),
		b(actKeepFilter, "keep the filter, back to the list", "keep filter", "tab"),
		b(actCancelFilter, "clear the search", "clear", "esc"),
		b(actUp, "move up", "", "up", "ctrl+p").show(1),
		b(actDown, "move down", "", "down", "ctrl+n").show(1),
		b(actDeleteChar, "delete a character", "", "backspace").hidden(),
	},
	scRepo: {
		b(actFocusMain, "focus the diff to scroll it", "diff", "enter"),
		b(actBack, "back to the list", "back", "esc", "q").show(1),
		b(actNextPanel, "next panel", "", "tab", "right").show(1),
		b(actPanel1, "Files panel", "", "1"),
		b(actPanel2, "Branches panel", "", "2"),
		b(actPanel3, "Commits panel", "", "3"),
		b(actPanel4, "Stash panel", "", "4"),
		b(actUp, "move up", "", "up"),
		b(actDown, "move down", "", "down"),
		b(actScrollUp, "scroll the diff up", "", "ctrl+u", "K").show(1),
		b(actScrollDown, "scroll the diff down", "", "ctrl+d", "J").show(1),
		b(actFetch, "fetch all remotes", "fetch", "f"),
		b(actPull, "pull --ff-only", "pull", "p"),
		b(actPullRebase, "pull --rebase", "", "ctrl+p"),
		b(actPush, "push (sets the upstream if missing; offers a force push when the remote refuses)", "push", "P"),
		b(actUndo, "undo the last history change (again to redo)", "undo", "z"),
		b(actBrowse, "browse every file of the repository, with syntax colours", "files", "B"),
		b(actCdQuit, "quit and cd into this repository (needs the shell integration)", "", "o"),
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
		b(actToggleDir, "on a directory: collapse / expand; on a file: stage it line by line", "lines", "enter"),
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
		b(actFastForward, "fast-forward to upstream without checkout", "fast-forward", "F"),
		b(actMerge, "merge into the current branch", "merge", "M"),
		b(actRebase, "rebase the current branch onto it", "rebase", "R"),
		b(actToggleRemote, "next tab: Local, Remote, Tags", "tabs", "]", "t").show(1),
		b(actPrevTab, "previous tab", "", "["),
	},
	scCommits: {
		b(actSquash, "squash into the parent (both messages)", "squash", "s"),
		b(actFixup, "fixup into the parent (parent's message)", "fixup", "F"),
		b(actDrop, "drop the commit", "drop", "d"),
		b(actReword, "reword (full message editor)", "reword", "r"),
		b(actRevert, "revert the commit", "revert", "t"),
		b(actUndoCommit, "undo the last commit, keep changes staged", "undo last", "u"),
		b(actRebasePlan, "interactive rebase from this commit to HEAD (plan, reorder)", "rebase -i", "i"),
		b(actMoveDown, "move the commit down (older)", "", "ctrl+j"),
		b(actMoveUp, "move the commit up (newer)", "", "ctrl+k"),
		b(actCopy, "copy the commit for cherry-pick (again: uncopy)", "copy", "c"),
		b(actPaste, "paste copied commits onto this branch (also from another repository)", "paste", "v"),
		b(actClearCopied, "forget the copied commits", "", "C"),
		b(actToggleReflog, "show HEAD's reflog (every move, with a graph of what changed)", "reflog", "]", "[").show(1),
	},
	scPanes: {
		b(actNewPane, "open another pane next to this one, starting on the list", "pane", "ctrl+n"),
		b(actClosePane, "close this pane (the last one quits)", "", "ctrl+w"),
		b(actPaneLeft, "move to the pane in that direction (alt+arrows work too)", "", "ctrl+left", "alt+left").labeled("ctrl+←→↑↓"),
		b(actPaneRight, "focus the pane on the right", "", "ctrl+right", "alt+right").hidden(),
		b(actPaneUp, "focus the pane above", "", "ctrl+up", "alt+up").hidden(),
		b(actPaneDown, "focus the pane below", "", "ctrl+down", "alt+down").hidden(),
		b(actPaneLayout, "pane layout: grid, side by side, stacked", "", "ctrl+g"),
	},
	scDash: {
		b(actUp, "move up", "", "up"),
		b(actDown, "move down", "", "down"),
		b(actTop, "first", "", "home", "g").show(1),
		b(actBottom, "last", "", "end", "G").show(1),
		b(actOpen, "open the repository", "open", "enter"),
		b(actSearch, "search the repositories", "search", "/"),
		b(actBack, "the repository list", "list", "esc", "H"),
		b(actRefresh, "read the activity again", "refresh", "r"),
		b(actQuit, "quit", "quit", "q"),
	},
	scBrowser: {
		b(actUp, "move up (or scroll the file)", "", "up"),
		b(actDown, "move down (or scroll the file)", "", "down"),
		b(actOpen, "fold / unfold a folder; on a file, scroll it", "open", "enter"),
		b(actEdit, "edit the file in your editor", "edit", "e"),
		b(actBack, "back (from the file to the tree, then to the repository)", "back", "esc", "q"),
	},
	scMenu: {
		b(actUp, "previous option", "", "up"),
		b(actDown, "next option", "", "down"),
		b(actSave, "choose", "choose", "enter", " "),
		b(actCancel, "close", "close", "esc", "q"),
	},
	scPullAll: {
		b(actUp, "previous branch", "", "up"),
		b(actDown, "next branch", "", "down"),
		b(actTogglePull, "include / leave out the branch", "toggle", " "),
		b(actToggleAllPulls, "include all / none", "all", "a"),
		b(actSave, "pull the selected branches (fast-forward only)", "pull", "enter"),
		b(actCancel, "cancel", "cancel", "esc", "q").show(1),
	},
	scStaging: {
		b(actToggleStage, "stage (or, on the staged side, unstage) the line, range or hunk", "stage", " "),
		b(actDown, "next change line (next hunk in hunk mode)", "", "down"),
		b(actUp, "previous change line", "", "up"),
		b(actPageDown, "half page down", "", "pgdown", "ctrl+d").show(1),
		b(actPageUp, "half page up", "", "pgup", "ctrl+u").show(1),
		b(actRange, "start / end a range selection", "range", "v"),
		b(actHunkMode, "select whole hunks / single lines", "hunk", "a"),
		b(actSwitchSide, "switch between the changes and the staged lines (space there unstages)", "staged side", "tab"),
		b(actDiscard, "discard the selected unstaged lines", "discard", "d"),
		b(actBack, "back to the files", "back", "esc", "q").show(1),
	},
	scPlan: {
		b(actUp, "previous commit", "", "up"),
		b(actDown, "next commit", "", "down"),
		b(actMoveUp, "move up (newer)", "", "ctrl+k", "K").show(1),
		b(actMoveDown, "move down (older)", "move", "ctrl+j", "J").show(1),
		b(actPick, "pick", "pick", "p"),
		b(actReword, "reword (opens the editor)", "reword", "r"),
		b(actEditStep, "edit: stop there to change the commit", "edit", "e"),
		b(actSquash, "squash into the commit below", "squash", "s"),
		b(actFixup, "fixup into the commit below", "fixup", "f"),
		b(actDrop, "drop", "drop", "d"),
		b(actSave, "run the plan", "run", "enter"),
		b(actCancel, "cancel", "cancel", "esc", "q").show(1),
	},
	scTags: {
		b(actNewTag, "new tag at HEAD (with a message: annotated)", "new", "n"),
		b(actCheckoutTag, "check out the tag (detached HEAD)", "checkout", " "),
		b(actPushTag, "push the tag", "push", "P"),
		b(actDeleteTag, "delete the local tag", "delete", "d"),
		b(actDeleteRemoteTag, "delete the tag on the remote", "", "D"),
	},
	scReflog: {
		b(actToggleReflog, "back to the commits", "commits", "[", "]").show(1),
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
		b(actBack, "back to the panels", "back", "esc", "q", "enter", "left", "tab").show(1),
		b(actDown, "scroll down", "", "down"),
		b(actUp, "scroll up", "", "up"),
		b(actPageDown, "half page down", "", "pgdown", "d", "ctrl+d", " ", "J").hidden(),
		b(actPageUp, "half page up", "", "pgup", "u", "ctrl+u", "K").hidden(),
		b(actTop, "top", "", "home", "g").show(1),
		b(actBottom, "bottom", "", "end", "G").show(1),
	},
	scOverlay: {
		b(actBack, "close", "close", "esc", "q", "enter").show(1),
		b(actDown, "scroll", "", "down").labeled("↑ ↓"),
		b(actUp, "scroll up", "", "up").hidden(),
		b(actPageDown, "page down", "", "pgdown", "ctrl+d", " ").hidden(),
		b(actPageUp, "page up", "", "pgup", "ctrl+u").hidden(),
	},
	scEditor: {
		b(actEditorEnter, "in the subject: save (the body is optional)", "", "enter"),
		b(actNextField, "move to the body", "", "tab"),
		b(actPrevField, "back to the subject", "", "shift+tab"),
		b(actSave, "save from either field", "save", "ctrl+s", "alt+enter").show(1),
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
	switch k {
	case " ":
		return "space"
	case "up":
		return "↑"
	case "down":
		return "↓"
	case "left":
		return "←"
	case "right":
		return "→"
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
				parts = append(parts, keyHint(bd.keys[0], bd.hint))
			}
		}
	}
	return strings.Join(parts, "  ")
}

// keyHint is one footer hint: the key in the accent colour, then what it does.
func keyHint(key, hint string) string {
	return fg(cAccent, keyLabel(key)) + " " + fg(cDim, hint)
}

// otherHints is hints for scs without the keys the panel takes over (f
// fast-forwards in Branches, so "f fetch" isn't offered there).
func otherHints(panel scope, scs ...scope) string {
	var parts []string
	for _, sc := range scs {
		for _, bd := range keymap[sc] {
			if bd.hint != "" && lookup(panel, bd.keys[0]) == actNone {
				parts = append(parts, keyHint(bd.keys[0], bd.hint))
			}
		}
	}
	return strings.Join(parts, "  ")
}

// helpLines renders every scope's bindings for the help overlay.
func helpLines() []string {
	order := []scope{scPanes, scDash, scList, scFilter, scMenu, scPullAll, scRepo, scBrowser, scFiles, scBranches, scCommits, scStash, scStaging, scPlan, scTags, scReflog, scState, scMain, scEditor, scPicker, scOverlay, scGlobal}
	var out []string
	for _, sc := range order {
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, bold(scopeTitles[sc]))
		for _, bd := range keymap[sc] {
			if bd.shown < 0 {
				continue
			}
			keys := bd.keys
			if bd.shown > 0 {
				keys = keys[:bd.shown]
			}
			labels := make([]string, len(keys))
			for i, k := range keys {
				labels[i] = keyLabel(k)
			}
			label := strings.Join(labels, " ")
			if bd.label != "" {
				label = bd.label
			}
			out = append(out, "  "+fg(cAccent, padRight(label, 14))+" "+bd.help)
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
