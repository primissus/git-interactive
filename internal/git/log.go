package git

import (
	"context"
	"strconv"
	"strings"
)

// Commit is one row of `git log` output.
type Commit struct {
	SHA        string
	ShortSHA   string
	Subject    string
	RelDate    string
	AbsDate    string // ISO 8601 committer date, for -F/--full
	CommitUnix int64  // committerdate:unix, for short/iso date formatting
	AuthorName string
	Refs       []string // local branch names pointing at this commit
}

const logFormat = "%H\x1f%h\x1f%s\x1f%cr\x1f%an\x1f%D\x1f%cI\x1f%ct"

// ListCommits returns the commit history reachable from HEAD, most recent first.
func ListCommits(ctx context.Context, r *Runner) ([]Commit, error) {
	if !hasCommits(ctx, r) {
		return nil, nil // unborn branch (freshly-init'd repo): an empty history
	}
	out, err := r.Run(ctx, "log", "--pretty=format:"+logFormat)
	if err != nil {
		return nil, err
	}
	return parseCommits(out), nil
}

// hasCommits reports whether HEAD resolves to a commit. It is false on an
// unborn branch — a `git init`'d repo with no commits yet — where `git log`
// would otherwise fail; callers use it to render an empty history instead.
func hasCommits(ctx context.Context, r *Runner) bool {
	_, err := r.Run(ctx, "rev-parse", "--verify", "--quiet", "HEAD")
	return err == nil
}

func parseCommits(out string) []Commit {
	if strings.TrimSpace(out) == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	commits := make([]Commit, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\x1f")
		for len(fields) < 8 {
			fields = append(fields, "")
		}
		commitUnix, _ := strconv.ParseInt(fields[7], 10, 64)
		commits = append(commits, Commit{
			SHA:        fields[0],
			ShortSHA:   fields[1],
			Subject:    fields[2],
			RelDate:    fields[3],
			AuthorName: fields[4],
			Refs:       parseRefs(fields[5]),
			AbsDate:    fields[6],
			CommitUnix: commitUnix,
		})
	}
	return commits
}

// CheckoutCommit checks out sha directly, detaching HEAD.
func CheckoutCommit(ctx context.Context, r *Runner, sha string) error {
	_, err := r.Run(ctx, "checkout", sha)
	return err
}

// CherryPick cherry-picks shas (oldest first) onto HEAD. noCommit stages the
// changes without creating a commit (git cherry-pick --no-commit).
func CherryPick(ctx context.Context, r *Runner, shas []string, noCommit bool) error {
	args := []string{"cherry-pick"}
	if noCommit {
		args = append(args, "--no-commit")
	}
	args = append(args, shas...)
	_, err := r.Run(ctx, args...)
	return err
}

// SquashHead folds HEAD into its parent, keeping the parent's message: a soft
// reset one commit back followed by an amend. Only HEAD can be squashed this
// way; squashing an arbitrary older commit needs a rebase (phase 7).
func SquashHead(ctx context.Context, r *Runner) error {
	if _, err := r.Run(ctx, "reset", "--soft", "HEAD~1"); err != nil {
		return err
	}
	_, err := r.Run(ctx, "commit", "--amend", "--no-edit")
	return err
}

// ResetMode selects how far ResetTo unwinds the working tree/index.
type ResetMode string

const (
	ResetSoft  ResetMode = "soft"
	ResetMixed ResetMode = "mixed"
	ResetHard  ResetMode = "hard"
)

// ResetTo resets the current branch to sha using the given mode.
func ResetTo(ctx context.Context, r *Runner, sha string, mode ResetMode) error {
	_, err := r.Run(ctx, "reset", "--"+string(mode), sha)
	return err
}

// parseRefs turns `%D` output (e.g. "HEAD -> main, origin/main, tag: v1")
// into local branch names, dropping HEAD, remotes, and tags.
func parseRefs(raw string) []string {
	if raw == "" {
		return nil
	}
	var refs []string
	for part := range strings.SplitSeq(raw, ",") {
		name := strings.TrimSpace(part)
		if name == "" || name == "HEAD" {
			continue
		}
		if idx := strings.Index(name, "-> "); idx != -1 {
			name = name[idx+3:]
		}
		if strings.HasPrefix(name, "tag: ") || strings.Contains(name, "/") {
			continue
		}
		refs = append(refs, name)
	}
	return refs
}
