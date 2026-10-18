package main

import (
	"runtime/debug"
	"strings"
)

// buildVersion fills in version information for builds made without the
// release ldflags, such as go install github.com/Paksonite/gitnest/v2/cmd/gitnest@v2.0.0:
// the module version, and the commit when the build has VCS data.
func buildVersion() (v, c, d string) {
	v, c, d = version, commit, date
	if v != "dev" {
		return v, c, d
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return v, c, d
	}
	if mv := info.Main.Version; mv != "" && mv != "(devel)" {
		v = strings.TrimPrefix(mv, "v")
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) >= 7 {
				c = s.Value[:7]
			}
		case "vcs.time":
			d = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				c += "-dirty"
			}
		}
	}
	return v, c, d
}
