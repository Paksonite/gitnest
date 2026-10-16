package gitx

// History-rewriting operations. Each one is checked up front, records an
// undo point before touching anything, and targets commits by hash.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Actions accepted by CheckRewrite and RebaseAction.
const (
	ActionDrop   = "drop"
	ActionReword = "reword"
	ActionSquash = "squash" // meld into the parent, combining messages
	ActionFixup  = "fixup"  // meld into the parent, keeping the parent's message
	ActionAmend  = "amend"  // any change to HEAD only (amend, undo commit)
)

// CheckRewrite reports whether commit rev can be rewritten with action. On
// success it returns the remote-tracking branches that already contain an
// affected commit: rewriting those will require a force push.
func CheckRewrite(ctx context.Context, dir, rev, action string) (onRemotes []string, err error) {
	if err := requireNoOperation(ctx, dir); err != nil {
		return nil, err
	}
	hash, err := resolve(ctx, dir, rev)
	if err != nil {
		return nil, err
	}
	head, err := resolve(ctx, dir, "HEAD")
	if err != nil {
		return nil, err
	}
	if !isAncestor(ctx, dir, hash, head) {
		return nil, fmt.Errorf("commit %s is not on the current branch", short(hash))
	}

	oldest := hash // the oldest commit whose hash changes
	switch action {
	case ActionSquash, ActionFixup:
		parent, err := resolve(ctx, dir, hash+"^")
		if err != nil {
			return nil, fmt.Errorf("commit %s has no parent to %s into", short(hash), action)
		}
		oldest = parent
	case ActionAmend:
		if hash != head {
			return nil, errors.New("only HEAD can be amended")
		}
	case ActionDrop, ActionReword:
	default:
		return nil, fmt.Errorf("unknown rewrite action %q", action)
	}

	// Amending HEAD (including reword of HEAD) doesn't replay anything.
	if action != ActionAmend && !(action == ActionReword && hash == head) {
		if hasMerges(ctx, dir, oldest) {
			return nil, fmt.Errorf("the history from %s to HEAD contains merge commits; "+
				"rewriting across merges is not supported", short(oldest))
		}
	}
	return remotesContaining(ctx, dir, oldest), nil
}

// CheckRebaseOnto validates rebasing the current branch onto `onto` and returns
// the remote-tracking branches that already contain commits it would rewrite.
func CheckRebaseOnto(ctx context.Context, dir, onto string) (onRemotes []string, err error) {
	if err := requireNoOperation(ctx, dir); err != nil {
		return nil, err
	}
	out, err := run(ctx, kRead, dir, nil, "rev-list", "--reverse", onto+"..HEAD")
	if err != nil {
		return nil, err
	}
	commits := strings.Fields(out)
	if len(commits) == 0 {
		return nil, nil // fast-forward: nothing is rewritten
	}
	return remotesContaining(ctx, dir, commits[0]), nil
}

// RebaseAction applies action (drop, reword, squash, fixup) to one commit via
// `git rebase -i`, editing the todo line that matches the commit's hash.
// msg is the new message for reword.
func RebaseAction(ctx context.Context, dir, rev, action, msg string) error {
	if action == ActionAmend {
		return errors.New("use Reword or AmendNoEdit to change HEAD")
	}
	if _, err := CheckRewrite(ctx, dir, rev, action); err != nil {
		return err
	}
	hash, err := resolve(ctx, dir, rev)
	if err != nil {
		return err
	}
	base := hash + "^"
	if action == ActionSquash || action == ActionFixup {
		base = hash + "^^"
	}
	args := []string{"rebase", "-i", "--autostash"}
	if _, err := resolve(ctx, dir, base); err != nil {
		args = append(args, "--root")
	} else {
		args = append(args, base)
	}
	env := []string{
		"GITNEST_HASH=" + hash,
		"GITNEST_ACTION=" + action,
		"GITNEST_AWK=" + todoAwk,
		"GIT_SEQUENCE_EDITOR=" + todoEditor,
	}
	if action == ActionReword {
		env = append(env,
			"GITNEST_MSG="+msg,
			`GIT_EDITOR=sh -c 'printf "%s\n" "$GITNEST_MSG" > "$1"' sh`)
	}
	if err := savePoint(ctx, dir, action+" "+short(hash)); err != nil {
		return err
	}
	_, err = run(ctx, kWrite, dir, env, args...)
	return err
}

// todoAwk rewrites the single `pick` line whose abbreviated hash is a prefix
// of $GITNEST_HASH and fails unless exactly one line matched, which makes git
// abort the rebase before anything is changed.
const todoAwk = `($1 == "pick" || $1 == "p") && length($2) >= 4 && index(h, $2) == 1 { $1 = a; n++ }
{ print }
END { if (n != 1) exit 3 }`

const todoEditor = `sh -c 'awk -v h="$GITNEST_HASH" -v a="$GITNEST_ACTION" "$GITNEST_AWK" "$1" > "$1.gitnest" && mv "$1.gitnest" "$1"' sh`

// Reword replaces the subject line of commit rev and keeps its body.
func Reword(ctx context.Context, dir, rev, subject string) error {
	subject = strings.TrimSpace(subject)
	if subject == "" || strings.Contains(subject, "\n") {
		return errors.New("reword: subject must be a single non-empty line")
	}
	full, err := Message(ctx, dir, rev)
	if err != nil {
		return err
	}
	return RewordMessage(ctx, dir, rev, replaceSubject(full, subject))
}

// RewordMessage replaces the whole message of commit rev. HEAD is amended
// with --only, so staged changes stay staged instead of being folded into the
// commit; older commits go through an interactive rebase.
func RewordMessage(ctx context.Context, dir, rev, msg string) error {
	msg = CleanMessage(msg)
	if msg == "" {
		return errors.New("reword: the message is empty")
	}
	hash, err := resolve(ctx, dir, rev)
	if err != nil {
		return err
	}
	head, err := resolve(ctx, dir, "HEAD")
	if err != nil {
		return err
	}
	if hash != head {
		return RebaseAction(ctx, dir, hash, ActionReword, msg)
	}
	if _, err := CheckRewrite(ctx, dir, hash, ActionReword); err != nil {
		return err
	}
	if err := savePoint(ctx, dir, "reword "+short(hash)); err != nil {
		return err
	}
	_, err = run(ctx, kWrite, dir, nil, "commit", "--amend", "--only", "--allow-empty", "-m", msg)
	return err
}

// CleanMessage trims trailing spaces from every line and drops blank lines
// at the start and end of a commit message.
func CleanMessage(msg string) string {
	lines := strings.Split(strings.ReplaceAll(msg, "\r\n", "\n"), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}

// AmendNoEdit folds the staged changes into HEAD, keeping its message.
func AmendNoEdit(ctx context.Context, dir string) error {
	if err := savePoint(ctx, dir, "amend"); err != nil {
		return err
	}
	_, err := run(ctx, kWrite, dir, nil, "commit", "--amend", "--no-edit")
	return err
}

// UndoCommit removes the last commit and keeps its changes staged.
func UndoCommit(ctx context.Context, dir string) error {
	if err := savePoint(ctx, dir, "undo commit"); err != nil {
		return err
	}
	_, err := run(ctx, kWrite, dir, nil, "reset", "--soft", "HEAD~1")
	return err
}

func Revert(ctx context.Context, dir, hash string) error {
	if err := savePoint(ctx, dir, "revert "+short(hash)); err != nil {
		return err
	}
	_, err := run(ctx, kWrite, dir, nil, "revert", "--no-edit", hash)
	return err
}

func Merge(ctx context.Context, dir, name string) error {
	if err := savePoint(ctx, dir, "merge "+name); err != nil {
		return err
	}
	_, err := run(ctx, kWrite, dir, nil, "merge", "--no-edit", name)
	return err
}

func RebaseOnto(ctx context.Context, dir, name string) error {
	if err := savePoint(ctx, dir, "rebase onto "+name); err != nil {
		return err
	}
	_, err := run(ctx, kWrite, dir, nil, "rebase", "--autostash", name)
	return err
}

// ------------------------------------------------------------- undo points

// UndoPoint is the state recorded before the last history-changing operation.
type UndoPoint struct {
	Label  string `json:"label"`  // e.g. "squash 1a2b3c4"
	Head   string `json:"head"`   // HEAD before the operation
	Branch string `json:"branch"` // checked-out branch, empty when detached
	Time   int64  `json:"time"`
}

// undoRef keeps the saved commit reachable (safe from gc). refs/worktree/ is
// private to each worktree, like the metadata file next to it.
const undoRef = "refs/worktree/gitnest/undo"

func undoFile(gitDir string) string { return filepath.Join(gitDir, "gitnest-undo.json") }

// savePoint records HEAD so the next operation can be undone. A repository
// without commits has nothing to record.
func savePoint(ctx context.Context, dir, label string) error {
	head, err := resolve(ctx, dir, "HEAD")
	if err != nil {
		return nil
	}
	gitDir, err := absGitDir(ctx, dir)
	if err != nil {
		return err
	}
	branch, _ := run(ctx, kRead, dir, nil, "symbolic-ref", "-q", "--short", "HEAD")
	if _, err := run(ctx, kWrite, dir, nil, "update-ref", "-m", "gitnest: "+label, undoRef, head); err != nil {
		return fmt.Errorf("saving undo point: %w", err)
	}
	b, _ := json.Marshal(UndoPoint{Label: label, Head: head, Branch: strings.TrimSpace(branch), Time: time.Now().Unix()})
	if err := os.WriteFile(undoFile(gitDir), b, 0o644); err != nil {
		return fmt.Errorf("saving undo point: %w", err)
	}
	return nil
}

// LastUndo returns the most recent undo point, or nil when there is none.
func LastUndo(ctx context.Context, dir string) (*UndoPoint, error) {
	gitDir, err := absGitDir(ctx, dir)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(undoFile(gitDir))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var p UndoPoint
	if err := json.Unmarshal(b, &p); err != nil || p.Head == "" {
		return nil, nil
	}
	if ref, err := resolve(ctx, dir, undoRef); err != nil || ref != p.Head {
		return nil, nil // ref deleted or moved by someone else: don't trust the file
	}
	return &p, nil
}

// Undo resets the branch to the last undo point. It refuses while an
// operation is in progress, with uncommitted changes to tracked files, or on
// a different branch than the one the point was recorded on. The current
// HEAD becomes the new undo point, so a second Undo redoes.
func Undo(ctx context.Context, dir string) error {
	p, err := LastUndo(ctx, dir)
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("nothing to undo")
	}
	if err := requireNoOperation(ctx, dir); err != nil {
		return err
	}
	branch, _ := run(ctx, kRead, dir, nil, "symbolic-ref", "-q", "--short", "HEAD")
	if b := strings.TrimSpace(branch); b != p.Branch {
		return fmt.Errorf("undo point is for %s, but %s is checked out", orDetached(p.Branch), orDetached(b))
	}
	dirty, err := run(ctx, kRead, dir, nil, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return err
	}
	if strings.TrimSpace(dirty) != "" {
		return errors.New("commit or stash your changes before undoing")
	}
	if err := savePoint(ctx, dir, "undo of "+p.Label); err != nil {
		return err
	}
	_, err = run(ctx, kWrite, dir, nil, "reset", "--hard", "-q", p.Head)
	return err
}

// ----------------------------------------------------------------- helpers

func resolve(ctx context.Context, dir, rev string) (string, error) {
	out, err := run(ctx, kRead, dir, nil, "rev-parse", "--verify", "-q", rev+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("unknown revision %s", rev)
	}
	return strings.TrimSpace(out), nil
}

func isAncestor(ctx context.Context, dir, a, b string) bool {
	_, err := run(ctx, kRead, dir, nil, "merge-base", "--is-ancestor", a, b)
	return err == nil
}

// hasMerges reports whether any commit from oldest (inclusive) to HEAD is a merge.
func hasMerges(ctx context.Context, dir, oldest string) bool {
	rng := oldest + "^..HEAD"
	if _, err := resolve(ctx, dir, oldest+"^"); err != nil {
		rng = "HEAD" // oldest is a root commit
	}
	out, err := run(ctx, kRead, dir, nil, "rev-list", "--min-parents=2", "-n", "1", rng)
	return err != nil || strings.TrimSpace(out) != ""
}

func remotesContaining(ctx context.Context, dir, hash string) []string {
	// Full names: refs/remotes/origin/HEAD shortens to just "origin".
	out, err := run(ctx, kRead, dir, nil, "for-each-ref", "--contains", hash, "--format=%(refname)", "refs/remotes")
	if err != nil {
		return nil
	}
	var refs []string
	for _, r := range strings.Fields(out) {
		if !strings.HasSuffix(r, "/HEAD") {
			refs = append(refs, strings.TrimPrefix(r, "refs/remotes/"))
		}
	}
	return refs
}

func requireNoOperation(ctx context.Context, dir string) error {
	st, err := GetState(ctx, dir)
	if err != nil {
		return err
	}
	if st != StateNone {
		return fmt.Errorf("finish or abort the %s in progress first", strings.ToLower(st.String()))
	}
	return nil
}

func absGitDir(ctx context.Context, dir string) (string, error) {
	out, err := run(ctx, kRead, dir, nil, "rev-parse", "--absolute-git-dir")
	return strings.TrimSpace(out), err
}

func short(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}

func orDetached(branch string) string {
	if branch == "" {
		return "detached HEAD"
	}
	return branch
}
