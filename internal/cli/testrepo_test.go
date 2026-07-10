package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"git-interact/internal/git"
)

// newTestRepo creates an initialized git repo in a temp dir and returns a
// Runner scoped to it, mirroring the git package's own fixture helper so cli
// glue functions can be exercised end-to-end against real git.
func newTestRepo(t *testing.T) *git.Runner {
	t.Helper()
	dir := t.TempDir()
	r := git.NewRunner(dir)
	mustGit(t, r, "init", "-b", "main")
	mustGit(t, r, "config", "user.name", "Test User")
	mustGit(t, r, "config", "user.email", "test@example.com")
	mustGit(t, r, "config", "commit.gpgsign", "false")
	return r
}

func mustGit(t *testing.T, r *git.Runner, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// commitFile writes name with contents, stages it, and commits it with subject.
func commitFile(t *testing.T, r *git.Runner, name, contents, subject string) {
	t.Helper()
	writeRepoFile(t, r, name, contents)
	mustGit(t, r, "add", name)
	mustGit(t, r, "commit", "-m", subject)
}

func writeRepoFile(t *testing.T, r *git.Runner, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(r.Dir, name), []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// headSubject returns the subject line of HEAD.
func headSubject(t *testing.T, r *git.Runner) string {
	t.Helper()
	out, err := r.Run(context.Background(), "log", "-1", "--pretty=%s")
	if err != nil {
		t.Fatalf("head subject: %v", err)
	}
	return strings.TrimSpace(out)
}

// commitCount returns the number of commits reachable from HEAD.
func commitCount(t *testing.T, r *git.Runner) int {
	t.Helper()
	out, err := r.Run(context.Background(), "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("commit count: %v", err)
	}
	n := 0
	for _, c := range strings.TrimSpace(out) {
		n = n*10 + int(c-'0')
	}
	return n
}
