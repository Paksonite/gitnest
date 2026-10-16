package gitx

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// State is a multi-step operation left in progress in a repository.
type State int

const (
	StateNone State = iota
	StateRebasing
	StateMerging
	StateCherryPicking
	StateReverting
	StateBisecting
	StateApplying // git am (e.g. commits copied from another repository)
)

func (s State) String() string {
	switch s {
	case StateRebasing:
		return "REBASING"
	case StateMerging:
		return "MERGING"
	case StateCherryPicking:
		return "CHERRY-PICKING"
	case StateReverting:
		return "REVERTING"
	case StateBisecting:
		return "BISECTING"
	case StateApplying:
		return "APPLYING PATCHES"
	}
	return ""
}

// CanSkip reports whether the current step of s can be skipped.
func (s State) CanSkip() bool {
	return s == StateRebasing || s == StateCherryPicking || s == StateReverting || s == StateApplying
}

// GetState inspects the repository's git dir for an operation in progress.
// It works for linked worktrees, whose state lives in their own git dir.
func GetState(ctx context.Context, dir string) (State, error) {
	out, err := run(ctx, kRead, dir, nil, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return StateNone, err
	}
	return stateIn(strings.TrimSpace(out)), nil
}

func stateIn(gitDir string) State {
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(gitDir, name))
		return err == nil
	}
	// Rebase first: a conflicting rebase step also leaves CHERRY_PICK_HEAD.
	switch {
	case exists("rebase-apply/applying"):
		return StateApplying
	case exists("rebase-merge"), exists("rebase-apply"):
		return StateRebasing
	case exists("MERGE_HEAD"):
		return StateMerging
	case exists("CHERRY_PICK_HEAD"):
		return StateCherryPicking
	case exists("REVERT_HEAD"):
		return StateReverting
	case exists("BISECT_LOG"):
		return StateBisecting
	}
	// A multi-commit cherry-pick/revert whose conflict was already committed
	// leaves only the sequencer todo behind.
	return sequencerState(filepath.Join(gitDir, "sequencer", "todo"))
}

func sequencerState(todo string) State {
	f, err := os.Open(todo)
	if err != nil {
		return StateNone
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		switch strings.Fields(sc.Text() + " #")[0] {
		case "pick", "p":
			return StateCherryPicking
		case "revert":
			return StateReverting
		}
	}
	return StateNone
}

// Continue resumes the operation in progress after conflicts were resolved.
func Continue(ctx context.Context, dir string, s State) error {
	var args []string
	switch s {
	case StateRebasing:
		args = []string{"rebase", "--continue"}
	case StateApplying:
		args = []string{"am", "--continue"}
	case StateMerging:
		args = []string{"merge", "--continue"}
	case StateCherryPicking:
		args = []string{"cherry-pick", "--continue"}
	case StateReverting:
		args = []string{"revert", "--continue"}
	default:
		return fmt.Errorf("nothing to continue (%s)", stateName(s))
	}
	_, err := run(ctx, kWrite, dir, nil, args...)
	return err
}

// Abort cancels the operation in progress and restores the previous state.
func Abort(ctx context.Context, dir string, s State) error {
	var args []string
	switch s {
	case StateRebasing:
		args = []string{"rebase", "--abort"}
	case StateApplying:
		args = []string{"am", "--abort"}
	case StateMerging:
		args = []string{"merge", "--abort"}
	case StateCherryPicking:
		args = []string{"cherry-pick", "--abort"}
	case StateReverting:
		args = []string{"revert", "--abort"}
	case StateBisecting:
		args = []string{"bisect", "reset"}
	default:
		return fmt.Errorf("nothing to abort (%s)", stateName(s))
	}
	_, err := run(ctx, kWrite, dir, nil, args...)
	return err
}

// Skip drops the current step of a rebase, cherry-pick or revert.
func Skip(ctx context.Context, dir string, s State) error {
	var args []string
	switch s {
	case StateRebasing:
		args = []string{"rebase", "--skip"}
	case StateApplying:
		args = []string{"am", "--skip"}
	case StateCherryPicking:
		args = []string{"cherry-pick", "--skip"}
	case StateReverting:
		args = []string{"revert", "--skip"}
	default:
		return fmt.Errorf("can't skip while %s", stateName(s))
	}
	_, err := run(ctx, kWrite, dir, nil, args...)
	return err
}

func stateName(s State) string {
	if s == StateNone {
		return "no operation in progress"
	}
	return strings.ToLower(s.String())
}
