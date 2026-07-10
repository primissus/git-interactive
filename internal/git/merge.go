package git

import "context"

// MergeMode selects how MergeBranch combines branch into the current branch.
type MergeMode string

const (
	MergeDefault MergeMode = "default" // an ordinary merge commit when not a fast-forward
	MergeFFOnly  MergeMode = "ff-only"
	MergeNoFF    MergeMode = "no-ff"
	MergeSquash  MergeMode = "squash"
)

// MergeBranch merges branch into the current branch per mode. Squash merges
// stage the changes and then commit them directly, since `git merge --squash`
// alone never creates a commit.
func MergeBranch(ctx context.Context, r *Runner, branch string, mode MergeMode) error {
	switch mode {
	case MergeFFOnly:
		_, err := r.Run(ctx, "merge", "--ff-only", branch)
		return err
	case MergeNoFF:
		_, err := r.Run(ctx, "merge", "--no-ff", "-m", "Merge branch '"+branch+"'", branch)
		return err
	case MergeSquash:
		if _, err := r.Run(ctx, "merge", "--squash", branch); err != nil {
			return err
		}
		_, err := r.Run(ctx, "commit", "-m", "Merge branch '"+branch+"' (squash)")
		return err
	default:
		_, err := r.Run(ctx, "merge", "-m", "Merge branch '"+branch+"'", branch)
		return err
	}
}
