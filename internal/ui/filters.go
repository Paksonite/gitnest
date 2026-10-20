package ui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paksonite/gitnest/v2/internal/gitx"
	"github.com/Paksonite/gitnest/v2/internal/scan"
)

// Status filters of the repository list (F, or d for dirty).
const (
	stAll = iota
	stDirty
	stStashed
	stAhead
	stBehind
	stNoRemote
	statusFilters
)

var statusFilterNames = [statusFilters]string{
	stAll: "Any status", stDirty: "Dirty", stStashed: "Stashed changes",
	stAhead: "Ahead of remote", stBehind: "Behind remote", stNoRemote: "No remote",
}

// statusFilterTags name an active filter in the list header.
var statusFilterTags = [statusFilters]string{
	stDirty: "dirty only", stStashed: "stashed", stAhead: "ahead", stBehind: "behind", stNoRemote: "no remote",
}

func statusMatch(f int, r scan.Repo) bool {
	switch f {
	case stDirty:
		return r.Changes > 0
	case stStashed:
		return r.Stashes > 0
	case stAhead:
		return r.Upstream && r.Ahead > 0
	case stBehind:
		return r.Upstream && r.Behind > 0
	case stNoRemote:
		return r.RemoteURL == ""
	}
	return true
}

// repoHost is the remote host a repository is filed under ("local" without
// a remote).
func repoHost(r scan.Repo) string {
	if h := gitx.RemoteHost(r.RemoteURL); h != "" {
		return h
	}
	return "local"
}

// filterSummary names the active filters: "no remote · github.com".
func (a *App) filterSummary() string {
	rs := &a.rs
	var parts []string
	if rs.favOnly {
		parts = append(parts, "favorites only")
	}
	if rs.status != stAll {
		parts = append(parts, statusFilterTags[rs.status])
	}
	if rs.host != "" {
		parts = append(parts, rs.host)
	}
	return strings.Join(parts, " · ")
}

// listFiltered reports whether a status or host filter hides repositories.
func (rs *reposState) listFiltered() bool { return rs.status != stAll || rs.host != "" || rs.favOnly }

// keeps reports whether the status and host filters let r through.
func (rs *reposState) keeps(r scan.Repo) bool {
	return statusMatch(rs.status, r) && (rs.host == "" || repoHost(r) == rs.host)
}

// filterItems are the filters of the view menu: status filters and remote
// hosts, each with how many repositories it keeps; choosing one applies it
// at once.
func (a *App) filterItems() []menuItem {
	rs := &a.rs
	var counts [statusFilters]int
	hosts := map[string]int{}
	for _, r := range rs.all {
		for f := range counts {
			if statusMatch(f, r) {
				counts[f]++
			}
		}
		hosts[repoHost(r)]++
	}
	apply := func() tea.Cmd {
		a.rebuildView(a.selectedKey())
		a.saveListView()
		return a.previewCmd()
	}
	m := &menu{}
	m.items = append(m.items, menuHeader("Status filter"))
	for f := 0; f < statusFilters; f++ {
		m.items = append(m.items, menuItem{label: statusFilterNames[f], right: countLabel(counts[f]), stay: true,
			on:   func() bool { return rs.status == f },
			pick: func() tea.Cmd { rs.status = f; return apply() }})
	}
	names := make([]string, 0, len(hosts))
	for h := range hosts {
		names = append(names, h)
	}
	sort.Slice(names, func(i, j int) bool {
		if hosts[names[i]] != hosts[names[j]] {
			return hosts[names[i]] > hosts[names[j]]
		}
		return names[i] < names[j]
	})
	m.items = append(m.items, menuHeader("Remote filter"), menuItem{label: "Any remote", right: countLabel(len(rs.all)), stay: true,
		on:   func() bool { return rs.host == "" },
		pick: func() tea.Cmd { rs.host = ""; return apply() }})
	for _, h := range names {
		m.items = append(m.items, menuItem{label: h, right: countLabel(hosts[h]), stay: true,
			on:   func() bool { return rs.host == h },
			pick: func() tea.Cmd { rs.host = h; return apply() }})
	}
	return m.items
}
