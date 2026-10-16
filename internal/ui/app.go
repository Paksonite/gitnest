// Package ui is the bubbletea TUI: a repository list screen and a lazygit-style repo screen.
package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/config"
	"github.com/Paksonite/gitnest/internal/scan"
	"github.com/Paksonite/gitnest/internal/state"
	"github.com/Paksonite/gitnest/internal/theme"
)

// Smallest window the layout supports; below it a notice is shown instead.
const (
	minWidth  = 50
	minHeight = 12
)

type screen int

const (
	scrRepos screen = iota
	scrRepo
)

// ----------------------------------------------------------------- messages

type reposMsg struct {
	repos  []scan.Repo
	paths  []string // repository paths when the list came from a scan or its cache
	cached bool     // paths came from the scan cache; a rescan should follow
}
type previewMsg struct{ path, text string }
type repoDataMsg struct {
	dir string
	d   repoData
}
type mainMsg struct{ key, text string }
type opMsg struct {
	label string
	msg   string
	err   error
}
type execDoneMsg struct{ err error }

// checkMsg carries the result of a pre-flight check started by App.check.
type checkMsg struct {
	pushed []string
	err    error
	next   func(pushed []string) tea.Cmd
}

// --------------------------------------------------------------------- model

type prompt struct {
	title      string
	input      []rune
	confirm    bool
	allowEmpty bool // enter with no text is an answer too
	onOK       func(string) tea.Cmd
}

type App struct {
	ctx      context.Context // canceled when the UI exits; kills running git commands
	scanner  *scan.Scanner
	cfg      config.Config
	st       *state.State
	noConfig bool
	startDir string

	w, h int
	scr  screen
	rs   reposState
	rp   repoState

	p         *prompt
	report    *report // failures of the last bulk operation, or details of an error
	ed        *editor // commit message editor
	picker    *themePicker
	themes    *theme.Set
	theme     theme.Theme // active theme
	dark      bool        // the dark theme slot is in use
	themeWarn string      // problem loading themes, shown at start
	lastErr   *report     // details of the last failed operation, shown with "!"
	toast     string
	toastErr  bool
	busy      string
}

// Run starts the TUI. startDir, if non-empty, opens that repository directly.
func Run(cfg config.Config, noConfig bool, startDir string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := &App{ctx: ctx, cfg: cfg, st: state.Load(), noConfig: noConfig, startDir: startDir,
		scanner: scan.NewScanner(scan.FromConfig(cfg))}
	a.loadThemes()
	setIcons(cfg.Icons)
	a.rs.marked = map[string]bool{}
	a.rs.isNew = map[string]bool{}
	a.rs.collapsed = map[string]bool{}
	a.rs.tree = a.st.ListTree
	_, err := tea.NewProgram(a, tea.WithAltScreen(), tea.WithReportFocus()).Run()
	return err
}

// loadThemes loads built-in and user themes and applies the configured one.
func (a *App) loadThemes() {
	set, errs := theme.Load(config.ThemesDir())
	a.themes = set
	a.dark = darkBackground(a.cfg)
	t, err := pickTheme(set, a.cfg, a.dark)
	if err != nil {
		errs = append([]error{err}, errs...)
	}
	a.theme = t
	a.previewTheme(t)
	if len(errs) > 0 {
		a.themeWarn = errs[0].Error()
		if len(errs) > 1 {
			a.themeWarn += fmt.Sprintf(" (+%d more: gitnest theme list)", len(errs)-1)
		}
	}
}

func (a *App) Init() tea.Cmd {
	cmds := []tea.Cmd{a.loadRepos(), a.rescanTick(), a.statusTick(), repoTick()}
	if a.startDir != "" {
		cmds = append(cmds, a.openDir(a.startDir))
	}
	if a.noConfig {
		a.setToast("no config yet — choose directories with: gitnest roots add ~/repos", false)
	}
	if a.themeWarn != "" {
		a.setToast(a.themeWarn, true)
	}
	return tea.Batch(cmds...)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = m.Width, m.Height
		a.sizeEditor()
	case reposMsg:
		return a, a.onRepos(m)
	case scanMsg:
		return a, a.onScan(m)
	case statusMsg:
		return a, a.onStatus(m)
	case repoTickMsg:
		return a, a.onRepoTick()
	case repoRefreshMsg:
		return a, a.onRepoRefresh(m)
	case tea.FocusMsg:
		// Back from another window: whatever happened there shows up now.
		if a.scr == scrRepo {
			return a, loadRepoData(a.ctx, a.rp.dir)
		}
		return a, a.statusCmd(a.visiblePaths(), false)
	case rescanTickMsg:
		return a, a.onRescanTick()
	case statusTickMsg:
		return a, a.onStatusTick()
	case previewMsg:
		if m.path == a.rs.previewPath {
			a.rs.preview = m.text
		}
	case repoDataMsg:
		return a, a.onRepoData(m)
	case mainMsg:
		if m.key == a.rp.mainKey {
			a.rp.main = m.text
		}
	case opMsg:
		return a, a.onOp(m)
	case bulkMsg:
		return a, a.onBulk(m)
	case afterMsg:
		a.busy = ""
		if m.err != nil {
			a.setToast(m.err.Error(), true)
			return a, nil
		}
		return a, m.next(m.ok)
	case checkMsg:
		a.busy = ""
		if m.err != nil {
			a.setToast(m.err.Error(), true)
			return a, nil
		}
		return a, m.next(m.pushed)
	case execDoneMsg:
		if m.err != nil {
			a.setToast(m.err.Error(), true)
		}
		return a, a.reloadCurrent()
	case tea.KeyMsg:
		return a, a.onKey(m)
	default:
		if a.ed != nil { // cursor blink and other input internals
			return a, a.ed.update(msg)
		}
	}
	return a, nil
}

func (a *App) onKey(k tea.KeyMsg) tea.Cmd {
	s := k.String()
	global := lookup(scGlobal, s)
	if global == actQuit {
		return tea.Quit
	}
	a.toast = ""
	if a.ed != nil {
		return a.editorKey(k)
	}
	if a.p != nil {
		return a.promptKey(k)
	}
	if a.report != nil {
		return a.reportKey(k)
	}
	if a.picker != nil {
		return a.pickerKey(k)
	}
	if a.scr == scrRepos && a.rs.filtering {
		return a.reposKey(k) // every key belongs to the search query, "?" too
	}
	if global == actThemes {
		a.openPicker()
		return nil
	}
	if global == actHelp {
		a.report = &report{title: "Keys", lines: helpLines()}
		return nil
	}
	if global == actErrorDetails {
		if a.lastErr == nil {
			a.setToast("no error to show", false)
		} else {
			a.report = a.lastErr
			a.report.off = 0
		}
		return nil
	}
	if a.scr == scrRepos {
		return a.reposKey(k)
	}
	return a.repoKey(k)
}

func (a *App) promptKey(k tea.KeyMsg) tea.Cmd {
	p := a.p
	if p.confirm {
		switch k.String() {
		case "y", "Y", "enter":
			a.p = nil
			return p.onOK("")
		case "n", "N", "esc":
			a.p = nil
		}
		return nil
	}
	switch k.Type {
	case tea.KeyEnter:
		v := strings.TrimSpace(string(p.input))
		if v == "" && !p.allowEmpty {
			return nil
		}
		a.p = nil
		return p.onOK(v)
	case tea.KeyEsc:
		a.p = nil
	case tea.KeyBackspace:
		if len(p.input) > 0 {
			p.input = p.input[:len(p.input)-1]
		}
	case tea.KeyCtrlU:
		p.input = nil
	case tea.KeySpace:
		p.input = append(p.input, ' ')
	case tea.KeyRunes:
		p.input = append(p.input, k.Runes...)
	}
	return nil
}

func (a *App) ask(title, initial string, ok func(string) tea.Cmd) {
	a.p = &prompt{title: title, input: []rune(initial), onOK: ok}
}

func (a *App) confirm(text string, ok func() tea.Cmd) {
	a.p = &prompt{title: text, confirm: true, onOK: func(string) tea.Cmd { return ok() }}
}

func (a *App) setToast(s string, isErr bool) {
	a.toast = strings.ReplaceAll(s, "\n", " ")
	a.toastErr = isErr
}

// doMsg runs fn in the background; the result arrives as an opMsg.
func (a *App) doMsg(label string, fn func() (string, error)) tea.Cmd {
	a.busy = label + "…"
	return func() tea.Msg {
		s, err := fn()
		return opMsg{label: label, msg: s, err: err}
	}
}

// check runs a history-rewrite pre-flight check in the background and hands the
// remote branches that already contain the affected commits to next.
func (a *App) check(fn func() ([]string, error), next func(pushed []string) tea.Cmd) tea.Cmd {
	a.busy = "checking…"
	return func() tea.Msg {
		pushed, err := fn()
		return checkMsg{pushed: pushed, err: err, next: next}
	}
}

// afterMsg carries the result of a background yes/no check started by App.after.
type afterMsg struct {
	ok   bool
	err  error
	next func(bool) tea.Cmd
}

// after runs a quick check in the background and hands its answer to next.
func (a *App) after(fn func() (bool, error), next func(bool) tea.Cmd) tea.Cmd {
	a.busy = "checking…"
	return func() tea.Msg {
		ok, err := fn()
		return afterMsg{ok: ok, err: err, next: next}
	}
}

// confirmRewrite asks question (skipped when empty) and, if the commits are
// already pushed, asks a second time with a force-push warning.
func (a *App) confirmRewrite(question string, pushed []string, run func() tea.Cmd) tea.Cmd {
	second := func() tea.Cmd {
		if len(pushed) == 0 {
			return run()
		}
		a.confirm("⚠ Already pushed to "+strings.Join(pushed, ", ")+". Rewriting will need a force push (P offers it afterwards). Continue?", run)
		return nil
	}
	if question == "" {
		return second()
	}
	a.confirm(question, second)
	return nil
}

func (a *App) do(label string, fn func() error) tea.Cmd {
	return a.doMsg(label, func() (string, error) { return "", fn() })
}

func (a *App) onOp(m opMsg) tea.Cmd {
	a.busy = ""
	switch {
	case m.err != nil:
		a.lastErr = errorReport(m.label, m.err)
		msg := m.label + ": " + m.err.Error()
		if len(a.lastErr.lines) > 3 {
			msg += "  ·  ! details"
		}
		a.setToast(msg, true)
	case m.msg != "":
		a.setToast(m.msg, false)
	default:
		a.setToast(m.label+" ✓", false)
	}
	return a.reloadCurrent()
}

func (a *App) reloadCurrent() tea.Cmd {
	if a.scr == scrRepo {
		a.rp.mainKey = ""
		return loadRepoData(a.ctx, a.rp.dir)
	}
	return a.gather(a.paths())
}

func (a *App) paths() []string {
	ps := make([]string, 0, len(a.rs.all))
	for _, r := range a.rs.all {
		ps = append(ps, r.Path)
	}
	return ps
}

func (a *App) gather(paths []string) tea.Cmd {
	ctx, jobs, recent := a.ctx, a.cfg.Jobs, a.st.Snapshot()
	return func() tea.Msg {
		repos := scan.Gather(ctx, paths, jobs, recent)
		_ = scan.SaveStatus(state.StatusCachePath(), repos)
		return reposMsg{repos: repos}
	}
}

// --------------------------------------------------------------------- views

func (a *App) View() string {
	if a.w == 0 || a.h == 0 {
		return ""
	}
	if a.w < minWidth || a.h < minHeight {
		lines := []string{fmt.Sprintf("gitnest needs at least %d×%d", minWidth, minHeight),
			fmt.Sprintf("(the window is %d×%d)", a.w, a.h)}
		if len(lines) > a.h {
			lines = lines[:a.h]
		}
		for i, l := range lines {
			lines[i] = fit(l, a.w)
		}
		return lipgloss.Place(a.w, a.h, lipgloss.Center, lipgloss.Center, strings.Join(lines, "\n"))
	}
	switch {
	case a.ed != nil:
		return a.viewEditor()
	case a.p != nil:
		return a.viewPrompt()
	case a.report != nil:
		return a.viewReport()
	case a.picker != nil:
		return a.viewPicker()
	case a.scr == scrRepos:
		return a.viewRepos()
	}
	return a.viewRepo()
}

func (a *App) viewPrompt() string {
	p := a.p
	w := clamp(a.w-8, 20, 70)
	var body string
	if p.confirm {
		body = bold(p.title) + "\n\n" + fg(cDim, "y/enter confirm · n/esc cancel")
	} else {
		body = bold(p.title) + "\n\n" + fg(cAccent, "› ") + string(p.input) + fg(cAccent, "▏") +
			"\n\n" + fg(cDim, "enter confirm · esc cancel")
	}
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cAccent).
		Padding(1, 2).Width(w).Render(body)
	return lipgloss.Place(a.w, a.h, lipgloss.Center, lipgloss.Center, box)
}

// footer renders a status line (busy/toast) plus a hints line: always 2 rows.
func (a *App) footer(hints string) string {
	t := ""
	switch {
	case a.busy != "":
		t = fg(cYellow, "⟳ "+a.busy)
	case a.toast != "" && a.toastErr:
		t = fg(cRed, a.toast)
	case a.toast != "":
		t = fg(cGreen, a.toast)
	}
	return fit(t, a.w) + "\n" + fit(fg(cDim, hints), a.w)
}
