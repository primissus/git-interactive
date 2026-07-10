package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// InProgressOp names a git operation that can stop on conflicts.
type InProgressOp string

const (
	OpMerge      InProgressOp = "merge"
	OpRebase     InProgressOp = "rebase"
	OpCherryPick InProgressOp = "cherry-pick"
	OpRevert     InProgressOp = "revert"
)

// InProgressState describes a stopped merge/rebase/cherry-pick/revert. status
// (phase 5) surfaces it with a plain continue/abort; the future
// resolve-conflicts command (PROMPT.md → Future) hooks into the same
// detection to offer per-conflict resolution, which is why detection and
// continue/abort live behind this type rather than being inlined into status.
type InProgressState struct {
	Op InProgressOp
}

// DetectInProgress reports the in-progress operation, if any, by checking for
// the marker files git itself uses. Returns (nil, nil) when nothing is in
// progress.
func DetectInProgress(ctx context.Context, r *Runner) (*InProgressState, error) {
	gitDir, err := gitDir(ctx, r)
	if err != nil {
		return nil, err
	}
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(gitDir, name))
		return err == nil
	}
	switch {
	case exists("MERGE_HEAD"):
		return &InProgressState{Op: OpMerge}, nil
	case exists("CHERRY_PICK_HEAD"):
		return &InProgressState{Op: OpCherryPick}, nil
	case exists("REVERT_HEAD"):
		return &InProgressState{Op: OpRevert}, nil
	case exists("rebase-merge"), exists("rebase-apply"):
		return &InProgressState{Op: OpRebase}, nil
	}
	return nil, nil
}

// Continue resolves the in-progress operation after its conflicts are fixed.
func (s *InProgressState) Continue(ctx context.Context, r *Runner) error {
	_, err := r.Run(ctx, string(s.Op), "--continue")
	return err
}

// Abort cancels the in-progress operation, restoring the pre-operation state.
func (s *InProgressState) Abort(ctx context.Context, r *Runner) error {
	_, err := r.Run(ctx, string(s.Op), "--abort")
	return err
}

func gitDir(ctx context.Context, r *Runner) (string, error) {
	out, err := r.Run(ctx, "rev-parse", "--git-dir")
	if err != nil {
		return "", err
	}
	dir := strings.TrimSpace(out)
	if filepath.IsAbs(dir) {
		return dir, nil
	}
	base := r.Dir
	if base == "" {
		base = "."
	}
	return filepath.Join(base, dir), nil
}
