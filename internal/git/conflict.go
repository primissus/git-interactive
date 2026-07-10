package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// noEditorEnv keeps sequencer/commit steps from launching a real editor inside
// the TUI: git keeps whatever message it already prepared (the merge message, a
// squash's combined message) instead of blocking on $EDITOR.
var noEditorEnv = []string{"GIT_EDITOR=true", "GIT_SEQUENCE_EDITOR=true"}

// InProgressOp names a git operation that can stop on conflicts.
type InProgressOp string

const (
	OpMerge      InProgressOp = "merge"
	OpRebase     InProgressOp = "rebase"
	OpCherryPick InProgressOp = "cherry-pick"
	OpRevert     InProgressOp = "revert"
)

// InProgressState describes a stopped merge/rebase/cherry-pick/revert. status
// (phase 5) surfaces it with a plain continue/abort; the future
// resolve-conflicts command (PROMPT.md → Future) hooks into the same
// detection to offer per-conflict resolution, which is why detection and
// continue/abort live behind this type rather than being inlined into status.
type InProgressState struct {
	Op InProgressOp
}

// DetectInProgress reports the in-progress operation, if any, by checking for
// the marker files git itself uses. Returns (nil, nil) when nothing is in
// progress.
func DetectInProgress(ctx context.Context, r *Runner) (*InProgressState, error) {
	gitDir, err := gitDir(ctx, r)
	if err != nil {
		return nil, err
	}
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(gitDir, name))
		return err == nil
	}
	switch {
	case exists("MERGE_HEAD"):
		return &InProgressState{Op: OpMerge}, nil
	case exists("CHERRY_PICK_HEAD"):
		return &InProgressState{Op: OpCherryPick}, nil
	case exists("REVERT_HEAD"):
		return &InProgressState{Op: OpRevert}, nil
	case exists("rebase-merge"), exists("rebase-apply"):
		return &InProgressState{Op: OpRebase}, nil
	}
	return nil, nil
}

// Continue resolves the in-progress operation after its conflicts are fixed.
// It suppresses the editor so continuing past a squash/merge commit (which
// would otherwise open $EDITOR for the message) never blocks the TUI.
func (s *InProgressState) Continue(ctx context.Context, r *Runner) error {
	_, err := r.RunEnv(ctx, noEditorEnv, string(s.Op), "--continue")
	return err
}

// Abort cancels the in-progress operation, restoring the pre-operation state.
func (s *InProgressState) Abort(ctx context.Context, r *Runner) error {
	_, err := r.Run(ctx, string(s.Op), "--abort")
	return err
}

// CanSkip reports whether the in-progress operation supports --skip (rebase,
// cherry-pick, and revert do; merge does not).
func (s *InProgressState) CanSkip() bool { return s.Op != OpMerge }

// Skip drops the current commit and advances the operation. Only valid when
// CanSkip reports true.
func (s *InProgressState) Skip(ctx context.Context, r *Runner) error {
	_, err := r.RunEnv(ctx, noEditorEnv, string(s.Op), "--skip")
	return err
}

// ConflictSides carries the human-readable labels for the two sides of a
// conflict. During a rebase git's "ours"/"theirs" are inverted relative to a
// merge (git's "ours" is the branch being rebased onto), so the UI must never
// show raw "ours"/"theirs" — it shows these resolved names instead. Ours is the
// side `git checkout --ours` takes; Theirs is the `--theirs` side.
type ConflictSides struct {
	Ours   string
	Theirs string
}

// ResolveSides returns branch/commit-name labels for the "ours"/"theirs" sides
// of the in-progress operation's conflicts, accounting for the rebase
// inversion. The git command semantics are unchanged (--ours is always git's
// ours); only the labels differ, so the user sees a name rather than a side
// that means the opposite of what they expect.
func ResolveSides(ctx context.Context, r *Runner, s *InProgressState) ConflictSides {
	current := currentLabel(ctx, r)
	switch s.Op {
	case OpRebase:
		// git's "ours" during rebase is the branch being rebased onto; "theirs"
		// is the commit from the branch being replayed.
		onto, branch := rebaseSides(ctx, r)
		if onto == "" {
			onto = "new base"
		}
		if branch == "" {
			branch = "rebased commit"
		}
		return ConflictSides{Ours: onto, Theirs: branch}
	case OpMerge:
		return ConflictSides{Ours: current, Theirs: mergeTheirs(ctx, r)}
	case OpCherryPick:
		return ConflictSides{Ours: current, Theirs: friendlyRef(ctx, r, "CHERRY_PICK_HEAD")}
	case OpRevert:
		return ConflictSides{Ours: current, Theirs: friendlyRef(ctx, r, "REVERT_HEAD")}
	default:
		return ConflictSides{Ours: "ours", Theirs: "theirs"}
	}
}

// currentLabel is the checked-out branch, or a short HEAD sha when detached.
func currentLabel(ctx context.Context, r *Runner) string {
	if b, err := CurrentBranch(ctx, r); err == nil && b != "" {
		return b
	}
	if sha, err := RevParse(ctx, r, "HEAD"); err == nil {
		return shortSHA(sha)
	}
	return "HEAD"
}

// mergeTheirs labels the side being merged in, preferring the branch name from
// MERGE_MSG ("Merge branch 'x'") and falling back to MERGE_HEAD's name.
func mergeTheirs(ctx context.Context, r *Runner) string {
	if msg, err := readGitFile(ctx, r, "MERGE_MSG"); err == nil {
		first := strings.SplitN(msg, "\n", 2)[0]
		if name := branchFromMergeMsg(first); name != "" {
			return name
		}
	}
	return friendlyRef(ctx, r, "MERGE_HEAD")
}

// branchFromMergeMsg extracts "x" from "Merge branch 'x'" / "Merge branch 'x'
// into y", returning "" when the line has no such quote.
func branchFromMergeMsg(line string) string {
	start := strings.Index(line, "'")
	if start == -1 {
		return ""
	}
	end := strings.Index(line[start+1:], "'")
	if end == -1 {
		return ""
	}
	return line[start+1 : start+1+end]
}

// rebaseSides reads the rebasing branch and its new base from the rebase state
// directory, resolving both to friendly names.
func rebaseSides(ctx context.Context, r *Runner) (onto, branch string) {
	if raw, err := readGitFile(ctx, r, filepath.Join("rebase-merge", "head-name")); err == nil {
		branch = strings.TrimPrefix(strings.TrimSpace(raw), "refs/heads/")
	}
	if raw, err := readGitFile(ctx, r, filepath.Join("rebase-merge", "onto")); err == nil {
		onto = friendlyName(ctx, r, strings.TrimSpace(raw))
	}
	return onto, branch
}

// friendlyRef resolves the sha stored in a pseudo-ref file (MERGE_HEAD,
// CHERRY_PICK_HEAD…) to a friendly name.
func friendlyRef(ctx context.Context, r *Runner, name string) string {
	raw, err := readGitFile(ctx, r, name)
	if err != nil {
		return name
	}
	return friendlyName(ctx, r, strings.TrimSpace(raw))
}

// friendlyName turns a sha into the nearest ref name (via name-rev), stripping
// any "~N"/"^N" suffix, and falls back to the short sha.
func friendlyName(ctx context.Context, r *Runner, sha string) string {
	if sha == "" {
		return ""
	}
	out, err := r.Run(ctx, "name-rev", "--name-only", "--no-undefined", sha)
	if err != nil {
		return shortSHA(sha)
	}
	name := strings.TrimSpace(out)
	name = strings.TrimPrefix(name, "remotes/")
	if i := strings.IndexAny(name, "~^"); i != -1 {
		name = name[:i]
	}
	if name == "" {
		return shortSHA(sha)
	}
	return name
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// ConflictedFiles lists the paths with unresolved merge conflicts.
func ConflictedFiles(ctx context.Context, r *Runner) ([]string, error) {
	out, err := r.Run(ctx, "diff", "--name-only", "--diff-filter=U")
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

// TakeOurs resolves path to the "ours" side (git's ours) and stages it.
func TakeOurs(ctx context.Context, r *Runner, path string) error {
	return takeSide(ctx, r, "--ours", path)
}

// TakeTheirs resolves path to the "theirs" side (git's theirs) and stages it.
func TakeTheirs(ctx context.Context, r *Runner, path string) error {
	return takeSide(ctx, r, "--theirs", path)
}

func takeSide(ctx context.Context, r *Runner, side, path string) error {
	if _, err := r.Run(ctx, "checkout", side, "--", path); err != nil {
		return err
	}
	return StageFile(ctx, r, path)
}

// TakeBoth resolves path by keeping both sides of every conflict hunk (dropping
// the conflict markers and any diff3 base section) and stages it.
func TakeBoth(ctx context.Context, r *Runner, path string) error {
	root, err := RepoRoot(ctx, r)
	if err != nil {
		return err
	}
	full := filepath.Join(root, path)
	data, err := os.ReadFile(full)
	if err != nil {
		return err
	}
	merged := mergeBothSides(string(data))
	if err := os.WriteFile(full, []byte(merged), 0o644); err != nil {
		return err
	}
	return StageFile(ctx, r, path)
}

// mergeBothSides strips conflict markers, keeping both the "ours" and "theirs"
// content and discarding any diff3 base section (the lines between "|||||||"
// and "=======").
func mergeBothSides(content string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	inBase := false
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "<<<<<<<"):
			inBase = false
		case strings.HasPrefix(line, "|||||||"):
			inBase = true
		case strings.HasPrefix(line, "======="):
			inBase = false
		case strings.HasPrefix(line, ">>>>>>>"):
			inBase = false
		case !inBase:
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// RepoRoot returns the repository's working-tree root.
func RepoRoot(ctx context.Context, r *Runner) (string, error) {
	out, err := r.Run(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// readGitFile reads a file inside the .git directory (relative path), returning
// its contents.
func readGitFile(ctx context.Context, r *Runner, name string) (string, error) {
	dir, err := gitDir(ctx, r)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func gitDir(ctx context.Context, r *Runner) (string, error) {
	out, err := r.Run(ctx, "rev-parse", "--git-dir")
	if err != nil {
		return "", err
	}
	dir := strings.TrimSpace(out)
	if filepath.IsAbs(dir) {
		return dir, nil
	}
	base := r.Dir
	if base == "" {
		base = "."
	}
	return filepath.Join(base, dir), nil
}
