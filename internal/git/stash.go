package git

import (
	"context"
	"regexp"
	"strings"
)

// Stash is one row of `git stash list` output.
type Stash struct {
	Index   int
	Ref     string // e.g. "stash@{0}"
	Branch  string
	Message string
	RelDate string
}

const stashFormat = "%gd\x1f%gs\x1f%cr"

var stashSubjectRe = regexp.MustCompile(`^(?:WIP )?[Oo]n ([^:]+): (.*)$`)

// ListStashes returns the stash list, most recent first.
func ListStashes(ctx context.Context, r *Runner) ([]Stash, error) {
	out, err := r.Run(ctx, "stash", "list", "--format="+stashFormat)
	if err != nil {
		return nil, err
	}
	return parseStashes(out), nil
}

func parseStashes(out string) []Stash {
	if strings.TrimSpace(out) == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	stashes := make([]Stash, 0, len(lines))
	for i, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\x1f")
		for len(fields) < 3 {
			fields = append(fields, "")
		}
		branch, message := "", fields[1]
		if m := stashSubjectRe.FindStringSubmatch(fields[1]); m != nil {
			branch, message = m[1], m[2]
		}
		stashes = append(stashes, Stash{
			Index:   i,
			Ref:     fields[0],
			Branch:  branch,
			Message: message,
			RelDate: fields[2],
		})
	}
	return stashes
}
