// Package gitx wraps the git CLI. Shelling out (like lazygit does) keeps behaviour
// identical to what the user's own git config would do.
package gitx

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Toplevel returns the repository root containing dir.
func Toplevel(ctx context.Context, dir string) (string, error) {
	out, err := run(ctx, kRead, dir, nil, "rev-parse", "--show-toplevel")
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
	State     State
}

func GetInfo(ctx context.Context, dir string) (Info, error) {
	out, err := run(ctx, kRead, dir, nil, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return Info{}, err
	}
	in := parseStatusV2(out)
	if o, err := run(ctx, kRead, dir, nil, "log", "-1", "--format=%ct%x1f%cr"); err == nil {
		parts := strings.SplitN(strings.TrimSpace(o), "\x1f", 2)
		if len(parts) == 2 {
			in.LastUnix, _ = strconv.ParseInt(parts[0], 10, 64)
			in.LastRel = parts[1]
		}
	}
	in.State, _ = GetState(ctx, dir)
	return in, nil
}

func parseStatusV2(out string) Info {
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
	return in
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

func Files(ctx context.Context, dir string) ([]File, error) {
	out, err := run(ctx, kRead, dir, nil, "status", "--porcelain=v1", "-z", "-uall")
	if err != nil {
		return nil, err
	}
	return parseFiles(out), nil
}

func parseFiles(out string) []File {
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
	return files
}

func DiffFile(ctx context.Context, dir string, f File) string {
	var b strings.Builder
	if f.Untracked() {
		out, _ := run(ctx, kRead, dir, nil, "-c", "color.ui=always", "diff", "--no-index", "--", "/dev/null", f.Path)
		return limitLines(out, 5000)
	}
	if f.Staged() {
		if o, _ := run(ctx, kRead, dir, nil, "-c", "color.ui=always", "diff", "--cached", "--", f.Path); o != "" {
			b.WriteString("── staged ──\n" + o)
		}
	}
	if f.Unstaged() {
		if o, _ := run(ctx, kRead, dir, nil, "-c", "color.ui=always", "diff", "--", f.Path); o != "" {
			b.WriteString("── unstaged ──\n" + o)
		}
	}
	if b.Len() == 0 {
		return "(no diff)"
	}
	return limitLines(b.String(), 5000)
}

func Stage(ctx context.Context, dir, path string) error {
	_, err := run(ctx, kWrite, dir, nil, "add", "--", path)
	return err
}

func Unstage(ctx context.Context, dir, path string) error {
	_, err := run(ctx, kWrite, dir, nil, "restore", "--staged", "--", path)
	if err != nil { // e.g. repository without commits
		_, err = run(ctx, kWrite, dir, nil, "rm", "--cached", "-q", "--", path)
	}
	return err
}

func StageAll(ctx context.Context, dir string) error {
	_, err := run(ctx, kWrite, dir, nil, "add", "-A")
	return err
}

func UnstageAll(ctx context.Context, dir string) error {
	_, err := run(ctx, kWrite, dir, nil, "reset", "-q")
	return err
}

func Discard(ctx context.Context, dir string, f File) error {
	var err error
	switch {
	case f.Untracked():
		_, err = run(ctx, kWrite, dir, nil, "clean", "-fd", "--", f.Path)
	case f.X == 'A':
		_, err = run(ctx, kWrite, dir, nil, "rm", "-f", "-q", "--", f.Path)
	default:
		_, err = run(ctx, kWrite, dir, nil, "restore", "--source=HEAD", "--staged", "--worktree", "--", f.Path)
	}
	return err
}

// ---------------------------------------------------------------- commits

type Commit struct {
	Hash, Short, Author, Rel, Subject, Refs string
	Merge                                   bool
}

const logFormat = "--format=%H%x1f%h%x1f%an%x1f%ar%x1f%s%x1f%D%x1f%P"

func Log(ctx context.Context, dir string, n int) ([]Commit, error) {
	out, err := run(ctx, kRead, dir, nil, "log", "-n", strconv.Itoa(n), logFormat)
	if err != nil {
		return nil, err
	}
	return parseLog(out), nil
}

func parseLog(out string) []Commit {
	var cs []Commit
	for _, l := range strings.Split(out, "\n") {
		f := strings.Split(l, "\x1f")
		if len(f) < 7 {
			continue
		}
		cs = append(cs, Commit{Hash: f[0], Short: f[1], Author: f[2], Rel: f[3],
			Subject: f[4], Refs: f[5], Merge: strings.Contains(strings.TrimSpace(f[6]), " ")})
	}
	return cs
}

// Unpushed returns the set of commit hashes not on the upstream (nil if no upstream).
func Unpushed(ctx context.Context, dir string) map[string]bool {
	out, err := run(ctx, kRead, dir, nil, "rev-list", "@{u}..HEAD")
	if err != nil {
		return nil
	}
	m := map[string]bool{}
	for _, h := range strings.Fields(out) {
		m[h] = true
	}
	return m
}

func Show(ctx context.Context, dir, hash string) string {
	out, _ := run(ctx, kRead, dir, nil, "-c", "color.ui=always", "show", "--stat", "--patch", hash)
	return limitLines(out, 5000)
}

func CommitMsg(ctx context.Context, dir, msg string) error {
	_, err := run(ctx, kWrite, dir, nil, "commit", "-m", msg)
	return err
}

// Message returns the full commit message of rev.
func Message(ctx context.Context, dir, rev string) (string, error) {
	return run(ctx, kRead, dir, nil, "log", "-1", "--format=%B", rev)
}

// replaceSubject swaps the first line of a commit message.
func replaceSubject(msg, subject string) string {
	msg = strings.TrimRight(msg, "\n")
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		return subject + msg[i:]
	}
	return subject
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

func Branches(ctx context.Context, dir string) (local, remote []Branch, err error) {
	out, err := run(ctx, kRead, dir, nil, "for-each-ref", "--sort=-committerdate", "--format="+branchFormat, "refs/heads")
	if err != nil {
		return nil, nil, err
	}
	local = parseBranches(out, false)
	if out, err = run(ctx, kRead, dir, nil, "for-each-ref", "--sort=-committerdate", "--format="+branchFormat, "refs/remotes"); err == nil {
		remote = parseBranches(out, true)
	}
	return local, remote, nil
}

func BranchLog(ctx context.Context, dir, name string) string {
	out, _ := run(ctx, kRead, dir, nil, "-c", "color.ui=always", "log", "--graph", "--oneline", "--decorate", "-n", "60", name)
	return out
}

func Checkout(ctx context.Context, dir string, b Branch) error {
	var err error
	if b.Remote {
		_, err = run(ctx, kWrite, dir, nil, "checkout", "--track", b.Name)
	} else {
		_, err = run(ctx, kWrite, dir, nil, "checkout", b.Name)
	}
	return err
}

func NewBranch(ctx context.Context, dir, name, base string) error {
	args := []string{"checkout", "-b", name}
	if base != "" {
		args = append(args, base)
	}
	_, err := run(ctx, kWrite, dir, nil, args...)
	return err
}

func DeleteBranch(ctx context.Context, dir, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := run(ctx, kWrite, dir, nil, "branch", flag, name)
	return err
}

func DeleteRemoteBranch(ctx context.Context, dir, name string) error {
	i := strings.Index(name, "/")
	if i < 0 {
		return errors.New("not a remote branch: " + name)
	}
	_, err := run(ctx, kNet, dir, nil, "push", name[:i], "--delete", name[i+1:])
	return err
}

func RenameBranch(ctx context.Context, dir, old, name string) error {
	_, err := run(ctx, kWrite, dir, nil, "branch", "-m", old, name)
	return err
}

// ------------------------------------------------------------ remote sync

func Fetch(ctx context.Context, dir string) error {
	_, err := run(ctx, kNet, dir, nil, "fetch", "--all", "--prune", "-q")
	return err
}

func Pull(ctx context.Context, dir string, rebase bool) error {
	args := []string{"pull", "-q"}
	if rebase {
		args = append(args, "--rebase", "--autostash")
	} else {
		args = append(args, "--ff-only")
	}
	_, err := run(ctx, kNet, dir, nil, args...)
	return err
}

func Push(ctx context.Context, dir string, force, hasUpstream bool) error {
	args := []string{"push"}
	if force {
		args = append(args, "--force-with-lease")
	}
	if !hasUpstream {
		remote := "origin"
		if o, err := run(ctx, kRead, dir, nil, "remote"); err == nil {
			if f := strings.Fields(o); len(f) > 0 {
				remote = f[0]
			}
		}
		args = append(args, "-u", remote, "HEAD")
	}
	_, err := run(ctx, kNet, dir, nil, args...)
	return err
}

// ------------------------------------------------------------------ stash

type Stash struct{ Ref, Msg, Rel string }

func Stashes(ctx context.Context, dir string) ([]Stash, error) {
	out, err := run(ctx, kRead, dir, nil, "stash", "list", "--format=%gd%x1f%s%x1f%cr")
	if err != nil {
		return nil, err
	}
	return parseStashes(out), nil
}

func parseStashes(out string) []Stash {
	var ss []Stash
	for _, l := range strings.Split(out, "\n") {
		f := strings.Split(l, "\x1f")
		if len(f) >= 3 {
			ss = append(ss, Stash{Ref: f[0], Msg: f[1], Rel: f[2]})
		}
	}
	return ss
}

func StashPush(ctx context.Context, dir, msg string) error {
	args := []string{"stash", "push", "-u"}
	if msg != "" {
		args = append(args, "-m", msg)
	}
	_, err := run(ctx, kWrite, dir, nil, args...)
	return err
}

func StashApply(ctx context.Context, dir, ref string) error {
	_, err := run(ctx, kWrite, dir, nil, "stash", "apply", ref)
	return err
}

func StashPop(ctx context.Context, dir, ref string) error {
	_, err := run(ctx, kWrite, dir, nil, "stash", "pop", ref)
	return err
}

func StashDrop(ctx context.Context, dir, ref string) error {
	_, err := run(ctx, kWrite, dir, nil, "stash", "drop", ref)
	return err
}

func StashShow(ctx context.Context, dir, ref string) string {
	out, _ := run(ctx, kRead, dir, nil, "-c", "color.ui=always", "stash", "show", "-p", ref)
	return limitLines(out, 5000)
}

// ---------------------------------------------------------------- preview

// Preview is the right-hand pane of the repository list.
func Preview(ctx context.Context, dir string) string {
	var b strings.Builder
	o, _ := run(ctx, kRead, dir, nil, "-c", "color.ui=always", "status", "-sb")
	b.WriteString(o)
	b.WriteString("\n")
	o, _ = run(ctx, kRead, dir, nil, "-c", "color.ui=always", "log", "--oneline", "--decorate", "-n", "15")
	b.WriteString(o)
	return b.String()
}
