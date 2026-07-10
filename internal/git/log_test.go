package git

import (
	"context"
	"testing"
)

func TestListCommitsEmptyRepo(t *testing.T) {
	// A freshly-init'd repo has an unborn branch; ListCommits must return an
	// empty history rather than surfacing git's "no commits yet" error.
	r := newTestRepo(t)
	commits, err := ListCommits(context.Background(), r)
	if err != nil {
		t.Fatalf("ListCommits on empty repo: %v", err)
	}
	if len(commits) != 0 {
		t.Fatalf("ListCommits on empty repo: got %d commits, want 0", len(commits))
	}
}

func TestListCommits(t *testing.T) {
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "first")
	writeFile(t, r, "a.txt", "two\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "second")

	commits, err := ListCommits(context.Background(), r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("want 2 commits, got %d: %+v", len(commits), commits)
	}

	if commits[0].Subject != "second" {
		t.Errorf("commits[0].Subject = %q, want %q", commits[0].Subject, "second")
	}
	if commits[1].Subject != "first" {
		t.Errorf("commits[1].Subject = %q, want %q", commits[1].Subject, "first")
	}
	for _, c := range commits {
		if c.ShortSHA == "" {
			t.Errorf("commit %+v missing short SHA", c)
		}
		if c.AuthorName != "Test User" {
			t.Errorf("author = %q, want %q", c.AuthorName, "Test User")
		}
	}
	found := false
	for _, ref := range commits[0].Refs {
		if ref == "main" {
			found = true
		}
	}
	if !found {
		t.Errorf("commits[0].Refs = %v, want to contain main", commits[0].Refs)
	}
}

func TestCheckoutCommit(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "first")
	commits, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if err := CheckoutCommit(ctx, r, commits[0].SHA); err != nil {
		t.Fatalf("CheckoutCommit: %v", err)
	}
	cur, err := CurrentBranch(ctx, r)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if cur != "" {
		t.Errorf("CurrentBranch after detached checkout = %q, want empty (detached)", cur)
	}
}

func TestCherryPick(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "first")
	mustGit(t, r, "branch", "feature")
	mustGit(t, r, "checkout", "feature")
	writeFile(t, r, "b.txt", "two\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "add b")
	commits, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	pickSHA := commits[0].SHA
	mustGit(t, r, "checkout", "main")

	if err := CherryPick(ctx, r, []string{pickSHA}, false); err != nil {
		t.Fatalf("CherryPick: %v", err)
	}
	main, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if main[0].Subject != "add b" {
		t.Errorf("main HEAD subject = %q, want %q", main[0].Subject, "add b")
	}
}

func TestSquashHead(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "first")
	writeFile(t, r, "a.txt", "two\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "second")

	if err := SquashHead(ctx, r); err != nil {
		t.Fatalf("SquashHead: %v", err)
	}
	commits, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("want 1 commit after squash, got %d: %+v", len(commits), commits)
	}
	if commits[0].Subject != "first" {
		t.Errorf("squashed commit subject = %q, want %q (amend keeps the parent's message)", commits[0].Subject, "first")
	}
}

func TestResetTo(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "first")
	first, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	writeFile(t, r, "a.txt", "two\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "second")

	if err := ResetTo(ctx, r, first[0].SHA, ResetHard); err != nil {
		t.Fatalf("ResetTo(hard): %v", err)
	}
	commits, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(commits) != 1 || commits[0].Subject != "first" {
		t.Fatalf("after hard reset: %+v, want just 'first'", commits)
	}
}

func TestParseRefs(t *testing.T) {
	tests := []struct {
		raw  string
		want []string
	}{
		{"", nil},
		{"HEAD -> main, origin/main", []string{"main"}},
		{"HEAD -> main, tag: v1.0, feature", []string{"main", "feature"}},
		{"origin/main", nil},
	}
	for _, tt := range tests {
		got := parseRefs(tt.raw)
		if len(got) != len(tt.want) {
			t.Errorf("parseRefs(%q) = %v, want %v", tt.raw, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseRefs(%q) = %v, want %v", tt.raw, got, tt.want)
				break
			}
		}
	}
}
