package git

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

// Tag is one row of `git for-each-ref refs/tags/` output.
type Tag struct {
	Name     string
	Head     bool   // the checked-out commit is this tag's target
	Subject  string // the tag's own annotation message, or the commit's subject for a lightweight tag
	Date     string // relative, e.g. "3 days ago" — tagger date for annotated, commit date for lightweight
	DateUnix int64  // for date-bucket filtering and sorting
	Author   string // tagger name for annotated tags, commit author for lightweight
	SHA      string // the tagged commit's SHA (dereferenced past the tag object for annotated tags)
}

// tagFormat asks for-each-ref for name/subject/date/author uniformly across
// annotated and lightweight tags. %(contents:subject) and %(creatordate) are
// each valid on both a tag object and a commit object (git resolves them to
// the ref's immediate target either way), so no dereference is needed there.
// %(taggername) only exists on tag objects and %(authorname) only on commits,
// so an %(if)/%(then)/%(else) picks whichever one the immediate target has.
// objectname is captured both directly and dereferenced (the trailing
// %(*objectname)) so parseTags can resolve the underlying commit SHA even
// through an annotated tag's intermediate tag object.
const tagFormat = "%(refname:short)\x1f%(contents:subject)\x1f%(creatordate:relative)\x1f" +
	"%(creatordate:unix)\x1f%(if)%(taggername)%(then)%(taggername)%(else)%(authorname)%(end)\x1f" +
	"%(objectname)\x1f%(*objectname)"

// ListTags returns tags sorted by creation date, most recent first.
func ListTags(ctx context.Context, r *Runner) ([]Tag, error) {
	out, err := r.Run(ctx, "for-each-ref", "--format="+tagFormat, "--sort=-creatordate", "refs/tags/")
	if err != nil {
		return nil, err
	}
	head, err := RevParse(ctx, r, "HEAD")
	if err != nil {
		head = "" // detached/unborn HEAD: nothing to mark current
	}
	return parseTags(out, head), nil
}

func parseTags(out, head string) []Tag {
	lines := strings.Split(out, "\n")
	tags := make([]Tag, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\x1f")
		for len(fields) < 7 {
			fields = append(fields, "")
		}
		dateUnix, _ := strconv.ParseInt(fields[3], 10, 64)
		sha := fields[6] // dereferenced (annotated tag → its commit)
		if sha == "" {
			sha = fields[5] // already a commit (lightweight tag)
		}
		tags = append(tags, Tag{
			Name:     fields[0],
			Subject:  fields[1],
			Date:     fields[2],
			DateUnix: dateUnix,
			Author:   fields[4],
			SHA:      sha,
			Head:     head != "" && sha == head,
		})
	}
	return tags
}

// TagExists reports whether a tag named name exists.
func TagExists(ctx context.Context, r *Runner, name string) (bool, error) {
	_, err := r.Run(ctx, "show-ref", "--verify", "--quiet", "refs/tags/"+name)
	if err == nil {
		return true, nil
	}
	var gitErr *Error
	if errors.As(err, &gitErr) {
		return false, nil
	}
	return false, err
}

// CreateTag creates a lightweight tag named name. If target is "" it points
// at HEAD.
func CreateTag(ctx context.Context, r *Runner, name, target string) error {
	args := []string{"tag", name}
	if target != "" {
		args = append(args, target)
	}
	_, err := r.Run(ctx, args...)
	return err
}

// DeleteTag deletes a tag.
func DeleteTag(ctx context.Context, r *Runner, name string) error {
	_, err := r.Run(ctx, "tag", "-d", name)
	return err
}

// CheckoutTag checks out a tag, entering detached HEAD at its commit.
func CheckoutTag(ctx context.Context, r *Runner, name string) error {
	_, err := r.Run(ctx, "checkout", name)
	return err
}

// PushTag pushes a tag to its upstream (origin).
func PushTag(ctx context.Context, r *Runner, name string) error {
	_, err := r.Run(ctx, "push", "origin", name)
	return err
}
