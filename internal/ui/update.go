package ui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paksonite/gitnest/v2/internal/config"
	"github.com/Paksonite/gitnest/v2/internal/state"
	"github.com/Paksonite/gitnest/v2/internal/update"
)

// Version is the running gitnest's version, set by main; development
// builds never look for updates.
var Version = "dev"

// updateMsg reports a release newer than the running gitnest.
type updateMsg struct{ version string }

// updateCheck asks GitHub (at most once a day, see update.Check) whether a
// newer release exists; preview builds (dev-<commit>) ask hourly for a newer
// dev build. Failures are quiet: being offline is not news.
func (a *App) updateCheck() tea.Cmd {
	if !config.UpdateCheck(a.cfg) {
		return nil
	}
	ctx := a.ctx
	if strings.HasPrefix(Version, "dev-") {
		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			b, newer, err := update.CheckDev(ctx, Version, state.DevUpdateCachePath(), time.Hour)
			if err != nil || !newer {
				return nil
			}
			return updateMsg{b.Version()}
		}
	}
	if !update.IsRelease(Version) {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		rel, newer, err := update.Check(ctx, Version, state.UpdateCachePath(), 24*time.Hour)
		if err != nil || !newer {
			return nil
		}
		return updateMsg{rel.Version}
	}
}

// updateBadge is the header note about a newer release.
func (a *App) updateBadge() string {
	if a.sh.update == "" {
		return ""
	}
	return "  " + badge("⬆ "+a.sh.update+" · "+updateCommand(a.sh.update), cGreen)
}

// updateCommand is what installs version v: dev builds update gitnest-dev.
func updateCommand(v string) string {
	if strings.HasPrefix(v, "dev-") {
		return "gitnest-dev update"
	}
	return "gitnest update"
}
