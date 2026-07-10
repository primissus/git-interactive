package git

import (
	"context"
	"strings"
)

// Commit creates a commit from the index. noVerify skips hooks.
func CommitStaged(ctx context.Context, r *Runner, message string, noVerify bool) error {
	args := []string{"commit", "-m", message}
	if noVerify {
		args = append(args, "--no-verify")
	}
	_, err := r.Run(ctx, args...)
	return err
}

// AmendCommit amends HEAD. An empty message keeps HEAD's existing message
// (--no-edit); noVerify skips hooks.
func AmendCommit(ctx context.Context, r *Runner, message string, noVerify bool) error {
	args := []string{"commit", "--amend"}
	if message != "" {
		args = append(args, "-m", message)
	} else {
		args = append(args, "--no-edit")
	}
	if noVerify {
		args = append(args, "--no-verify")
	}
	_, err := r.Run(ctx, args...)
	return err
}

// IsCommitPushed reports whether sha is reachable from any remote-tracking
// branch — the "already pushed" check behind commit's amend warning
// (PROMPT.md → commit: "warn if the commit is already pushed").
func IsCommitPushed(ctx context.Context, r *Runner, sha string) (bool, error) {
	out, err := r.Run(ctx, "branch", "-r", "--contains", sha)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}
