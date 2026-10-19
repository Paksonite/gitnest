package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// ColorProfile picks how many colours gitnest draws with, and says why.
//
// termenv trusts COLORTERM for true colour, but it is often missing where
// the terminal has true colour all the same: over SSH (it isn't sent), in
// tmux and screen, in some Linux terminals. The themes then fall to the
// nearest of 256 colours — a different, darker look than on the Mac. So a
// 256-colour TERM counts as true colour, as every current terminal draws it
// (tmux turns it into 256 colours itself where the outer terminal can't).
// Terminal.app before macOS 26 is the one that can't; it keeps 256.
// GITNEST_COLOR=truecolor|256|16|none decides instead.
func ColorProfile() (termenv.Profile, string) {
	return colorProfile(lipgloss.ColorProfile(), os.Getenv)
}

// colorProfile decides from what termenv found and the environment.
func colorProfile(p termenv.Profile, getenv func(string) string) (termenv.Profile, string) {
	switch strings.ToLower(getenv("GITNEST_COLOR")) {
	case "truecolor", "24bit":
		return termenv.TrueColor, "GITNEST_COLOR"
	case "256":
		return termenv.ANSI256, "GITNEST_COLOR"
	case "16":
		return termenv.ANSI, "GITNEST_COLOR"
	case "none", "0":
		return termenv.Ascii, "GITNEST_COLOR"
	}
	if p != termenv.ANSI256 {
		return p, "terminal"
	}
	if getenv("TERM_PROGRAM") == "Apple_Terminal" && !appleTerminalTrueColor(getenv("TERM_PROGRAM_VERSION")) {
		return p, "Terminal.app before macOS 26"
	}
	if t := getenv("TERM"); strings.HasSuffix(t, "-256color") || strings.HasSuffix(t, "-direct") {
		return termenv.TrueColor, "TERM=" + t + " without COLORTERM"
	}
	return p, "terminal"
}

// appleTerminalTrueColor: Terminal.app draws 24-bit colour from macOS 26
// (Terminal 460 and later).
func appleTerminalTrueColor(version string) bool {
	major, _, _ := strings.Cut(version, ".")
	n := 0
	for _, r := range major {
		if r < '0' || r > '9' {
			return false
		}
		n = n*10 + int(r-'0')
	}
	return n >= 460
}

// ProfileName is how gitnest doctor shows a profile.
func ProfileName(p termenv.Profile) string {
	switch p {
	case termenv.TrueColor:
		return "true colour"
	case termenv.ANSI256:
		return "256 colours"
	case termenv.ANSI:
		return "16 colours"
	}
	return "no colour"
}
