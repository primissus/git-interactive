package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// rebaseWindow caps how many commits the planning view offers when the branch
// has no upstream to fork from — a full-history rebase list is rarely wanted
// and slow to render.
const rebaseWindow = 20

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
// the log view), plus the base they sit on.
type RebasePlan struct {
	Commits []Commit
	Base    string // the commit to rebase onto; empty when Root is true
	Root    bool   // rebase from the root commit (--root) — Base has no parent
}

// PlanRebase gathers the commits eligible for interactive rebase. When the
// branch has an upstream it offers the commits ahead of the merge-base;
// otherwise it offers the most recent commits (capped at rebaseWindow) rebased
// onto the parent of the oldest.
func PlanRebase(ctx context.Context, r *Runner) (RebasePlan, error) {
	if upstream, err := r.Run(ctx, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}"); err == nil {
		up := strings.TrimSpace(upstream)
		base, err := mergeBase(ctx, r, "HEAD", up)
		if err != nil {
			return RebasePlan{}, err
		}
		commits, err := ListCommitsRange(ctx, r, base+"..HEAD")
		if err != nil {
			return RebasePlan{}, err
		}
		return RebasePlan{Commits: commits, Base: base}, nil
	}

	all, err := ListCommits(ctx, r)
	if err != nil {
		return RebasePlan{}, err
	}
	if len(all) == 0 {
		return RebasePlan{}, nil
	}
	if len(all) > rebaseWindow {
		commits := all[:rebaseWindow]
		return RebasePlan{Commits: commits, Base: commits[len(commits)-1].SHA + "^"}, nil
	}

	oldest := all[len(all)-1]
	root, err := rootCommit(ctx, r)
	if err != nil {
		return RebasePlan{}, err
	}
	if oldest.SHA == root {
		return RebasePlan{Commits: all, Root: true}, nil
	}
	return RebasePlan{Commits: all, Base: oldest.SHA + "^"}, nil
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

func mergeBase(ctx context.Context, r *Runner, a, b string) (string, error) {
	out, err := r.Run(ctx, "merge-base", a, b)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func rootCommit(ctx context.Context, r *Runner) (string, error) {
	out, err := r.Run(ctx, "rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		return "", err
	}
	// A repo can have several root commits; the last line is the oldest along
	// HEAD's first-parent history.
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[len(fields)-1], nil
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

	args := []string{"rebase", "-i"}
	if plan.Root {
		args = append(args, "--root")
	} else {
		args = append(args, plan.Base)
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
	Branch  string // the branch being rebased
	Onto    string // the base it is being replayed onto
	Current int    // 1-based index of the commit git stopped on
	Total   int    // total commits in the rebase
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
