package git

import (
	"context"
	"strings"
)

// graphSentinel marks where a graph line's commit data begins, so parseGraph
// can split git's ASCII graph glyphs (which precede the format string on each
// commit line, and are the entire content of pure connector lines) from the
// commit fields.
const graphSentinel = "\x02"

// GraphRow is one line of a `git log --graph` view: the ASCII graph glyphs
// for that line, plus the commit it depicts when the line represents one
// (connector-only lines between graph nodes carry no commit).
type GraphRow struct {
	Prefix    string
	Commit    Commit
	HasCommit bool
}

// ListCommitGraph returns the commit graph. all selects every local branch as
// a starting point (PROMPT.md's -A/--not-all default of "all"); false bases
// the graph on HEAD alone. simplify collapses commits with no ref decoration,
// which is graph-branch's "each branch's last commit only" view.
func ListCommitGraph(ctx context.Context, r *Runner, all, simplify bool) ([]GraphRow, error) {
	if !hasCommits(ctx, r) {
		return nil, nil // unborn branch: nothing to graph yet
	}
	args := []string{"log", "--graph", "--pretty=format:" + graphSentinel + logFormat}
	if all {
		args = append(args, "--all")
	}
	if simplify {
		args = append(args, "--simplify-by-decoration")
	}
	out, err := r.Run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parseGraph(out), nil
}

func parseGraph(out string) []GraphRow {
	if strings.TrimSpace(out) == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	rows := make([]GraphRow, 0, len(lines))
	for _, line := range lines {
		idx := strings.Index(line, graphSentinel)
		if idx == -1 {
			rows = append(rows, GraphRow{Prefix: line})
			continue
		}
		prefix := line[:idx]
		fields := strings.Split(line[idx+len(graphSentinel):], "\x1f")
		for len(fields) < 7 {
			fields = append(fields, "")
		}
		rows = append(rows, GraphRow{
			Prefix:    prefix,
			HasCommit: true,
			Commit: Commit{
				SHA: fields[0], ShortSHA: fields[1], Subject: fields[2],
				RelDate: fields[3], AuthorName: fields[4], Refs: parseRefs(fields[5]),
				AbsDate: fields[6],
			},
		})
	}
	return rows
}
