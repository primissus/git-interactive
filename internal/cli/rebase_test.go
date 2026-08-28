package cli

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

func TestRebaseStepItemColumns(t *testing.T) {
	c := git.Commit{ShortSHA: "abc1234", Subject: "original subject"}

	pick := rebaseStepItem{step: &git.RebaseStep{Commit: c, Op: git.RebasePick}}
	cols := pick.Columns()
	if cols[0] != "pick" || cols[1] != "abc1234" || cols[2] != "original subject" {
		t.Fatalf("pick columns = %v", cols)
	}

	reword := rebaseStepItem{step: &git.RebaseStep{Commit: c, Op: git.RebaseReword, Message: "new subject"}}
	if got := reword.Columns()[2]; got != "new subject  (was: original subject)" {
		t.Fatalf("reword message column = %q", got)
	}

	// A reword with no message yet falls back to the original subject.
	empty := rebaseStepItem{step: &git.RebaseStep{Commit: c, Op: git.RebaseReword}}
	if got := empty.Columns()[2]; got != "original subject" {
		t.Fatalf("empty reword column = %q", got)
	}
}

func TestConflictPath(t *testing.T) {
	if _, ok := conflictPath(nil); ok {
		t.Error("conflictPath(nil) should be false")
	}
	if _, ok := conflictPath([]tui.Item{conflictItem{path: "a", conflict: true}, conflictItem{path: "b", conflict: true}}); ok {
		t.Error("conflictPath with two rows should be false")
	}
	got, ok := conflictPath([]tui.Item{conflictItem{path: "f.txt", conflict: true}})
	if !ok || got != "f.txt" {
		t.Errorf("conflictPath = %q, %v; want f.txt, true", got, ok)
	}
	// An informational edit-stop row (changed file, no conflict) is rejected:
	// the per-file resolution ops only apply to conflicted files.
	if _, ok := conflictPath([]tui.Item{conflictItem{path: "f.txt"}}); ok {
		t.Error("conflictPath(non-conflict row) should be false")
	}
}

func TestStatusConflictPath(t *testing.T) {
	conflict := statusItem{e: git.StatusEntry{Code: "UU", Path: "c.txt"}, conflict: true}
	if got, ok := statusConflictPath([]tui.Item{conflict}); !ok || got != "c.txt" {
		t.Errorf("statusConflictPath(conflict) = %q, %v; want c.txt, true", got, ok)
	}
	// A non-conflict row is rejected: the file-resolution ops only apply to
	// conflicted files.
	clean := statusItem{e: git.StatusEntry{Code: "M.", Path: "m.txt"}}
	if _, ok := statusConflictPath([]tui.Item{clean}); ok {
		t.Error("statusConflictPath(non-conflict) should be false")
	}
}

func TestRebaseSelectItemColumns(t *testing.T) {
	b := git.Branch{Name: "feat/x", Subject: "some work", CommitDate: "2 days ago", AuthorName: "Test User"}

	plain := rebaseSelectItem{b: b}
	if got := plain.Columns()[0]; got != "feat/x" {
		t.Fatalf("unmarked name column = %q, want feat/x", got)
	}
	base := rebaseSelectItem{b: b, role: "base"}
	if got := base.Columns()[0]; got != "(base) feat/x" {
		t.Fatalf("base name column = %q, want (base) feat/x", got)
	}
	target := rebaseSelectItem{b: b, role: "target"}
	if got := target.Columns()[0]; got != "(target) feat/x" {
		t.Fatalf("target name column = %q, want (target) feat/x", got)
	}
	// The fuzzy filter matches the bare branch name, not the mark prefix.
	if plain.FilterValue() != "feat/x" {
		t.Fatalf("FilterValue = %q, want feat/x", plain.FilterValue())
	}
}

func TestRebaseMarksRole(t *testing.T) {
	m := &rebaseMarks{base: "main", target: "feat/x"}
	if got := m.role("main"); got != "base" {
		t.Errorf("role(main) = %q, want base", got)
	}
	if got := m.role("feat/x"); got != "target" {
		t.Errorf("role(feat/x) = %q, want target", got)
	}
	if got := m.role("other"); got != "" {
		t.Errorf("role(other) = %q, want empty", got)
	}
	// Empty marks mark nothing.
	empty := &rebaseMarks{}
	if got := empty.role("main"); got != "" {
		t.Errorf("empty marks role(main) = %q, want empty", got)
	}
}

// runMarkOp drives a selector mark operation against a single row, returning
// the resulting status/items command effects (executed synchronously — the
// ops emit tea.Batch of plain msgs).
func runMarkOp(t *testing.T, opName string, marks *rebaseMarks, branches []git.Branch, row int) {
	t.Helper()
	applied := false
	ops := buildRebaseSelectOps(marks, branches, &applied)
	for _, op := range ops {
		if op.Name != opName {
			continue
		}
		items := rebaseSelectItems(branches, marks)
		cmd := op.Run(tui.OpContext{Items: []tui.Item{items[row]}})
		if cmd == nil {
			return
		}
		// Drain the returned command's messages; SetItems/Status need no
		// further handling here — marks are mutated synchronously by Run.
		_ = cmd()
		return
	}
	t.Fatalf("operation %q not found", opName)
}

func TestRebaseSelectMarkOps(t *testing.T) {
	branches := []git.Branch{{Name: "main"}, {Name: "feat/x"}, {Name: "feat/y"}}
	marks := &rebaseMarks{}

	// Mark base and target.
	runMarkOp(t, "onto different branch", marks, branches, 0)
	if marks.base != "main" {
		t.Fatalf("marks.base = %q, want main", marks.base)
	}
	runMarkOp(t, "rebase this branch instead", marks, branches, 1)
	if marks.target != "feat/x" {
		t.Fatalf("marks.target = %q, want feat/x", marks.target)
	}

	// Overwriting a mark is allowed.
	runMarkOp(t, "onto different branch", marks, branches, 2)
	if marks.base != "feat/y" {
		t.Fatalf("marks.base after overwrite = %q, want feat/y", marks.base)
	}

	// Marking the other role's branch is rejected and keeps the old mark.
	runMarkOp(t, "onto different branch", marks, branches, 1)
	if marks.base != "feat/y" {
		t.Fatalf("marks.base after rejected mark = %q, want feat/y (unchanged)", marks.base)
	}
}

func TestDefaultRebaseMarks(t *testing.T) {
	branches := []git.Branch{{Name: "main"}, {Name: "feat/x", Head: true}}
	m := defaultRebaseMarks(branches)
	if m.target != "feat/x" {
		t.Fatalf("default target = %q, want feat/x", m.target)
	}
	if m.base != "" {
		t.Fatalf("default base = %q, want empty", m.base)
	}
	// Detached HEAD: no current branch, so nothing is pre-marked.
	detached := []git.Branch{{Name: "main"}}
	if m2 := defaultRebaseMarks(detached); m2.target != "" || m2.base != "" {
		t.Fatalf("detached default marks = target %q base %q, want both empty", m2.target, m2.base)
	}
}

func TestRebaseApplyRequiresBothMarks(t *testing.T) {
	branches := []git.Branch{{Name: "main"}, {Name: "feat/x", Head: true}}

	// The zero-arg selector pre-marks the target; that must not let apply
	// succeed on its own — a base is still required.
	runApply := func(marks *rebaseMarks) (applied bool, cmd tea.Cmd) {
		applied = false
		ops := buildRebaseSelectOps(marks, branches, &applied)
		for _, op := range ops {
			if op.Name == "apply" {
				return applied, op.Run(tui.OpContext{})
			}
		}
		t.Fatal("apply operation not found")
		return false, nil
	}

	applied, cmd := runApply(defaultRebaseMarks(branches))
	if applied {
		t.Fatal("apply succeeded with only a target marked; a base must be required")
	}
	if cmd == nil {
		t.Fatal("apply with only a target should emit a status, got nil command")
	}

	// Once a base is also marked, apply succeeds.
	marks := defaultRebaseMarks(branches)
	marks.base = "main"
	applied, cmd = runApply(marks)
	if !applied {
		t.Fatal("apply with both marks should succeed")
	}
	if cmd == nil {
		t.Fatal("apply with both marks should emit a quit command")
	}
}
