// Package gitx wraps the git CLI. Shelling out (like lazygit does) keeps behaviour
// identical to what the user's own git config would do.
package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func run(dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	e := append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0", "GIT_EDITOR=true", "LC_MESSAGES=C", "GIT_OPTIONAL_LOCKS=0")
	if os.Getenv("GIT_SSH_COMMAND") == "" {
		e = append(e, "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
	}
	cmd.Env = append(e, env...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = strings.TrimSpace(out.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return out.String(), errors.New(firstLines(msg, 3))
	}
	return out.String(), nil
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, " ")
}

func limitLines(s string, n int) string {
	if strings.Count(s, "\n") <= n {
		return s
	}
	lines := strings.SplitN(s, "\n", n+1)
	return strings.Join(lines[:n], "\n") + "\n… (truncated)"
}

// Toplevel returns the repository root containing dir.
func Toplevel(dir string) (string, error) {
	out, err := run(dir, nil, "rev-parse", "--show-toplevel")
	return strings.TrimSpace(out), err
}

// ---------------------------------------------------------------- repo info

type Info struct {
	Branch    string
	Detached  bool
	Upstream  bool
	Ahead     int
	Behind    int
	Changes   int
	Conflicts bool
	LastUnix  int64
	LastRel   string
}

func GetInfo(dir string) (Info, error) {
	out, err := run(dir, nil, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return Info{}, err
	}
	var in Info
	for _, l := range strings.Split(out, "\n") {
		switch {
		case l == "":
		case strings.HasPrefix(l, "# branch.head "):
			in.Branch = strings.TrimPrefix(l, "# branch.head ")
			in.Detached = in.Branch == "(detached)"
		case strings.HasPrefix(l, "# branch.ab "):
			in.Upstream = true
			fmt.Sscanf(strings.TrimPrefix(l, "# branch.ab "), "+%d -%d", &in.Ahead, &in.Behind)
		case strings.HasPrefix(l, "#"):
		default:
			in.Changes++
			if strings.HasPrefix(l, "u ") {
				in.Conflicts = true
			}
		}
	}
	if o, err := run(dir, nil, "log", "-1", "--format=%ct%x1f%cr"); err == nil {
		parts := strings.SplitN(strings.TrimSpace(o), "\x1f", 2)
		if len(parts) == 2 {
			in.LastUnix, _ = strconv.ParseInt(parts[0], 10, 64)
			in.LastRel = parts[1]
		}
	}
	return in, nil
}

// Sync returns a compact plain-text summary like "↑1 ↓2 ~3".
func (i Info) Sync() string {
	var p []string
	if i.Upstream {
		if i.Ahead > 0 {
			p = append(p, fmt.Sprintf("↑%d", i.Ahead))
		}
		if i.Behind > 0 {
			p = append(p, fmt.Sprintf("↓%d", i.Behind))
		}
	} else if !i.Detached {
		p = append(p, "∅")
	}
	if i.Changes > 0 {
		p = append(p, fmt.Sprintf("~%d", i.Changes))
	}
	return strings.Join(p, " ")
}

// ------------------------------------------------------------------- files

type File struct {
	Path, Orig string
	X, Y       byte
}

func (f File) Untracked() bool { return f.X == '?' }
func (f File) Staged() bool    { return f.X != ' ' && f.X != '?' }
func (f File) Unstaged() bool  { return f.Y != ' ' }
func (f File) Conflict() bool {
	return f.X == 'U' || f.Y == 'U' || (f.X == 'A' && f.Y == 'A') || (f.X == 'D' && f.Y == 'D')
}

func Files(dir string) ([]File, error) {
	out, err := run(dir, nil, "status", "--porcelain=v1", "-z", "-uall")
	if err != nil {
		return nil, err
	}
	parts := strings.Split(out, "\x00")
	var files []File
	for i := 0; i < len(parts); i++ {
		p := parts[i]
		if len(p) < 4 {
			continue
		}
		f := File{X: p[0], Y: p[1], Path: p[3:]}
		if (f.X == 'R' || f.X == 'C') && i+1 < len(parts) {
			f.Orig = parts[i+1]
			i++
		}
		files = append(files, f)
	}
	return files, nil
}

func DiffFile(dir string, f File) string {
	var b strings.Builder
	if f.Untracked() {
		out, _ := run(dir, nil, "-c", "color.ui=always", "diff", "--no-index", "--", "/dev/null", f.Path)
		return limitLines(out, 5000)
	}
	if f.Staged() {
		if o, _ := run(dir, nil, "-c", "color.ui=always", "diff", "--cached", "--", f.Path); o != "" {
			b.WriteString("── staged ──\n" + o)
		}
	}
	if f.Unstaged() {
		if o, _ := run(dir, nil, "-c", "color.ui=always", "diff", "--", f.Path); o != "" {
			b.WriteString("── unstaged ──\n" + o)
		}
	}
	if b.Len() == 0 {
		return "(no diff)"
	}
	return limitLines(b.String(), 5000)
}

func Stage(dir, path string) error {
	_, err := run(dir, nil, "add", "--", path)
	return err
}

func Unstage(dir, path string) error {
	_, err := run(dir, nil, "restore", "--staged", "--", path)
	if err != nil { // e.g. repository without commits
		_, err = run(dir, nil, "rm", "--cached", "-q", "--", path)
	}
	return err
}

func StageAll(dir string) error {
	_, err := run(dir, nil, "add", "-A")
	return err
}

func UnstageAll(dir string) error {
	_, err := run(dir, nil, "reset", "-q")
	return err
}

func Discard(dir string, f File) error {
	var err error
	switch {
	case f.Untracked():
		_, err = run(dir, nil, "clean", "-fd", "--", f.Path)
	case f.X == 'A':
		_, err = run(dir, nil, "rm", "-f", "-q", "--", f.Path)
	default:
		_, err = run(dir, nil, "restore", "--source=HEAD", "--staged", "--worktree", "--", f.Path)
	}
	return err
}

// ---------------------------------------------------------------- commits

type Commit struct {
	Hash, Short, Author, Rel, Subject, Refs string
	Merge                                   bool
}

func Log(dir string, n int) ([]Commit, error) {
	out, err := run(dir, nil, "log", "-n", strconv.Itoa(n),
		"--format=%H%x1f%h%x1f%an%x1f%ar%x1f%s%x1f%D%x1f%P")
	if err != nil {
		return nil, err
	}
	var cs []Commit
	for _, l := range strings.Split(out, "\n") {
		f := strings.Split(l, "\x1f")
		if len(f) < 7 {
			continue
		}
		cs = append(cs, Commit{Hash: f[0], Short: f[1], Author: f[2], Rel: f[3],
			Subject: f[4], Refs: f[5], Merge: strings.Contains(strings.TrimSpace(f[6]), " ")})
	}
	return cs, nil
}

// Unpushed returns the set of commit hashes not on the upstream (nil if no upstream).
func Unpushed(dir string) map[string]bool {
	out, err := run(dir, nil, "rev-list", "@{u}..HEAD")
	if err != nil {
		return nil
	}
	m := map[string]bool{}
	for _, h := range strings.Fields(out) {
		m[h] = true
	}
	return m
}

func Show(dir, hash string) string {
	out, _ := run(dir, nil, "-c", "color.ui=always", "show", "--stat", "--patch", hash)
	return limitLines(out, 5000)
}

func CommitMsg(dir, msg string) error {
	_, err := run(dir, nil, "commit", "-m", msg)
	return err
}

func AmendNoEdit(dir string) error {
	_, err := run(dir, nil, "commit", "--amend", "--no-edit")
	return err
}

func AmendMessage(dir, msg string) error {
	_, err := run(dir, nil, "commit", "--amend", "-m", msg)
	return err
}

func UndoCommit(dir string) error {
	_, err := run(dir, nil, "reset", "--soft", "HEAD~1")
	return err
}

func Revert(dir, hash string) error {
	_, err := run(dir, nil, "revert", "--no-edit", hash)
	return err
}

// RebaseAction rewrites history non-interactively via `git rebase -i` with a
// scripted todo editor. action is one of: drop, reword, squash, fixup.
// squash/fixup meld the commit into the one before it.
func RebaseAction(dir, hash, action, msg string) error {
	line := 1
	spec := hash + "^"
	if action == "squash" || action == "fixup" {
		line = 2
		spec = hash + "^^"
	}
	args := []string{"rebase", "-i", "--autostash"}
	if _, err := run(dir, nil, "rev-parse", "--verify", "-q", spec); err != nil {
		args = append(args, "--root")
	} else {
		args = append(args, spec)
	}
	env := []string{fmt.Sprintf("GIT_SEQUENCE_EDITOR=sed -i.bak '%ds/^pick/%s/'", line, action)}
	if action == "reword" {
		env = append(env,
			"GITNEST_MSG="+msg,
			`GIT_EDITOR=sh -c 'printf "%s\n" "$GITNEST_MSG" > "$1"' sh`)
	}
	_, err := run(dir, env, args...)
	return err
}

func RebaseInProgress(dir string) bool {
	for _, p := range []string{"rebase-merge", "rebase-apply"} {
		out, err := run(dir, nil, "rev-parse", "--git-path", p)
		if err != nil {
			continue
		}
		path := strings.TrimSpace(out)
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, path)
		}
		if st, err := os.Stat(path); err == nil && st.IsDir() {
			return true
		}
	}
	return false
}

func RebaseContinue(dir string) error {
	_, err := run(dir, nil, "rebase", "--continue")
	return err
}

func RebaseAbort(dir string) error {
	_, err := run(dir, nil, "rebase", "--abort")
	return err
}

// ---------------------------------------------------------------- branches

type Branch struct {
	Name, Upstream, Track, Rel, Subject string
	Current, Remote                     bool
}

const branchFormat = "%(HEAD)%x1f%(refname:short)%x1f%(upstream:short)%x1f%(upstream:track)%x1f%(committerdate:relative)%x1f%(subject)"

func parseBranches(out string, remote bool) []Branch {
	var bs []Branch
	for _, l := range strings.Split(out, "\n") {
		f := strings.Split(l, "\x1f")
		if len(f) < 6 {
			continue
		}
		if remote && (!strings.Contains(f[1], "/") || strings.HasSuffix(f[1], "/HEAD")) {
			continue
		}
		bs = append(bs, Branch{Current: strings.TrimSpace(f[0]) == "*", Name: f[1], Upstream: f[2],
			Track: f[3], Rel: f[4], Subject: f[5], Remote: remote})
	}
	return bs
}

func Branches(dir string) (local, remote []Branch, err error) {
	out, err := run(dir, nil, "for-each-ref", "--sort=-committerdate", "--format="+branchFormat, "refs/heads")
	if err != nil {
		return nil, nil, err
	}
	local = parseBranches(out, false)
	if out, err = run(dir, nil, "for-each-ref", "--sort=-committerdate", "--format="+branchFormat, "refs/remotes"); err == nil {
		remote = parseBranches(out, true)
	}
	return local, remote, nil
}

func BranchLog(dir, name string) string {
	out, _ := run(dir, nil, "-c", "color.ui=always", "log", "--graph", "--oneline", "--decorate", "-n", "60", name)
	return out
}

func Checkout(dir string, b Branch) error {
	var err error
	if b.Remote {
		_, err = run(dir, nil, "checkout", "--track", b.Name)
	} else {
		_, err = run(dir, nil, "checkout", b.Name)
	}
	return err
}

func NewBranch(dir, name, base string) error {
	args := []string{"checkout", "-b", name}
	if base != "" {
		args = append(args, base)
	}
	_, err := run(dir, nil, args...)
	return err
}

func DeleteBranch(dir, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := run(dir, nil, "branch", flag, name)
	return err
}

func DeleteRemoteBranch(dir, name string) error {
	i := strings.Index(name, "/")
	if i < 0 {
		return errors.New("not a remote branch: " + name)
	}
	_, err := run(dir, nil, "push", name[:i], "--delete", name[i+1:])
	return err
}

func RenameBranch(dir, old, name string) error {
	_, err := run(dir, nil, "branch", "-m", old, name)
	return err
}

func Merge(dir, name string) error {
	_, err := run(dir, nil, "merge", "--no-edit", name)
	return err
}

func RebaseOnto(dir, name string) error {
	_, err := run(dir, nil, "rebase", "--autostash", name)
	return err
}

// ------------------------------------------------------------ remote sync

func Fetch(dir string) error {
	_, err := run(dir, nil, "fetch", "--all", "--prune", "-q")
	return err
}

func Pull(dir string, rebase bool) error {
	args := []string{"pull", "-q"}
	if rebase {
		args = append(args, "--rebase", "--autostash")
	} else {
		args = append(args, "--ff-only")
	}
	_, err := run(dir, nil, args...)
	return err
}

func Push(dir string, force, hasUpstream bool) error {
	args := []string{"push"}
	if force {
		args = append(args, "--force-with-lease")
	}
	if !hasUpstream {
		remote := "origin"
		if o, err := run(dir, nil, "remote"); err == nil {
			if f := strings.Fields(o); len(f) > 0 {
				remote = f[0]
			}
		}
		args = append(args, "-u", remote, "HEAD")
	}
	_, err := run(dir, nil, args...)
	return err
}

// ------------------------------------------------------------------ stash

type Stash struct{ Ref, Msg, Rel string }

func Stashes(dir string) ([]Stash, error) {
	out, err := run(dir, nil, "stash", "list", "--format=%gd%x1f%s%x1f%cr")
	if err != nil {
		return nil, err
	}
	var ss []Stash
	for _, l := range strings.Split(out, "\n") {
		f := strings.Split(l, "\x1f")
		if len(f) >= 3 {
			ss = append(ss, Stash{Ref: f[0], Msg: f[1], Rel: f[2]})
		}
	}
	return ss, nil
}

func StashPush(dir, msg string) error {
	args := []string{"stash", "push", "-u"}
	if msg != "" {
		args = append(args, "-m", msg)
	}
	_, err := run(dir, nil, args...)
	return err
}

func StashApply(dir, ref string) error {
	_, err := run(dir, nil, "stash", "apply", ref)
	return err
}

func StashPop(dir, ref string) error {
	_, err := run(dir, nil, "stash", "pop", ref)
	return err
}

func StashDrop(dir, ref string) error {
	_, err := run(dir, nil, "stash", "drop", ref)
	return err
}

func StashShow(dir, ref string) string {
	out, _ := run(dir, nil, "-c", "color.ui=always", "stash", "show", "-p", ref)
	return limitLines(out, 5000)
}

// ---------------------------------------------------------------- preview

// Preview is the right-hand pane of the repository list.
func Preview(dir string) string {
	var b strings.Builder
	o, _ := run(dir, nil, "-c", "color.ui=always", "status", "-sb")
	b.WriteString(o)
	b.WriteString("\n")
	o, _ = run(dir, nil, "-c", "color.ui=always", "log", "--oneline", "--decorate", "-n", "15")
	b.WriteString(o)
	return b.String()
}
