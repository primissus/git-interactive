package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"git-interact/internal/git"
)

// TestRunCommitChoice drives commit's core flow — the confirmation-choice →
// git-operation mapping shared by `gint commit` and status's inline commit —
// end to end against a real repo.
func TestRunCommitChoice(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	commitFile(t, r, "seed.txt", "seed", "seed commit")

	// "no" (empty choice) is a no-op: it must not create a commit.
	writeRepoFile(t, r, "a.txt", "one")
	mustGit(t, r, "add", "a.txt")
	before := commitCount(t, r)
	if status, err := runCommitChoice(ctx, r, "should not happen", ""); err != nil || status != "" {
		t.Fatalf(`runCommitChoice "no": status=%q err=%v, want ""/nil`, status, err)
	}
	if got := commitCount(t, r); got != before {
		t.Fatalf(`"no" created a commit: count %d -> %d`, before, got)
	}

	// "yes" commits the staged change.
	if _, err := runCommitChoice(ctx, r, "add a", "yes"); err != nil {
		t.Fatalf(`runCommitChoice "yes": %v`, err)
	}
	if got := headSubject(t, r); got != "add a" {
		t.Fatalf(`after "yes": HEAD subject %q, want "add a"`, got)
	}
	if got := commitCount(t, r); got != before+1 {
		t.Fatalf(`"yes" commit count: %d, want %d`, got, before+1)
	}

	// "amend" rewrites HEAD's message without adding a commit.
	countBeforeAmend := commitCount(t, r)
	if _, err := runCommitChoice(ctx, r, "add a (reworded)", "amend"); err != nil {
		t.Fatalf(`runCommitChoice "amend": %v`, err)
	}
	if got := headSubject(t, r); got != "add a (reworded)" {
		t.Fatalf(`after "amend": HEAD subject %q, want reworded`, got)
	}
	if got := commitCount(t, r); got != countBeforeAmend {
		t.Fatalf(`"amend" changed commit count: %d, want %d`, got, countBeforeAmend)
	}

	// "no-verify" commits the next staged change (hooks skipped).
	writeRepoFile(t, r, "b.txt", "two")
	mustGit(t, r, "add", "b.txt")
	countBeforeNV := commitCount(t, r)
	if _, err := runCommitChoice(ctx, r, "add b", "no-verify"); err != nil {
		t.Fatalf(`runCommitChoice "no-verify": %v`, err)
	}
	if got := commitCount(t, r); got != countBeforeNV+1 {
		t.Fatalf(`"no-verify" commit count: %d, want %d`, got, countBeforeNV+1)
	}
}

// TestRunMergeChoice drives merge's core flow — the choice → MergeMode mapping —
// end to end, covering both a fast-forward and a squash merge.
func TestRunMergeChoice(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	commitFile(t, r, "base.txt", "base", "base")

	// A feature branch one commit ahead of main.
	mustGit(t, r, "checkout", "-b", "feature")
	commitFile(t, r, "feat.txt", "feat", "feature work")
	mustGit(t, r, "checkout", "main")

	// "no" is a no-op.
	if status, err := runMergeChoice(ctx, r, "feature", ""); err != nil || status != "" {
		t.Fatalf(`runMergeChoice "no": status=%q err=%v, want ""/nil`, status, err)
	}
	if fileExists(r, "feat.txt") {
		t.Fatalf(`"no" merged feature into main`)
	}

	// "ff-only" fast-forwards main to feature.
	if _, err := runMergeChoice(ctx, r, "feature", "ff-only"); err != nil {
		t.Fatalf(`runMergeChoice "ff-only": %v`, err)
	}
	if got := headSubject(t, r); got != "feature work" {
		t.Fatalf(`after ff-only: HEAD subject %q, want "feature work"`, got)
	}

	// Set up a second divergent branch to squash-merge.
	mustGit(t, r, "checkout", "-b", "topic")
	commitFile(t, r, "topic1.txt", "1", "topic one")
	commitFile(t, r, "topic2.txt", "2", "topic two")
	mustGit(t, r, "checkout", "main")
	commitFile(t, r, "main-side.txt", "m", "main side")

	countBefore := commitCount(t, r)
	if _, err := runMergeChoice(ctx, r, "topic", "squash"); err != nil {
		t.Fatalf(`runMergeChoice "squash": %v`, err)
	}
	// gint's squash stages the combined change and commits it as a single
	// squash-merge commit (git merge --squash alone never commits).
	if !fileExists(r, "topic1.txt") {
		t.Fatalf(`"squash" did not bring in topic files`)
	}
	if got := commitCount(t, r); got != countBefore+1 {
		t.Fatalf(`"squash" commit count: %d, want %d`, got, countBefore+1)
	}
	if got := headSubject(t, r); got != "Merge branch 'topic' (squash)" {
		t.Fatalf(`"squash" HEAD subject %q`, got)
	}
}

func fileExists(r *git.Runner, name string) bool {
	_, err := os.Stat(filepath.Join(r.Dir, name))
	return err == nil
}
