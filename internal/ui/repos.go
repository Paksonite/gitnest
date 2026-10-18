package ui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/config"
	"github.com/Paksonite/gitnest/internal/gitx"
	"github.com/Paksonite/gitnest/internal/scan"
	"github.com/Paksonite/gitnest/internal/state"
)

type reposState struct {
	all         []scan.Repo
	known       []string // sorted repository paths from the last scan
	view        []int    // indexes into all, after filtering
	cur, off    int
	listH       int // rows of the list on screen (set while rendering)
	listW       int // width of the list panel (set while rendering)
	filter      string
	filtering   bool
	status      int         // status filter: stAll, stDirty… (F, d)
	host        string      // remote host filter ("" = all, "local" = no remote)
	favOnly     bool        // only starred repositories
	score       map[int]int // search score of each shown repository (rs.all index)
	loading     bool        // initial load or full status refresh running
	scanning    bool        // background scan for new/removed repositories running
	marked      map[string]bool
	isNew       map[string]bool   // found by a rescan, not opened yet
	label       map[string]string // tells apart clones that share a name, e.g. "forgejo"
	hl          map[string][]int  // name runes matched by the filter
	sortMode    int
	tree        bool            // group repositories (by directory or remote)
	group       int             // groupFolders or groupRemote, when tree
	rows        []listRow       // what is shown: repositories and, in tree mode, directories
	collapsed   map[string]bool // collapsed directories in tree mode
	details     *gitx.Details   // of previewPath, nil while loading
	previewPath string
}

type (
	scanMsg struct {
		paths []string
		full  bool
	}
	// statusMsg refreshes some repositories in place; asked lists the paths
	// whose status was requested (missing from repos = no longer a repo).
	statusMsg struct {
		repos []scan.Repo
		asked []string
		added bool
	}
	rescanTickMsg struct{}
	statusTickMsg struct{}
)

// loadRepos shows the repositories known from the scan cache right away and
// then rescans in the background; without a cache it scans first.
func (a *App) loadRepos() tea.Cmd {
	ctx, sc, jobs, recent := a.ctx, a.scanner, a.cfg.Jobs, a.st.Snapshot()
	a.rs.loading = true
	return func() tea.Msg {
		if sc.Load(state.ScanCachePath()) {
			// Show the last known list and status immediately; fresh status
			// and a rescan follow in the background.
			paths := sc.Repos()
			repos := scan.Cached(state.StatusCachePath(), paths, recent)
			scan.Sort(repos)
			return reposMsg{repos: repos, paths: paths, cached: true}
		}
		paths, _ := sc.Scan(false)
		_ = sc.Save(state.ScanCachePath())
		repos := scan.Gather(ctx, paths, jobs, recent)
		_ = scan.SaveStatus(state.StatusCachePath(), repos)
		return reposMsg{repos: repos, paths: paths}
	}
}

// scanCmd looks for new and removed repositories. Incremental scans only
// list directories whose mtime changed; full scans list everything.
func (a *App) scanCmd(full bool) tea.Cmd {
	if a.rs.scanning {
		return nil
	}
	a.rs.scanning = true
	sc := a.scanner
	return func() tea.Msg {
		paths, _ := sc.Scan(full)
		_ = sc.Save(state.ScanCachePath())
		return scanMsg{paths: paths, full: full}
	}
}

func (a *App) statusCmd(paths []string, added bool) tea.Cmd {
	if len(paths) == 0 {
		return nil
	}
	ctx, jobs, recent := a.ctx, a.cfg.Jobs, a.st.Snapshot()
	return func() tea.Msg {
		return statusMsg{repos: scan.Status(ctx, paths, jobs, recent), asked: paths, added: added}
	}
}

func (a *App) onRepos(m reposMsg) tea.Cmd {
	rs := &a.rs
	rs.loading = m.cached // cached rows are shown while fresh status loads
	keep := a.selectedKey()
	rs.all = m.repos
	if m.paths != nil {
		rs.known = m.paths
	}
	a.rebuildView(keep)
	var cmds []tea.Cmd
	if m.cached {
		cmds = append(cmds, a.gather(m.paths), a.scanCmd(false))
	} else if m.paths != nil {
		a.setToast(fmt.Sprintf("%d repositories found", len(m.repos)), false)
	}
	cmds = append(cmds, a.previewCmd())
	return tea.Batch(cmds...)
}

func (a *App) onScan(m scanMsg) tea.Cmd {
	rs := &a.rs
	rs.scanning = false
	added, removed := scan.Diff(rs.known, m.paths)
	rs.known = m.paths
	if len(added) == 0 && len(removed) == 0 {
		if m.full {
			a.setToast(fmt.Sprintf("%d repositories, no changes", len(m.paths)), false)
		}
		return nil
	}
	if len(removed) > 0 {
		gone := map[string]bool{}
		for _, p := range removed {
			gone[p] = true
			delete(rs.marked, p)
			delete(rs.isNew, p)
		}
		keep := rs.all[:0]
		for _, r := range rs.all {
			if !gone[r.Path] {
				keep = append(keep, r)
			}
		}
		rs.all = keep
		a.rebuildView(a.selectedKey())
	}
	for _, p := range added {
		rs.isNew[p] = true
	}
	a.setToast(changeSummary(added, removed), false)
	return tea.Batch(a.statusCmd(added, true), a.previewCmd())
}

func changeSummary(added, removed []string) string {
	var parts []string
	if n := len(added); n > 0 {
		names := make([]string, 0, 3)
		for _, p := range added {
			if len(names) == 3 {
				names = append(names, "…")
				break
			}
			names = append(names, filepath.Base(p))
		}
		parts = append(parts, fmt.Sprintf("+%d new: %s", n, strings.Join(names, ", ")))
	}
	if n := len(removed); n > 0 {
		parts = append(parts, fmt.Sprintf("−%d removed", n))
	}
	return strings.Join(parts, "  ·  ")
}

// onStatus merges refreshed repositories into the list without reordering
// it, so rows don't jump under the cursor; new repositories are sorted in.
func (a *App) onStatus(m statusMsg) tea.Cmd {
	rs := &a.rs
	keep := a.selectedKey()
	got := map[string]scan.Repo{}
	for _, r := range m.repos {
		got[r.Path] = r
	}
	asked := map[string]bool{}
	for _, p := range m.asked {
		asked[p] = true
	}
	before := len(rs.all)
	out := rs.all[:0]
	for _, r := range rs.all {
		if nr, ok := got[r.Path]; ok {
			out = append(out, nr)
			delete(got, r.Path)
		} else if !asked[r.Path] {
			out = append(out, r)
		}
	}
	for _, p := range m.asked { // newly found repositories, in scan order
		if r, ok := got[p]; ok {
			out = append(out, r)
		}
	}
	rs.all = out
	// A plain refresh updates rows in place: indexes in rs.view stay valid
	// and rows don't jump around under the cursor.
	if m.added || len(out) != before || rs.listFiltered() {
		a.rebuildView(keep)
	}
	return nil
}

// visiblePaths are the repositories currently on screen.
func (a *App) visiblePaths() []string {
	rs := &a.rs
	var ps []string
	for i := rs.off; i < len(rs.rows) && i < rs.off+rs.listH; i++ {
		if r := rs.rows[i]; r.repo >= 0 {
			ps = append(ps, rs.all[r.repo].Path)
		}
	}
	return ps
}

func (a *App) rescanTick() tea.Cmd {
	if a.cfg.RescanInterval <= 0 {
		return nil
	}
	return tea.Tick(time.Duration(a.cfg.RescanInterval)*time.Second, func(time.Time) tea.Msg { return rescanTickMsg{} })
}

func (a *App) statusTick() tea.Cmd {
	if a.cfg.StatusInterval <= 0 {
		return nil
	}
	return tea.Tick(time.Duration(a.cfg.StatusInterval)*time.Second, func(time.Time) tea.Msg { return statusTickMsg{} })
}

func (a *App) onRescanTick() tea.Cmd {
	return tea.Batch(a.scanCmd(false), a.rescanTick())
}

// onStatusTick refreshes the visible rows while the list is on screen and idle.
func (a *App) onStatusTick() tea.Cmd {
	var cmd tea.Cmd
	if a.scr == scrRepos && !a.rs.loading && a.busy == "" && a.p == nil {
		cmd = a.statusCmd(a.visiblePaths(), false)
	}
	return tea.Batch(cmd, a.statusTick())
}

// Sort modes for the repository list, cycled with "s".
const (
	sortRecent = iota // opened in gitnest recently, then newest commit
	sortName
	sortCommit // newest commit first
	sortStatus // operations in progress, conflicts, changes, unpushed first
	sortModes
)

var sortNames = [sortModes]string{"recent", "name", "last commit", "status"}

func lessRepo(mode int, a, b scan.Repo) bool {
	switch mode {
	case sortName:
		if x, y := strings.ToLower(a.Name), strings.ToLower(b.Name); x != y {
			return x < y
		}
		return a.Path < b.Path
	case sortCommit:
		if a.LastUnix != b.LastUnix {
			return a.LastUnix > b.LastUnix
		}
	case sortStatus:
		if x, y := attention(a), attention(b); x != y {
			return x > y
		}
	}
	if a.Used != b.Used {
		return a.Used > b.Used
	}
	if a.LastUnix != b.LastUnix {
		return a.LastUnix > b.LastUnix
	}
	return a.Name < b.Name
}

// attention ranks how much a repository needs looking at.
func attention(r scan.Repo) int {
	n := 0
	if r.State != gitx.StateNone {
		n += 8
	}
	if r.Conflicts {
		n += 4
	}
	if r.Changes > 0 {
		n += 2
	}
	if r.Ahead > 0 || r.Behind > 0 {
		n++
	}
	return n
}

// rebuildView filters and orders the list. With a filter, rows are ranked by
// match quality; otherwise (and for ties) by the sort mode.
func (a *App) rebuildView(keep string) {
	rs := &a.rs
	rs.label = disambiguate(rs.all)
	rs.hl = map[string][]int{}
	terms := strings.Fields(rs.filter)
	type cand struct{ idx, score int }
	rs.score = map[int]int{}
	var cs []cand
	for i, r := range rs.all {
		if !rs.keeps(r) || (rs.favOnly && !a.sh.fav[r.Path]) {
			continue
		}
		score, ok := a.matchRepo(r, terms)
		if !ok {
			continue
		}
		cs = append(cs, cand{i, score})
		rs.score[i] = score
	}
	sort.SliceStable(cs, func(x, y int) bool {
		if cs[x].score != cs[y].score {
			return cs[x].score > cs[y].score
		}
		return lessRepo(rs.sortMode, rs.all[cs[x].idx], rs.all[cs[y].idx])
	})
	rs.view = rs.view[:0]
	for _, c := range cs {
		rs.view = append(rs.view, c.idx)
	}
	a.buildRows()
	rs.cur = clamp(rs.cur, 0, len(rs.rows)-1)
	if keep != "" {
		for i, r := range rs.rows {
			if r.key(a) == keep {
				rs.cur = i
				break
			}
		}
	}
}

// matchRepo scores r against the filter terms; every term must match. A term
// matches the name (highlighted), or else the clone label, remote host and
// parent directories at a lower score; "@term" matches the current branch.
func (a *App) matchRepo(r scan.Repo, terms []string) (int, bool) {
	total := 0
	var namePos []int
	for _, t := range terms {
		if strings.HasPrefix(t, "@") {
			if t == "@" {
				continue
			}
			sc, _, ok := fuzzyMatch(r.Branch, t[1:])
			if !ok {
				return 0, false
			}
			total += sc / 2
			continue
		}
		if sc, pos, ok := fuzzyMatch(r.Name, t); ok {
			total += sc
			namePos = append(namePos, pos...)
			continue
		}
		// Elsewhere a term must match within one part of the clone label,
		// remote host or parent path ("grp" finds the folder "group"):
		// letters scattered over a whole host or path match almost anything.
		where := a.rs.label[r.Path] + " " + gitx.RemoteHost(r.RemoteURL) + " " + config.Abbrev(filepath.Dir(r.Path))
		best, found := 0, false
		for _, part := range strings.FieldsFunc(where, func(c rune) bool { return c == ' ' || c == '/' || c == '.' }) {
			if sc, _, ok := fuzzyMatch(part, t); ok && (!found || sc > best) {
				best, found = sc, true
			}
		}
		if !found {
			return 0, false
		}
		total += best / 3
	}
	if len(namePos) > 0 {
		a.rs.hl[r.Path] = namePos
	}
	return total, true
}

// disambiguate labels repositories that share a name with something that
// tells them apart: the remote host's first label ("forgejo" vs "gitea"),
// the full host, the parent directory, or the parent path.
func disambiguate(repos []scan.Repo) map[string]string {
	groups := map[string][]scan.Repo{}
	for _, r := range repos {
		k := strings.ToLower(r.Name)
		groups[k] = append(groups[k], r)
	}
	strategies := []func(scan.Repo) string{
		func(r scan.Repo) string {
			h, _, _ := strings.Cut(gitx.RemoteHost(r.RemoteURL), ".")
			return h
		},
		func(r scan.Repo) string { return gitx.RemoteHost(r.RemoteURL) },
		func(r scan.Repo) string { return filepath.Base(filepath.Dir(r.Path)) },
		func(r scan.Repo) string { return config.Abbrev(filepath.Dir(r.Path)) },
	}
	out := map[string]string{}
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		for _, label := range strategies {
			seen := map[string]bool{}
			ok := true
			for _, r := range g {
				l := label(r)
				if l == "" || seen[l] {
					ok = false
					break
				}
				seen[l] = true
			}
			if ok {
				for _, r := range g {
					out[r.Path] = label(r)
				}
				break
			}
		}
	}
	return out
}

func (a *App) selectedPath() string {
	rs := &a.rs
	if rs.cur >= 0 && rs.cur < len(rs.rows) && rs.rows[rs.cur].repo >= 0 {
		return rs.all[rs.rows[rs.cur].repo].Path
	}
	return ""
}

// selectedKey identifies the selected row, repository or directory.
func (a *App) selectedKey() string {
	rs := &a.rs
	if rs.cur >= 0 && rs.cur < len(rs.rows) {
		return rs.rows[rs.cur].key(a)
	}
	return ""
}

// selectedDir returns the selected row's index when it is a directory.
func (a *App) selectedDir() (int, bool) {
	rs := &a.rs
	if rs.cur >= 0 && rs.cur < len(rs.rows) && rs.rows[rs.cur].repo < 0 {
		return rs.cur, true
	}
	return 0, false
}

func (a *App) previewCmd() tea.Cmd {
	p := a.selectedPath()
	if p == "" {
		a.rs.details, a.rs.previewPath = nil, ""
		return nil
	}
	if p != a.rs.previewPath {
		a.rs.details = nil
	}
	a.rs.previewPath = p
	ctx := a.ctx
	return func() tea.Msg { return previewMsg{path: p, details: gitx.RepoDetails(ctx, p, 12)} }
}

func (a *App) openDir(path string) tea.Cmd {
	a.st.Touch(path)
	delete(a.rs.isNew, path)
	if a.scr != scrRepo {
		a.from = a.scr
	}
	a.scr = scrRepo
	a.rp = repoState{dir: path, name: filepath.Base(path)}
	return loadRepoData(a.ctx, path)
}

// saveCollapsed remembers the folded directories for the next start.
func (a *App) saveCollapsed() {
	dirs := make([]string, 0, len(a.rs.collapsed))
	for d := range a.rs.collapsed {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	a.st.SetCollapsed(dirs)
}

func (a *App) moveRepos(delta int) tea.Cmd {
	rs := &a.rs
	rs.cur = clamp(rs.cur+delta, 0, len(rs.rows)-1)
	return a.previewCmd()
}

// targets returns marked repos, or fallback() when nothing is marked.
func (a *App) targets(fallback func() []string) []string {
	var ps []string
	for _, r := range a.rs.all {
		if a.rs.marked[r.Path] {
			ps = append(ps, r.Path)
		}
	}
	if len(ps) == 0 {
		return fallback()
	}
	return ps
}

func (a *App) reposKey(k tea.KeyMsg) tea.Cmd {
	if a.rs.filtering {
		return a.filterKey(k)
	}
	return a.listAction(lookup(scList, k.String()))
}

// listAction runs one of the list's actions, from a key or a menu.
func (a *App) listAction(act action) tea.Cmd {
	rs := &a.rs
	ctx := a.ctx
	switch act {
	case actUp:
		return a.moveRepos(-1)
	case actDown:
		return a.moveRepos(1)
	case actPageUp:
		return a.moveRepos(-10)
	case actPageDown:
		return a.moveRepos(10)
	case actTop:
		return a.moveRepos(-len(rs.rows))
	case actBottom:
		return a.moveRepos(len(rs.rows))
	case actQuit:
		return tea.Quit
	case actCdQuit:
		return a.cdQuit(a.selectedPath())
	case actClearSearch:
		if rs.filter != "" {
			rs.filter = ""
			a.rebuildView(a.selectedKey())
			return a.previewCmd()
		}
	case actSearch: // a new search; tab kept the old one on screen until now
		rs.filtering = true
		if rs.filter != "" {
			rs.filter = ""
			a.rebuildView(a.selectedKey())
		}
	case actOpen:
		if i, ok := a.selectedDir(); ok { // enter on a directory folds it
			d := rs.rows[i].dir
			if rs.collapsed[d] {
				delete(rs.collapsed, d)
			} else {
				rs.collapsed[d] = true
			}
			a.saveCollapsed()
			a.rebuildView("d:" + d)
			return nil
		}
		if p := a.selectedPath(); p != "" {
			return a.openDir(p)
		}
	case actToggleTree: // flat → by folder → by remote → flat
		switch {
		case !rs.tree:
			rs.tree, rs.group = true, groupFolders
		case rs.group == groupFolders:
			rs.group = groupRemote
		default:
			rs.tree = false
		}
		a.setListView(rs.tree, rs.group)
		a.setToast("list: "+a.listViewName()+" (` changes)", false)
		return a.previewCmd()
	case actDirtyOnly:
		if rs.status == stDirty {
			rs.status = stAll
		} else {
			rs.status = stDirty
		}
		a.rebuildView(a.selectedKey())
		return a.previewCmd()
	case actFilters:
		a.openFilters()
		return nil
	case actSort:
		rs.sortMode = (rs.sortMode + 1) % sortModes
		a.rebuildView(a.selectedKey())
		a.setToast("sorted by "+sortNames[rs.sortMode], false)
		return a.previewCmd()
	case actMark:
		if i, ok := a.selectedDir(); ok { // a directory marks (or unmarks) everything below
			paths := a.reposUnder(i)
			all := true
			for _, p := range paths {
				all = all && rs.marked[p]
			}
			for _, p := range paths {
				if all {
					delete(rs.marked, p)
				} else {
					rs.marked[p] = true
				}
			}
		} else if p := a.selectedPath(); p != "" {
			if rs.marked[p] {
				delete(rs.marked, p)
			} else {
				rs.marked[p] = true
			}
		}
		return a.moveRepos(1)
	case actRescan:
		a.setToast("rescanning all roots…", false)
		return a.scanCmd(true)
	case actRefreshList:
		return a.gather(a.paths())
	case actFetchMany:
		return a.bulk("fetch", "fetched", a.targets(a.paths), func(p string) error { return gitx.Fetch(ctx, p) })
	case actPullMany:
		ps := a.targets(func() []string {
			if p := a.selectedPath(); p != "" {
				return []string{p}
			}
			return nil
		})
		return a.bulk("pull", "pulled", ps, func(p string) error { return gitx.Pull(ctx, p, false) })
	case actPullAll:
		return a.startPullAll()
	case actActions:
		a.openActions()
	case actViews:
		a.openViews()
	case actDashboard:
		return a.openDashboard()
	case actBrowse:
		if p := a.selectedPath(); p != "" {
			return tea.Batch(a.openDir(p), a.openBrowser(p))
		}
	case actFavorite:
		return a.toggleFavorite()
	case actUnmarkAll:
		rs.marked = map[string]bool{}
	case actMarkAll:
		for _, i := range rs.view {
			rs.marked[rs.all[i].Path] = true
		}
	}
	return nil
}

// filterKey edits the search query. Every printable key, including j, k, q
// and ?, goes into the query; enter opens the selected repository, tab keeps
// the filter and returns to navigation, esc clears it.
func (a *App) filterKey(k tea.KeyMsg) tea.Cmd {
	rs := &a.rs
	refilter := func() tea.Cmd {
		a.rebuildView("")
		rs.cur = 0
		for i, r := range rs.rows { // in the tree, land on the best match, not its folder
			if r.repo >= 0 {
				rs.cur = i
				break
			}
		}
		return a.previewCmd()
	}
	switch lookup(scFilter, k.String()) {
	case actCancelFilter:
		rs.filtering, rs.filter = false, ""
		a.rebuildView(a.selectedKey())
		return a.previewCmd()
	case actOpenMatch:
		rs.filtering = false
		if p := a.selectedPath(); p != "" {
			return a.openDir(p)
		}
		return nil
	case actKeepFilter:
		rs.filtering = false
		return nil
	case actUp:
		return a.moveRepos(-1)
	case actDown:
		return a.moveRepos(1)
	case actDeleteChar:
		if r := []rune(rs.filter); len(r) > 0 {
			rs.filter = string(r[:len(r)-1])
			return refilter()
		}
		return nil
	case actClearQuery:
		rs.filter = ""
		return refilter()
	case actDeleteWord:
		f := strings.TrimRight(rs.filter, " ")
		if i := strings.LastIndex(f, " "); i >= 0 {
			rs.filter = f[:i+1]
		} else {
			rs.filter = ""
		}
		return refilter()
	}
	switch k.Type {
	case tea.KeyRunes:
		rs.filter += string(k.Runes)
		return refilter()
	case tea.KeySpace:
		rs.filter += " "
		return refilter()
	}
	return nil
}

// ---------------------------------------------------------------------- view

func syncStyled(i gitx.Info) string {
	var parts []string
	if i.Upstream {
		if i.Ahead > 0 {
			parts = append(parts, fg(cGreen, fmt.Sprintf("↑%d", i.Ahead)))
		}
		if i.Behind > 0 {
			parts = append(parts, fg(cRed, fmt.Sprintf("↓%d", i.Behind)))
		}
	} else if !i.Detached {
		parts = append(parts, fg(cDim, "∅"))
	}
	if i.Changes > 0 {
		parts = append(parts, fg(cYellow, fmt.Sprintf("~%d", i.Changes)))
	}
	return strings.Join(parts, " ")
}

// relTime is a compact age: 45s, 12m, 5h, 3d, 6w, 4mo, 2y.
func relTime(unix int64, now time.Time) string {
	if unix == 0 {
		return "—"
	}
	d := now.Sub(time.Unix(unix, 0))
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 14*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d < 60*24*time.Hour:
		return fmt.Sprintf("%dw", int(d.Hours()/24/7))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(d.Hours()/24/30))
	}
	return fmt.Sprintf("%dy", int(d.Hours()/24/365))
}

type rowOpts struct {
	marked, isNew bool
	fav           bool   // starred
	label         string // clone label, shown after the name
	hl            []int  // name runes matched by the filter
	nameW         int
	now           time.Time
	indent        int // tree depth
}

func repoRow(r scan.Repo, o rowOpts) rowItem {
	sel := " "
	pad := strings.Repeat("  ", o.indent)
	if o.marked {
		sel = "✓"
	}
	suffix, styledSuffix := "", ""
	if o.label != "" {
		suffix += " · " + o.label
		styledSuffix += fg(cDim, " · "+o.label)
	}
	if o.fav {
		suffix += " ★"
		styledSuffix += fg(cYellow, " ★")
	}
	if o.isNew {
		suffix += " ✦"
		styledSuffix += fg(cAccent, " ✦")
	}
	hlStyle := lipgloss.NewStyle().Bold(true).Foreground(cAccent).Underline(true)
	name := fit(pad+r.Name+suffix, o.nameW)
	styledName := fit(pad+highlight(r.Name, o.hl, bold, func(s string) string { return hlStyle.Render(s) })+styledSuffix, o.nameW)
	br := fit(r.Branch, 16)
	sp, ss := statusCell(r)
	rel := fmt.Sprintf("%6s", relTime(r.LastUnix, o.now))
	plain := sel + " " + name + " " + br + " " + fit(sp, statusW) + " " + rel
	styled := fg(cGreen, sel) + " " + styledName + " " + fg(cCyan, br) + " " +
		fit(ss, statusW) + " " + fg(cDim, rel)
	return rowItem{plain: plain, styled: styled}
}

// statusW is the width of the list's status column.
const statusW = 12

// statusCell is a repository's state in the list's status column, the
// one place it is shown: an operation in progress or a conflict (!), else
// changed files (~3), commits to push (↑1) and pull (↓2), no upstream (∅),
// or ✓ when clean and in sync.
func statusCell(r scan.Repo) (plain, styled string) {
	switch {
	case r.State != gitx.StateNone:
		s := "! " + strings.ToLower(r.State.String())
		return s, fg(cRed, s)
	case r.Conflicts:
		return "! conflict", fg(cRed, "! conflict")
	}
	plain, styled = r.Info.Sync(), syncStyled(r.Info)
	if plain == "" && r.Upstream {
		return "✓", fg(cGreen, "✓")
	}
	return plain, styled
}

func (a *App) viewRepos() string {
	rs := &a.rs
	bodyH := a.h - 3
	dirty := 0
	for _, r := range rs.all {
		if r.Changes > 0 {
			dirty++
		}
	}

	head := brand() +
		fg(cDim, fmt.Sprintf("  %d repos · %d dirty", len(rs.all), dirty))
	head += a.copiedBadge() + a.updateBadge()
	if rs.sortMode != sortRecent {
		head += "  " + badge("sort: "+sortNames[rs.sortMode], cDim)
	}
	if rs.status != stAll {
		head += "  " + badge(statusFilterTags[rs.status], cYellow)
	}
	if rs.host != "" {
		head += "  " + badge(rs.host, cCyan)
	}
	if rs.favOnly {
		head += "  " + badge("★ favorites", cYellow)
	}
	if rs.filtering || rs.filter != "" {
		cursor := ""
		if rs.filtering {
			cursor = "▏"
		}
		head += "  " + fg(cAccent, "/"+rs.filter+cursor)
	}
	if rs.loading && len(rs.all) == 0 {
		head += "  " + fg(cYellow, "⟳ scanning…")
	} else if rs.loading {
		head += "  " + fg(cYellow, "⟳ refreshing…")
	}

	listW := a.w
	if a.w >= 90 {
		listW = a.w * 3 / 5
	}
	inner := panelInner(listW)
	nameW := clamp(inner-39, 10, 60)
	now := time.Now()
	lh := bodyH - 4 // borders, panel title and the column headers
	rs.listH, rs.listW = lh, listW
	rs.off = scrollTo(rs.cur, rs.off, lh)
	// Only the rows on screen are built: styling every row of a long list
	// on each frame is what makes big lists sluggish.
	end := min(rs.off+lh, len(rs.rows))
	var items []rowItem
	for _, row := range rs.rows[min(rs.off, end):end] {
		if row.repo < 0 {
			items = append(items, dirRow(row, rs.collapsed[row.dir] && rs.filter == "", inner, nameW))
			continue
		}
		r := rs.all[row.repo]
		items = append(items, repoRow(r, rowOpts{marked: rs.marked[r.Path], isNew: rs.isNew[r.Path], fav: a.sh.fav[r.Path],
			label: rs.label[r.Path], hl: rs.hl[r.Path], nameW: nameW, now: now, indent: row.depth}))
	}
	lines := append([]string{listHeader(nameW)}, renderRows(items, rs.cur-rs.off, 0, lh, inner, true)...)
	if len(rs.view) == 0 {
		msg := "No repositories match."
		if rs.loading {
			msg = "Scanning…"
		} else if len(a.cfg.Roots) == 0 {
			msg = "No directories to scan — run: gitnest roots add ~/repos"
		} else if len(rs.all) == 0 {
			msg = "No repositories found under " + rootsList(a.cfg.Roots)
		}
		lines = []string{fg(cDim, msg)}
	}
	note := fmt.Sprintf("%d · %s", len(rs.view), a.listViewName())
	if len(rs.view) != len(rs.all) {
		note = fmt.Sprintf("%d of %d · %s", len(rs.view), len(rs.all), a.listViewName())
	}
	list := panel(subtitle("Repositories", note), lines, listW, bodyH, true)

	body := list
	if a.w >= 90 {
		var pl []string
		pw, ph := panelInner(a.w-listW), bodyH-3
		title := "Details"
		if p := a.selectedPath(); p != "" {
			r := rs.all[rs.rows[rs.cur].repo]
			var d *gitx.Details
			if p == rs.previewPath {
				d = rs.details
			}
			pl = detailsCard(r, rs.label[p], d, pw, ph, now)
		} else if i, ok := a.selectedDir(); ok {
			repos := map[string]scan.Repo{}
			for _, r := range rs.all {
				repos[r.Path] = r
			}
			title = "Group"
			pl = groupCard(rs.rows[i].name, a.reposUnder(i), repos, pw, ph)
		}
		body = lipgloss.JoinHorizontal(lipgloss.Top, list, panel(title, pl, a.w-listW, bodyH, false))
	}

	footer := hints(scList, scPanes, scGlobal)
	if rs.filtering {
		footer = "type to search (@branch)  ·  ↑/↓ move  ·  " + hints(scFilter)
	}
	return strings.Join([]string{fit(head, a.w), body, a.footer(footer)}, "\n")
}

// listHeader names the columns of repoRow: the age is that of the last commit.
func listHeader(nameW int) string {
	return fg(cDim, "  "+fit("repository", nameW)+" "+fit("branch", 16)+" "+fit("status", statusW)+" "+fmt.Sprintf("%6s", "commit"))
}

func rootsList(roots []string) string {
	out := make([]string, len(roots))
	for i, r := range roots {
		out[i] = config.Abbrev(r)
	}
	return strings.Join(out, ", ")
}
