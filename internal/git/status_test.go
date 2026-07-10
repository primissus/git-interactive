package git

import (
	"context"
	"testing"
)

func TestGetStatus(t *testing.T) {
	r := newTestRepo(t)
	writeFile(t, r, "tracked.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")

	// staged
	writeFile(t, r, "staged.txt", "staged\n")
	mustGit(t, r, "add", "staged.txt")

	// unstaged modification
	writeFile(t, r, "tracked.txt", "modified\n")

	// untracked
	writeFile(t, r, "untracked.txt", "new\n")

	st, err := GetStatus(context.Background(), r)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if st.Branch != "main" {
		t.Errorf("Branch = %q, want main", st.Branch)
	}
	if len(st.Staged) != 1 || st.Staged[0].Path != "staged.txt" {
		t.Errorf("Staged = %+v, want [staged.txt]", st.Staged)
	}
	if len(st.Unstaged) != 1 || st.Unstaged[0].Path != "tracked.txt" {
		t.Errorf("Unstaged = %+v, want [tracked.txt]", st.Unstaged)
	}
	if len(st.Untracked) != 1 || st.Untracked[0].Path != "untracked.txt" {
		t.Errorf("Untracked = %+v, want [untracked.txt]", st.Untracked)
	}
}

func TestGetStatusConflict(t *testing.T) {
	r := newTestRepo(t)
	writeFile(t, r, "f.txt", "base\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "base")

	mustGit(t, r, "checkout", "-b", "feature")
	writeFile(t, r, "f.txt", "feature change\n")
	mustGit(t, r, "commit", "-am", "feature change")

	mustGit(t, r, "checkout", "main")
	writeFile(t, r, "f.txt", "main change\n")
	mustGit(t, r, "commit", "-am", "main change")

	cmd := &Runner{Dir: r.Dir}
	_, _ = cmd.Run(context.Background(), "merge", "feature")

	st, err := GetStatus(context.Background(), r)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(st.Conflicts) != 1 || st.Conflicts[0].Path != "f.txt" {
		t.Errorf("Conflicts = %+v, want [f.txt]", st.Conflicts)
	}
}

func TestStageUnstageFile(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial")
	writeFile(t, r, "a.txt", "two\n")

	if err := StageFile(ctx, r, "a.txt"); err != nil {
		t.Fatalf("StageFile: %v", err)
	}
	st, err := GetStatus(ctx, r)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(st.Staged) != 1 {
		t.Fatalf("Staged = %+v, want 1 entry", st.Staged)
	}

	if err := UnstageFile(ctx, r, "a.txt"); err != nil {
		t.Fatalf("UnstageFile: %v", err)
	}
	st, err = GetStatus(ctx, r)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(st.Staged) != 0 {
		t.Fatalf("Staged after unstage = %+v, want none", st.Staged)
	}
	if len(st.Unstaged) != 1 {
		t.Fatalf("Unstaged after unstage = %+v, want 1 entry", st.Unstaged)
	}
}

func TestStageAllUnstageAll(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	writeFile(t, r, "b.txt", "two\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial")
	writeFile(t, r, "a.txt", "changed\n")
	writeFile(t, r, "b.txt", "changed\n")

	if err := StageAll(ctx, r); err != nil {
		t.Fatalf("StageAll: %v", err)
	}
	st, err := GetStatus(ctx, r)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(st.Staged) != 2 {
		t.Fatalf("Staged after StageAll = %+v, want 2 entries", st.Staged)
	}

	if err := UnstageAll(ctx, r); err != nil {
		t.Fatalf("UnstageAll: %v", err)
	}
	st, err = GetStatus(ctx, r)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(st.Staged) != 0 {
		t.Fatalf("Staged after UnstageAll = %+v, want none", st.Staged)
	}
}

func TestDiscardFile(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial")
	writeFile(t, r, "a.txt", "changed\n")

	if err := DiscardFile(ctx, r, "a.txt", false); err != nil {
		t.Fatalf("DiscardFile(tracked): %v", err)
	}
	st, err := GetStatus(ctx, r)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(st.Unstaged) != 0 {
		t.Fatalf("Unstaged after discard = %+v, want none", st.Unstaged)
	}

	writeFile(t, r, "untracked.txt", "new\n")
	if err := DiscardFile(ctx, r, "untracked.txt", true); err != nil {
		t.Fatalf("DiscardFile(untracked): %v", err)
	}
	st, err = GetStatus(ctx, r)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(st.Untracked) != 0 {
		t.Fatalf("Untracked after discard = %+v, want none", st.Untracked)
	}
}

func TestCleanUntracked(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial")
	writeFile(t, r, "b.txt", "new\n")

	if err := CleanUntracked(ctx, r); err != nil {
		t.Fatalf("CleanUntracked: %v", err)
	}
	st, err := GetStatus(ctx, r)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(st.Untracked) != 0 {
		t.Fatalf("Untracked after clean = %+v, want none", st.Untracked)
	}
}

func TestDiffFile(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial")
	writeFile(t, r, "a.txt", "two\n")

	diff, err := DiffFile(ctx, r, "a.txt", false)
	if err != nil {
		t.Fatalf("DiffFile(unstaged): %v", err)
	}
	if diff == "" {
		t.Errorf("DiffFile(unstaged) is empty, want a diff")
	}

	mustGit(t, r, "add", "a.txt")
	diff, err = DiffFile(ctx, r, "a.txt", true)
	if err != nil {
		t.Fatalf("DiffFile(staged): %v", err)
	}
	if diff == "" {
		t.Errorf("DiffFile(staged) is empty, want a diff")
	}
}
