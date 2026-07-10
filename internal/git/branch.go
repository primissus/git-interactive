package git

import (
	"context"
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

// branchFormat mirrors git-br.py's field order (name, subject, date, author,
// worktree) plus machine-readable fields (unix timestamps, upstream tracking
// state) for later sorting/filtering.
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
