package ui

// End-to-end tests: the app is driven with key presses on real repositories,
// and every command it returns is executed and its messages fed back, like
// the Bubble Tea runtime does (without a terminal).

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paksonite/gitnest/internal/config"
)

// drive runs cmd and everything that follows from it. Timer messages are
// dropped (ticks would loop forever); commands still running after a few
// seconds are abandoned.
func drive(t *testing.T, a tea.Model, cmd tea.Cmd) {
	t.Helper()
	queue := []tea.Cmd{cmd}
	for steps := 0; len(queue) > 0; steps++ {
		if steps > 500 {
			t.Fatal("too many messages: a command loop?")
		}
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		done := make(chan tea.Msg, 1)
		go func() { done <- c() }()
		var msg tea.Msg
		select {
		case msg = <-done:
		case <-time.After(3 * time.Second):
			continue // a tick or a long timer
		}
		inner := msg
		if pm, ok := msg.(paneMsg); ok {
			inner = pm.msg
		}
		if inner != nil && strings.Contains(reflect.TypeOf(inner).PkgPath(), "bubbles/cursor") {
			continue // cursor blinking re-arms itself forever
		}
		switch inner.(type) {
		case rescanTickMsg, statusTickMsg, repoTickMsg, shellRescanTickMsg, shellStatusTickMsg:
			continue
		case tea.QuitMsg:
			if _, isPane := msg.(paneMsg); !isPane {
				continue
			}
		}
		switch m := msg.(type) {
		case nil, tea.QuitMsg:
			continue
		case tea.BatchMsg:
			queue = append(queue, m...)
			continue
		}
		// tea.Sequence's message type is unexported: a slice of commands.
		if v := reflect.ValueOf(msg); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeOf(tea.Cmd(nil)) {
			for i := 0; i < v.Len(); i++ {
				queue = append(queue, v.Index(i).Interface().(tea.Cmd))
			}
			continue
		}
		_, next := a.Update(msg)
		queue = append(queue, next)
	}
}

// press sends keys: single runes, or names like "enter", "esc", "tab",
// "space", "ctrl+s"; "type:text" types text.
func press(t *testing.T, a tea.Model, keys ...string) {
	t.Helper()
	for _, k := range keys {
		var msg tea.KeyMsg
		switch {
		case strings.HasPrefix(k, "type:"):
			for _, r := range strings.TrimPrefix(k, "type:") {
				_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
				drive(t, a, cmd)
			}
			continue
		case k == "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case k == "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case k == "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		case k == "space":
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}
		case k == "ctrl+s":
			msg = tea.KeyMsg{Type: tea.KeyCtrlS}
		case k == "ctrl+n":
			msg = tea.KeyMsg{Type: tea.KeyCtrlN}
		case k == "ctrl+w":
			msg = tea.KeyMsg{Type: tea.KeyCtrlW}
		case k == "ctrl+g":
			msg = tea.KeyMsg{Type: tea.KeyCtrlG}
		case k == "ctrl+left":
			msg = tea.KeyMsg{Type: tea.KeyCtrlLeft}
		case k == "ctrl+right":
			msg = tea.KeyMsg{Type: tea.KeyCtrlRight}
		case k == "ctrl+up":
			msg = tea.KeyMsg{Type: tea.KeyCtrlUp}
		case k == "ctrl+down":
			msg = tea.KeyMsg{Type: tea.KeyCtrlDown}
		case k == "alt+right":
			msg = tea.KeyMsg{Type: tea.KeyRight, Alt: true}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		_, cmd := a.Update(msg)
		drive(t, a, cmd)
	}
}

func sh(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// e2eSetup makes HOME, config and two repositories: app (with a bare origin)
// and lib, under one root.
func e2eSetup(t *testing.T) (a *App, root, origin string) {
	home := t.TempDir()
	for k, v := range map[string]string{
		"HOME": home, "XDG_CONFIG_HOME": filepath.Join(home, ".config"), "XDG_STATE_HOME": filepath.Join(home, ".state"),
		"XDG_CACHE_HOME": filepath.Join(home, ".cache"), "GIT_CONFIG_GLOBAL": os.DevNull, "GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME": "Ann Example", "GIT_AUTHOR_EMAIL": "ann@example.com",
		"GIT_COMMITTER_NAME": "Ann Example", "GIT_COMMITTER_EMAIL": "ann@example.com", "GITNEST_ICONS": "",
	} {
		t.Setenv(k, v)
	}
	root = filepath.Join(home, "repos")
	seed := filepath.Join(home, "seed")
	os.MkdirAll(seed, 0o755)
	sh(t, seed, "init", "-q", "-b", "main")
	os.WriteFile(filepath.Join(seed, "README.md"), []byte("app\n"), 0o644)
	sh(t, seed, "add", ".")
	sh(t, seed, "commit", "-q", "-m", "initial")
	origin = filepath.Join(home, "origin.git")
	sh(t, home, "clone", "-q", "--bare", seed, origin)
	os.MkdirAll(root, 0o755)
	sh(t, root, "clone", "-q", origin, "app")
	lib := filepath.Join(root, "lib")
	os.MkdirAll(lib, 0o755)
	sh(t, lib, "init", "-q", "-b", "main")
	os.WriteFile(filepath.Join(lib, "lib.txt"), []byte("lib\n"), 0o644)
	sh(t, lib, "add", ".")
	sh(t, lib, "commit", "-q", "-m", "lib initial")

	if err := config.SetRoots([]string{root}); err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := config.Load()
	a = newApp(context.Background(), cfg, false, "", "")
	a.w, a.h = 140, 40
	drive(t, a, a.loadRepos())
	return a, root, origin
}

func TestE2EWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, origin := e2eSetup(t)
	app := filepath.Join(root, "app")
	lib := filepath.Join(root, "lib")
	if len(a.rs.all) != 2 {
		t.Fatalf("repositories: %d", len(a.rs.all))
	}

	// Search and open "app".
	press(t, a, "/", "type:app", "enter")
	if a.scr != scrRepo || a.rp.dir != app {
		t.Fatalf("not in app: scr=%v dir=%q", a.scr, a.rp.dir)
	}

	// A file created outside shows up through the live refresh.
	os.WriteFile(filepath.Join(app, "notes.txt"), []byte("one\ntwo\n"), 0o644)
	drive(t, a, a.onRepoTick())
	if n, ok := a.rp.fileNode(); !ok || n.Path != "notes.txt" {
		t.Fatalf("live refresh: %+v", a.rp.files)
	}

	// Stage one line of the new file, then everything, and commit.
	press(t, a, "enter")
	if a.rp.staging == nil {
		t.Fatal("enter on a file should open the staging view")
	}
	press(t, a, "space")
	if got := sh(t, app, "show", ":notes.txt"); got != "one" {
		t.Errorf("index after staging one line: %q", got)
	}
	press(t, a, "esc", "a", "c", "type:Add notes", "enter")
	if got := sh(t, app, "log", "-1", "--format=%s"); got != "Add notes" {
		t.Fatalf("commit: %q", got)
	}

	// Push, then reword the pushed commit and push again (force with lease).
	press(t, a, "P")
	if sh(t, origin, "log", "-1", "--format=%s", "main") != "Add notes" {
		t.Fatal("push did not reach origin")
	}
	press(t, a, "3", "r", "tab", "type:Why: test.", "ctrl+s")
	if a.p == nil || !strings.Contains(a.p.title, "Already pushed") {
		t.Fatalf("expected the pushed-commit warning, got %+v", a.p)
	}
	press(t, a, "y")
	if got := sh(t, app, "log", "-1", "--format=%B"); got != "Add notes\n\nWhy: test." {
		t.Fatalf("reword: %q", got)
	}
	press(t, a, "P")
	if a.p == nil || !strings.Contains(a.p.title, "diverged") {
		t.Fatalf("expected the force-push offer, got %+v (toast %q)", a.p, a.toast)
	}
	press(t, a, "y")
	if got := sh(t, origin, "log", "-1", "--format=%B", "main"); got != "Add notes\n\nWhy: test." {
		t.Fatalf("origin after force push: %q", got)
	}

	// Tag it and push the tag.
	press(t, a, "2", "]", "]", "n", "type:v1.0.0", "enter", "type:First release", "enter")
	if got := sh(t, app, "tag", "--list"); got != "v1.0.0" {
		t.Fatalf("tags: %q", got)
	}
	press(t, a, "P")
	if got := sh(t, origin, "tag", "--list"); got != "v1.0.0" {
		t.Fatalf("remote tags: %q", got)
	}

	// Copy the commit and paste it into lib (another repository).
	press(t, a, "3", "c", "esc")
	if a.scr != scrRepos || a.sh.copied == nil {
		t.Fatalf("back on the list with a copied commit: scr=%v copied=%+v", a.scr, a.sh.copied)
	}
	press(t, a, "/", "type:lib", "enter", "3", "v")
	if a.p == nil || !strings.Contains(a.p.title, "as patches from app") {
		t.Fatalf("paste prompt: %+v", a.p)
	}
	press(t, a, "y")
	if got := sh(t, lib, "log", "-1", "--format=%an|%s"); got != "Ann Example|Add notes" {
		t.Fatalf("lib after paste: %q", got)
	}

	// z undoes the paste.
	press(t, a, "z", "y")
	if got := sh(t, lib, "log", "-1", "--format=%s"); got != "lib initial" {
		t.Fatalf("after undo: %q", got)
	}

	// Interactive rebase in lib: add two commits, drop the older one.
	for _, n := range []string{"x", "y"} {
		os.WriteFile(filepath.Join(lib, n+".txt"), []byte(n), 0o644)
		sh(t, lib, "add", ".")
		sh(t, lib, "commit", "-q", "-m", "add "+n)
	}
	drive(t, a, loadRepoData(a.ctx, lib))
	press(t, a, "3", "j", "i", "j", "d", "enter")
	if a.p == nil || !strings.Contains(a.p.title, "Run the rebase plan on 2 commits") {
		t.Fatalf("plan confirmation: %+v", a.p)
	}
	press(t, a, "y")
	if got := sh(t, lib, "log", "--format=%s"); got != "add y\nlib initial" {
		t.Fatalf("after the plan: %q", got)
	}
}

func TestE2EList(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, _ := e2eSetup(t)

	// A clone made while gitnest runs is picked up by the next scan.
	sh(t, root, "clone", "-q", filepath.Join(root, "lib"), "lib-copy")
	drive(t, a, a.scanCmd(false))
	if len(a.rs.all) != 3 || !a.rs.isNew[filepath.Join(root, "lib-copy")] {
		t.Fatalf("new clone: %d repos, new=%v", len(a.rs.all), a.rs.isNew)
	}

	// Mark two repositories and pull: lib has no upstream, so the report opens.
	press(t, a, "s") // sort by name: app, lib, lib-copy
	press(t, a, "g", "space", "space", "p")
	if a.report == nil || !strings.Contains(a.report.title, "1 of 2 failed") {
		t.Fatalf("report: %+v (toast %q)", a.report, a.toast)
	}
	press(t, a, "esc")

	// Tree view groups by directory; with one level nothing nests.
	press(t, a, "`")
	if !a.rs.tree || len(a.rs.rows) != 3 || !a.st.ListTree {
		t.Errorf("tree: %v rows=%d", a.rs.tree, len(a.rs.rows))
	}

	// o writes the selected repository for the gn shell function.
	a.cdFile = filepath.Join(t.TempDir(), "cd")
	press(t, a, "g", "o")
	b, _ := os.ReadFile(a.cdFile)
	if strings.TrimSpace(string(b)) != filepath.Join(root, "app") {
		t.Errorf("cd file: %q", b)
	}
}

// TestMain shortens the live-refresh timer so tests don't wait for it (its
// messages are dropped by drive anyway).
func TestMain(m *testing.M) {
	liveRefresh = time.Millisecond
	os.Exit(m.Run())
}
