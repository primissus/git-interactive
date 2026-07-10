package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// threeCommitRepo builds base→c1(adds a.txt)→c2(adds b.txt) on main and
// returns the runner plus the base sha (parent of c1), suitable for driving a
// rebase of c1 and c2.
func threeCommitRepo(t *testing.T) (*Runner, string) {
	t.Helper()
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "f.txt", "base\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "base")
	base, err := RevParse(ctx, r, "HEAD")
	if err != nil {
		t.Fatalf("RevParse base: %v", err)
	}
	writeFile(t, r, "a.txt", "a\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "c1")
	writeFile(t, r, "b.txt", "b\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "c2")
	return r, base
}

func TestRunRebasePlanSquash(t *testing.T) {
	ctx := context.Background()
	r, base := threeCommitRepo(t)

	commits, err := ListCommitsRange(ctx, r, base+"..HEAD")
	if err != nil {
		t.Fatalf("ListCommitsRange: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("commits = %d, want 2 (c1, c2)", len(commits))
	}

	// commits are newest first: [c2, c1]. Squash c2 into c1.
	steps := []RebaseStep{
		{Commit: commits[0], Op: RebaseSquash},
		{Commit: commits[1], Op: RebasePick},
	}
	if err := RunRebasePlan(ctx, r, RebasePlan{Base: base}, steps); err != nil {
		t.Fatalf("RunRebasePlan: %v", err)
	}

	all, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("commits after squash = %d, want 2 (base + squashed c1c2)", len(all))
	}
	if state, _ := DetectInProgress(ctx, r); state != nil {
		t.Fatalf("rebase still in progress after clean squash: %+v", state)
	}
}

func TestRunRebasePlanDrop(t *testing.T) {
	ctx := context.Background()
	r, base := threeCommitRepo(t)
	commits, _ := ListCommitsRange(ctx, r, base+"..HEAD")

	// Drop c1 (the older commit); keep c2.
	steps := []RebaseStep{
		{Commit: commits[0], Op: RebasePick},
		{Commit: commits[1], Op: RebaseDrop},
	}
	if err := RunRebasePlan(ctx, r, RebasePlan{Base: base}, steps); err != nil {
		t.Fatalf("RunRebasePlan(drop): %v", err)
	}
	all, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("commits after drop = %d, want 2 (base + c2)", len(all))
	}
	root, err := RepoRoot(ctx, r)
	if err != nil {
		t.Fatalf("RepoRoot: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt")); !os.IsNotExist(err) {
		t.Fatalf("a.txt should be gone after dropping c1 (stat err=%v)", err)
	}
}

func TestRunRebasePlanReword(t *testing.T) {
	ctx := context.Background()
	r, base := threeCommitRepo(t)
	commits, _ := ListCommitsRange(ctx, r, base+"..HEAD")

	steps := []RebaseStep{
		{Commit: commits[0], Op: RebaseReword, Message: "c2 reworded"},
		{Commit: commits[1], Op: RebasePick},
	}
	if err := RunRebasePlan(ctx, r, RebasePlan{Base: base}, steps); err != nil {
		t.Fatalf("RunRebasePlan(reword): %v", err)
	}
	all, err := ListCommits(ctx, r)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if all[0].Subject != "c2 reworded" {
		t.Fatalf("HEAD subject = %q, want %q", all[0].Subject, "c2 reworded")
	}
}

// setupConflictRebase diverges feature and main on the same file so rebasing
// feature onto main conflicts. It returns the runner and main's sha, and leaves
// feature checked out.
func setupConflictRebase(t *testing.T) (*Runner, string) {
	t.Helper()
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "f.txt", "base\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "base")
	mustGit(t, r, "branch", "feature")

	mustGit(t, r, "checkout", "feature")
	writeFile(t, r, "f.txt", "feature\n")
	mustGit(t, r, "commit", "-am", "feature change")

	mustGit(t, r, "checkout", "main")
	writeFile(t, r, "f.txt", "main\n")
	mustGit(t, r, "commit", "-am", "main change")
	mainSHA, err := RevParse(ctx, r, "main")
	if err != nil {
		t.Fatalf("RevParse main: %v", err)
	}

	mustGit(t, r, "checkout", "feature")
	return r, mainSHA
}

func TestRebaseConflictResolveContinue(t *testing.T) {
	ctx := context.Background()
	r, mainSHA := setupConflictRebase(t)

	commits, err := ListCommitsRange(ctx, r, mainSHA+"..HEAD")
	if err != nil {
		t.Fatalf("ListCommitsRange: %v", err)
	}
	steps := []RebaseStep{{Commit: commits[0], Op: RebasePick}}

	// The rebase stops on the conflict, so a non-nil error is expected here.
	if err := RunRebasePlan(ctx, r, RebasePlan{Base: mainSHA}, steps); err == nil {
		t.Fatalf("RunRebasePlan: expected a conflict stop, got nil error")
	}

	state, err := DetectInProgress(ctx, r)
	if err != nil {
		t.Fatalf("DetectInProgress: %v", err)
	}
	if state == nil || state.Op != OpRebase {
		t.Fatalf("DetectInProgress = %+v, want rebase in progress", state)
	}

	// Side labels must reflect the rebase inversion: git's "ours" is the branch
	// rebased onto (main), "theirs" is the replayed commit (feature).
	sides := ResolveSides(ctx, r, state)
	if sides.Ours != "main" {
		t.Fatalf("sides.Ours = %q, want %q (the onto branch)", sides.Ours, "main")
	}
	if sides.Theirs != "feature" {
		t.Fatalf("sides.Theirs = %q, want %q (the replayed branch)", sides.Theirs, "feature")
	}

	files, err := ConflictedFiles(ctx, r)
	if err != nil {
		t.Fatalf("ConflictedFiles: %v", err)
	}
	if len(files) != 1 || files[0] != "f.txt" {
		t.Fatalf("ConflictedFiles = %v, want [f.txt]", files)
	}

	// Take "theirs" — the feature side, thanks to the inversion.
	if err := TakeTheirs(ctx, r, "f.txt"); err != nil {
		t.Fatalf("TakeTheirs: %v", err)
	}
	if err := state.Continue(ctx, r); err != nil {
		t.Fatalf("Continue: %v", err)
	}
	if s, _ := DetectInProgress(ctx, r); s != nil {
		t.Fatalf("rebase still in progress after continue: %+v", s)
	}

	root, _ := RepoRoot(ctx, r)
	data, err := os.ReadFile(filepath.Join(root, "f.txt"))
	if err != nil {
		t.Fatalf("read f.txt: %v", err)
	}
	if strings.TrimSpace(string(data)) != "feature" {
		t.Fatalf("f.txt = %q, want %q (feature side taken)", string(data), "feature")
	}
}

func TestReadRebaseProgress(t *testing.T) {
	ctx := context.Background()
	r, mainSHA := setupConflictRebase(t)
	commits, _ := ListCommitsRange(ctx, r, mainSHA+"..HEAD")
	_ = RunRebasePlan(ctx, r, RebasePlan{Base: mainSHA}, []RebaseStep{{Commit: commits[0], Op: RebasePick}})

	p, err := ReadRebaseProgress(ctx, r)
	if err != nil {
		t.Fatalf("ReadRebaseProgress: %v", err)
	}
	if p.Branch != "feature" {
		t.Fatalf("progress.Branch = %q, want feature", p.Branch)
	}
	if p.Total < 1 || p.Current < 1 {
		t.Fatalf("progress = %+v, want current/total >= 1", p)
	}
}

func TestResolveSidesMerge(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "f.txt", "base\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "base")
	mustGit(t, r, "branch", "feature")
	mustGit(t, r, "checkout", "feature")
	writeFile(t, r, "f.txt", "feature\n")
	mustGit(t, r, "commit", "-am", "feature")
	mustGit(t, r, "checkout", "main")
	writeFile(t, r, "f.txt", "main\n")
	mustGit(t, r, "commit", "-am", "main")

	_, _ = r.Run(ctx, "merge", "feature") // conflicts

	state, err := DetectInProgress(ctx, r)
	if err != nil || state == nil || state.Op != OpMerge {
		t.Fatalf("DetectInProgress = %+v, %v; want merge", state, err)
	}
	sides := ResolveSides(ctx, r, state)
	if sides.Ours != "main" {
		t.Fatalf("merge sides.Ours = %q, want main (current branch)", sides.Ours)
	}
	if sides.Theirs != "feature" {
		t.Fatalf("merge sides.Theirs = %q, want feature (merged branch)", sides.Theirs)
	}
}

func TestMergeBothSides(t *testing.T) {
	// Standard (2-way) conflict.
	got := mergeBothSides("top\n<<<<<<< HEAD\nours\n=======\ntheirs\n>>>>>>> other\nbottom\n")
	want := "top\nours\ntheirs\nbottom\n"
	if got != want {
		t.Fatalf("mergeBothSides = %q, want %q", got, want)
	}

	// diff3 conflict: the base section between ||||||| and ======= is dropped.
	got = mergeBothSides("<<<<<<< HEAD\nours\n||||||| base\norig\n=======\ntheirs\n>>>>>>> other\n")
	want = "ours\ntheirs\n"
	if got != want {
		t.Fatalf("mergeBothSides(diff3) = %q, want %q", got, want)
	}
}

func TestValidateRebaseSteps(t *testing.T) {
	c := Commit{SHA: "x", Subject: "s"}
	// Oldest kept commit (last in display order) may not be squash/fixup.
	err := ValidateRebaseSteps([]RebaseStep{{Commit: c, Op: RebasePick}, {Commit: c, Op: RebaseSquash}})
	if err == nil {
		t.Fatal("ValidateRebaseSteps: expected error for oldest=squash")
	}
	// All-drop is rejected.
	if err := ValidateRebaseSteps([]RebaseStep{{Commit: c, Op: RebaseDrop}}); err == nil {
		t.Fatal("ValidateRebaseSteps: expected error for all-drop")
	}
	// A valid plan passes.
	if err := ValidateRebaseSteps([]RebaseStep{{Commit: c, Op: RebaseSquash}, {Commit: c, Op: RebasePick}}); err != nil {
		t.Fatalf("ValidateRebaseSteps(valid): %v", err)
	}
}
