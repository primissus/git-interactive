package git

import (
	"context"
	"testing"
)

func TestCommitStaged(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")

	if err := CommitStaged(ctx, r, "first", false); err != nil {
		t.Fatalf("CommitStaged: %v", err)
	}
	commits, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(commits) != 1 || commits[0].Subject != "first" {
		t.Fatalf("commits = %+v, want [first]", commits)
	}
}

func TestAmendCommit(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "original")
	writeFile(t, r, "a.txt", "two\n")
	mustGit(t, r, "add", ".")

	if err := AmendCommit(ctx, r, "amended message", false); err != nil {
		t.Fatalf("AmendCommit: %v", err)
	}
	commits, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(commits) != 1 || commits[0].Subject != "amended message" {
		t.Fatalf("commits = %+v, want [amended message]", commits)
	}
}

func TestAmendCommitNoEdit(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "original")
	writeFile(t, r, "a.txt", "two\n")
	mustGit(t, r, "add", ".")

	if err := AmendCommit(ctx, r, "", false); err != nil {
		t.Fatalf("AmendCommit(no message): %v", err)
	}
	commits, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(commits) != 1 || commits[0].Subject != "original" {
		t.Fatalf("commits = %+v, want message kept as [original]", commits)
	}
}

func TestIsCommitPushed(t *testing.T) {
	ctx := context.Background()
	remoteDir := t.TempDir()
	mustGit(t, &Runner{Dir: remoteDir}, "init", "-q", "--bare", "-b", "main")

	r := newTestRepo(t)
	mustGit(t, r, "remote", "add", "origin", remoteDir)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "first")

	sha, err := RevParse(ctx, r, "HEAD")
	if err != nil {
		t.Fatalf("RevParse: %v", err)
	}
	pushed, err := IsCommitPushed(ctx, r, sha)
	if err != nil {
		t.Fatalf("IsCommitPushed(before push): %v", err)
	}
	if pushed {
		t.Errorf("IsCommitPushed(before push) = true, want false")
	}

	mustGit(t, r, "push", "origin", "main")
	pushed, err = IsCommitPushed(ctx, r, sha)
	if err != nil {
		t.Fatalf("IsCommitPushed(after push): %v", err)
	}
	if !pushed {
		t.Errorf("IsCommitPushed(after push) = false, want true")
	}
}
