package git

import (
	"os"
	"os/exec"
	"testing"
)

// newTestRepo creates an initialized git repo in a temp dir and returns a
// Runner scoped to it. Commits are made with a fixed author/committer so
// test assertions stay deterministic.
func newTestRepo(t *testing.T) *Runner {
	t.Helper()
	dir := t.TempDir()

	r := &Runner{Dir: dir}
	mustGit(t, r, "init", "-b", "main")
	mustGit(t, r, "config", "user.name", "Test User")
	mustGit(t, r, "config", "user.email", "test@example.com")
	mustGit(t, r, "config", "commit.gpgsign", "false")

	return r
}

func mustGit(t *testing.T, r *Runner, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// writeFile creates or overwrites a file relative to the repo root.
func writeFile(t *testing.T, r *Runner, name, contents string) {
	t.Helper()
	if err := os.WriteFile(r.Dir+"/"+name, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
