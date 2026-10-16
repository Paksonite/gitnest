package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paksonite/gitnest/internal/config"
	"github.com/Paksonite/gitnest/internal/state"
	"github.com/Paksonite/gitnest/internal/update"
)

// Version is the running gitnest's version, set by main; development
// builds never look for updates.
var Version = "dev"

// updateMsg reports a release newer than the running gitnest.
type updateMsg struct{ version string }

// updateCheck asks GitHub (at most once a day, see update.Check) whether a
// newer release exists. Failures are quiet: being offline is not news.
func (a *App) updateCheck() tea.Cmd {
	if !update.IsRelease(Version) || !config.UpdateCheck(a.cfg) {
		return nil
	}
	ctx := a.ctx
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
	return "  " + fg(cGreen, "⬆ "+a.sh.update+" · gitnest update")
}
