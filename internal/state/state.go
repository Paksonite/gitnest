// Package state persists "recently used" timestamps and locates the scan cache.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

func dir(env, fallback string) string {
	if x := os.Getenv(env); x != "" {
		return filepath.Join(x, "gitnest")
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, fallback, "gitnest")
}

func statePath() string { return filepath.Join(dir("XDG_STATE_HOME", ".local/state"), "recent.json") }

// StatusCachePath holds the last known status of every repository, shown
// at startup until fresh status arrives.
func StatusCachePath() string { return filepath.Join(dir("XDG_CACHE_HOME", ".cache"), "status.json") }

// UpdateCachePath remembers the last release check, so GitHub is asked at
// most once a day.
func UpdateCachePath() string { return filepath.Join(dir("XDG_CACHE_HOME", ".cache"), "update.json") }

// ScanCachePath is where the repository scanner keeps its directory listings.
func ScanCachePath() string { return filepath.Join(dir("XDG_CACHE_HOME", ".cache"), "scan.json") }

type State struct {
	Recent   map[string]int64 `json:"recent"`
	ListTree bool             `json:"list_tree"` // repository list grouped by directory
}

// SetListTree remembers the repository list layout for the next start.
func (s *State) SetListTree(on bool) {
	s.ListTree = on
	_ = s.save()
}

func Load() *State {
	s := &State{Recent: map[string]int64{}}
	if b, err := os.ReadFile(statePath()); err == nil {
		_ = json.Unmarshal(b, s)
	}
	if s.Recent == nil {
		s.Recent = map[string]int64{}
	}
	return s
}

// Snapshot returns a copy of the recent map (safe to hand to goroutines).
func (s *State) Snapshot() map[string]int64 {
	m := make(map[string]int64, len(s.Recent))
	for k, v := range s.Recent {
		m[k] = v
	}
	return m
}

// Touch marks a repo as used now and saves the state.
func (s *State) Touch(path string) {
	s.Recent[path] = time.Now().Unix()
	if len(s.Recent) > 1000 {
		var oldest string
		var ot int64 = 1 << 62
		for k, v := range s.Recent {
			if v < ot {
				oldest, ot = k, v
			}
		}
		delete(s.Recent, oldest)
	}
	_ = s.save()
}

func (s *State) save() error {
	p := statePath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}
