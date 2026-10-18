package ui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/config"
	"github.com/Paksonite/gitnest/internal/gitx"
	"github.com/Paksonite/gitnest/internal/scan"
)

// detailsCard is the preview of the selected repository in the list: who and
// where it is, how it stands, its latest commits and changes. d may be nil
// while the details load.
func detailsCard(r scan.Repo, label string, d *gitx.Details, w, h int, now time.Time) []string {
	inner := max(10, w)
	var out []string
	add := func(s string) { out = append(out, fit(s, inner)) }

	title := bold(r.Name)
	if label != "" {
		title += fg(cDim, " · "+label)
	}
	if r.Branch != "" {
		title += "  " + badge(branchGlyph(false)+r.Branch, cCyan)
	}
	if s := syncStyled(r.Info); s != "" {
		title += "  " + s
	}
	add(title)
	if g := gitx.RemoteGroup(r.RemoteURL); g != nil {
		add(fg(cDim, strings.Join(g, "/")+"/"+strings.TrimSuffix(filepath.Base(r.RemoteURL), ".git")))
	} else {
		add(fg(cDim, "no remote"))
	}
	add("")

	const keyW = 12
	field := func(k, v string) {
		if v != "" {
			add(fg(cDim, padRight(k, keyW)) + v)
		}
	}
	field("Path", config.Abbrev(r.Path))
	field("Remote", r.RemoteURL)
	if d != nil {
		up := d.Upstream
		if up == "" && !r.Detached {
			up = fg(cDim, "none")
		}
		field("Upstream", up)
	}
	field("Status", repoStatusText(r))
	if r.Stashes > 0 {
		field("Stashes", strconv.Itoa(r.Stashes))
	}
	if d != nil && len(d.Commits) > 0 {
		c := d.Commits[0]
		field("Last commit", relTime(c.Unix, now)+" ago · "+fg(authorColor(c.Author), c.Author))
	}
	if d != nil {
		field("Tag", d.Tag)
		if d.Description != "" {
			for i, l := range wrapWords(d.Description, inner-keyW) {
				if i == 2 {
					break
				}
				k := ""
				if i == 0 {
					k = "About"
				}
				add(fg(cDim, padRight(k, keyW)) + l)
			}
		}
	}
	if d == nil {
		add("")
		add(fg(cDim, "loading…"))
		return clip(out, h)
	}

	if len(d.Changes) > 0 {
		add("")
		add(bold(fg(cAccent, "Changes")))
		for _, c := range d.Changes {
			code, path := c, ""
			if len(c) > 3 {
				code, path = c[:2], c[3:]
			}
			col := cRed
			switch {
			case code == "??":
			case code[0] != ' ' && code[1] == ' ':
				col = cGreen
			case code[0] != ' ':
				col = cYellow
			}
			add(fg(col, code) + " " + path)
		}
		if d.More > 0 {
			add(fg(cDim, fmt.Sprintf("… and %d more", d.More)))
		}
	}

	if len(d.Commits) > 0 {
		add("")
		add(bold(fg(cAccent, "Recent commits")))
		ageW := 4
		for _, c := range d.Commits {
			age := fmt.Sprintf("%*s", ageW, relTime(c.Unix, now))
			subject := c.Subject
			if refs := compactRefs(c.Refs); refs != "" {
				subject = fg(cYellow, "("+refs+")") + " " + subject
			}
			ac := authorColor(c.Author)
			line := fg(ac, "●") + " " + fg(cYellow, c.Short) + " " +
				lipgloss.NewStyle().Bold(true).Foreground(ac).Render(initials(c.Author)) + "  "
			room := inner - lipgloss.Width(line) - ageW - 1
			add(line + fit(subject, max(1, room)) + " " + fg(cDim, age))
		}
	}
	return clip(out, h)
}

// repoStatusText says in words how a repository stands.
func repoStatusText(r scan.Repo) string {
	var parts []string
	if r.State != gitx.StateNone {
		parts = append(parts, fg(cRed, r.State.String()))
	}
	if r.Conflicts {
		parts = append(parts, fg(cRed, "conflicts"))
	}
	if s := syncStyled(r.Info); s != "" {
		parts = append(parts, s)
	}
	switch {
	case r.Changes == 0 && r.Upstream && r.Ahead == 0 && r.Behind == 0:
		parts = append(parts, fg(cGreen, "clean, in sync"))
	case r.Changes == 0:
		parts = append(parts, fg(cDim, "clean"))
	}
	return strings.Join(parts, "  ")
}

// groupCard is the preview of a group row in the list tree.
func groupCard(name string, paths []string, all map[string]scan.Repo, w, h int) []string {
	inner := max(10, w)
	var out []string
	dirty := 0
	var sc syncCount
	for _, p := range paths {
		if all[p].Changes > 0 {
			dirty++
		}
		sc = sc.add(all[p])
	}
	summary := fmt.Sprintf("%d repositories · %d dirty", len(paths), dirty)
	switch {
	case sc.tracked == 0:
	case sc.ahead == 0 && sc.behind == 0:
		summary = fg(cDim, summary+" · ") + fg(cGreen, "✓ all up to date")
	default:
		summary = fg(cDim, fmt.Sprintf("%s · %d to push · %d to pull", summary, sc.ahead, sc.behind))
	}
	out = append(out, fit(bold(name+"/"), inner), fit(fg(cDim, summary), inner), "")
	for _, p := range paths {
		r := all[p]
		mark := fg(cDim, "·")
		if r.Changes > 0 {
			mark = fg(cYellow, "●")
		}
		sync := syncStyled(r.Info)
		if r.Upstream && r.Ahead == 0 && r.Behind == 0 {
			sync = strings.TrimSpace(fg(cGreen, "✓") + " " + sync)
		}
		out = append(out, fit(mark+" "+r.Name+"  "+fg(cCyan, r.Branch)+" "+sync, inner))
	}
	return clip(out, h)
}

func clip(lines []string, h int) []string {
	if len(lines) > h {
		return lines[:h]
	}
	return lines
}

// wrapWords breaks s into lines of at most w columns at spaces.
func wrapWords(s string, w int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(s) {
		switch {
		case line == "":
			line = word
		case lipgloss.Width(line)+1+lipgloss.Width(word) <= w:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// compactRefs shortens git's %D for a narrow card: "HEAD -> main,
// origin/main, origin/HEAD, tag: v1" becomes "main, origin/main, v1".
func compactRefs(refs string) string {
	var out []string
	for _, r := range strings.Split(refs, ", ") {
		r = strings.TrimPrefix(strings.TrimPrefix(r, "HEAD -> "), "tag: ")
		if r == "" || r == "HEAD" || strings.HasSuffix(r, "/HEAD") {
			continue
		}
		out = append(out, r)
	}
	return strings.Join(out, ", ")
}
