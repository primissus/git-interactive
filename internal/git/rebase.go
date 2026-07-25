package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RebaseOp is the action chosen for one commit in the rebase plan. The values
// match git's interactive-rebase todo keywords, except OpReword, which we drive
// with an `exec git commit --amend` line so the message can be captured up
// front rather than through a blocking editor.
type RebaseOp string

const (
	RebasePick   RebaseOp = "pick"
	RebaseReword RebaseOp = "reword"
	RebaseEdit   RebaseOp = "edit"
	RebaseSquash RebaseOp = "squash"
	RebaseFixup  RebaseOp = "fixup"
	RebaseDrop   RebaseOp = "drop"
)

// RebaseStep is one commit's entry in the rebase plan.
type RebaseStep struct {
	Commit  Commit
	Op      RebaseOp
	Message string // new subject when Op is RebaseReword
}

// RebasePlan is the set of commits a rebase will replay, newest first (matching
// the log view), plus the base they sit on and the branch being rebased.
type RebasePlan struct {
	Commits []Commit
	Base    string // the branch/ref to rebase onto
	Target  string // the branch being rebased; "" means HEAD
}

// PlanRebaseRange gathers the commits an interactive rebase of target onto
// base will replay — exactly `git log base..target`, newest first.
func PlanRebaseRange(ctx context.Context, r *Runner, target, base string) (RebasePlan, error) {
	commits, err := ListCommitsRange(ctx, r, base+".."+target)
	if err != nil {
		return RebasePlan{}, err
	}
	return RebasePlan{Commits: commits, Base: base, Target: target}, nil
}

// ListCommitsRange returns the commits in revRange (e.g. "base..HEAD"), newest
// first.
func ListCommitsRange(ctx context.Context, r *Runner, revRange string) ([]Commit, error) {
	out, err := r.Run(ctx, "log", "--pretty=format:"+logFormat, revRange)
	if err != nil {
		return nil, err
	}
	return parseCommits(out), nil
}

// ChangedFiles lists the paths a commit touched, for the edit-stop status view.
// --root makes a root commit report its files as a creation event instead of
// an empty diff.
func ChangedFiles(ctx context.Context, r *Runner, sha string) ([]string, error) {
	out, err := r.Run(ctx, "diff-tree", "--root", "--no-commit-id", "--name-only", "-r", sha)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// RunRebasePlan starts an interactive rebase driven by the plan's per-commit
// operations. It generates the todo list itself (via GIT_SEQUENCE_EDITOR) so no
// editor ever opens. It returns the git error verbatim when the rebase stops on
// a conflict or an `edit`; the caller inspects DetectInProgress to tell a stop
// from a hard failure.
func RunRebasePlan(ctx context.Context, r *Runner, plan RebasePlan, steps []RebaseStep) error {
	todo, err := renderTodo(steps)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp("", "gint-rebase-todo-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.WriteString(todo); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	// git invokes the sequence editor as `<GIT_SEQUENCE_EDITOR> <todofile>`, so
	// `cp -- <our todo>` overwrites git's generated todo with ours. GIT_EDITOR
	// is neutralized so squash message-combining never blocks.
	env := []string{
		"GIT_SEQUENCE_EDITOR=cp -- " + shellQuote(tmp.Name()),
		"GIT_EDITOR=true",
	}

	args := []string{"rebase", "-i", plan.Base}
	if plan.Target != "" {
		args = append(args, plan.Target)
	}
	_, err = r.RunEnv(ctx, env, args...)
	return err
}

// ValidateRebaseSteps reports whether a plan can be turned into a valid todo.
// steps are in display order (newest first). The only structural rule is that
// the oldest kept commit cannot be a squash/fixup, since nothing precedes it to
// fold into.
func ValidateRebaseSteps(steps []RebaseStep) error {
	dropped := 0
	for i := len(steps) - 1; i >= 0; i-- {
		s := steps[i]
		if s.Op == RebaseDrop {
			dropped++
			continue
		}
		if s.Op == RebaseSquash || s.Op == RebaseFixup {
			return fmt.Errorf("the oldest kept commit cannot be %s — nothing precedes it to fold into", s.Op)
		}
		break
	}
	if dropped == len(steps) && len(steps) > 0 {
		return fmt.Errorf("every commit is dropped — nothing would remain")
	}
	return nil
}

// renderTodo builds the interactive-rebase todo body from steps. steps are in
// display order (newest first); the todo is emitted oldest first, as git
// expects. Reword is expressed as pick + an `exec git commit --amend` so its
// message can be supplied without an editor.
func renderTodo(steps []RebaseStep) (string, error) {
	if err := ValidateRebaseSteps(steps); err != nil {
		return "", err
	}
	ordered := make([]RebaseStep, len(steps))
	for i, s := range steps {
		ordered[len(steps)-1-i] = s
	}

	var b strings.Builder
	for _, s := range ordered {
		switch s.Op {
		case RebaseReword:
			fmt.Fprintf(&b, "pick %s %s\n", s.Commit.SHA, s.Commit.Subject)
			fmt.Fprintf(&b, "exec git commit --amend -m %s\n", shellQuote(rewordMessage(s)))
		case RebaseDrop:
			fmt.Fprintf(&b, "drop %s %s\n", s.Commit.SHA, s.Commit.Subject)
		default:
			fmt.Fprintf(&b, "%s %s %s\n", s.Op, s.Commit.SHA, s.Commit.Subject)
		}
	}
	return b.String(), nil
}

func rewordMessage(s RebaseStep) string {
	if strings.TrimSpace(s.Message) != "" {
		return s.Message
	}
	return s.Commit.Subject
}

// shellQuote single-quotes s for the /bin/sh command line git builds for its
// editor hooks.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// RebaseProgress describes a stopped rebase for the resume header.
type RebaseProgress struct {
	Branch         string // the branch being rebased
	Onto           string // the base it is being replayed onto
	Current        int    // 1-based index of the commit git stopped on
	Total          int    // total commits in the rebase
	StoppedSHA     string // short sha of the commit git stopped on ("" when unknown)
	StoppedSubject string // that commit's subject line
}

// ReadRebaseProgress reads the current rebase's branch, onto, and progress
// counters from the rebase state directory. It tolerates a partially-populated
// state and returns whatever it can read.
func ReadRebaseProgress(ctx context.Context, r *Runner) (RebaseProgress, error) {
	dir, err := gitDir(ctx, r)
	if err != nil {
		return RebaseProgress{}, err
	}
	var p RebaseProgress
	onto, branch := rebaseSides(ctx, r)
	p.Onto, p.Branch = onto, branch

	// Interactive/merge rebases use rebase-merge/{msgnum,end}; am-based rebases
	// use rebase-apply/{next,last}.
	if cur, ok := readInt(filepath.Join(dir, "rebase-merge", "msgnum")); ok {
		p.Current = cur
		if total, ok := readInt(filepath.Join(dir, "rebase-merge", "end")); ok {
			p.Total = total
		}
	} else if cur, ok := readInt(filepath.Join(dir, "rebase-apply", "next")); ok {
		p.Current = cur
		if total, ok := readInt(filepath.Join(dir, "rebase-apply", "last")); ok {
			p.Total = total
		}
	}

	// REBASE_HEAD names the commit currently being replayed, for both
	// merge-based and am-based rebases; it may be absent in odd states.
	if out, err := r.Run(ctx, "log", "-1", "--pretty=format:%h\x1f%s", "REBASE_HEAD"); err == nil {
		if sha, subject, ok := strings.Cut(strings.TrimSpace(out), "\x1f"); ok {
			p.StoppedSHA, p.StoppedSubject = sha, subject
		}
	}
	return p, nil
}

func readInt(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}
	return n, true
}
