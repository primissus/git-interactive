package git

import (
	"context"
	"testing"
)

func TestListStashes(t *testing.T) {
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")

	writeFile(t, r, "a.txt", "two\n")
	mustGit(t, r, "stash", "push", "-m", "wip changes")

	stashes, err := ListStashes(context.Background(), r)
	if err != nil {
		t.Fatalf("ListStashes: %v", err)
	}
	if len(stashes) != 1 {
		t.Fatalf("want 1 stash, got %d: %+v", len(stashes), stashes)
	}
	s := stashes[0]
	if s.Ref != "stash@{0}" {
		t.Errorf("Ref = %q, want stash@{0}", s.Ref)
	}
	if s.Branch != "main" {
		t.Errorf("Branch = %q, want main", s.Branch)
	}
	if s.Message != "wip changes" {
		t.Errorf("Message = %q, want %q", s.Message, "wip changes")
	}
}

func TestStashApplyPopDrop(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial")
	writeFile(t, r, "a.txt", "two\n")
	if err := StashPush(ctx, r, "wip"); err != nil {
		t.Fatalf("StashPush: %v", err)
	}

	if err := StashApply(ctx, r, "stash@{0}"); err != nil {
		t.Fatalf("StashApply: %v", err)
	}
	st, err := GetStatus(ctx, r)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(st.Unstaged) != 1 {
		t.Fatalf("Unstaged after apply = %+v, want 1 entry", st.Unstaged)
	}
	// apply doesn't remove the stash.
	stashes, err := ListStashes(ctx, r)
	if err != nil {
		t.Fatalf("ListStashes: %v", err)
	}
	if len(stashes) != 1 {
		t.Fatalf("stashes after apply = %+v, want still 1", stashes)
	}

	if err := StashDrop(ctx, r, "stash@{0}"); err != nil {
		t.Fatalf("StashDrop: %v", err)
	}
	stashes, err = ListStashes(ctx, r)
	if err != nil {
		t.Fatalf("ListStashes: %v", err)
	}
	if len(stashes) != 0 {
		t.Fatalf("stashes after drop = %+v, want none", stashes)
	}
}

func TestStashPop(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial")
	writeFile(t, r, "a.txt", "two\n")
	if err := StashPush(ctx, r, "wip"); err != nil {
		t.Fatalf("StashPush: %v", err)
	}

	if err := StashPop(ctx, r, "stash@{0}"); err != nil {
		t.Fatalf("StashPop: %v", err)
	}
	stashes, err := ListStashes(ctx, r)
	if err != nil {
		t.Fatalf("ListStashes: %v", err)
	}
	if len(stashes) != 0 {
		t.Fatalf("stashes after pop = %+v, want none", stashes)
	}
}

func TestStashClear(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial")
	writeFile(t, r, "a.txt", "two\n")
	if err := StashPush(ctx, r, "wip"); err != nil {
		t.Fatalf("StashPush: %v", err)
	}

	if err := StashClear(ctx, r); err != nil {
		t.Fatalf("StashClear: %v", err)
	}
	stashes, err := ListStashes(ctx, r)
	if err != nil {
		t.Fatalf("ListStashes: %v", err)
	}
	if len(stashes) != 0 {
		t.Fatalf("stashes after clear = %+v, want none", stashes)
	}
}

func TestStashPushSingleFile(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	writeFile(t, r, "b.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial")
	writeFile(t, r, "a.txt", "changed\n")
	writeFile(t, r, "b.txt", "changed\n")

	if err := StashPush(ctx, r, "", "a.txt"); err != nil {
		t.Fatalf("StashPush(single file): %v", err)
	}
	st, err := GetStatus(ctx, r)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(st.Unstaged) != 1 || st.Unstaged[0].Path != "b.txt" {
		t.Fatalf("Unstaged after single-file stash = %+v, want just b.txt", st.Unstaged)
	}
}

func TestStashDiff(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial")
	writeFile(t, r, "a.txt", "two\n")
	if err := StashPush(ctx, r, "wip"); err != nil {
		t.Fatalf("StashPush: %v", err)
	}

	diff, err := StashDiff(ctx, r, "stash@{0}")
	if err != nil {
		t.Fatalf("StashDiff: %v", err)
	}
	if diff == "" {
		t.Errorf("StashDiff is empty, want a diff")
	}
}
