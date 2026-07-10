package git

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

// Branch is one row of `git branch` output.
type Branch struct {
	Name         string
	Head         bool
	Subject      string
	CommitDate   string // relative, e.g. "3 days ago"
	CommitUnix   int64  // committerdate, for date-bucket filtering and "last commit" sort
	AuthorUnix   int64  // authordate of the tip commit, used as the "created" sort heuristic
	AuthorName   string
	WorktreePath string // "" if not checked out in another worktree
	// UpstreamTrack is git's %(upstream:track) value, e.g. "[gone]",
	// "[ahead 1, behind 2]", or "" when there is no upstream / it is current.
	UpstreamTrack string
}

// Gone reports whether the branch's upstream was deleted.
func (b Branch) Gone() bool { return strings.Contains(b.UpstreamTrack, "gone") }

// branchFormat mirrors git-br.py's field order (name, subject, date, author,
// worktree) plus the extra machine-readable fields phase 3 needs for
// sorting/filtering: unix timestamps and upstream tracking state.
const branchFormat = "%(HEAD)\x1f%(refname:short)\x1f%(contents:subject)\x1f" +
	"%(committerdate:relative)\x1f%(committerdate:unix)\x1f%(authordate:unix)\x1f" +
	"%(authorname)\x1f%(worktreepath)\x1f%(upstream:track)"

// ListBranches returns local branches sorted by most recent commit first.
func ListBranches(ctx context.Context, r *Runner) ([]Branch, error) {
	out, err := r.Run(ctx, "branch", "--format="+branchFormat, "--sort=-committerdate")
	if err != nil {
		return nil, err
	}
	return parseBranches(out), nil
}

func parseBranches(out string) []Branch {
	lines := strings.Split(out, "\n")
	branches := make([]Branch, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\x1f")
		for len(fields) < 9 {
			fields = append(fields, "")
		}
		commitUnix, _ := strconv.ParseInt(fields[4], 10, 64)
		authorUnix, _ := strconv.ParseInt(fields[5], 10, 64)
		branches = append(branches, Branch{
			Head:          fields[0] == "*",
			Name:          fields[1],
			Subject:       fields[2],
			CommitDate:    fields[3],
			CommitUnix:    commitUnix,
			AuthorUnix:    authorUnix,
			AuthorName:    fields[6],
			WorktreePath:  fields[7],
			UpstreamTrack: fields[8],
		})
	}
	return branches
}

// MergedBranches returns the set of local branch names merged into HEAD (the
// current branch), for the branch view's merged/not-merged filter.
func MergedBranches(ctx context.Context, r *Runner) (map[string]bool, error) {
	out, err := r.Run(ctx, "branch", "--format=%(refname:short)", "--merged")
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			set[line] = true
		}
	}
	return set, nil
}

// CurrentBranch returns the name of the checked-out branch, or "" if detached.
func CurrentBranch(ctx context.Context, r *Runner) (string, error) {
	out, err := r.Run(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(out)
	if name == "HEAD" {
		return "", nil
	}
	return name, nil
}

// RevParse resolves ref to its full object SHA.
func RevParse(ctx context.Context, r *Runner, ref string) (string, error) {
	out, err := r.Run(ctx, "rev-parse", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// BranchExists reports whether a local branch named name exists.
func BranchExists(ctx context.Context, r *Runner, name string) (bool, error) {
	_, err := r.Run(ctx, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	if err == nil {
		return true, nil
	}
	var gitErr *Error
	if errors.As(err, &gitErr) {
		return false, nil
	}
	return false, err
}

// CreateBranch creates a new branch named name. If startPoint is "" it starts
// from HEAD.
func CreateBranch(ctx context.Context, r *Runner, name, startPoint string) error {
	args := []string{"branch", name}
	if startPoint != "" {
		args = append(args, startPoint)
	}
	_, err := r.Run(ctx, args...)
	return err
}

// CheckoutBranch checks out an existing local branch.
func CheckoutBranch(ctx context.Context, r *Runner, name string) error {
	_, err := r.Run(ctx, "checkout", name)
	return err
}

// DeleteBranch deletes a local branch; force uses -D instead of -d.
func DeleteBranch(ctx context.Context, r *Runner, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := r.Run(ctx, "branch", flag, name)
	return err
}

// RenameBranch renames a local branch.
func RenameBranch(ctx context.Context, r *Runner, oldName, newName string) error {
	_, err := r.Run(ctx, "branch", "-m", oldName, newName)
	return err
}

// PullBranch updates name from its upstream. When name is the current branch
// this is a plain `git pull`; otherwise it fetches the upstream directly into
// the local ref without checking it out.
func PullBranch(ctx context.Context, r *Runner, name string, current bool) error {
	if current {
		_, err := r.Run(ctx, "pull")
		return err
	}
	_, err := r.Run(ctx, "fetch", "origin", name+":"+name)
	return err
}

// PushBranch pushes name to its upstream (origin), setting the upstream on
// the first push.
func PushBranch(ctx context.Context, r *Runner, name string) error {
	_, err := r.Run(ctx, "push", "-u", "origin", name)
	return err
}

// TagRef creates a lightweight tag named tag pointing at target.
func TagRef(ctx context.Context, r *Runner, tag, target string) error {
	_, err := r.Run(ctx, "tag", tag, target)
	return err
}
