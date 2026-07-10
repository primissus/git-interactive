package cli

import (
	"testing"

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
	if _, ok := conflictPath([]tui.Item{conflictItem{"a"}, conflictItem{"b"}}); ok {
		t.Error("conflictPath with two rows should be false")
	}
	got, ok := conflictPath([]tui.Item{conflictItem{"f.txt"}})
	if !ok || got != "f.txt" {
		t.Errorf("conflictPath = %q, %v; want f.txt, true", got, ok)
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
