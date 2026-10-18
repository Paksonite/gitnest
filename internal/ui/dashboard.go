package ui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/gitx"
	"github.com/Paksonite/gitnest/internal/scan"
)

// The dashboard (start screen, H in the list): commit activity across every
// repository as a heatmap, an overview, what needs attention, the most
// active repositories with weekly sparklines, and the latest commits.
type dashState struct {
	loading bool
	loaded  time.Time
	act     map[string][]gitx.ActivityCommit // by repository path
	cur     int
	items   []string  // repositories the cursor can be on, in screen order
	hits    []dashHit // where they are on screen, for the mouse
}

// dashHit is a selectable row on screen: item i at row y, columns x0..x1.
type dashHit struct{ i, y, x0, x1 int }

type activityMsg struct {
	act map[string][]gitx.ActivityCommit
}

const (
	heatWeeks   = 26
	activeWeeks = 12
)

// loadActivity collects the last heatWeeks weeks of commits of every
// repository, in parallel.
func (a *App) loadActivity() tea.Cmd {
	if a.dash.loading || len(a.rs.all) == 0 {
		return nil
	}
	a.dash.loading = true
	ctx, jobs := a.ctx, a.cfg.Jobs
	paths := make([]string, len(a.rs.all))
	for i, r := range a.rs.all {
		paths[i] = r.Path
	}
	since := weekStart(time.Now()).AddDate(0, 0, -7*(heatWeeks-1))
	return func() tea.Msg {
		act := make(map[string][]gitx.ActivityCommit, len(paths))
		var mu sync.Mutex
		scan.ForEach(len(paths), jobs, func(i int) {
			cs, err := gitx.Activity(ctx, paths[i], since, 5000)
			if err != nil {
				return
			}
			mu.Lock()
			act[paths[i]] = cs
			mu.Unlock()
		})
		return activityMsg{act: act}
	}
}

func (a *App) onActivity(m activityMsg) {
	a.dash.loading, a.dash.loaded, a.dash.act = false, time.Now(), m.act
}

// openDashboard shows the dashboard, loading the activity when it is
// missing or older than ten minutes.
func (a *App) openDashboard() tea.Cmd {
	a.scr = scrDash
	if a.dash.act == nil || time.Since(a.dash.loaded) > 10*time.Minute {
		return a.loadActivity()
	}
	return nil
}

// weekStart is the Monday 00:00 of t's week, in local time.
func weekStart(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

func (a *App) dashKey(k tea.KeyMsg) tea.Cmd {
	d := &a.dash
	switch lookup(scDash, k.String()) {
	case actUp:
		d.cur = clamp(d.cur-1, 0, len(d.items)-1)
	case actDown:
		d.cur = clamp(d.cur+1, 0, len(d.items)-1)
	case actTop:
		d.cur = 0
	case actBottom:
		d.cur = max(0, len(d.items)-1)
	case actOpen:
		if d.cur >= 0 && d.cur < len(d.items) {
			return a.openDir(d.items[d.cur])
		}
	case actSearch:
		a.scr = scrRepos
		a.rs.filtering, a.rs.filter = true, ""
		a.rebuildView(a.selectedKey())
	case actBack, actDashboard:
		a.scr = scrRepos
		return a.previewCmd()
	case actRefreshList:
		a.dash.act = nil
		return a.loadActivity()
	case actQuit:
		return tea.Quit
	}
	return nil
}

// ------------------------------------------------------------------ view

func (a *App) viewDashboard() string {
	d := &a.dash
	now := time.Now()
	bodyH := a.h - 3
	var sum overview
	for _, r := range a.rs.all {
		sum.add(r)
	}
	head := brand() + fg(cDim, " › ") + bold("dashboard") + fg(cDim, fmt.Sprintf("  %d repos · %d dirty", len(a.rs.all), sum.dirty))
	if d.loading {
		head += "  " + fg(cYellow, "⟳ reading activity…")
	}
	head += a.updateBadge()

	byDay := map[time.Time]int{}
	var all []feedCommit
	for p, cs := range d.act {
		for _, c := range cs {
			t := time.Unix(c.Unix, 0)
			byDay[time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())]++
			all = append(all, feedCommit{c, p})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Unix > all[j].Unix })

	// Selectable rows, filled in while the panels are drawn.
	d.items, d.hits = d.items[:0], d.hits[:0]
	pick := func(path string) (selected bool) {
		d.items = append(d.items, path)
		return len(d.items)-1 == d.cur
	}

	wide := a.w >= 110
	heatW := min(a.w, 2*heatWeeks+10)
	if wide {
		heatW = 2*heatWeeks + 10
	}
	heatH := 12
	var top string
	heat := panel(subtitle("Activity", fmt.Sprintf("%d commits · %d weeks", len(all), heatWeeks)),
		heatmap(byDay, now, panelInner(heatW), d.loading && d.act == nil), heatW, heatH, false)
	if wide {
		top = lipgloss.JoinHorizontal(lipgloss.Top, heat,
			panel("Overview", overviewLines(sum, byDay, now, panelInner(a.w-heatW)), a.w-heatW, heatH, false))
	} else {
		top = heat
	}
	restH := max(3, bodyH-heatH)
	y0 := 1 + heatH // the first row below the heatmap, on screen

	leftW := a.w / 2
	if !wide {
		leftW = a.w
	}
	attnH := restH / 2
	if !wide {
		attnH = restH / 3
	}
	recentH := restH - attnH
	activeH := restH
	if !wide {
		recentH, activeH = restH/3, restH-attnH-restH/3
	}

	attn := needAttention(a.rs.all)
	stW := 1
	for _, r := range attn {
		sp, _ := statusCell(r)
		stW = max(stW, lipgloss.Width(sp))
	}
	var attnLines []string
	for i, r := range attn {
		if i >= attnH-3 {
			break
		}
		sp, ss := statusCell(r)
		row := rowItem{plain: " " + fit(sp, stW) + "  " + r.Name + "  " + r.Branch,
			styled: " " + fit(ss, stW) + "  " + bold(r.Name) + "  " + fg(cCyan, r.Branch)}
		attnLines = append(attnLines, a.dashRow(row, pick(r.Path), panelInner(leftW)))
		d.hits = append(d.hits, dashHit{len(d.items) - 1, y0 + 2 + i, 0, leftW})
	}
	if len(attn) == 0 {
		attnLines = []string{fg(cGreen, "✓ everything is committed and in sync")}
	}
	attnPanel := panel(subtitle("Needs attention", fmt.Sprint(len(attn))), attnLines, leftW, attnH, false)

	recentW := leftW
	var recentLines []string
	for i, c := range all {
		if i >= recentH-3 {
			break
		}
		ac := authorColor(c.Author)
		age := fmt.Sprintf("%4s", relTime(c.Unix, now))
		repo := filepath.Base(c.path)
		plain := "● " + initials(c.Author) + " " + c.Short + " " + repo + " · " + c.Subject
		styled := fg(ac, "●") + " " + lipgloss.NewStyle().Bold(true).Foreground(ac).Render(initials(c.Author)) + " " +
			fg(cYellow, c.Short) + " " + bold(repo) + fg(cDim, " · ") + c.Subject
		room := panelInner(recentW) - 5
		row := rowItem{plain: fit(plain, room) + " " + age, styled: fit(styled, room) + " " + fg(cDim, age)}
		y := y0 + attnH + 2 + i
		if !wide {
			y = y0 + attnH + activeH + 2 + i
		}
		recentLines = append(recentLines, a.dashRow(row, pick(c.path), panelInner(recentW)))
		d.hits = append(d.hits, dashHit{len(d.items) - 1, y, 0, recentW})
	}
	if d.act == nil {
		recentLines = []string{fg(cDim, "reading the history of every repository…")}
	}
	recentPanel := panel("Recent commits", recentLines, recentW, recentH, false)

	activeW := a.w - leftW
	if !wide {
		activeW = a.w
	}
	var activeLines []string
	if d.act != nil {
		for i, s := range mostActive(d.act, a.rs.all, now) {
			if i >= activeH-3 {
				break
			}
			inner := panelInner(activeW)
			spark := sparkline(s.weeks)
			count := fmt.Sprintf("%4d", s.total)
			age := fmt.Sprintf("%4s", relTime(s.last, now))
			nameW := max(8, inner-len([]rune(spark))-4-4-3)
			row := rowItem{plain: fit(s.name, nameW) + " " + spark + " " + count + " " + age,
				styled: fit(bold(s.name), nameW) + " " + fg(cAccent, spark) + " " + count + " " + fg(cDim, age)}
			x0, y := leftW, y0+2+i
			if !wide {
				x0, y = 0, y0+attnH+2+i
			}
			activeLines = append(activeLines, a.dashRow(row, pick(s.path), inner))
			d.hits = append(d.hits, dashHit{len(d.items) - 1, y, x0, x0 + activeW})
		}
	}
	activePanel := panel(subtitle("Most active", fmt.Sprintf("commits per week · %d weeks", activeWeeks)), activeLines, activeW, activeH, false)

	var body string
	if wide {
		left := lipgloss.JoinVertical(lipgloss.Left, attnPanel, recentPanel)
		body = lipgloss.JoinVertical(lipgloss.Left, top, lipgloss.JoinHorizontal(lipgloss.Top, left, activePanel))
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, top, attnPanel, activePanel, recentPanel)
	}
	d.cur = clamp(d.cur, 0, max(0, len(d.items)-1))
	lines := strings.Split(body, "\n")
	if len(lines) > bodyH {
		lines = lines[:bodyH]
	}
	for len(lines) < bodyH {
		lines = append(lines, "")
	}
	return strings.Join([]string{fit(head, a.w), strings.Join(lines, "\n"), a.footer(hints(scDash, scGlobal))}, "\n")
}

// dashRow highlights the row under the cursor.
func (a *App) dashRow(it rowItem, selected bool, w int) string {
	return renderRows([]rowItem{it}, 0, 0, 1, w, selected)[0]
}

type feedCommit struct {
	gitx.ActivityCommit
	path string
}

// overview counts the repositories by how they stand.
type overview struct{ dirty, ahead, behind, noUpstream, trouble int }

func (o *overview) add(r scan.Repo) {
	if r.Changes > 0 {
		o.dirty++
	}
	if r.Upstream && r.Ahead > 0 {
		o.ahead++
	}
	if r.Upstream && r.Behind > 0 {
		o.behind++
	}
	if !r.Upstream && !r.Detached {
		o.noUpstream++
	}
	if r.Conflicts || r.State != gitx.StateNone {
		o.trouble++
	}
}

func overviewLines(o overview, byDay map[time.Time]int, now time.Time, w int) []string {
	this := weekStart(now)
	var thisWeek, lastWeek int
	byWeekday := [7]int{}
	for day, n := range byDay {
		switch {
		case !day.Before(this):
			thisWeek += n
		case !day.Before(this.AddDate(0, 0, -7)):
			lastWeek += n
		}
		byWeekday[(int(day.Weekday())+6)%7] += n
	}
	busiest := 0
	for i, n := range byWeekday {
		if n > byWeekday[busiest] {
			busiest = i
		}
	}
	streak := 0
	for day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()); byDay[day] > 0; day = day.AddDate(0, 0, -1) {
		streak++
	}
	num := func(n int, c lipgloss.TerminalColor) string {
		if n == 0 {
			return fg(cDim, "0")
		}
		return fg(c, fmt.Sprint(n))
	}
	field := func(k, v string) string { return fit(fg(cDim, padRight(k, 13))+v, w) }
	lines := []string{
		field("Dirty", num(o.dirty, cYellow)),
		field("To push", num(o.ahead, cGreen)),
		field("To pull", num(o.behind, cRed)),
		field("No upstream", num(o.noUpstream, cDim)),
		field("In trouble", num(o.trouble, cRed)),
		field("This week", fmt.Sprintf("%d commits", thisWeek)),
		field("Last week", fmt.Sprintf("%d commits", lastWeek)),
	}
	if len(byDay) > 0 {
		lines = append(lines, field("Busiest day", [...]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}[busiest]))
	}
	if streak > 0 {
		lines = append(lines, field("Streak", fmt.Sprintf("%d day(s) in a row", streak)))
	}
	return lines
}

// heatmap draws commits per day, a GitHub-style grid: weeks across, days
// of the week down, darker squares for busier days.
func heatmap(byDay map[time.Time]int, now time.Time, w int, loading bool) []string {
	if loading {
		return []string{fg(cDim, "reading the history of every repository…")}
	}
	weeks := clamp((w-4)/2, 4, heatWeeks)
	start := weekStart(now).AddDate(0, 0, -7*(weeks-1))
	top := 0
	for _, n := range byDay {
		top = max(top, n)
	}
	levels := []lipgloss.TerminalColor{tint(cAccent, 0.12), tint(cAccent, 0.35), tint(cAccent, 0.6), tint(cAccent, 0.8), cAccent}
	level := func(n int) int {
		if n == 0 || top == 0 {
			return 0
		}
		return min(4, 1+(n*4-1)/top)
	}
	// Month names above the week where a month starts.
	months := []rune(strings.Repeat(" ", 4+2*weeks))
	for c := 0; c < weeks; c++ {
		first := start.AddDate(0, 0, 7*c)
		if c == 0 || first.Month() != first.AddDate(0, 0, -7).Month() {
			name := []rune(first.Format("Jan"))
			at := 4 + 2*c
			if at+len(name) <= len(months) {
				copy(months[at:], name)
			}
		}
	}
	out := []string{fg(cDim, string(months))}
	labels := [7]string{"Mon", "", "Wed", "", "Fri", "", "Sun"}
	for row := 0; row < 7; row++ {
		var b strings.Builder
		b.WriteString(fg(cDim, padRight(labels[row], 4)))
		for c := 0; c < weeks; c++ {
			day := start.AddDate(0, 0, 7*c+row)
			if day.After(now) {
				b.WriteString("  ")
				continue
			}
			b.WriteString(fg(levels[level(byDay[day])], "■") + " ")
		}
		out = append(out, b.String())
	}
	legend := fg(cDim, "    less ")
	for _, c := range levels {
		legend += fg(c, "■") + " "
	}
	return append(out, legend+fg(cDim, "more"))
}

// needAttention lists the repositories that need something done: operations
// in progress and conflicts, then uncommitted changes, then commits to
// pull, then to push.
func needAttention(repos []scan.Repo) []scan.Repo {
	rank := func(r scan.Repo) int {
		switch {
		case r.State != gitx.StateNone || r.Conflicts:
			return 0
		case r.Changes > 0:
			return 1
		case r.Upstream && r.Behind > 0:
			return 2
		case r.Upstream && r.Ahead > 0:
			return 3
		}
		return -1
	}
	var out []scan.Repo
	for _, r := range repos {
		if rank(r) >= 0 {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if rank(out[i]) != rank(out[j]) {
			return rank(out[i]) < rank(out[j])
		}
		return out[i].Name < out[j].Name
	})
	return out
}

type repoActivity struct {
	path, name string
	weeks      []int // commits per week, oldest first
	total      int
	last       int64
}

// mostActive ranks repositories by their commits in the last activeWeeks
// weeks.
func mostActive(act map[string][]gitx.ActivityCommit, repos []scan.Repo, now time.Time) []repoActivity {
	start := weekStart(now).AddDate(0, 0, -7*(activeWeeks-1))
	var out []repoActivity
	for _, r := range repos {
		ra := repoActivity{path: r.Path, name: r.Name, weeks: make([]int, activeWeeks)}
		for _, c := range act[r.Path] {
			t := time.Unix(c.Unix, 0)
			if t.Before(start) {
				continue
			}
			wk := int(t.Sub(start).Hours() / 24 / 7)
			if wk >= 0 && wk < activeWeeks {
				ra.weeks[wk]++
				ra.total++
			}
			ra.last = max(ra.last, c.Unix)
		}
		if ra.total > 0 {
			out = append(out, ra)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].total != out[j].total {
			return out[i].total > out[j].total
		}
		return out[i].last > out[j].last
	})
	return out
}

// sparkline draws counts as bars ▁▂▃▄▅▆▇█, scaled to the largest; empty
// weeks are a dim dot.
func sparkline(counts []int) string {
	bars := []rune("▁▂▃▄▅▆▇█")
	top := 0
	for _, n := range counts {
		top = max(top, n)
	}
	var b strings.Builder
	for _, n := range counts {
		if n == 0 || top == 0 {
			b.WriteRune('·')
			continue
		}
		b.WriteRune(bars[min(len(bars)-1, (n*len(bars)-1)/top)])
	}
	return b.String()
}
