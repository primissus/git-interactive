package git

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

// Stash is one row of `git stash list` output.
type Stash struct {
	Index   int
	Ref     string // e.g. "stash@{0}"
	Branch  string
	Message string
	RelDate string
	Unix    int64 // committerdate:unix, for short/iso date formatting
}

const stashFormat = "%gd\x1f%gs\x1f%cr\x1f%ct"

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
		for len(fields) < 4 {
			fields = append(fields, "")
		}
		branch, message := "", fields[1]
		if m := stashSubjectRe.FindStringSubmatch(fields[1]); m != nil {
			branch, message = m[1], m[2]
		}
		unix, _ := strconv.ParseInt(fields[3], 10, 64)
		stashes = append(stashes, Stash{
			Index:   i,
			Ref:     fields[0],
			Branch:  branch,
			Message: message,
			RelDate: fields[2],
			Unix:    unix,
		})
	}
	return stashes
}

// StashPush creates a new stash. When paths is non-empty, only those paths
// are stashed (the per-file "stash the selected file" operation); an empty
// paths stashes every change.
func StashPush(ctx context.Context, r *Runner, message string, paths ...string) error {
	args := []string{"stash", "push"}
	if message != "" {
		args = append(args, "-m", message)
	}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}
	_, err := r.Run(ctx, args...)
	return err
}

// StashApply applies ref without removing it from the stash list.
func StashApply(ctx context.Context, r *Runner, ref string) error {
	_, err := r.Run(ctx, "stash", "apply", ref)
	return err
}

// StashPop applies ref and removes it from the stash list.
func StashPop(ctx context.Context, r *Runner, ref string) error {
	_, err := r.Run(ctx, "stash", "pop", ref)
	return err
}

// StashDrop removes ref from the stash list without applying it.
func StashDrop(ctx context.Context, r *Runner, ref string) error {
	_, err := r.Run(ctx, "stash", "drop", ref)
	return err
}

// StashClear removes every stash.
func StashClear(ctx context.Context, r *Runner) error {
	_, err := r.Run(ctx, "stash", "clear")
	return err
}

// StashDiff returns ref's diff against the commit it was stashed from.
func StashDiff(ctx context.Context, r *Runner, ref string) (string, error) {
	return r.Run(ctx, "stash", "show", "-p", ref)
}
