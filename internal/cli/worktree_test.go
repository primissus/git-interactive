package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"git-interact/internal/gh"
	"git-interact/internal/git"
)

func TestShortSHA(t *testing.T) {
	if got := shortSHA("abcdef0123456789"); got != "abcdef0" {
		t.Errorf("shortSHA(long) = %q, want %q", got, "abcdef0")
	}
	if got := shortSHA("abc"); got != "abc" {
		t.Errorf("shortSHA(short) = %q, want %q", got, "abc")
	}
}

// TestWorktreeItemPRColumn verifies the 5th cell renders the PR number and
// state, empty when the row has no open PR.
func TestWorktreeItemPRColumn(t *testing.T) {
	none := worktreeItem{w: git.Worktree{Path: "/x", Branch: "main"}}
	if cols := none.Columns(); cols[4] != "" {
		t.Errorf("PR cell for no PR = %q, want empty", cols[4])
	}

	open := worktreeItem{w: git.Worktree{Path: "/x", Branch: "feature"}, pr: gh.PR{Number: 412}}
	if cols := open.Columns(); cols[4] != "#412 open" {
		t.Errorf("PR cell = %q, want %q", cols[4], "#412 open")
	}
}

// TestWorktreeItemSearchValueMatchesShortestPath is the regression test for
// the p15 search-audit finding: FilterValue carries the raw absolute path,
// but the column renders it through FormatWorktreePath, so under the default
// "shortest" format a query typed against the displayed "~"-abbreviated path
// matched nothing. SearchValue widens the fuzzy filter to also cover the
// formatted display form (and the PR number), while FilterValue stays the
// raw path + branch since it also labels this row in resilient bulk ops.
func TestWorktreeItemSearchValueMatchesShortestPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	wtPath := filepath.Join(home, "src", "wt", "foo")
	wi := worktreeItem{w: git.Worktree{Path: wtPath, Branch: "feature/x"}, cwd: filepath.Join(home, "elsewhere"), pr: gh.PR{Number: 9}}

	wantDisplay := filepath.Join("~", "src", "wt", "foo")
	sv := wi.SearchValue()
	for _, want := range []string{wantDisplay, "feature/x", "#9"} {
		if !strings.Contains(sv, want) {
			t.Errorf("SearchValue() = %q, want it to contain %q", sv, want)
		}
	}

	if wi.FilterValue() != wtPath+" feature/x" {
		t.Errorf("FilterValue() = %q, want unchanged raw path + branch", wi.FilterValue())
	}
}
