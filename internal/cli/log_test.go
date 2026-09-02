package cli

import (
	"testing"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

func TestShortAuthor(t *testing.T) {
	cases := map[string]string{
		"Test User":        "Test U.",
		"Ada Lovelace":     "Ada L.",
		"Cher":             "Cher",
		"Mary Jane Watson": "Mary Jane W.",
	}
	for in, want := range cases {
		if got := tui.ShortAuthor(in); got != want {
			t.Errorf("ShortAuthor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFirstRef(t *testing.T) {
	if got, ok := firstRef(nil); ok || got != "" {
		t.Errorf("firstRef(nil) = %q, %v; want \"\", false", got, ok)
	}
	if got, ok := firstRef([]string{"main", "other"}); !ok || got != "main" {
		t.Errorf("firstRef([main, other]) = %q, %v; want main, true", got, ok)
	}
}

func TestPlural2(t *testing.T) {
	if got := plural2(1); got != "1 commit" {
		t.Errorf("plural2(1) = %q, want %q", got, "1 commit")
	}
	if got := plural2(3); got != "3 commits" {
		t.Errorf("plural2(3) = %q, want %q", got, "3 commits")
	}
}

// TestBranchesCell verifies the branches cell joins the refs with ", " under
// the default (full) branch format — the short/ultra-short variants are covered
// in the tui package's FormatBranch tests.
func TestBranchesCell(t *testing.T) {
	if got := branchesCell(nil); got != "" {
		t.Errorf("branchesCell(nil) = %q, want empty", got)
	}
	if got := branchesCell([]string{"main"}); got != "main" {
		t.Errorf("branchesCell([main]) = %q, want %q", got, "main")
	}
	if got := branchesCell([]string{"feat/a", "feat/b"}); got != "feat/a, feat/b" {
		t.Errorf("branchesCell = %q, want %q", got, "feat/a, feat/b")
	}
}

// TestLogItemWorktreeColumn verifies the log row's worktree cell formats its
// absolute path per the active worktree-path format (shortest by default).
func TestLogItemWorktreeColumn(t *testing.T) {
	item := logItem{
		c:      git.Commit{ShortSHA: "abc1234", Subject: "subj", RelDate: "yesterday", AuthorName: "Test User", Refs: []string{"main"}},
		full:   false,
		wtDir:  "/home/u/proj",
		cwd:    "/home/u",
		isHead: true,
	}
	cols := item.Columns()
	if len(cols) != 6 {
		t.Fatalf("logItem.Columns() = %v, want 6 cells", cols)
	}
	if cols[4] != "main" {
		t.Errorf("branches cell = %q, want %q", cols[4], "main")
	}
	if cols[5] != "proj" {
		t.Errorf("worktree cell = %q, want %q (shortest)", cols[5], "proj")
	}
}
