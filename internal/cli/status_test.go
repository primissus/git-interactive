package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

// withKeymapOverride points XDG_CONFIG_HOME at a temp keymap.json with
// contents, loads it, and resets to defaults on cleanup (tui.LoadKeymap's
// active keymap is process-global, so tests must not leak overrides).
func withKeymapOverride(t *testing.T, contents string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	gintDir := filepath.Join(dir, "gint")
	if err := os.MkdirAll(gintDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gintDir, "keymap.json"), []byte(contents), 0o644); err != nil {
		t.Fatalf("write keymap.json: %v", err)
	}
	if err := tui.LoadKeymap(); err != nil {
		t.Fatalf("LoadKeymap: %v", err)
	}
	t.Cleanup(func() {
		if err := os.WriteFile(filepath.Join(gintDir, "keymap.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("reset keymap.json: %v", err)
		}
		if err := tui.LoadKeymap(); err != nil {
			t.Fatalf("LoadKeymap (reset): %v", err)
		}
	})
}

func TestBuildStatusOperationsAppliesKeymapOverride(t *testing.T) {
	withKeymapOverride(t, `{"operations": {"status.toggle stage": {"key": "z"}}}`)

	ops := buildStatusOperations(context.Background(), newTestRepo(t))
	var toggle tui.Operation
	found := false
	for _, op := range ops {
		if op.Name == "toggle stage" {
			toggle, found = op, true
		}
	}
	if !found {
		t.Fatal("\"toggle stage\" operation not found")
	}
	if toggle.Key != "z" {
		t.Errorf("toggle stage Key = %q, want \"z\" (overridden)", toggle.Key)
	}
}

func mkEntry(code string) git.StatusEntry {
	return git.StatusEntry{Code: code, Path: "file.go"}
}

func TestStatusItemDisplayCode(t *testing.T) {
	cases := []struct {
		name string
		item statusItem
		want string
	}{
		{"staged side of a fully-staged file", statusItem{e: mkEntry("M."), grouped: true, staged: true}, "M"},
		{"unstaged side of a fully-unstaged file", statusItem{e: mkEntry(".M"), grouped: true}, "M"},
		{"staged side of a dual-status file", statusItem{e: mkEntry("MM"), grouped: true, staged: true}, "M"},
		{"unstaged side of a dual-status file", statusItem{e: mkEntry("MM"), grouped: true}, "M"},
		{"conflict keeps its raw code", statusItem{e: mkEntry("UU"), grouped: true, conflict: true}, "UU"},
		{"untracked keeps its raw code", statusItem{e: mkEntry("??"), grouped: true, untracked: true}, "??"},
		{"non-grouped (flat/plain) row always keeps the raw XY code", statusItem{e: mkEntry("MM"), staged: true}, "MM"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.item.displayCode(); got != c.want {
				t.Errorf("displayCode() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestLoadGroupedStatusItemsSectionsAndDualStatus(t *testing.T) {
	r := newTestRepo(t)
	commitFile(t, r, "staged.txt", "v1", "add staged.txt")
	commitFile(t, r, "dual.txt", "v1", "add dual.txt")

	// staged.txt: staged-only change.
	writeRepoFile(t, r, "staged.txt", "v2")
	mustGit(t, r, "add", "staged.txt")
	// dual.txt: staged, then further modified — appears in both buckets.
	writeRepoFile(t, r, "dual.txt", "v2")
	mustGit(t, r, "add", "dual.txt")
	writeRepoFile(t, r, "dual.txt", "v3")
	// untracked.txt: untracked.
	writeRepoFile(t, r, "untracked.txt", "v1")

	items, err := loadGroupedStatusItems(context.Background(), r)
	if err != nil {
		t.Fatalf("loadGroupedStatusItems: %v", err)
	}

	var headers []string
	rows := map[string][]statusItem{} // header label -> rows under it
	var section string
	for _, it := range items {
		if h, ok := it.(tui.HeaderItem); ok {
			section = h.Label
			headers = append(headers, section)
			continue
		}
		rows[section] = append(rows[section], it.(statusItem))
	}

	wantHeaders := []string{"Staged", "Unstaged", "Untracked"}
	if len(headers) != len(wantHeaders) {
		t.Fatalf("headers = %v, want %v", headers, wantHeaders)
	}
	for i, h := range wantHeaders {
		if headers[i] != h {
			t.Errorf("headers[%d] = %q, want %q", i, headers[i], h)
		}
	}
	if got := len(rows["Conflicts"]); got != 0 {
		t.Errorf("no Conflicts header expected, got %d rows under it", got)
	}
	if got, want := len(rows["Staged"]), 2; got != want {
		t.Errorf("Staged rows = %d, want %d", got, want)
	}
	if got, want := len(rows["Unstaged"]), 1; got != want {
		t.Errorf("Unstaged rows = %d, want %d", got, want)
	}
	if got, want := len(rows["Untracked"]), 1; got != want {
		t.Errorf("Untracked rows = %d, want %d", got, want)
	}

	foundDualStaged, foundDualUnstaged := false, false
	for _, s := range rows["Staged"] {
		if s.e.Path == "dual.txt" {
			foundDualStaged = true
			if !s.staged {
				t.Error("dual.txt row under Staged should have staged=true")
			}
		}
	}
	for _, s := range rows["Unstaged"] {
		if s.e.Path == "dual.txt" {
			foundDualUnstaged = true
			if s.staged {
				t.Error("dual.txt row under Unstaged should have staged=false")
			}
		}
	}
	if !foundDualStaged || !foundDualUnstaged {
		t.Errorf("dual.txt should appear once under Staged and once under Unstaged, got staged=%v unstaged=%v", foundDualStaged, foundDualUnstaged)
	}
}
