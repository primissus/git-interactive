package git

import (
	"context"
	"testing"
)

func TestGetStatus(t *testing.T) {
	r := newTestRepo(t)
	writeFile(t, r, "tracked.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")

	// staged
	writeFile(t, r, "staged.txt", "staged\n")
	mustGit(t, r, "add", "staged.txt")

	// unstaged modification
	writeFile(t, r, "tracked.txt", "modified\n")

	// untracked
	writeFile(t, r, "untracked.txt", "new\n")

	st, err := GetStatus(context.Background(), r)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if st.Branch != "main" {
		t.Errorf("Branch = %q, want main", st.Branch)
	}
	if len(st.Staged) != 1 || st.Staged[0].Path != "staged.txt" {
		t.Errorf("Staged = %+v, want [staged.txt]", st.Staged)
	}
	if len(st.Unstaged) != 1 || st.Unstaged[0].Path != "tracked.txt" {
		t.Errorf("Unstaged = %+v, want [tracked.txt]", st.Unstaged)
	}
	if len(st.Untracked) != 1 || st.Untracked[0].Path != "untracked.txt" {
		t.Errorf("Untracked = %+v, want [untracked.txt]", st.Untracked)
	}
}

func TestGetStatusConflict(t *testing.T) {
	r := newTestRepo(t)
	writeFile(t, r, "f.txt", "base\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "base")

	mustGit(t, r, "checkout", "-b", "feature")
	writeFile(t, r, "f.txt", "feature change\n")
	mustGit(t, r, "commit", "-am", "feature change")

	mustGit(t, r, "checkout", "main")
	writeFile(t, r, "f.txt", "main change\n")
	mustGit(t, r, "commit", "-am", "main change")

	cmd := &Runner{Dir: r.Dir}
	_, _ = cmd.Run(context.Background(), "merge", "feature")

	st, err := GetStatus(context.Background(), r)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(st.Conflicts) != 1 || st.Conflicts[0].Path != "f.txt" {
		t.Errorf("Conflicts = %+v, want [f.txt]", st.Conflicts)
	}
}
