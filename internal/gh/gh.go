// Package gh shells out to the GitHub CLI (`gh`) to fetch pull-request data.
// It mirrors internal/git's shape (a Runner with Dir, free functions taking
// (ctx, r, ...), a private parser) but does not extend git.Runner, whose exec
// target is hardcoded to "git".
package gh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Runner executes gh commands in a fixed working directory.
type Runner struct {
	Dir string
}

// NewRunner returns a Runner that runs gh commands in dir ("" for the current
// process's working directory).
func NewRunner(dir string) *Runner {
	return &Runner{Dir: dir}
}

// Available reports whether the gh binary is on PATH, the same optional-binary
// idiom as cli.defaultPager's bat/batcat lookup.
func Available() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// PR is one open pull request as reported by `gh pr list`.
type PR struct {
	Number      int    `json:"number"`
	State       string `json:"state"`
	IsDraft     bool   `json:"isDraft"`
	Title       string `json:"title"`
	HeadRefName string `json:"headRefName"`
	URL         string `json:"url"`
}

// run executes `gh <args...>` and returns stdout, or an error carrying stderr
// on failure.
func run(ctx context.Context, r *Runner, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	if r != nil && r.Dir != "" {
		cmd.Dir = r.Dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		stderrText := stderr.String()
		if stderrText != "" {
			return "", fmt.Errorf("gh %v: %s", args, stderrText)
		}
		return "", fmt.Errorf("gh %v: %w", args, err)
	}
	return stdout.String(), nil
}

// ListPRs returns every open pull request in the repository at r.Dir. The six
// field names are verified against gh 2.100.0, which rejects unknown fields.
func ListPRs(ctx context.Context, r *Runner) ([]PR, error) {
	out, err := run(ctx, r, "pr", "list", "--state", "open", "--limit", "200",
		"--json", "number,state,isDraft,title,headRefName,url")
	if err != nil {
		return nil, err
	}
	return parsePRs([]byte(out))
}

// parsePRs is the unit-testable seam: it never invokes gh.
func parsePRs(data []byte) ([]PR, error) {
	var prs []PR
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, fmt.Errorf("gh: parsing pr list: %w", err)
	}
	return prs, nil
}

// PRsByBranch returns the open PRs in r.Dir keyed by their head branch name.
// It returns nil on any error (no gh on PATH, not a GitHub remote, not
// authenticated, or any other non-zero exit), so callers need no error
// branch — a nil map behaves like an empty one for lookups.
func PRsByBranch(ctx context.Context, r *Runner) map[string]PR {
	prs, err := ListPRs(ctx, r)
	if err != nil {
		return nil
	}
	out := make(map[string]PR, len(prs))
	for _, pr := range prs {
		out[pr.HeadRefName] = pr
	}
	return out
}

// OpenPR opens pull request number in the user's browser.
func OpenPR(ctx context.Context, r *Runner, number int) error {
	cmd := exec.CommandContext(ctx, "gh", "pr", "view", fmt.Sprintf("%d", number), "--web")
	if r != nil && r.Dir != "" {
		cmd.Dir = r.Dir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
