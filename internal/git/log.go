package git

import (
	"context"
	"strings"
)

// Commit is one row of `git log` output.
type Commit struct {
	ShortSHA   string
	Subject    string
	RelDate    string
	AuthorName string
	Refs       []string // local branch names pointing at this commit
}

const logFormat = "%h\x1f%s\x1f%cr\x1f%an\x1f%D"

// ListCommits returns the commit history reachable from HEAD, most recent first.
func ListCommits(ctx context.Context, r *Runner) ([]Commit, error) {
	out, err := r.Run(ctx, "log", "--pretty=format:"+logFormat)
	if err != nil {
		return nil, err
	}
	return parseCommits(out), nil
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
		for len(fields) < 5 {
			fields = append(fields, "")
		}
		commits = append(commits, Commit{
			ShortSHA:   fields[0],
			Subject:    fields[1],
			RelDate:    fields[2],
			AuthorName: fields[3],
			Refs:       parseRefs(fields[4]),
		})
	}
	return commits
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
