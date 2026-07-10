package git

import (
	"context"
	"testing"
)

func TestListCommitGraph(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "first")
	mustGit(t, r, "branch", "feature")
	writeFile(t, r, "a.txt", "two\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "second")

	rows, err := ListCommitGraph(ctx, r, true, false)
	if err != nil {
		t.Fatalf("ListCommitGraph: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d: %+v", len(rows), rows)
	}
	for _, row := range rows {
		if !row.HasCommit {
			t.Errorf("row %+v: want HasCommit", row)
		}
		if row.Prefix == "" {
			t.Errorf("row %+v: want a non-empty graph prefix", row)
		}
	}
	if rows[0].Commit.Subject != "second" {
		t.Errorf("rows[0].Commit.Subject = %q, want %q", rows[0].Commit.Subject, "second")
	}
}

func TestListCommitGraphSimplify(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "first")
	mustGit(t, r, "branch", "feature")
	writeFile(t, r, "a.txt", "two\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "second, undecorated")
	writeFile(t, r, "a.txt", "three\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "third")

	rows, err := ListCommitGraph(ctx, r, true, true)
	if err != nil {
		t.Fatalf("ListCommitGraph: %v", err)
	}
	for _, row := range rows {
		if row.HasCommit && row.Commit.Subject == "second, undecorated" {
			t.Errorf("simplify-by-decoration should have dropped the undecorated commit: %+v", rows)
		}
	}
}

func TestParseGraphConnectorLine(t *testing.T) {
	// A merge produces connector-only lines (no sentinel) alongside commit
	// lines; parseGraph must keep both without erroring.
	out := "* \x02abc\x1fabc1\x1fsubject\x1f1 day ago\x1fAuthor\x1f\x1f2026-01-01\n|\\  \n| * \x02def\x1fdef1\x1fother\x1f2 days ago\x1fAuthor\x1f\x1f2026-01-02"
	rows := parseGraph(out)
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d: %+v", len(rows), rows)
	}
	if !rows[0].HasCommit || rows[0].Commit.Subject != "subject" {
		t.Errorf("rows[0] = %+v, want a commit with subject %q", rows[0], "subject")
	}
	if rows[1].HasCommit {
		t.Errorf("rows[1] = %+v, want a connector-only row", rows[1])
	}
	if !rows[2].HasCommit || rows[2].Commit.Subject != "other" {
		t.Errorf("rows[2] = %+v, want a commit with subject %q", rows[2], "other")
	}
}
