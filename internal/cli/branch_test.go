package cli

import (
	"testing"
	"time"

	"git-interact/internal/git"
)

func TestFilterBranchesAuthor(t *testing.T) {
	branches := []git.Branch{
		{Name: "a", AuthorName: "Alice"},
		{Name: "b", AuthorName: "Bob"},
	}
	out := filterBranches(branches, branchFilters{author: "ali"}, nil)
	if len(out) != 1 || out[0].Name != "a" {
		t.Fatalf("filterBranches(author) = %+v, want just 'a'", out)
	}
}

func TestFilterBranchesSince(t *testing.T) {
	now := time.Now()
	branches := []git.Branch{
		{Name: "recent", CommitUnix: now.Unix()},
		{Name: "old", CommitUnix: now.AddDate(0, 0, -10).Unix()},
	}
	out := filterBranches(branches, branchFilters{since: "3d"}, nil)
	if len(out) != 1 || out[0].Name != "recent" {
		t.Fatalf("filterBranches(since=3d) = %+v, want just 'recent'", out)
	}
}

func TestFilterBranchesMergedNotMerged(t *testing.T) {
	branches := []git.Branch{{Name: "a"}, {Name: "b"}}
	merged := map[string]bool{"a": true}

	out := filterBranches(branches, branchFilters{merged: true}, merged)
	if len(out) != 1 || out[0].Name != "a" {
		t.Fatalf("filterBranches(merged) = %+v, want just 'a'", out)
	}

	out = filterBranches(branches, branchFilters{notMerged: true}, merged)
	if len(out) != 1 || out[0].Name != "b" {
		t.Fatalf("filterBranches(notMerged) = %+v, want just 'b'", out)
	}
}

func TestFilterBranchesGone(t *testing.T) {
	branches := []git.Branch{
		{Name: "a", UpstreamTrack: "[gone]"},
		{Name: "b", UpstreamTrack: ""},
	}
	out := filterBranches(branches, branchFilters{gone: true}, nil)
	if len(out) != 1 || out[0].Name != "a" {
		t.Fatalf("filterBranches(gone) = %+v, want just 'a'", out)
	}
}

func TestBranchFiltersValidate(t *testing.T) {
	if err := (branchFilters{merged: true, notMerged: true}).validate(); err == nil {
		t.Errorf("merged+notMerged should be mutually exclusive")
	}
	if err := (branchFilters{since: "bogus"}).validate(); err == nil {
		t.Errorf("unknown since bucket should error")
	}
	if err := (branchFilters{since: "1w"}).validate(); err != nil {
		t.Errorf("valid since bucket should not error: %v", err)
	}
}

func TestSortBranches(t *testing.T) {
	branches := []git.Branch{
		{Name: "b", AuthorName: "Bob", CommitUnix: 1, AuthorUnix: 2},
		{Name: "a", AuthorName: "Alice", CommitUnix: 2, AuthorUnix: 1},
	}

	byAuthor := append([]git.Branch(nil), branches...)
	sortBranches(byAuthor, "author")
	if byAuthor[0].Name != "a" {
		t.Errorf("sort by author: got %+v, want 'a' first", byAuthor)
	}

	byCreated := append([]git.Branch(nil), branches...)
	sortBranches(byCreated, "created")
	if byCreated[0].Name != "b" {
		t.Errorf("sort by created: got %+v, want 'b' first (higher AuthorUnix)", byCreated)
	}

	byLastCommit := append([]git.Branch(nil), branches...)
	sortBranches(byLastCommit, "last-commit")
	if byLastCommit[0].Name != "a" {
		t.Errorf("sort by last-commit: got %+v, want 'a' first (higher CommitUnix)", byLastCommit)
	}
}

func TestSortModeFromFlag(t *testing.T) {
	cases := map[string]string{
		"created": "created", "author": "author", "off": "off",
		"": "last-commit", "bogus": "last-commit",
	}
	for in, want := range cases {
		if got := sortModeFromFlag(in); got != want {
			t.Errorf("sortModeFromFlag(%q) = %q, want %q", in, got, want)
		}
	}
}
