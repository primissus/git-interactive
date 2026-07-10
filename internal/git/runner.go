// Package git shells out to the git binary and parses its porcelain output
// into typed values for the rest of gint to consume.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Error wraps a failed git invocation, carrying its stderr for display.
type Error struct {
	Args   []string
	Stderr string
	Err    error
}

func (e *Error) Error() string {
	stderr := strings.TrimSpace(e.Stderr)
	if stderr != "" {
		return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), stderr)
	}
	return fmt.Sprintf("git %s: %v", strings.Join(e.Args, " "), e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

// Runner executes git commands in a fixed working directory.
type Runner struct {
	Dir string
}

// NewRunner returns a Runner that runs git commands in dir ("" for the
// current process's working directory).
func NewRunner(dir string) *Runner {
	return &Runner{Dir: dir}
}

// Run executes `git <args...>` and returns stdout, or an *Error carrying
// stderr on failure.
func (r *Runner) Run(ctx context.Context, args ...string) (string, error) {
	return r.RunEnv(ctx, nil, args...)
}

// RunEnv is Run with extra environment variables (each "KEY=value") appended
// to the process environment. It is how the sequencer-driven commands
// (interactive rebase) inject GIT_SEQUENCE_EDITOR/GIT_EDITOR so git never
// blocks on a real editor.
func (r *Runner) RunEnv(ctx context.Context, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", &Error{Args: args, Stderr: stderr.String(), Err: err}
	}

	return stdout.String(), nil
}
