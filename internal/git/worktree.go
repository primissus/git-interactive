package git

import (
	"context"
	"strings"
)

// Worktree is one entry from `git worktree list --porcelain`.
type Worktree struct {
	Path           string
	Head           string
	Branch         string // "" if detached
	Detached       bool
	Bare           bool
	Locked         bool
	LockedReason   string
	Prunable       bool
	PrunableReason string
}

// ListWorktrees returns all worktrees registered against the repo.
func ListWorktrees(ctx context.Context, r *Runner) ([]Worktree, error) {
	out, err := r.Run(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktrees(out), nil
}

// AddWorktree creates a worktree at path. When newBranch is true it creates
// branch as a new branch (`-b`) starting from startPoint (HEAD if ""); when
// false it checks out the existing branch there.
func AddWorktree(ctx context.Context, r *Runner, path, branch string, newBranch bool, startPoint string) error {
	args := []string{"worktree", "add"}
	if newBranch {
		args = append(args, "-b", branch, path)
		if startPoint != "" {
			args = append(args, startPoint)
		}
	} else {
		args = append(args, path, branch)
	}
	_, err := r.Run(ctx, args...)
	return err
}

// RemoveWorktree removes the worktree at path; force removes it even with
// untracked/modified files or a locked worktree.
func RemoveWorktree(ctx context.Context, r *Runner, path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	_, err := r.Run(ctx, args...)
	return err
}

// PruneWorktrees removes administrative files for worktrees whose directory
// no longer exists.
func PruneWorktrees(ctx context.Context, r *Runner) error {
	_, err := r.Run(ctx, "worktree", "prune")
	return err
}

// LockWorktree locks the worktree at path against `worktree prune`/`remove`,
// optionally recording reason.
func LockWorktree(ctx context.Context, r *Runner, path, reason string) error {
	args := []string{"worktree", "lock"}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	args = append(args, path)
	_, err := r.Run(ctx, args...)
	return err
}

// UnlockWorktree unlocks the worktree at path.
func UnlockWorktree(ctx context.Context, r *Runner, path string) error {
	_, err := r.Run(ctx, "worktree", "unlock", path)
	return err
}

func parseWorktrees(out string) []Worktree {
	var worktrees []Worktree
	var cur *Worktree

	flush := func() {
		if cur != nil {
			worktrees = append(worktrees, *cur)
			cur = nil
		}
	}

	for line := range strings.SplitSeq(out, "\n") {
		if line == "" {
			flush()
			continue
		}
		key, rest, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			flush()
			cur = &Worktree{Path: rest}
		case "HEAD":
			if cur != nil {
				cur.Head = rest
			}
		case "branch":
			if cur != nil {
				cur.Branch = strings.TrimPrefix(rest, "refs/heads/")
			}
		case "detached":
			if cur != nil {
				cur.Detached = true
			}
		case "bare":
			if cur != nil {
				cur.Bare = true
			}
		case "locked":
			if cur != nil {
				cur.Locked = true
				cur.LockedReason = rest
			}
		case "prunable":
			if cur != nil {
				cur.Prunable = true
				cur.PrunableReason = rest
			}
		}
	}
	flush()

	return worktrees
}
