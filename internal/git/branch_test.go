package git

import (
	"context"
	"testing"
)

func TestListBranches(t *testing.T) {
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "hello\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")
	mustGit(t, r, "branch", "feature")

	branches, err := ListBranches(context.Background(), r)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	if len(branches) != 2 {
		t.Fatalf("want 2 branches, got %d: %+v", len(branches), branches)
	}

	byName := map[string]Branch{}
	for _, b := range branches {
		byName[b.Name] = b
	}

	main, ok := byName["main"]
	if !ok {
		t.Fatalf("missing main branch: %+v", branches)
	}
	if !main.Head {
		t.Errorf("main should be HEAD")
	}
	if main.Subject != "initial commit" {
		t.Errorf("subject = %q, want %q", main.Subject, "initial commit")
	}
	if main.AuthorName != "Test User" {
		t.Errorf("author = %q, want %q", main.AuthorName, "Test User")
	}

	feature, ok := byName["feature"]
	if !ok {
		t.Fatalf("missing feature branch: %+v", branches)
	}
	if feature.Head {
		t.Errorf("feature should not be HEAD")
	}
}

func TestBranchExists(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "hello\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")
	mustGit(t, r, "branch", "feature")

	exists, err := BranchExists(ctx, r, "feature")
	if err != nil || !exists {
		t.Fatalf("BranchExists(feature) = %v, %v; want true, nil", exists, err)
	}
	exists, err = BranchExists(ctx, r, "nope")
	if err != nil || exists {
		t.Fatalf("BranchExists(nope) = %v, %v; want false, nil", exists, err)
	}
}

func TestCreateCheckoutDeleteBranch(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "hello\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")

	if err := CreateBranch(ctx, r, "feature", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	exists, err := BranchExists(ctx, r, "feature")
	if err != nil || !exists {
		t.Fatalf("branch not created: %v, %v", exists, err)
	}

	if err := CheckoutBranch(ctx, r, "feature"); err != nil {
		t.Fatalf("CheckoutBranch: %v", err)
	}
	cur, err := CurrentBranch(ctx, r)
	if err != nil || cur != "feature" {
		t.Fatalf("CurrentBranch = %q, %v; want feature", cur, err)
	}

	if err := CheckoutBranch(ctx, r, "main"); err != nil {
		t.Fatalf("CheckoutBranch(main): %v", err)
	}
	if err := DeleteBranch(ctx, r, "feature", false); err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}
	exists, err = BranchExists(ctx, r, "feature")
	if err != nil || exists {
		t.Fatalf("branch not deleted: %v, %v", exists, err)
	}
}

func TestDeleteBranchUnmergedRequiresForce(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "hello\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")
	mustGit(t, r, "branch", "feature")
	mustGit(t, r, "checkout", "feature")
	writeFile(t, r, "b.txt", "world\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "unmerged commit")
	mustGit(t, r, "checkout", "main")

	if err := DeleteBranch(ctx, r, "feature", false); err == nil {
		t.Fatalf("DeleteBranch(force=false) on unmerged branch: want error, got nil")
	}
	if err := DeleteBranch(ctx, r, "feature", true); err != nil {
		t.Fatalf("DeleteBranch(force=true): %v", err)
	}
}

func TestRenameBranch(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "hello\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")
	mustGit(t, r, "branch", "old-name")

	if err := RenameBranch(ctx, r, "old-name", "new-name"); err != nil {
		t.Fatalf("RenameBranch: %v", err)
	}
	if exists, _ := BranchExists(ctx, r, "old-name"); exists {
		t.Errorf("old-name should no longer exist")
	}
	if exists, _ := BranchExists(ctx, r, "new-name"); !exists {
		t.Errorf("new-name should exist")
	}
}

func TestMergedBranches(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "hello\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")
	mustGit(t, r, "branch", "merged-branch")
	mustGit(t, r, "branch", "unmerged-branch")
	mustGit(t, r, "checkout", "unmerged-branch")
	writeFile(t, r, "b.txt", "world\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "unmerged commit")
	mustGit(t, r, "checkout", "main")

	merged, err := MergedBranches(ctx, r)
	if err != nil {
		t.Fatalf("MergedBranches: %v", err)
	}
	if !merged["merged-branch"] {
		t.Errorf("merged-branch should be merged: %+v", merged)
	}
	if merged["unmerged-branch"] {
		t.Errorf("unmerged-branch should not be merged: %+v", merged)
	}
}

func TestRevParseAndTagRef(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "hello\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")

	sha, err := RevParse(ctx, r, "main")
	if err != nil || len(sha) != 40 {
		t.Fatalf("RevParse = %q, %v; want a 40-char sha", sha, err)
	}

	if err := TagRef(ctx, r, "archive/main", sha); err != nil {
		t.Fatalf("TagRef: %v", err)
	}
	tagSHA, err := RevParse(ctx, r, "archive/main")
	if err != nil || tagSHA != sha {
		t.Fatalf("archive/main = %q, %v; want %q", tagSHA, err, sha)
	}
}
