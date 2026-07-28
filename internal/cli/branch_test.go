package cli

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"git-interact/internal/git"
	"git-interact/internal/tui"
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

// runBranchOp finds an operation by name in the branch view's registry and
// drives it with the given items, returning its cmd synchronously.
func runBranchOp(t *testing.T, ctx context.Context, r *git.Runner, opName string, f branchFilters, state *branchViewState, items []tui.Item) (tea.Cmd, error) {
	t.Helper()
	ops := buildBranchOperations(ctx, r, f, state, true)
	for _, op := range ops {
		if op.Name != opName {
			continue
		}
		return op.Run(tui.OpContext{Items: items}), nil
	}
	return nil, fmt.Errorf("operation %q not found", opName)
}

func TestBranchRebaseOpSetsState(t *testing.T) {
	r := newTestRepo(t)
	commitFile(t, r, "a", "a", "first commit")
	mustGit(t, r, "checkout", "-b", "feature")
	commitFile(t, r, "b", "b", "second commit")
	mustGit(t, r, "checkout", "main")

	ctx := context.Background()
	state := &branchViewState{}

	items, err := loadBranchItems(ctx, r, branchFilters{}, "last-commit", true)
	if err != nil {
		t.Fatalf("loadBranchItems: %v", err)
	}

	// Find the feature branch item (index 1: row 0 is create row, 1 is main,
	// 2 is feature).
	var featureItem tui.Item
	for _, it := range items {
		if b, ok := it.(branchItem); ok && b.b.Name == "feature" {
			featureItem = it
			break
		}
	}
	if featureItem == nil {
		t.Fatal("feature branch not found in items")
	}

	cmd, err := runBranchOp(t, ctx, r, "rebase onto this branch", branchFilters{}, state, []tui.Item{featureItem})
	if err != nil {
		t.Fatalf("runBranchOp: %v", err)
	}
	if state.rebaseBase != "feature" {
		t.Fatalf("state.rebaseBase = %q, want %q", state.rebaseBase, "feature")
	}
	if cmd == nil {
		t.Fatal("command is nil, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("command msg is not tea.QuitMsg")
	}
}

func TestBranchRebaseOpGuardCurrentBranch(t *testing.T) {
	r := newTestRepo(t)
	commitFile(t, r, "a", "a", "first commit")
	mustGit(t, r, "checkout", "-b", "feature")
	commitFile(t, r, "b", "b", "second commit")
	mustGit(t, r, "checkout", "main")

	ctx := context.Background()
	state := &branchViewState{}

	items, err := loadBranchItems(ctx, r, branchFilters{}, "last-commit", true)
	if err != nil {
		t.Fatalf("loadBranchItems: %v", err)
	}

	var mainItem tui.Item
	for _, it := range items {
		if b, ok := it.(branchItem); ok && b.b.Name == "main" {
			mainItem = it
			break
		}
	}
	if mainItem == nil {
		t.Fatal("main branch not found in items")
	}

	cmd, err := runBranchOp(t, ctx, r, "rebase onto this branch", branchFilters{}, state, []tui.Item{mainItem})
	if err != nil {
		t.Fatalf("runBranchOp: %v", err)
	}
	if state.rebaseBase != "" {
		t.Fatalf("state.rebaseBase = %q, want empty (should not be set)", state.rebaseBase)
	}
	if cmd == nil {
		t.Fatal("command is nil, want status msg")
	}
	if _, ok := cmd().(tea.QuitMsg); ok {
		t.Fatal("guard op returned tea.QuitMsg, should return status msg")
	}
}
