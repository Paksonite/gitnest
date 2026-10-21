// Package ui is the bubbletea TUI: the repository list and the repository view.
package ui

import (
	"errors"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/Paksonite/gitnest/v2/internal/config"
	"github.com/Paksonite/gitnest/v2/internal/gitx"
	"github.com/Paksonite/gitnest/v2/internal/scan"
	"github.com/Paksonite/gitnest/v2/internal/state"
	"github.com/Paksonite/gitnest/v2/internal/theme"
	"github.com/Paksonite/gitnest/v2/internal/update"
)

// shared is what all panes see: the commit clipboard (copy in one pane,
// paste in another).
type shared struct {
	copied *clipboard
	update string          // newer gitnest release, shown in the header
	// updateNow: the user said yes to installing it; Run returns
	// ErrUpdateRequested once gitnest has quit.
	updateNow bool
	fav    map[string]bool // starred repositories (*)
}

// Smallest window the layout supports; below it a notice is shown instead.
const (
	minWidth  = 50
	minHeight = 12
)

type screen int

const (
	scrRepos screen = iota
	scrRepo
	scrDash // the dashboard: activity and what needs attention
)

// ----------------------------------------------------------------- messages

type reposMsg struct {
	repos  []scan.Repo
	paths  []string // repository paths when the list came from a scan or its cache
	cached bool     // paths came from the scan cache; a rescan should follow
}
type previewMsg struct {
	path    string
	details gitx.Details
}
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
	cdFile    string      // gn shell function: write the repository to cd into here
	sh        *shared     // state shared by every pane
	quitAll   bool        // quitting leaves gitnest entirely (o for gn), not just this pane
	plan      *planner    // interactive rebase being planned
	pull      *pullAll    // choosing what "pull all" updates
	menu      *menu       // filters, actions or the sort & view chooser
	browse    *browser    // every file of the open repository (B)
	dash      dashState   // the dashboard
	sel       textSel     // text selected with the mouse in the right-hand panel
	from      screen      // where esc in a repository goes back to
	lastErr   *report     // details of the last failed operation, shown with "!"
	toast     string
	toastErr  bool
	busy      string
}

// Run starts the TUI. startDir, if non-empty, opens that repository directly.
// cdFile, when set, is where "o" writes the chosen repository for the gn
// shell function to cd into.
func Run(cfg config.Config, noConfig bool, startDir, cdFile string) error {
	profile, _ := ColorProfile()
	lipgloss.SetColorProfile(profile)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := newApp(ctx, cfg, noConfig, startDir, cdFile)
	if startDir == "" {
		switch config.StartScreen(cfg) {
		case "dashboard":
			a.scr = scrDash
		case "last":
			if a.st.Screen != "list" { // the dashboard the first time
				a.scr = scrDash
			}
		}
	}
	opts := []tea.ProgramOption{tea.WithAltScreen(), tea.WithReportFocus()}
	if config.MouseOn(cfg) {
		opts = append(opts, tea.WithMouseCellMotion())
		// While gitnest takes the mouse, ask for the plain arrow pointer
		// rather than a text or selection one (OSC 22: kitty, Ghostty,
		// WezTerm, foot, xterm; other terminals ignore it), and give the
		// text pointer back afterwards.
		fmt.Fprint(os.Stdout, "\x1b]22;default\x1b\\")
		defer fmt.Fprint(os.Stdout, "\x1b]22;text\x1b\\")
	}
	_, err := tea.NewProgram(newShell(a), opts...).Run()
	if err == nil && a.sh.updateNow {
		return ErrUpdateRequested
	}
	return err
}

func newApp(ctx context.Context, cfg config.Config, noConfig bool, startDir, cdFile string) *App {
	a := &App{ctx: ctx, cfg: cfg, st: state.Load(), noConfig: noConfig, startDir: startDir, cdFile: cdFile,
		scanner: scan.NewScanner(scan.FromConfig(cfg)), sh: &shared{}}
	a.loadThemes()
	mode, _ := config.Icons(cfg)
	setIcons(mode)
	a.rs.marked = map[string]bool{}
	a.rs.isNew = map[string]bool{}
	a.rs.collapsed = map[string]bool{}
	for _, d := range a.st.Collapsed {
		a.rs.collapsed[d] = true
	}
	a.sh.fav = map[string]bool{}
	for _, p := range a.st.Favorites {
		a.sh.fav[p] = true
	}
	a.rs.tree = a.st.ListTree
	if a.st.ListGroup == groupNames[groupRemote] {
		a.rs.group = groupRemote
	}
	a.loadListView(a.st.ListView)
	return a
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

// Init starts a single App on its own (the program runs a Shell, see panes.go).
func (a *App) Init() tea.Cmd {
	return tea.Batch(a.startCmds(), a.rescanTick(), a.statusTick())
}

// startCmds loads the repository list, starts the live refresh, opens the
// start directory and shows start-up notes.
func (a *App) startCmds() tea.Cmd {
	cmds := []tea.Cmd{a.loadRepos(), repoTick()}
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
		cmd := a.onRepos(m)
		if a.scr == scrDash && !a.dash.ready && !a.dash.loading {
			cmd = tea.Batch(cmd, a.loadActivity())
		}
		return a, cmd
	case activityMsg:
		a.onActivity(m)
	case previewDueMsg:
		return a, a.onPreviewDue(m)
	case tea.MouseMsg:
		return a, a.onMouse(m)
	case scanMsg:
		return a, a.onScan(m)
	case statusMsg:
		return a, a.onStatus(m)
	case repoTickMsg:
		return a, a.onRepoTick()
	case stagingMsg:
		return a, a.onStaging(m)
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
	case browseFilesMsg:
		a.onBrowseFiles(m)
	case openPanesMsg: // a lone App (no shell) opens the first one itself
		if len(m.paths) > 0 {
			return a, a.openDir(m.paths[0])
		}
	case previewMsg:
		if m.path == a.rs.previewPath {
			a.rs.details = &m.details
		}
	case repoDataMsg:
		return a, a.onRepoData(m)
	case openFetchedMsg:
		return a, a.onOpenFetched(m)
	case mainMsg:
		if m.key == a.rp.mainKey {
			a.rp.main = m.text
		}
	case opMsg:
		return a, a.onOp(m)
	case bulkMsg:
		return a, a.onBulk(m)
	case pullScanMsg:
		return a, a.onPullScan(m)
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
		if b := a.browse; b != nil { // show the edited file as it is now
			b.path = ""
			b.show()
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
	a.sel.on = false
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
	if a.plan != nil {
		return a.plannerKey(k)
	}
	if a.pull != nil {
		return a.pullAllKey(k)
	}
	if a.menu != nil {
		return a.menuKey(k)
	}
	if a.browse != nil && global != actHelp && global != actThemes {
		return a.browserKey(k)
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
	switch a.scr {
	case scrRepos:
		return a.reposKey(k)
	case scrDash:
		return a.dashKey(k)
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

// ErrUpdateRequested is what Run returns when gitnest quit to update itself
// (yes to the new release's question): the caller runs the update.
var ErrUpdateRequested = errors.New("update requested")

func (a *App) confirm(text string, ok func() tea.Cmd) {
	a.p = &prompt{title: text, confirm: true, onOK: func(string) tea.Cmd { return ok() }}
}

func (a *App) setToast(s string, isErr bool) {
	a.toast = gitx.Redact(strings.ReplaceAll(s, "\n", " "))
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
		// The repository's row in the list follows at once (one git
		// status, ~20 ms), so the list is right when you go back to it
		// instead of only after every repository was read again.
		return tea.Batch(loadRepoData(a.ctx, a.rp.dir), a.statusCmd([]string{a.rp.dir}, false))
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

func (a *App) View() string { return a.withSelection(a.screen()) }

// screen renders the current screen.
func (a *App) screen() string {
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
	case a.plan != nil:
		return a.viewPlanner()
	case a.pull != nil:
		return a.viewPullAll()
	case a.menu != nil:
		return a.viewMenu()
	case a.browse != nil:
		return a.viewBrowser()
	case a.scr == scrRepos:
		return a.viewRepos()
	case a.scr == scrDash:
		return a.viewDashboard()
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
	// The hints, and the version in the bottom-right corner. The very last
	// column stays empty: writing the bottom-right cell makes some terminals
	// leave marks behind while the window is resized.
	ver := versionLabel()
	room := a.w - 1 - lipgloss.Width(ver) - 2
	if room < 20 { // a narrow window: the hints matter more
		return fit(t, a.w) + "\n" + fitHints(hints, a.w-1)
	}
	return fit(t, a.w) + "\n" + fitHints(hints, room) + "  " + fg(cDim, ver)
}

// versionLabel is the running version as the footer shows it: v1.2.0, or
// dev-<commit> for preview builds.
func versionLabel() string {
	if update.IsRelease(Version) {
		return "v" + strings.TrimPrefix(Version, "v")
	}
	return Version
}

// fitHints shortens a footer to w columns by leaving out hints from the end,
// most specific first, but keeps "? help" last so the rest is a key away.
func fitHints(hints string, w int) string {
	if lipgloss.Width(hints) <= w {
		return fit(hints, w)
	}
	parts := strings.Split(hints, "  ")
	var help string
	if last := parts[len(parts)-1]; strings.Contains(ansi.Strip(last), "? help") {
		help, parts = last, parts[:len(parts)-1]
	}
	for len(parts) > 1 {
		s := strings.Join(parts, "  ")
		if help != "" {
			s += "  " + help
		}
		if lipgloss.Width(s) <= w {
			return fit(s, w)
		}
		parts = parts[:len(parts)-1]
		for len(parts) > 1 && strings.TrimSpace(ansi.Strip(parts[len(parts)-1])) == "·" {
			parts = parts[:len(parts)-1] // no separator left dangling
		}
	}
	return fit(strings.Join(parts, "  "), w)
}

// cdQuit leaves gitnest so the gn shell function cds into path.
func (a *App) cdQuit(path string) tea.Cmd {
	if path == "" {
		return nil
	}
	if a.cdFile == "" {
		shell, rc := "zsh", "~/.zshrc"
		if filepath.Base(os.Getenv("SHELL")) == "bash" {
			shell, rc = "bash", "~/.bashrc"
		}
		a.setToast(fmt.Sprintf(`o needs the shell integration: add  eval "$(gitnest shell %s)"  to %s and open a new terminal`, shell, rc), true)
		return nil
	}
	if err := os.WriteFile(a.cdFile, []byte(path+"\n"), 0o600); err != nil {
		a.setToast("writing "+a.cdFile+": "+err.Error(), true)
		return nil
	}
	a.quitAll = true
	return tea.Quit
}
