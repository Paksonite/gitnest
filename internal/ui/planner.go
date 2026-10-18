package ui

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/v2/internal/gitx"
)

// planner is the interactive rebase editor (i in the Commits panel): the
// commits from the selected one up to HEAD, newest first like the panel,
// each with an action, reorderable. enter runs the plan.
type planner struct {
	rows   []planRow
	cur    int
	oldest string // the range's oldest commit, for the pre-flight check
}

type planRow struct {
	c      gitx.Commit
	action string
	msg    string // reword message
}

func (a *App) openPlanner(i int) {
	rp := &a.rp
	p := &planner{oldest: rp.commits[i].Hash}
	for _, c := range rp.commits[:i+1] {
		p.rows = append(p.rows, planRow{c: c, action: "pick"})
	}
	a.plan = p
}

// steps turns the rows into a gitx plan, oldest first.
func (p *planner) steps() []gitx.PlanStep {
	out := make([]gitx.PlanStep, 0, len(p.rows))
	for i := len(p.rows) - 1; i >= 0; i-- {
		r := p.rows[i]
		out = append(out, gitx.PlanStep{Hash: r.c.Hash, Action: r.action, Message: r.msg})
	}
	return out
}

func (a *App) plannerKey(k tea.KeyMsg) tea.Cmd {
	p := a.plan
	row := &p.rows[p.cur]
	set := func(action string) { row.action, row.msg = action, "" }
	switch lookup(scPlan, k.String()) {
	case actUp:
		p.cur = clamp(p.cur-1, 0, len(p.rows)-1)
	case actDown:
		p.cur = clamp(p.cur+1, 0, len(p.rows)-1)
	case actMoveUp: // newer
		if p.cur > 0 {
			p.rows[p.cur], p.rows[p.cur-1] = p.rows[p.cur-1], p.rows[p.cur]
			p.cur--
		}
	case actMoveDown: // older
		if p.cur < len(p.rows)-1 {
			p.rows[p.cur], p.rows[p.cur+1] = p.rows[p.cur+1], p.rows[p.cur]
			p.cur++
		}
	case actPick:
		set("pick")
	case actEditStep:
		set("edit")
	case actSquash:
		set(gitx.ActionSquash)
	case actFixup:
		set(gitx.ActionFixup)
	case actDrop:
		set(gitx.ActionDrop)
	case actReword:
		dir, hash := a.rp.dir, row.c.Hash
		current := row.msg
		if current == "" {
			current, _ = gitx.Message(a.ctx, dir, hash)
		}
		i := p.cur
		return a.openEditor("Reword "+row.c.Short+" (in the plan)", gitx.CleanMessage(current), func(msg string) tea.Cmd {
			p.rows[i].action, p.rows[i].msg = gitx.ActionReword, msg
			return nil
		})
	case actCancel:
		a.plan = nil
	case actSave:
		return a.runPlan()
	}
	return nil
}

func (a *App) runPlan() tea.Cmd {
	p := a.plan
	steps := p.steps()
	switch steps[0].Action {
	case gitx.ActionSquash, gitx.ActionFixup:
		a.setToast("the bottom (oldest) commit has nothing to "+steps[0].Action+" into", true)
		return nil
	}
	a.plan = nil
	dir, ctx, oldest := a.rp.dir, a.ctx, p.oldest
	return a.check(func() ([]string, error) { return gitx.CheckRewrite(ctx, dir, oldest, gitx.ActionDrop) },
		func(pushed []string) tea.Cmd {
			return a.confirmRewrite(fmt.Sprintf("Run the rebase plan on %d commits?", len(steps)), pushed, func() tea.Cmd {
				return a.doMsg("rebase plan", func() (string, error) {
					err := gitx.RebasePlan(ctx, dir, steps)
					if errors.Is(err, gitx.ErrStopped) {
						return "stopped for editing — amend or commit, then C continues (X aborts)", nil
					}
					return "rebase plan ✓  ·  z to undo", err
				})
			})
		})
}

func actionColor(action string) lipgloss.TerminalColor {
	switch action {
	case gitx.ActionDrop:
		return cRed
	case gitx.ActionSquash, gitx.ActionFixup:
		return cYellow
	case gitx.ActionReword, "edit":
		return cCyan
	}
	return cGreen
}

func (a *App) viewPlanner() string {
	p := a.plan
	w := min(90, a.w-2)
	listH := clamp(a.h-10-2*editorPad(a.h), 1, len(p.rows))
	off := centerOff(p.cur, len(p.rows), listH)
	var lines []string
	for i := off; i < len(p.rows) && i < off+listH; i++ {
		r := p.rows[i]
		subject := r.c.Subject
		if r.msg != "" {
			subject, _, _ = strings.Cut(r.msg, "\n")
		}
		plain := fmt.Sprintf("%-7s %s %s", r.action, r.c.Short, subject)
		if i == p.cur {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Background(cSel).Foreground(cText).Render(fit(plain, w-6)))
			continue
		}
		styled := fg(actionColor(r.action), fmt.Sprintf("%-7s", r.action)) + " " + fg(cYellow, r.c.Short) + " " + subject
		if r.action == gitx.ActionDrop {
			styled = fg(cRed, fmt.Sprintf("%-7s", r.action)) + " " + fg(cDim, r.c.Short+" "+subject)
		}
		lines = append(lines, fit(styled, w-6))
	}
	body := bold("Interactive rebase") + fg(cDim, fmt.Sprintf("  %d commits, newest on top", len(p.rows))) + "\n\n" +
		strings.Join(lines, "\n") + "\n\n" + fg(cDim, hints(scPlan))
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cAccent).
		Padding(editorPad(a.h), 2).Width(w).Render(body)
	return lipgloss.Place(a.w, a.h, lipgloss.Center, lipgloss.Center, box)
}

// moveCommit swaps commit i with its older (delta 1) or newer (delta -1)
// neighbour by running a plan over the affected range.
func (a *App) moveCommit(i, delta int) tea.Cmd {
	rp := &a.rp
	j := i + delta
	if j < 0 || j >= len(rp.commits) {
		return nil
	}
	oldest := max(i, j)
	order := make([]gitx.Commit, oldest+1)
	copy(order, rp.commits[:oldest+1])
	order[i], order[j] = order[j], order[i]
	steps := make([]gitx.PlanStep, 0, len(order))
	for k := len(order) - 1; k >= 0; k-- {
		steps = append(steps, gitx.PlanStep{Hash: order[k].Hash, Action: "pick"})
	}
	// Values the background check needs are copied now: rp.commits may be
	// replaced by a refresh while it runs.
	dir, ctx, c, oldestHash := rp.dir, a.ctx, rp.commits[i], rp.commits[oldest].Hash
	return a.check(func() ([]string, error) { return gitx.CheckRewrite(ctx, dir, oldestHash, gitx.ActionDrop) },
		func(pushed []string) tea.Cmd {
			return a.confirmRewrite("", pushed, func() tea.Cmd {
				rp.cur[pCommits] = j // follow the moved commit
				return a.doUndoable("move "+c.Short, func() error { return gitx.RebasePlan(ctx, dir, steps) })
			})
		})
}
