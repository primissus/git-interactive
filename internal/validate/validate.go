// Package validate holds gint's input validators. They are plain functions
// (no struct tags, no reflection) applied to TUI text inputs before an
// operation runs — see .context/decisions.md for why go-playground/validator
// was rejected in favor of this.
package validate

import (
	"fmt"
	"strings"
)

// BranchName reports whether name is a usable git branch name, applying the
// subset of ref-format rules that `git check-ref-format --branch` enforces. It
// runs in-process (no `git` shell-out) so it works without a repository and
// returns a precise message naming the first rule the input violates. name is
// expected already trimmed of surrounding whitespace.
func BranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name is empty")
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("branch name cannot start with '-'")
	}
	if name == "@" {
		return fmt.Errorf("branch name cannot be '@'")
	}
	return refFormat(name)
}

// refFormat applies the shared ref-name rules (git-check-ref-format(1)): no
// forbidden bytes, no `..`/`@{`/`//`, no leading/trailing slash, no trailing
// dot, and per-component rules (non-empty, not starting with '.', not ending
// with ".lock").
func refFormat(ref string) error {
	for _, r := range ref {
		switch {
		case r < 0o40 || r == 0o177:
			return fmt.Errorf("branch name contains a control character")
		case r == ' ':
			return fmt.Errorf("branch name cannot contain spaces")
		case strings.ContainsRune("~^:?*[\\", r):
			return fmt.Errorf("branch name cannot contain %q", string(r))
		}
	}
	switch {
	case strings.Contains(ref, ".."):
		return fmt.Errorf("branch name cannot contain '..'")
	case strings.Contains(ref, "@{"):
		return fmt.Errorf("branch name cannot contain '@{'")
	case strings.Contains(ref, "//"):
		return fmt.Errorf("branch name cannot contain '//'")
	case strings.HasPrefix(ref, "/") || strings.HasSuffix(ref, "/"):
		return fmt.Errorf("branch name cannot start or end with '/'")
	case strings.HasSuffix(ref, "."):
		return fmt.Errorf("branch name cannot end with '.'")
	}
	for comp := range strings.SplitSeq(ref, "/") {
		switch {
		case comp == "":
			return fmt.Errorf("branch name has an empty path component")
		case strings.HasPrefix(comp, "."):
			return fmt.Errorf("branch name component cannot start with '.'")
		case strings.HasSuffix(comp, ".lock"):
			return fmt.Errorf("branch name component cannot end with '.lock'")
		}
	}
	return nil
}
