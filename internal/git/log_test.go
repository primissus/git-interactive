package git

import (
	"context"
	"testing"
)

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
