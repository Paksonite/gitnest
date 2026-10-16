package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/gitx"
)

const (
	pFiles = iota
	pBranches
	pCommits
	pStash
)

type repoData struct {
	info     gitx.Info
	files    []gitx.File
	local    []gitx.Branch
	remote   []gitx.Branch
	commits  []gitx.Commit
	unpushed map[string]bool
	stashes  []gitx.Stash
}

type repoState struct {
	dir, name  string
	focus      int
	info       gitx.Info
	files      []gitx.File
	local      []gitx.Branch
	remote     []gitx.Branch
	showRemote bool
	commits    []gitx.Commit
	unpushed   map[string]bool
	stashes    []gitx.Stash
	cur, off   [4]int
	main       string
	mainKey    string
	mainOff    int
	mainFocus  bool            // keys scroll the main (diff) panel
	fileRows   []fileNode      // Files panel rows (tree or flat)
	flat       bool            // Files panel as a flat list instead of a tree
	collapsed  map[string]bool // collapsed directories in the tree
	refreshing bool            // a live refresh is running
	mainH      int             // rows of the main panel on screen (set while rendering)
	mainLen    int             // lines in the main panel
}

func (rp *repoState) branches() []gitx.Branch {
	if rp.showRemote {
		return rp.remote
	}
	return rp.local
}

func (rp *repoState) count(panel int) int {
	switch panel {
	case pFiles:
		return len(rp.fileRows)
	case pBranches:
		return len(rp.branches())
	case pCommits:
		return len(rp.commits)
	}
	return len(rp.stashes)
}

// setFiles replaces the changed files and rebuilds the Files rows, keeping
// the cursor on the same file or directory.
func (rp *repoState) setFiles(files []gitx.File) {
	keep := ""
	if n, ok := rp.fileNode(); ok {
		keep = n.key()
	}
	rp.files = files
	rp.rebuildFiles(keep)
}

func (rp *repoState) rebuildFiles(keep string) {
	if rp.collapsed == nil {
		rp.collapsed = map[string]bool{}
	}
	rp.fileRows = buildFileRows(rp.files, rp.flat, rp.collapsed)
	for i, n := range rp.fileRows {
		if n.key() == keep {
			rp.cur[pFiles] = i
			return
		}
	}
	rp.cur[pFiles] = clamp(rp.cur[pFiles], 0, len(rp.fileRows)-1)
}

// fileNode is the selected row of the Files panel.
func (rp *repoState) fileNode() (fileNode, bool) {
	i := rp.cur[pFiles]
	if i < 0 || i >= len(rp.fileRows) {
		return fileNode{}, false
	}
	return rp.fileRows[i], true
}

func loadRepoData(ctx context.Context, dir string) tea.Cmd {
	return func() tea.Msg {
		var d repoData
		var wg sync.WaitGroup
		par := func(f func()) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				f()
			}()
		}
		par(func() { d.info, _ = gitx.GetInfo(ctx, dir) })
		par(func() { d.files, _ = gitx.Files(ctx, dir) })
		par(func() { d.local, d.remote, _ = gitx.Branches(ctx, dir) })
		par(func() { d.commits, _ = gitx.Log(ctx, dir, 300) })
		par(func() { d.unpushed = gitx.Unpushed(ctx, dir) })
		par(func() { d.stashes, _ = gitx.Stashes(ctx, dir) })
		wg.Wait()
		return repoDataMsg{dir: dir, d: d}
	}
}

func (a *App) onRepoData(m repoDataMsg) tea.Cmd {
	if m.dir != a.rp.dir || a.scr != scrRepo {
		return nil
	}
	rp := &a.rp
	d := m.d
	rp.info, rp.local, rp.remote = d.info, d.local, d.remote
	rp.setFiles(d.files)
	rp.commits, rp.unpushed, rp.stashes = d.commits, d.unpushed, d.stashes
	for i := 0; i < 4; i++ {
		rp.cur[i] = clamp(rp.cur[i], 0, rp.count(i)-1)
	}
	return a.mainCmd()
}

// mainCmd (re)computes the right-hand pane for the selected item, if it changed.
func (a *App) mainCmd() tea.Cmd {
	rp := &a.rp
	key, fn := a.mainSource()
	if key == rp.mainKey {
		return nil
	}
	rp.mainKey, rp.main, rp.mainOff = key, "", 0
	return func() tea.Msg { return mainMsg{key: key, text: fn()} }
}

// mainSource says what the main panel shows for the current selection: a
// key identifying it and a function producing the text.
func (a *App) mainSource() (string, func() string) {
	rp := &a.rp
	dir, ctx := rp.dir, a.ctx
	i := rp.cur[rp.focus]
	switch rp.focus {
	case pFiles:
		n, ok := rp.fileNode()
		switch {
		case !ok:
			return "files:none", func() string { return "Working tree clean ✓" }
		case n.Dir:
			var untracked []string
			for _, f := range n.Files {
				if f.Untracked() {
					untracked = append(untracked, f.Path)
				}
			}
			return fmt.Sprintf("d:%s:%d", n.Path, len(n.Files)), func() string { return gitx.DiffDir(ctx, dir, n.Path, untracked) }
		default:
			f := n.File
			return "f:" + f.Path + ":" + string([]byte{f.X, f.Y}), func() string { return gitx.DiffFile(ctx, dir, f) }
		}
	case pBranches:
		bl := rp.branches()
		if len(bl) == 0 {
			return "branches:none", func() string { return "(no branches)" }
		}
		b := bl[i]
		return "b:" + b.Name, func() string { return gitx.BranchLog(ctx, dir, b.Name) }
	case pCommits:
		if len(rp.commits) == 0 {
			return "commits:none", func() string { return "(no commits)" }
		}
		c := rp.commits[i]
		return "c:" + c.Hash, func() string { return gitx.Show(ctx, dir, c.Hash) }
	}
	if len(rp.stashes) == 0 {
		return "stash:none", func() string { return "(no stashes)" }
	}
	st := rp.stashes[i]
	return "s:" + st.Ref + ":" + st.Msg, func() string { return gitx.StashShow(ctx, dir, st.Ref) }
}

func (a *App) move(delta int) tea.Cmd {
	rp := &a.rp
	rp.cur[rp.focus] = clamp(rp.cur[rp.focus]+delta, 0, rp.count(rp.focus)-1)
	return a.mainCmd()
}

func (a *App) editor() string {
	for _, e := range []string{a.cfg.Editor, os.Getenv("VISUAL"), os.Getenv("EDITOR")} {
		if e != "" {
			return e
		}
	}
	return "vi"
}

func (a *App) execCmd(c *exec.Cmd) tea.Cmd {
	return tea.ExecProcess(c, func(err error) tea.Msg { return execDoneMsg{err: err} })
}

// ---------------------------------------------------------------------- keys

func (a *App) repoKey(k tea.KeyMsg) tea.Cmd {
	rp := &a.rp
	dir, ctx := rp.dir, a.ctx
	s := k.String()
	half := clamp((a.h-3)/2, 3, 100)

	if rp.mainFocus {
		return a.mainKey(s)
	}
	act := lookup(scRepo, s)
	if rp.focus == pFiles && lookup(scFiles, s) == actToggleDir {
		if n, ok := rp.fileNode(); ok && n.Dir {
			rp.collapsed[n.Path] = !rp.collapsed[n.Path]
			rp.rebuildFiles(n.key())
			return nil
		}
	}

	// navigation
	switch act {
	case actFocusMain:
		rp.mainFocus = true
		return nil
	case actBack:
		a.scr = scrRepos
		return a.gather(a.paths())
	case actNextPanel:
		rp.focus = (rp.focus + 1) % 4
		return a.mainCmd()
	case actPrevPanel:
		rp.focus = (rp.focus + 3) % 4
		return a.mainCmd()
	case actPanel1, actPanel2, actPanel3, actPanel4:
		rp.focus = int(act - actPanel1)
		return a.mainCmd()
	case actUp:
		return a.move(-1)
	case actDown:
		return a.move(1)
	case actTop:
		return a.move(-rp.count(rp.focus))
	case actBottom:
		return a.move(rp.count(rp.focus))
	case actScrollUp:
		a.scrollMain(-half)
		return nil
	case actScrollDown:
		a.scrollMain(half)
		return nil
	}

	// rebase/merge/cherry-pick/revert/bisect in progress
	if st := rp.info.State; st != gitx.StateNone {
		name := strings.ToLower(st.String())
		switch sa := lookup(scState, s); {
		case sa == actContinue && st != gitx.StateBisecting:
			return a.do("continue "+name, func() error { return gitx.Continue(ctx, dir, st) })
		case sa == actAbort:
			a.confirm("Abort "+name+"?", func() tea.Cmd {
				return a.do("abort "+name, func() error { return gitx.Abort(ctx, dir, st) })
			})
			return nil
		case sa == actSkip && st.CanSkip():
			return a.do("skip", func() error { return gitx.Skip(ctx, dir, st) })
		}
	}

	// panel-specific keys take precedence over global ones
	var cmd tea.Cmd
	handled := false
	switch rp.focus {
	case pFiles:
		cmd, handled = a.filesKey(s)
	case pBranches:
		cmd, handled = a.branchesKey(s)
	case pCommits:
		cmd, handled = a.commitsKey(s)
	default:
		cmd, handled = a.stashKey(s)
	}
	if handled {
		return cmd
	}

	// global
	switch act {
	case actFetch:
		return a.do("fetch", func() error { return gitx.Fetch(ctx, dir) })
	case actPull, actPullRebase:
		if !rp.info.Upstream {
			a.setToast("no upstream for "+rp.info.Branch+" — set one in Branches with u", true)
			return nil
		}
		if act == actPullRebase {
			return a.do("pull --rebase", func() error { return gitx.Pull(ctx, dir, true) })
		}
		return a.do("pull", func() error { return gitx.Pull(ctx, dir, false) })
	case actPush:
		return a.push(false)
	case actForcePush:
		return a.push(true)
	case actReload:
		rp.mainKey = ""
		return loadRepoData(ctx, dir)
	case actUndo:
		return a.undoLast()
	case actLazygit:
		if _, err := exec.LookPath("lazygit"); err != nil {
			a.setToast("lazygit not found in PATH", true)
			return nil
		}
		c := exec.Command("lazygit", "-p", dir)
		return a.execCmd(c)
	}
	return nil
}

func (a *App) filesKey(s string) (tea.Cmd, bool) {
	rp := &a.rp
	dir, ctx := rp.dir, a.ctx
	has := len(rp.files) > 0
	n, onRow := rp.fileNode()
	f := n.File
	switch lookup(scFiles, s) {
	case actToggleStage:
		if !onRow {
			return nil, true
		}
		unstaged := f.Unstaged()
		if n.Dir {
			_, unstaged = dirStage(n.Files)
		}
		if unstaged {
			return a.do("stage", func() error { return gitx.Stage(ctx, dir, n.Path) }), true
		}
		return a.do("unstage", func() error { return gitx.Unstage(ctx, dir, n.Path) }), true
	case actToggleTree:
		rp.flat = !rp.flat
		rp.rebuildFiles(n.key())
		return a.mainCmd(), true
	case actStageAll:
		anyUnstaged := false
		for _, x := range rp.files {
			if x.Unstaged() {
				anyUnstaged = true
				break
			}
		}
		if anyUnstaged {
			return a.do("stage all", func() error { return gitx.StageAll(ctx, dir) }), true
		}
		return a.do("unstage all", func() error { return gitx.UnstageAll(ctx, dir) }), true
	case actCommit:
		staged := false
		for _, x := range rp.files {
			if x.Staged() {
				staged = true
				break
			}
		}
		if staged {
			return a.openEditor("Commit", "", func(msg string) tea.Cmd {
				return a.do("commit", func() error { return gitx.CommitMsg(ctx, dir, msg) })
			}), true
		}
		if !has {
			a.setToast("nothing to commit", false)
			return nil, true
		}
		a.confirm("Nothing is staged. Stage all changes and commit?", func() tea.Cmd {
			return a.openEditor("Commit all changes", "", func(msg string) tea.Cmd {
				return a.do("commit", func() error {
					if err := gitx.StageAll(ctx, dir); err != nil {
						return err
					}
					return gitx.CommitMsg(ctx, dir, msg)
				})
			})
		})
		return nil, true
	case actStashStaged:
		a.ask("Stash staged changes — message", "", func(msg string) tea.Cmd {
			return a.do("stash staged", func() error { return gitx.StashStaged(ctx, dir, msg) })
		})
		return nil, true
	case actCommitEditor:
		c := exec.Command("git", "commit")
		c.Dir = dir
		return a.execCmd(c), true
	case actAmend:
		return a.check(func() ([]string, error) { return gitx.CheckRewrite(ctx, dir, "HEAD", gitx.ActionAmend) },
			func(pushed []string) tea.Cmd {
				return a.confirmRewrite("Amend last commit with the staged changes?", pushed, func() tea.Cmd {
					return a.doUndoable("amend", func() error { return gitx.AmendNoEdit(ctx, dir) })
				})
			}), true
	case actDiscard:
		switch {
		case !onRow:
		case n.Dir:
			files := n.Files
			a.confirm(fmt.Sprintf("Discard all changes in %s/ (%d files)?", n.Path, len(files)), func() tea.Cmd {
				return a.do("discard", func() error {
					for _, f := range files {
						if err := gitx.Discard(ctx, dir, f); err != nil {
							return err
						}
					}
					return nil
				})
			})
		default:
			a.confirm("Discard all changes to "+f.Path+"?", func() tea.Cmd {
				return a.do("discard", func() error { return gitx.Discard(ctx, dir, f) })
			})
		}
		return nil, true
	case actStash:
		a.ask("Stash message", "wip", func(msg string) tea.Cmd {
			return a.do("stash", func() error { return gitx.StashPush(ctx, dir, msg) })
		})
		return nil, true
	case actEdit:
		if !onRow || n.Dir {
			return nil, true
		}
		c := exec.Command("sh", "-c", a.editor()+` "$@"`, "sh", f.Path)
		c.Dir = dir
		return a.execCmd(c), true
	}
	return nil, false
}

func (a *App) branchesKey(s string) (tea.Cmd, bool) {
	rp := &a.rp
	dir, ctx := rp.dir, a.ctx
	bl := rp.branches()
	has := len(bl) > 0
	var b gitx.Branch
	if has {
		b = bl[rp.cur[pBranches]]
	}
	switch lookup(scBranches, s) {
	case actToggleRemote:
		rp.showRemote = !rp.showRemote
		rp.cur[pBranches], rp.off[pBranches] = 0, 0
		return a.mainCmd(), true
	case actCheckout:
		if !has {
			return nil, true
		}
		return a.do("checkout "+b.Name, func() error { return gitx.Checkout(ctx, dir, b) }), true
	case actNewBranch:
		base := ""
		if has {
			base = b.Name
		}
		a.ask("New branch name (from "+base+")", "", func(name string) tea.Cmd {
			return a.do("new branch", func() error { return gitx.NewBranch(ctx, dir, name, base) })
		})
		return nil, true
	case actDeleteBranch, actForceDeleteBranch:
		if !has {
			return nil, true
		}
		if b.Current {
			a.setToast("can't delete the checked-out branch", true)
			return nil, true
		}
		force := lookup(scBranches, s) == actForceDeleteBranch
		a.confirm("Delete branch "+b.Name+"?", func() tea.Cmd {
			if b.Remote {
				return a.do("delete remote branch", func() error { return gitx.DeleteRemoteBranch(ctx, dir, b.Name) })
			}
			return a.do("delete branch", func() error { return gitx.DeleteBranch(ctx, dir, b.Name, force) })
		})
		return nil, true
	case actRenameBranch:
		if !has || b.Remote {
			return nil, true
		}
		a.ask("Rename "+b.Name+" to", b.Name, func(name string) tea.Cmd {
			return a.do("rename", func() error { return gitx.RenameBranch(ctx, dir, b.Name, name) })
		})
		return nil, true
	case actSetUpstream:
		if !has || b.Remote {
			return nil, true
		}
		def := b.Upstream
		if def == "" {
			def = "origin/" + b.Name
		}
		a.ask("Upstream for "+b.Name, def, func(up string) tea.Cmd {
			return a.do("set upstream", func() error { return gitx.SetUpstream(ctx, dir, b.Name, up) })
		})
		return nil, true
	case actUnsetUpstream:
		if !has || b.Remote || b.Upstream == "" {
			return nil, true
		}
		a.confirm("Stop tracking "+b.Upstream+" for "+b.Name+"?", func() tea.Cmd {
			return a.do("unset upstream", func() error { return gitx.UnsetUpstream(ctx, dir, b.Name) })
		})
		return nil, true
	case actFastForward:
		if !has || b.Remote {
			return nil, true
		}
		return a.do("fast-forward "+b.Name, func() error { return gitx.FastForward(ctx, dir, b) }), true
	case actMerge:
		if !has || b.Current {
			return nil, true
		}
		a.confirm("Merge "+b.Name+" into the current branch?", func() tea.Cmd {
			return a.doUndoable("merge "+b.Name, func() error { return gitx.Merge(ctx, dir, b.Name) })
		})
		return nil, true
	case actRebase:
		if !has || b.Current {
			return nil, true
		}
		return a.check(func() ([]string, error) { return gitx.CheckRebaseOnto(ctx, dir, b.Name) },
			func(pushed []string) tea.Cmd {
				return a.confirmRewrite("Rebase the current branch onto "+b.Name+"?", pushed, func() tea.Cmd {
					return a.doUndoable("rebase onto "+b.Name, func() error { return gitx.RebaseOnto(ctx, dir, b.Name) })
				})
			}), true
	}
	return nil, false
}

func (a *App) commitsKey(s string) (tea.Cmd, bool) {
	rp := &a.rp
	dir, ctx := rp.dir, a.ctx
	if len(rp.commits) == 0 {
		return nil, false
	}
	idx := rp.cur[pCommits]
	c := rp.commits[idx]
	// rewrite checks the commit first (merges, operation in progress, pushed)
	// and only then asks; then runs op.
	rewrite := func(action, question string, op func() tea.Cmd) tea.Cmd {
		return a.check(func() ([]string, error) { return gitx.CheckRewrite(ctx, dir, c.Hash, action) },
			func(pushed []string) tea.Cmd { return a.confirmRewrite(question, pushed, op) })
	}
	meld := func(action string) tea.Cmd {
		q := strings.ToUpper(action[:1]) + action[1:] + " commit " + c.Short + " (" + c.Subject + ")?"
		return rewrite(action, q, func() tea.Cmd {
			return a.doUndoable(action+" "+c.Short, func() error { return gitx.RebaseAction(ctx, dir, c.Hash, action, "") })
		})
	}
	switch lookup(scCommits, s) {
	case actSquash:
		return meld(gitx.ActionSquash), true
	case actFixup:
		return meld(gitx.ActionFixup), true
	case actDrop:
		return meld(gitx.ActionDrop), true
	case actReword:
		var current string // filled in by the check, read after it completes
		return a.check(func() ([]string, error) {
			pushed, err := gitx.CheckRewrite(ctx, dir, c.Hash, gitx.ActionReword)
			if err == nil {
				current, err = gitx.Message(ctx, dir, c.Hash)
			}
			return pushed, err
		}, func(pushed []string) tea.Cmd {
			return a.openEditor("Reword "+c.Short, gitx.CleanMessage(current), func(msg string) tea.Cmd {
				return a.confirmRewrite("", pushed, func() tea.Cmd {
					return a.doUndoable("reword "+c.Short, func() error { return gitx.RewordMessage(ctx, dir, c.Hash, msg) })
				})
			})
		}), true
	case actRevert:
		a.confirm("Revert commit "+c.Short+"?", func() tea.Cmd {
			return a.doUndoable("revert "+c.Short, func() error { return gitx.Revert(ctx, dir, c.Hash) })
		})
		return nil, true
	case actUndoCommit:
		return a.check(func() ([]string, error) { return gitx.CheckRewrite(ctx, dir, "HEAD", gitx.ActionAmend) },
			func(pushed []string) tea.Cmd {
				return a.confirmRewrite("Undo the last commit (soft reset, changes stay staged)?", pushed, func() tea.Cmd {
					return a.doUndoable("undo commit", func() error { return gitx.UndoCommit(ctx, dir) })
				})
			}), true
	}
	return nil, false
}

func (a *App) stashKey(s string) (tea.Cmd, bool) {
	rp := &a.rp
	dir, ctx := rp.dir, a.ctx
	if len(rp.stashes) == 0 {
		return nil, false
	}
	st := rp.stashes[rp.cur[pStash]]
	switch lookup(scStash, s) {
	case actApply:
		return a.do("stash apply", func() error { return gitx.StashApply(ctx, dir, st.Ref) }), true
	case actPop:
		return a.do("stash pop", func() error { return gitx.StashPop(ctx, dir, st.Ref) }), true
	case actStashBranch:
		a.ask("New branch from "+st.Ref, "", func(name string) tea.Cmd {
			return a.do("stash branch", func() error { return gitx.StashBranch(ctx, dir, name, st.Ref) })
		})
		return nil, true
	case actStashDrop:
		a.confirm("Drop "+st.Ref+" ("+st.Msg+")?", func() tea.Cmd {
			return a.do("stash drop", func() error { return gitx.StashDrop(ctx, dir, st.Ref) })
		})
		return nil, true
	}
	return nil, false
}

// ---------------------------------------------------------------------- view

func (a *App) panelItems(panel int) []rowItem {
	rp := &a.rp
	var items []rowItem
	switch panel {
	case pFiles:
		for _, n := range rp.fileRows {
			items = append(items, fileRow(n, rp.collapsed[n.Path]))
		}
	case pBranches:
		for _, b := range rp.branches() {
			mark := "  "
			name := b.Name
			if b.Current {
				mark = "* "
				name = fg(cGreen, bold(b.Name))
			}
			items = append(items, rowItem{
				plain:  mark + b.Name + " " + b.Track,
				styled: mark + name + " " + fg(cDim, b.Track),
			})
		}
	case pCommits:
		for _, c := range rp.commits {
			hc := cYellow
			if rp.unpushed[c.Hash] {
				hc = cRed
			}
			refs := ""
			if c.Refs != "" {
				refs = " (" + c.Refs + ")"
			}
			items = append(items, rowItem{
				plain:  c.Short + " " + c.Subject + refs,
				styled: fg(hc, c.Short) + " " + c.Subject + fg(cCyan, refs),
			})
		}
	default:
		for _, s := range rp.stashes {
			items = append(items, rowItem{
				plain:  s.Ref + " " + s.Msg,
				styled: fg(cYellow, s.Ref) + " " + s.Msg,
			})
		}
	}
	return items
}

func (a *App) repoHints() string {
	rp := &a.rp
	if rp.mainFocus {
		return hints(scMain)
	}
	if st := rp.info.State; st != gitx.StateNone {
		return stateKeys(st) + "  ·  resolve conflicts in Files (e edit, space stage)  ·  " + hints(scGlobal)
	}
	panel := [4]scope{scFiles, scBranches, scCommits, scStash}[rp.focus]
	return hints(panel) + "  ·  " + hints(scRepo, scGlobal)
}

func (a *App) viewRepo() string {
	rp := &a.rp
	bodyH := a.h - 3

	head := lipgloss.NewStyle().Bold(true).Foreground(cAccent).Render("gitnest") + fg(cDim, " › ") +
		bold(rp.name) + "  " + fg(cCyan, rp.info.Branch) + " " + syncStyled(rp.info)
	if st := rp.info.State; st != gitx.StateNone {
		head += "  " + lipgloss.NewStyle().Bold(true).Foreground(cYellow).Render(st.String()+" — "+stateKeys(st))
	}

	leftW := clamp(a.w*2/5, 34, 64)
	if a.w < 80 {
		leftW = a.w / 2
	}
	mainW := a.w - leftW

	branchTitle := "[2] Branches (local)"
	if rp.showRemote {
		branchTitle = "[2] Branches (remote)"
	}
	titles := [4]string{
		fmt.Sprintf("[1] Files (%d)", len(rp.files)),
		branchTitle,
		fmt.Sprintf("[3] Commits (%d)", len(rp.commits)),
		fmt.Sprintf("[4] Stash (%d)", len(rp.stashes)),
	}
	heights := panelHeights(bodyH, rp.focus, [4]int{rp.count(0), rp.count(1), rp.count(2), rp.count(3)})
	panels := make([]string, 4)
	for i := 0; i < 4; i++ {
		lh := heights[i] - 3 // borders and the title row
		var lines []string
		if lh > 0 {
			rp.off[i] = scrollTo(rp.cur[i], rp.off[i], lh)
			lines = renderRows(a.panelItems(i), rp.cur[i], rp.off[i], lh, leftW-2, i == rp.focus)
		}
		if heights[i] > 0 {
			panels[i] = panel(titles[i], lines, leftW, heights[i], i == rp.focus)
		}
	}
	var shown []string
	for _, p := range panels {
		if p != "" {
			shown = append(shown, p)
		}
	}
	left := lipgloss.JoinVertical(lipgloss.Left, shown...)

	mainTitle := [4]string{"Diff", "Log", "Commit", "Stash"}[rp.focus]
	mainLines := strings.Split(strings.TrimRight(expandTabs(rp.main), "\n"), "\n")
	mh := lipgloss.Height(left)
	rp.mainH = mh - 3
	rp.mainLen = len(mainLines)
	rp.mainOff = clamp(rp.mainOff, 0, rp.mainLen-rp.mainH)
	if rp.mainLen > rp.mainH {
		mainTitle += fmt.Sprintf(" · %d–%d of %d", rp.mainOff+1, min(rp.mainOff+rp.mainH, rp.mainLen), rp.mainLen)
	}
	if rp.mainFocus {
		mainTitle += "  (j/k scroll · esc back)"
	}
	mainLines = mainLines[rp.mainOff:]
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, panel(mainTitle, mainLines, mainW, mh, rp.mainFocus))

	return strings.Join([]string{fit(head, a.w), body, a.footer(a.repoHints())}, "\n")
}

// stateKeys lists the keys that act on an operation in progress.
// stateKeys lists the keys that act on an operation in progress.
func stateKeys(st gitx.State) string {
	var k []string
	for _, bd := range keymap[scState] {
		switch {
		case bd.act == actContinue && st == gitx.StateBisecting:
		case bd.act == actSkip && !st.CanSkip():
		default:
			k = append(k, keyLabel(bd.keys[0])+" "+bd.hint)
		}
	}
	return strings.Join(k, " · ")
}

// doUndoable runs a history-changing operation and reminds that z undoes it.
func (a *App) doUndoable(label string, fn func() error) tea.Cmd {
	return a.doMsg(label, func() (string, error) { return label + " ✓  ·  z to undo", fn() })
}

// undoLast asks to reset the branch to the point saved before the last
// history-changing operation.
func (a *App) undoLast() tea.Cmd {
	dir, ctx := a.rp.dir, a.ctx
	p, err := gitx.LastUndo(ctx, dir)
	switch {
	case err != nil:
		a.setToast(err.Error(), true)
		return nil
	case p == nil:
		a.setToast("nothing to undo", false)
		return nil
	}
	target := p.Branch
	if target == "" {
		target = "detached HEAD"
	}
	a.confirm(fmt.Sprintf("Undo %q? %s will be reset to %.7s.", p.Label, target, p.Head), func() tea.Cmd {
		return a.doMsg("undo", func() (string, error) {
			return "undid " + p.Label + "  ·  z to redo", gitx.Undo(ctx, dir)
		})
	})
	return nil
}

// panelHeights splits the left column between the four panels the way
// lazygit does: every panel shows its items, the focused one gets the most
// room, and panels with few items give their unused rows away.
func panelHeights(total, focus int, counts [4]int) [4]int {
	const minH = 3 // border + title, no rows
	var h [4]int
	for i := range h {
		h[i] = minH
	}
	if total < 4*minH+4 { // tiny terminal: only the focused panel is shown
		h = [4]int{}
		h[focus] = total
		return h
	}
	rest := total - 4*minH
	h[focus] += rest * 45 / 100 // the focused panel always gets a good share
	rest -= h[focus] - minH
	// Hand out the remaining rows one at a time to the other panels that
	// still have items to show, never making one taller than the focused one.
	for rest > 0 {
		gave := false
		for i := range h {
			if rest > 0 && i != focus && h[i]-minH < counts[i] && h[i] < h[focus] {
				h[i]++
				rest--
				gave = true
			}
		}
		if !gave {
			break
		}
	}
	h[focus] += rest // nobody else needs them
	return h
}

func (a *App) scrollMain(delta int) {
	rp := &a.rp
	rp.mainOff = clamp(rp.mainOff+delta, 0, rp.mainLen-rp.mainH)
}

// mainKey handles keys while the main (diff) panel is focused.
func (a *App) mainKey(s string) tea.Cmd {
	rp := &a.rp
	half := clamp(rp.mainH/2, 1, 1<<30)
	switch lookup(scMain, s) {
	case actBack:
		rp.mainFocus = false
	case actDown:
		a.scrollMain(1)
	case actUp:
		a.scrollMain(-1)
	case actPageDown:
		a.scrollMain(half)
	case actPageUp:
		a.scrollMain(-half)
	case actTop:
		rp.mainOff = 0
	case actBottom:
		a.scrollMain(1 << 30)
	}
	return nil
}

// How often the open repository's working tree is re-read, so new, edited
// and deleted files show up without leaving the repository.
const liveRefresh = 2 * time.Second

type (
	repoTickMsg    struct{}
	repoRefreshMsg struct {
		dir   string
		info  gitx.Info
		files []gitx.File
		key   string // main panel source the text belongs to ("" = not refreshed)
		text  string
	}
)

func repoTick() tea.Cmd {
	return tea.Tick(liveRefresh, func(time.Time) tea.Msg { return repoTickMsg{} })
}

// onRepoTick re-reads status and changed files of the open repository (and
// the diff shown for a selected file) while nothing else is going on.
func (a *App) onRepoTick() tea.Cmd {
	rp := &a.rp
	idle := a.scr == scrRepo && !rp.refreshing && a.busy == "" && a.p == nil && a.ed == nil && a.report == nil
	if !idle {
		return repoTick()
	}
	rp.refreshing = true
	ctx, dir := a.ctx, rp.dir
	key, fn := "", func() string { return "" }
	if rp.focus == pFiles {
		key, fn = a.mainSource()
	}
	return tea.Batch(repoTick(), func() tea.Msg {
		m := repoRefreshMsg{dir: dir, key: key}
		m.info, _ = gitx.GetInfo(ctx, dir)
		m.files, _ = gitx.Files(ctx, dir)
		if key != "" {
			m.text = fn()
		}
		return m
	})
}

func (a *App) onRepoRefresh(m repoRefreshMsg) tea.Cmd {
	rp := &a.rp
	if m.dir != rp.dir {
		return nil
	}
	rp.refreshing = false
	if a.scr != scrRepo {
		return nil
	}
	rp.info = m.info
	rp.setFiles(m.files)
	if m.key != "" && m.key == rp.mainKey && m.text != rp.main {
		rp.main = m.text // same file, new content: keep the scroll position
	}
	return a.mainCmd() // the selection itself may have changed (file gone, new status)
}

// push pushes the current branch. When it has diverged from its upstream it
// checks whether that is only because pushed commits were rewritten here
// (squash, reword, amend…): then a force push with lease is offered; if the
// remote has commits that were never local, it refuses and suggests a pull.
func (a *App) push(force bool) tea.Cmd {
	rp := &a.rp
	dir, ctx := rp.dir, a.ctx
	info := rp.info
	up := info.Upstream
	upName := "the remote"
	for _, b := range rp.local {
		if b.Current && b.Upstream != "" {
			upName = b.Upstream
		}
	}
	doPush := func(force bool) tea.Cmd {
		label := "push"
		if force {
			label = "force push"
		}
		return a.doMsg(label, func() (string, error) {
			err := gitx.Push(ctx, dir, force, up)
			if err != nil && !force && gitx.PushRejected(err) {
				return "", fmt.Errorf("rejected: %s has commits you haven't fetched — fetch (f), then pull or force push", upName)
			}
			return "", err
		})
	}
	if !up || info.Behind == 0 {
		if !force {
			return doPush(false)
		}
		a.confirm("Force push "+info.Branch+" (--force-with-lease)?", func() tea.Cmd { return doPush(true) })
		return nil
	}
	diverged := fmt.Sprintf("%s and %s have diverged (↑%d ↓%d)", info.Branch, upName, info.Ahead, info.Behind)
	return a.after(func() (bool, error) { return gitx.UpstreamSeenLocally(ctx, dir) }, func(seen bool) tea.Cmd {
		switch {
		case seen:
			a.confirm(diverged+" because commits that were already pushed were rewritten here. Force push (--force-with-lease)?",
				func() tea.Cmd { return doPush(true) })
		case force:
			a.confirm("⚠ "+upName+" has commits that were never in your branch; force pushing deletes them from the remote. Force push anyway?",
				func() tea.Cmd { return doPush(true) })
		default:
			a.setToast(diverged+": "+upName+" has commits you don't have — pull first (p / ctrl+p), or F to overwrite them", true)
		}
		return nil
	})
}
