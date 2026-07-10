package git

import (
	"context"
	"testing"
)

// setupMergeFixture diverges main and feature so a merge produces a real
// merge commit rather than fast-forwarding.
func setupMergeFixture(t *testing.T) *Runner {
	t.Helper()
	r := newTestRepo(t)
	writeFile(t, r, "base.txt", "base\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "base")
	mustGit(t, r, "branch", "feature")
	mustGit(t, r, "checkout", "feature")
	writeFile(t, r, "feature.txt", "feature\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "feature commit")
	mustGit(t, r, "checkout", "main")
	writeFile(t, r, "main.txt", "main\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "main commit")
	return r
}

func TestMergeBranchDefault(t *testing.T) {
	ctx := context.Background()
	r := setupMergeFixture(t)

	if err := MergeBranch(ctx, r, "feature", MergeDefault); err != nil {
		t.Fatalf("MergeBranch: %v", err)
	}
	commits, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(commits) != 4 {
		t.Fatalf("commits = %+v, want 4 (base, main commit, feature commit, merge)", commits)
	}
}

func TestMergeBranchFFOnly(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "base")
	mustGit(t, r, "branch", "feature")
	mustGit(t, r, "checkout", "feature")
	writeFile(t, r, "b.txt", "two\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "feature commit")
	mustGit(t, r, "checkout", "main")

	if err := MergeBranch(ctx, r, "feature", MergeFFOnly); err != nil {
		t.Fatalf("MergeBranch(ff-only): %v", err)
	}
	commits, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("commits = %+v, want 2 (fast-forwarded, no merge commit)", commits)
	}
}

func TestMergeBranchNoFF(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "base")
	mustGit(t, r, "branch", "feature")
	mustGit(t, r, "checkout", "feature")
	writeFile(t, r, "b.txt", "two\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "feature commit")
	mustGit(t, r, "checkout", "main")

	if err := MergeBranch(ctx, r, "feature", MergeNoFF); err != nil {
		t.Fatalf("MergeBranch(no-ff): %v", err)
	}
	commits, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(commits) != 3 {
		t.Fatalf("commits = %+v, want 3 (base, feature, forced merge commit)", commits)
	}
}

func TestMergeBranchSquash(t *testing.T) {
	ctx := context.Background()
	r := setupMergeFixture(t)

	if err := MergeBranch(ctx, r, "feature", MergeSquash); err != nil {
		t.Fatalf("MergeBranch(squash): %v", err)
	}
	commits, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(commits) != 3 {
		t.Fatalf("commits = %+v, want 3 (base, main commit, single squash commit)", commits)
	}
	st, err := GetStatus(ctx, r)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(st.Staged) != 0 || len(st.Unstaged) != 0 {
		t.Fatalf("status after squash = %+v, want clean", st)
	}
}
