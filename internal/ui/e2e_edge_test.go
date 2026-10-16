package ui

// Edge cases, end to end: empty repositories, detached HEAD, no remote,
// huge diffs, unusual file names, slow networks and tiny windows.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Paksonite/gitnest/internal/gitx"
)

// newRepoIn makes an empty repository root/name and rescans, so the app
// lists it.
func newRepoIn(t *testing.T, a *App, root, name string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	sh(t, dir, "init", "-q", "-b", "main")
	drive(t, a, a.scanCmd(true))
	return dir
}

// everyKey presses every key of the repository view's panels once, in
// every panel, answering no to any question; nothing may panic.
func everyKey(t *testing.T, a *App) {
	t.Helper()
	keys := []string{"space", "a", "A", "s", "S", "e", "`", "n", "r", "d", "D", "u", "U", "f", "M", "R", "]", "[",
		"t", "i", "c", "v", "C", "X", "g", "b", "P", "F", "p", "z", "ctrl+j", "ctrl+k", "K", "J", "enter"}
	for _, panel := range []string{"1", "2", "3", "4"} {
		for _, k := range keys {
			if k == "e" {
				continue // would start an editor
			}
			press(t, a, panel, k)
			for i := 0; i < 3 && (a.p != nil || a.ed != nil || a.plan != nil || a.report != nil || a.rp.staging != nil || a.rp.mainFocus); i++ {
				press(t, a, "esc")
				if a.p != nil {
					press(t, a, "n")
				}
			}
			if a.scr != scrRepo {
				openRepo(t, a, a.rp.dir)
			}
			_ = a.View()
		}
	}
}

func TestE2EEmptyRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, _ := e2eSetup(t)
	empty := newRepoIn(t, a, root, "empty")
	press(t, a, "/", "type:empty", "enter")
	if a.rp.dir != empty {
		t.Fatalf("open: %q", a.rp.dir)
	}
	if v := a.View(); !strings.Contains(v, "main") || !strings.Contains(v, "Working tree clean") {
		t.Errorf("empty repository view:\n%s", v)
	}
	everyKey(t, a)

	// Stage and unstage in a repository without commits, then commit.
	writeFile(t, filepath.Join(empty, "first.txt"), "first\n")
	refresh(t, a)
	press(t, a, "1", "space")
	if got := sh(t, empty, "status", "--porcelain"); got != "A  first.txt" {
		t.Fatalf("stage: %q", got)
	}
	press(t, a, "space")
	if got := sh(t, empty, "status", "--porcelain"); got != "?? first.txt" {
		t.Fatalf("unstage without commits: %q (toast %q)", got, a.toast)
	}
	press(t, a, "enter", "space", "esc") // line staging of an untracked file
	if got := sh(t, empty, "show", ":first.txt"); got != "first" {
		t.Fatalf("line staging: %q", got)
	}
	press(t, a, "c", "type:First", "enter")
	if got := subjects(t, empty, "1"); got != "First" {
		t.Fatalf("first commit: %q (toast %q)", got, a.toast)
	}
	// Undoing the only commit makes the branch unborn again, changes staged.
	press(t, a, "3", "u", "y")
	if got := sh(t, empty, "status", "--porcelain"); got != "A  first.txt" {
		t.Fatalf("undo of the root commit: %q (toast %q)", got, a.toast)
	}
	if _, err := gitx.Log(a.ctx, empty, 1); err == nil && len(a.rp.commits) != 0 {
		t.Errorf("commits after undoing the root commit: %d", len(a.rp.commits))
	}
	press(t, a, "1", "c", "type:First again", "enter")
	if got := subjects(t, empty, "1"); got != "First again" {
		t.Fatalf("commit after undo: %q", got)
	}
	// The root commit can be reworded but not squashed into anything.
	press(t, a, "3", "s")
	if !strings.Contains(a.toast, "no parent") {
		t.Errorf("squash of the root commit: %q", a.toast)
	}
	press(t, a, "r", "ctrl+u", "type:Root", "ctrl+s")
	if got := subjects(t, empty, "1"); got != "Root" {
		t.Errorf("reword the root commit: %q (toast %q)", got, a.toast)
	}
	everyKey(t, a)
}

func TestE2EDetachedHEADAndNoRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, _ := e2eSetup(t)
	app, lib := filepath.Join(root, "app"), filepath.Join(root, "lib")
	sh(t, app, "checkout", "-q", "--detach")
	openRepo(t, a, app)
	if !a.rp.info.Detached || !strings.Contains(a.View(), "(detached)") {
		t.Fatalf("detached: %+v", a.rp.info)
	}
	press(t, a, "p")
	if !strings.Contains(a.toast, "detached") {
		t.Errorf("pull while detached: %q", a.toast)
	}
	press(t, a, "P")
	if !strings.Contains(a.toast, "detached") || strings.Contains(a.toast, "✓") {
		t.Errorf("push while detached: %q", a.toast)
	}
	// Committing on a detached HEAD works; undo still finds its way back.
	commitFile(t, app, "d.txt", "d\n", "detached work")
	refresh(t, a)
	press(t, a, "3", "d", "y")
	if got := subjects(t, app, "1"); got != "initial" {
		t.Fatalf("drop on a detached HEAD: %q (toast %q)", got, a.toast)
	}
	press(t, a, "z", "y")
	if got := subjects(t, app, "1"); got != "detached work" {
		t.Fatalf("undo on a detached HEAD: %q (toast %q)", got, a.toast)
	}
	everyKey(t, a)

	// lib has no remote: push, tag push and fetch say so plainly.
	press(t, a, "esc")
	openRepo(t, a, lib)
	press(t, a, "P")
	if !strings.Contains(a.toast, "no remote") {
		t.Errorf("push without a remote: %q", a.toast)
	}
	press(t, a, "2", "]", "]", "n", "type:v1", "enter", "enter", "P")
	if !strings.Contains(a.toast, "no remote") {
		t.Errorf("tag push without a remote: %q", a.toast)
	}
	press(t, a, "ctrl+f")
	if a.toastErr {
		t.Errorf("fetch without a remote: %q", a.toast)
	}
	everyKey(t, a)
}

func TestE2EHugeDiff(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, _ := e2eSetup(t)
	lib := filepath.Join(root, "lib")
	var b strings.Builder
	for i := 0; i < 30000; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	writeFile(t, filepath.Join(lib, "big.txt"), b.String())
	// One line of several megabytes, like minified JavaScript.
	writeFile(t, filepath.Join(lib, "min.js"), strings.Repeat("var a=1;", 600_000)+"\n")
	openRepo(t, a, lib)

	start := time.Now()
	selectRow(t, a, "big.txt")
	if !strings.Contains(a.rp.main, "(truncated)") {
		t.Errorf("a huge diff should be truncated: %d bytes", len(a.rp.main))
	}
	press(t, a, "enter", "G", "esc")
	_ = a.View()
	selectRow(t, a, "min.js")
	if len(a.rp.main) > 2<<20 {
		t.Errorf("the main panel holds %d bytes", len(a.rp.main))
	}
	for i := 0; i < 5; i++ {
		_ = a.View()
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Errorf("huge diffs took %s", d)
	}

	// Staging the huge file line by line still works.
	selectRow(t, a, "big.txt")
	press(t, a, "enter", "space", "esc")
	if got := sh(t, lib, "show", ":big.txt"); got != "line 0" {
		t.Errorf("staging one line of a huge file: %q", got)
	}
}

func TestE2EUnicodeAndSpaces(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, _ := e2eSetup(t)
	lib := filepath.Join(root, "lib")
	names := []string{"my file.txt", "zażółć gęślą.txt", "dir with space/über.txt", "日本語/ファイル.md", "tab\there.txt"}
	for _, n := range names {
		commitFile(t, lib, n, "one\n", "add "+n)
		writeFile(t, filepath.Join(lib, n), "one\ntwo\nthree\n")
	}
	openRepo(t, a, lib)
	if got := len(a.rp.files); got != len(names) {
		t.Fatalf("files: %+v", a.rp.files)
	}
	for _, n := range names {
		base := filepath.Base(n)
		selectRow(t, a, printable(base)) // a tab shows as \t
		if a.rp.main == "(no diff)" || a.rp.main == "" {
			t.Errorf("%s: no diff shown", n)
		}
		// Stage one line, then discard the other unstaged line.
		press(t, a, "enter", "space")
		if got := sh(t, lib, "show", ":"+n); got != "one\ntwo" {
			t.Errorf("%s: staged %q (toast %q)", n, got, a.toast)
		}
		press(t, a, "d", "y", "esc")
		if got, _ := os.ReadFile(filepath.Join(lib, n)); string(got) != "one\ntwo\n" {
			t.Errorf("%s: worktree after discarding a line %q (toast %q)", n, got, a.toast)
		}
	}
	press(t, a, "c", "type:Two", "enter")
	if got := sh(t, lib, "status", "--porcelain"); got != "" {
		t.Errorf("after commit: %q", got)
	}
	// Renamed and deleted files with odd names.
	sh(t, lib, "mv", "my file.txt", "renamed ✓.txt")
	os.Remove(filepath.Join(lib, "zażółć gęślą.txt"))
	refresh(t, a)
	selectRow(t, a, "renamed ✓.txt")
	if !strings.Contains(a.panelItems(pFiles)[a.rp.cur[pFiles]].plain, "← my file.txt") {
		t.Errorf("rename row: %q", a.panelItems(pFiles)[a.rp.cur[pFiles]].plain)
	}
	press(t, a, "d", "y")
	selectRow(t, a, "zażółć")
	press(t, a, "d", "y")
	if got := sh(t, lib, "status", "--porcelain"); got != "" {
		t.Errorf("discarding a rename and a deletion: %q (toast %q)", got, a.toast)
	}

	// Repository names that change length when lower-cased.
	press(t, a, "esc")
	newRepoIn(t, a, root, "İstanbul")
	newRepoIn(t, a, root, "ǅemal-ẞtraße")
	for _, q := range []string{"i", "is", "İs", "st", "ss", "ǆ", "ß"} {
		press(t, a, "/", "type:"+q)
		_ = a.View()
		press(t, a, "esc")
	}
	press(t, a, "/", "type:istanbul", "enter")
	if filepath.Base(a.rp.dir) != "İstanbul" {
		t.Errorf("search İstanbul: %q", a.rp.dir)
	}
}

// fakeSSH makes a GIT_SSH_COMMAND that never answers, like a host behind a
// firewall that drops packets.
func fakeSSH(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "ssh")
	writeFile(t, p, "#!/bin/sh\nsleep 60\n")
	os.Chmod(p, 0o755)
	return p
}

func TestE2ENetworkTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, _ := e2eSetup(t)
	t.Setenv("GIT_SSH_COMMAND", fakeSSH(t))
	defer func(d time.Duration) { gitx.NetTimeout = d }(gitx.NetTimeout)
	gitx.NetTimeout = time.Second
	app := filepath.Join(root, "app")
	sh(t, app, "remote", "set-url", "origin", "ssh://git@example.com/app.git")

	openRepo(t, a, app)
	start := time.Now()
	press(t, a, "ctrl+f")
	if !strings.Contains(a.toast, "timed out after 1s") {
		t.Errorf("fetch: %q", a.toast)
	}
	commitFile(t, app, "x.txt", "x\n", "x")
	refresh(t, a)
	press(t, a, "P")
	if !strings.Contains(a.toast, "timed out") {
		t.Errorf("push: %q", a.toast)
	}
	press(t, a, "!")
	if a.report == nil || !strings.Contains(strings.Join(a.report.lines, "\n"), "timed out") {
		t.Errorf("error details: %+v", a.report)
	}
	press(t, a, "esc", "esc")

	// Fetching many repositories reports the one that timed out.
	press(t, a, "f")
	if a.report == nil || !strings.Contains(a.report.title, "1 of 2 failed") {
		t.Fatalf("bulk fetch report: %+v (toast %q)", a.report, a.toast)
	}
	if !strings.Contains(strings.Join(a.report.lines, "\n"), "timed out after 1s") {
		t.Errorf("report: %q", a.report.lines)
	}
	if d := time.Since(start); d > 20*time.Second {
		t.Errorf("timeouts took %s", d)
	}
}

// Every screen and overlay fits windows from the 50x12 minimum up, and
// smaller windows get the size notice instead.
func TestE2ETinyWindows(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end")
	}
	a, root, _ := e2eSetup(t)
	lib := filepath.Join(root, "lib")
	for i := 0; i < 5; i++ {
		commitFile(t, lib, fmt.Sprintf("f%d.txt", i), "x\n", fmt.Sprintf("commit %d with a subject long enough to be cut", i))
	}
	writeFile(t, filepath.Join(lib, "lib.txt"), "a\nb\nc\n")
	sh(t, lib, "tag", "v1")
	sh(t, lib, "stash", "push", "-q", "-m", "a stash")
	writeFile(t, filepath.Join(lib, "lib.txt"), "a\nb\nc\n")

	screens := map[string]func(){
		"list":     func() { press(t, a, "esc", "esc") },
		"search":   func() { press(t, a, "esc", "esc", "/", "type:li") },
		"repo":     func() { openRepo(t, a, lib) },
		"staging":  func() { openRepo(t, a, lib); selectRow(t, a, "lib.txt"); press(t, a, "enter") },
		"branches": func() { openRepo(t, a, lib); press(t, a, "2") },
		"tags":     func() { openRepo(t, a, lib); press(t, a, "2", "]", "]") },
		"reflog":   func() { openRepo(t, a, lib); press(t, a, "3", "]") },
		"stash":    func() { openRepo(t, a, lib); press(t, a, "4") },
		"diff":     func() { openRepo(t, a, lib); press(t, a, "enter") },
		"planner":  func() { openRepo(t, a, lib); press(t, a, "3", "end", "i") },
		"editor":   func() { openRepo(t, a, lib); press(t, a, "3", "r") },
		"prompt":   func() { openRepo(t, a, lib); press(t, a, "2", "n") },
		"confirm":  func() { openRepo(t, a, lib); press(t, a, "3", "t") },
		"help":     func() { press(t, a, "?") },
		"themes":   func() { press(t, a, "T") },
		"error": func() {
			a.lastErr = errorReport("push", fmt.Errorf("%s", strings.Repeat("a long error line\n", 40)))
			press(t, a, "!")
		},
		"too small": func() {},
	}
	reset := func() {
		a.p, a.ed, a.plan, a.report, a.picker = nil, nil, nil, nil, nil
		a.rp.staging, a.rp.mainFocus, a.rs.filtering, a.rs.filter = nil, false, false, ""
	}
	for _, size := range [][2]int{{50, 12}, {51, 13}, {60, 15}, {79, 20}, {49, 30}, {120, 11}, {10, 3}, {1, 1}} {
		for name, open := range screens {
			reset()
			a.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
			open()
			a.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			frame := a.View()
			checkFrame(t, name, a, frame)
			small := size[0] < minWidth || size[1] < minHeight
			notice := strings.Contains(frame, "needs at least")
			if (small && size[0] >= 30 && !notice) || (!small && notice) {
				t.Errorf("%s at %dx%d: size notice shown = %v", name, size[0], size[1], notice)
			}
			if !small && name == "staging" && !strings.Contains(frame, "Unstaged") {
				t.Errorf("staging at %dx%d lost its title:\n%s", size[0], size[1], frame)
			}
		}
	}
	reset()
}
