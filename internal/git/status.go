package git

import (
	"context"
	"strconv"
	"strings"
)

// StatusEntry is one changed path from `git status --porcelain=v2`.
type StatusEntry struct {
	Code     string // XY, e.g. "M.", ".M", "??", "UU"
	Path     string
	OrigPath string // set for renames/copies
}

// Status is the parsed result of `git status --porcelain=v2 --branch`.
type Status struct {
	Branch    string // "" if detached
	Detached  bool
	OID       string
	Upstream  string
	Ahead     int
	Behind    int
	Staged    []StatusEntry
	Unstaged  []StatusEntry
	Conflicts []StatusEntry
	Untracked []StatusEntry
	Ignored   []StatusEntry
}

// GetStatus returns the parsed working tree status.
func GetStatus(ctx context.Context, r *Runner) (Status, error) {
	out, err := r.Run(ctx, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return Status{}, err
	}
	return parseStatus(out), nil
}

func parseStatus(out string) Status {
	var st Status

	for line := range strings.SplitSeq(out, "\n") {
		if line == "" {
			continue
		}
		switch line[0] {
		case '#':
			parseStatusHeader(&st, line)
		case '1':
			st.appendEntry(parseOrdinaryEntry(line))
		case '2':
			st.appendEntry(parseRenameEntry(line))
		case 'u':
			st.Conflicts = append(st.Conflicts, parseUnmergedEntry(line))
		case '?':
			st.Untracked = append(st.Untracked, StatusEntry{Code: "??", Path: line[2:]})
		case '!':
			st.Ignored = append(st.Ignored, StatusEntry{Code: "!!", Path: line[2:]})
		}
	}

	return st
}

func (st *Status) appendEntry(e StatusEntry) {
	if e.Code == "" {
		return
	}
	if e.Code[0] != '.' {
		st.Staged = append(st.Staged, e)
	}
	if e.Code[1] != '.' {
		st.Unstaged = append(st.Unstaged, e)
	}
}

func parseStatusHeader(st *Status, line string) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return
	}
	switch fields[1] {
	case "branch.oid":
		if len(fields) > 2 {
			st.OID = fields[2]
		}
	case "branch.head":
		if len(fields) > 2 {
			if fields[2] == "(detached)" {
				st.Detached = true
			} else {
				st.Branch = fields[2]
			}
		}
	case "branch.upstream":
		if len(fields) > 2 {
			st.Upstream = fields[2]
		}
	case "branch.ab":
		for _, f := range fields[2:] {
			n, _ := strconv.Atoi(strings.TrimLeft(f, "+-"))
			if strings.HasPrefix(f, "+") {
				st.Ahead = n
			} else if strings.HasPrefix(f, "-") {
				st.Behind = n
			}
		}
	}
}

// parseOrdinaryEntry parses a "1 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <path>" line.
func parseOrdinaryEntry(line string) StatusEntry {
	fields := strings.SplitN(line, " ", 9)
	if len(fields) < 9 {
		return StatusEntry{}
	}
	return StatusEntry{Code: fields[1], Path: fields[8]}
}

// parseRenameEntry parses a "2 <XY> <sub> ... <path><tab><origPath>" line.
func parseRenameEntry(line string) StatusEntry {
	fields := strings.SplitN(line, " ", 10)
	if len(fields) < 10 {
		return StatusEntry{}
	}
	pathAndOrig := strings.SplitN(fields[9], "\t", 2)
	e := StatusEntry{Code: fields[1], Path: pathAndOrig[0]}
	if len(pathAndOrig) > 1 {
		e.OrigPath = pathAndOrig[1]
	}
	return e
}

// parseUnmergedEntry parses a "u <XY> <sub> <m1> <m2> <m3> <mW> <hH> <hI> <hM> <path>" line.
func parseUnmergedEntry(line string) StatusEntry {
	fields := strings.SplitN(line, " ", 11)
	if len(fields) < 11 {
		return StatusEntry{}
	}
	return StatusEntry{Code: fields[1], Path: fields[10]}
}
