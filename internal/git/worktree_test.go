package git

import (
	"context"
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
