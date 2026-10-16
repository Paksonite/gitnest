package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

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
		return len(rp.files)
	case pBranches:
		return len(rp.branches())
	case pCommits:
		return len(rp.commits)
	}
	return len(rp.stashes)
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
	rp.info, rp.files, rp.local, rp.remote = d.info, d.files, d.local, d.remote
	rp.commits, rp.unpushed, rp.stashes = d.commits, d.unpushed, d.stashes
	for i := 0; i < 4; i++ {
		rp.cur[i] = clamp(rp.cur[i], 0, rp.count(i)-1)
	}
	return a.mainCmd()
}

// mainCmd (re)computes the right-hand pane for the selected item, if it changed.
func (a *App) mainCmd() tea.Cmd {
	rp := &a.rp
	dir, ctx := rp.dir, a.ctx
	var key string
	var fn func() string
	i := rp.cur[rp.focus]
	switch rp.focus {
	case pFiles:
		if len(rp.files) == 0 {
			key = "files:none"
			fn = func() string { return "Working tree clean ✓" }
		} else {
			f := rp.files[i]
			key = "f:" + f.Path + ":" + string([]byte{f.X, f.Y})
			fn = func() string { return gitx.DiffFile(ctx, dir, f) }
		}
	case pBranches:
		bl := rp.branches()
		if len(bl) == 0 {
			key = "branches:none"
			fn = func() string { return "(no branches)" }
		} else {
			b := bl[i]
			key = "b:" + b.Name
			fn = func() string { return gitx.BranchLog(ctx, dir, b.Name) }
		}
	case pCommits:
		if len(rp.commits) == 0 {
			key = "commits:none"
			fn = func() string { return "(no commits)" }
		} else {
			c := rp.commits[i]
			key = "c:" + c.Hash
			fn = func() string { return gitx.Show(ctx, dir, c.Hash) }
		}
	default:
		if len(rp.stashes) == 0 {
			key = "stash:none"
			fn = func() string { return "(no stashes)" }
		} else {
			s := rp.stashes[i]
			key = "s:" + s.Ref + ":" + s.Msg
			fn = func() string { return gitx.StashShow(ctx, dir, s.Ref) }
		}
	}
	if key == rp.mainKey {
		return nil
	}
	rp.mainKey, rp.main, rp.mainOff = key, "", 0
	return func() tea.Msg { return mainMsg{key: key, text: fn()} }
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

	// navigation
	switch s {
	case "esc", "q":
		a.scr = scrRepos
		return a.gather(a.paths())
	case "tab", "right", "l":
		rp.focus = (rp.focus + 1) % 4
		return a.mainCmd()
	case "shift+tab", "left", "h":
		rp.focus = (rp.focus + 3) % 4
		return a.mainCmd()
	case "1", "2", "3", "4":
		rp.focus = int(s[0] - '1')
		return a.mainCmd()
	case "up", "k":
		return a.move(-1)
	case "down", "j":
		return a.move(1)
	case "home":
		return a.move(-rp.count(rp.focus))
	case "end":
		return a.move(rp.count(rp.focus))
	case "pgup", "ctrl+u", "K":
		rp.mainOff = clamp(rp.mainOff-half, 0, 1<<30)
		return nil
	case "pgdown", "ctrl+d", "J":
		rp.mainOff += half
		return nil
	}

	// rebase/merge/cherry-pick/revert/bisect in progress
	if st := rp.info.State; st != gitx.StateNone {
		name := strings.ToLower(st.String())
		switch {
		case s == "C" && st != gitx.StateBisecting:
			return a.do("continue "+name, func() error { return gitx.Continue(ctx, dir, st) })
		case s == "X":
			a.confirm("Abort "+name+"?", func() tea.Cmd {
				return a.do("abort "+name, func() error { return gitx.Abort(ctx, dir, st) })
			})
			return nil
		case s == "S" && st.CanSkip():
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
	switch s {
	case "f", "ctrl+f":
		return a.do("fetch", func() error { return gitx.Fetch(ctx, dir) })
	case "p":
		return a.do("pull", func() error { return gitx.Pull(ctx, dir, false) })
	case "ctrl+p":
		return a.do("pull --rebase", func() error { return gitx.Pull(ctx, dir, true) })
	case "P":
		up := rp.info.Upstream
		return a.do("push", func() error { return gitx.Push(ctx, dir, false, up) })
	case "F":
		up := rp.info.Upstream
		a.confirm("Force push (--force-with-lease)?", func() tea.Cmd {
			return a.do("force push", func() error { return gitx.Push(ctx, dir, true, up) })
		})
	case "ctrl+r":
		rp.mainKey = ""
		return loadRepoData(ctx, dir)
	case "z":
		return a.undoLast()
	case "L":
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
	var f gitx.File
	if has {
		f = rp.files[rp.cur[pFiles]]
	}
	switch s {
	case " ":
		if !has {
			return nil, true
		}
		if f.Unstaged() {
			return a.do("stage", func() error { return gitx.Stage(ctx, dir, f.Path) }), true
		}
		return a.do("unstage", func() error { return gitx.Unstage(ctx, dir, f.Path) }), true
	case "a":
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
	case "c":
		a.ask("Commit message", "", func(msg string) tea.Cmd {
			return a.do("commit", func() error { return gitx.CommitMsg(ctx, dir, msg) })
		})
		return nil, true
	case "C":
		c := exec.Command("git", "commit")
		c.Dir = dir
		return a.execCmd(c), true
	case "A":
		return a.check(func() ([]string, error) { return gitx.CheckRewrite(ctx, dir, "HEAD", gitx.ActionAmend) },
			func(pushed []string) tea.Cmd {
				return a.confirmRewrite("Amend last commit with the staged changes?", pushed, func() tea.Cmd {
					return a.doUndoable("amend", func() error { return gitx.AmendNoEdit(ctx, dir) })
				})
			}), true
	case "d":
		if !has {
			return nil, true
		}
		a.confirm("Discard all changes to "+f.Path+"?", func() tea.Cmd {
			return a.do("discard", func() error { return gitx.Discard(ctx, dir, f) })
		})
		return nil, true
	case "s":
		a.ask("Stash message", "wip", func(msg string) tea.Cmd {
			return a.do("stash", func() error { return gitx.StashPush(ctx, dir, msg) })
		})
		return nil, true
	case "e":
		if !has {
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
	switch s {
	case "t", "[", "]":
		rp.showRemote = !rp.showRemote
		rp.cur[pBranches], rp.off[pBranches] = 0, 0
		return a.mainCmd(), true
	case " ":
		if !has {
			return nil, true
		}
		return a.do("checkout "+b.Name, func() error { return gitx.Checkout(ctx, dir, b) }), true
	case "n":
		base := ""
		if has {
			base = b.Name
		}
		a.ask("New branch name (from "+base+")", "", func(name string) tea.Cmd {
			return a.do("new branch", func() error { return gitx.NewBranch(ctx, dir, name, base) })
		})
		return nil, true
	case "d", "D":
		if !has {
			return nil, true
		}
		if b.Current {
			a.setToast("can't delete the checked-out branch", true)
			return nil, true
		}
		force := s == "D"
		a.confirm("Delete branch "+b.Name+"?", func() tea.Cmd {
			if b.Remote {
				return a.do("delete remote branch", func() error { return gitx.DeleteRemoteBranch(ctx, dir, b.Name) })
			}
			return a.do("delete branch", func() error { return gitx.DeleteBranch(ctx, dir, b.Name, force) })
		})
		return nil, true
	case "r":
		if !has || b.Remote {
			return nil, true
		}
		a.ask("Rename "+b.Name+" to", b.Name, func(name string) tea.Cmd {
			return a.do("rename", func() error { return gitx.RenameBranch(ctx, dir, b.Name, name) })
		})
		return nil, true
	case "M":
		if !has || b.Current {
			return nil, true
		}
		a.confirm("Merge "+b.Name+" into the current branch?", func() tea.Cmd {
			return a.doUndoable("merge "+b.Name, func() error { return gitx.Merge(ctx, dir, b.Name) })
		})
		return nil, true
	case "R":
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
	switch s {
	case "s":
		return meld(gitx.ActionSquash), true
	case "f":
		return meld(gitx.ActionFixup), true
	case "d":
		return meld(gitx.ActionDrop), true
	case "r":
		return a.check(func() ([]string, error) { return gitx.CheckRewrite(ctx, dir, c.Hash, gitx.ActionReword) },
			func(pushed []string) tea.Cmd {
				a.ask("Reword "+c.Short, c.Subject, func(msg string) tea.Cmd {
					return a.confirmRewrite("", pushed, func() tea.Cmd {
						return a.doUndoable("reword "+c.Short, func() error { return gitx.Reword(ctx, dir, c.Hash, msg) })
					})
				})
				return nil
			}), true
	case "t":
		a.confirm("Revert commit "+c.Short+"?", func() tea.Cmd {
			return a.doUndoable("revert "+c.Short, func() error { return gitx.Revert(ctx, dir, c.Hash) })
		})
		return nil, true
	case "u":
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
	switch s {
	case " ":
		return a.do("stash apply", func() error { return gitx.StashApply(ctx, dir, st.Ref) }), true
	case "g":
		return a.do("stash pop", func() error { return gitx.StashPop(ctx, dir, st.Ref) }), true
	case "d":
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
		for _, f := range rp.files {
			code := string([]byte{f.X, f.Y})
			var st string
			switch {
			case f.Conflict():
				st = lipgloss.NewStyle().Bold(true).Foreground(cYellow).Render(code)
			case f.Untracked():
				st = fg(cRed, code)
			default:
				x, y := string(f.X), string(f.Y)
				if f.X != ' ' {
					x = fg(cGreen, x)
				}
				if f.Y != ' ' {
					y = fg(cRed, y)
				}
				st = x + y
			}
			items = append(items, rowItem{plain: code + " " + f.Path, styled: st + " " + f.Path})
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
	if st := a.rp.info.State; st != gitx.StateNone {
		return stateKeys(st) + "  ·  resolve conflicts in Files (e edit, space stage)  ·  ? help"
	}
	switch a.rp.focus {
	case pFiles:
		return "space stage  a all  c commit  C editor  A amend  d discard  s stash  e edit  ·  p pull  P push  ctrl+f fetch  L lazygit  esc back  ? help"
	case pBranches:
		return "space checkout  n new  d delete  r rename  M merge  R rebase  t local/remote  ·  p pull  P push  esc back  ? help"
	case pCommits:
		return "s squash  f fixup  d drop  r reword  t revert  u undo last  z undo rewrite  ·  p pull  P push  esc back  ? help"
	}
	return "space apply  g pop  d drop  ·  p pull  P push  esc back  ? help"
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

	const collapsed = 3
	focH := bodyH - 3*collapsed
	if focH < 5 {
		focH = 5
	}
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
	panels := make([]string, 4)
	for i := 0; i < 4; i++ {
		h := collapsed
		if i == rp.focus {
			h = focH
		}
		var lines []string
		if h > collapsed {
			items := a.panelItems(i)
			lh := h - 3
			rp.off[i] = scrollTo(rp.cur[i], rp.off[i], lh)
			lines = renderRows(items, rp.cur[i], rp.off[i], lh, leftW-2, true)
		}
		panels[i] = panel(titles[i], lines, leftW, h, i == rp.focus)
	}
	left := lipgloss.JoinVertical(lipgloss.Left, panels...)

	mainTitle := [4]string{"Diff", "Log", "Commit", "Stash"}[rp.focus]
	mainLines := strings.Split(strings.TrimRight(expandTabs(rp.main), "\n"), "\n")
	mh := lipgloss.Height(left)
	if rp.mainOff > len(mainLines)-1 {
		rp.mainOff = clamp(len(mainLines)-1, 0, 1<<30)
	}
	mainLines = mainLines[rp.mainOff:]
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, panel(mainTitle, mainLines, mainW, mh, false))

	return strings.Join([]string{fit(head, a.w), body, a.footer(a.repoHints())}, "\n")
}

// stateKeys lists the keys that act on an operation in progress.
func stateKeys(st gitx.State) string {
	var k []string
	if st != gitx.StateBisecting {
		k = append(k, "C continue")
	}
	if st.CanSkip() {
		k = append(k, "S skip")
	}
	return strings.Join(append(k, "X abort"), " · ")
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
