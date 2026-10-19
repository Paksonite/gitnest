package ui

import (
	"testing"

	"github.com/muesli/termenv"
)

func TestColorProfile(t *testing.T) {
	for _, c := range []struct {
		name string
		base termenv.Profile
		env  map[string]string
		want termenv.Profile
	}{
		{"COLORTERM=truecolor", termenv.TrueColor, map[string]string{"TERM": "xterm-256color"}, termenv.TrueColor},
		{"SSH to Debian, no COLORTERM", termenv.ANSI256, map[string]string{"TERM": "xterm-256color"}, termenv.TrueColor},
		{"tmux", termenv.ANSI256, map[string]string{"TERM": "tmux-256color"}, termenv.TrueColor},
		{"Terminal.app on macOS 15", termenv.ANSI256, map[string]string{"TERM": "xterm-256color", "TERM_PROGRAM": "Apple_Terminal", "TERM_PROGRAM_VERSION": "455.1"}, termenv.ANSI256},
		{"Terminal.app on macOS 26+", termenv.ANSI256, map[string]string{"TERM": "xterm-256color", "TERM_PROGRAM": "Apple_Terminal", "TERM_PROGRAM_VERSION": "488"}, termenv.TrueColor},
		{"Linux console", termenv.ANSI, map[string]string{"TERM": "linux"}, termenv.ANSI},
		{"plain xterm", termenv.ANSI256, map[string]string{"TERM": "xterm"}, termenv.ANSI256},
		{"forced 256", termenv.TrueColor, map[string]string{"GITNEST_COLOR": "256"}, termenv.ANSI256},
		{"forced true colour", termenv.ANSI, map[string]string{"TERM": "linux", "GITNEST_COLOR": "truecolor"}, termenv.TrueColor},
		{"no colour", termenv.TrueColor, map[string]string{"GITNEST_COLOR": "none"}, termenv.Ascii},
	} {
		got, why := colorProfile(c.base, func(k string) string { return c.env[k] })
		if got != c.want {
			t.Errorf("%s: %s (%s), want %s", c.name, ProfileName(got), why, ProfileName(c.want))
		}
	}
}
