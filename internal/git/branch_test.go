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
