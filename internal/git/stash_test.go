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
