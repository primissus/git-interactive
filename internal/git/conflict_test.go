package git

import (
	"context"
	"testing"
)

func TestDetectInProgressNone(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "one\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial")

	state, err := DetectInProgress(ctx, r)
	if err != nil {
		t.Fatalf("DetectInProgress: %v", err)
	}
	if state != nil {
		t.Fatalf("DetectInProgress = %+v, want nil", state)
	}
}

func TestDetectInProgressMergeAndAbort(t *testing.T) {
	ctx := context.Background()
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

	_, _ = r.Run(ctx, "merge", "feature")

	state, err := DetectInProgress(ctx, r)
	if err != nil {
		t.Fatalf("DetectInProgress: %v", err)
	}
	if state == nil || state.Op != OpMerge {
		t.Fatalf("DetectInProgress = %+v, want merge in progress", state)
	}

	if err := state.Abort(ctx, r); err != nil {
		t.Fatalf("Abort: %v", err)
	}
	state, err = DetectInProgress(ctx, r)
	if err != nil {
		t.Fatalf("DetectInProgress after abort: %v", err)
	}
	if state != nil {
		t.Fatalf("DetectInProgress after abort = %+v, want nil", state)
	}
}
