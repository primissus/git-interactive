package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestListWorktrees(t *testing.T) {
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "hello\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")

	wtDir := t.TempDir()
	mustGit(t, r, "worktree", "add", "-b", "feature", wtDir)
	resolvedWtDir, err := filepath.EvalSymlinks(wtDir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}

	worktrees, err := ListWorktrees(context.Background(), r)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if len(worktrees) != 2 {
		t.Fatalf("want 2 worktrees, got %d: %+v", len(worktrees), worktrees)
	}

	byBranch := map[string]Worktree{}
	for _, w := range worktrees {
		byBranch[w.Branch] = w
	}
	if _, ok := byBranch["main"]; !ok {
		t.Errorf("missing main worktree: %+v", worktrees)
	}
	feature, ok := byBranch["feature"]
	if !ok {
		t.Fatalf("missing feature worktree: %+v", worktrees)
	}
	if feature.Path != resolvedWtDir {
		t.Errorf("feature.Path = %q, want %q", feature.Path, resolvedWtDir)
	}
}

func TestAddAndRemoveWorktree(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "hello\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")

	wtDir := t.TempDir()
	resolvedWtDir, err := filepath.EvalSymlinks(wtDir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if err := AddWorktree(ctx, r, wtDir, "feature", true, ""); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	worktrees, err := ListWorktrees(ctx, r)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	found := false
	for _, w := range worktrees {
		if w.Path == resolvedWtDir {
			found = true
		}
	}
	if !found {
		t.Fatalf("worktree not found after AddWorktree: %+v", worktrees)
	}

	if err := RemoveWorktree(ctx, r, wtDir, false); err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}
	worktrees, err = ListWorktrees(ctx, r)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	for _, w := range worktrees {
		if w.Path == resolvedWtDir {
			t.Fatalf("worktree still present after RemoveWorktree: %+v", worktrees)
		}
	}
}

func TestLockUnlockWorktree(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "hello\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")

	wtDir := t.TempDir()
	if err := AddWorktree(ctx, r, wtDir, "feature", true, ""); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	if err := LockWorktree(ctx, r, wtDir, "testing"); err != nil {
		t.Fatalf("LockWorktree: %v", err)
	}
	worktrees, err := ListWorktrees(ctx, r)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	var locked *Worktree
	for i := range worktrees {
		if worktrees[i].Branch == "feature" {
			locked = &worktrees[i]
		}
	}
	if locked == nil || !locked.Locked {
		t.Fatalf("worktree not locked: %+v", worktrees)
	}

	if err := UnlockWorktree(ctx, r, wtDir); err != nil {
		t.Fatalf("UnlockWorktree: %v", err)
	}
	worktrees, err = ListWorktrees(ctx, r)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	for _, w := range worktrees {
		if w.Branch == "feature" && w.Locked {
			t.Fatalf("worktree still locked after UnlockWorktree: %+v", worktrees)
		}
	}
}

func TestPruneWorktrees(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "hello\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")

	wtDir := t.TempDir()
	if err := AddWorktree(ctx, r, wtDir, "feature", true, ""); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}
	if err := os.RemoveAll(wtDir); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}

	if err := PruneWorktrees(ctx, r); err != nil {
		t.Fatalf("PruneWorktrees: %v", err)
	}
	worktrees, err := ListWorktrees(ctx, r)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	for _, w := range worktrees {
		if w.Branch == "feature" {
			t.Fatalf("pruned worktree still listed: %+v", worktrees)
		}
	}
}
