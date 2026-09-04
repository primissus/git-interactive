package cli

import (
	"strings"
	"testing"
)

func TestShellInitScript(t *testing.T) {
	for _, sh := range []string{"bash", "zsh", "fish"} {
		script, err := shellInitScript(sh)
		if err != nil {
			t.Fatalf("shellInitScript(%q) error: %v", sh, err)
		}
		// The wrapper must call the real binary and hand it a --cd-file so the
		// checkout path survives back to the parent shell.
		for _, want := range []string{"command gint", "--cd-file"} {
			if !strings.Contains(script, want) {
				t.Errorf("shellInitScript(%q) missing %q", sh, want)
			}
		}
	}
}

// TestShellInitMatchesBranch verifies both wrappers' case arms cover
// branch/br alongside worktree/wt, so checkout's "cd there" prompt (phase 15)
// actually moves the shell once the wrapper is (re-)installed.
func TestShellInitMatchesBranch(t *testing.T) {
	for _, sh := range []string{"bash", "zsh", "fish"} {
		script, err := shellInitScript(sh)
		if err != nil {
			t.Fatalf("shellInitScript(%q) error: %v", sh, err)
		}
		if !strings.Contains(script, "branch") || !strings.Contains(script, "br") {
			t.Errorf("shellInitScript(%q) case arm missing branch/br:\n%s", sh, script)
		}
	}
}

func TestShellInitUnsupported(t *testing.T) {
	if _, err := shellInitScript("tcsh"); err == nil {
		t.Fatal("shellInitScript(\"tcsh\") = nil error, want unsupported-shell error")
	}
}
