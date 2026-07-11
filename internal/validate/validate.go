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
	return refName("branch", name)
}

// TagName reports whether name is a usable git tag name. Tags and branches
// share the same ref-format rules (git-check-ref-format(1)), so this applies
// the identical checks as BranchName, just worded for a tag.
func TagName(name string) error {
	return refName("tag", name)
}

// refName applies the shared ref-format rules for a ref of the given kind
// ("branch" or "tag"), used only to word the returned error.
func refName(kind, name string) error {
	if name == "" {
		return fmt.Errorf("%s name is empty", kind)
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("%s name cannot start with '-'", kind)
	}
	if name == "@" {
		return fmt.Errorf("%s name cannot be '@'", kind)
	}
	return refFormat(kind, name)
}

// refFormat applies the shared ref-name rules (git-check-ref-format(1)): no
// forbidden bytes, no `..`/`@{`/`//`, no leading/trailing slash, no trailing
// dot, and per-component rules (non-empty, not starting with '.', not ending
// with ".lock").
func refFormat(kind, ref string) error {
	for _, r := range ref {
		switch {
		case r < 0o40 || r == 0o177:
			return fmt.Errorf("%s name contains a control character", kind)
		case r == ' ':
			return fmt.Errorf("%s name cannot contain spaces", kind)
		case strings.ContainsRune("~^:?*[\\", r):
			return fmt.Errorf("%s name cannot contain %q", kind, string(r))
		}
	}
	switch {
	case strings.Contains(ref, ".."):
		return fmt.Errorf("%s name cannot contain '..'", kind)
	case strings.Contains(ref, "@{"):
		return fmt.Errorf("%s name cannot contain '@{'", kind)
	case strings.Contains(ref, "//"):
		return fmt.Errorf("%s name cannot contain '//'", kind)
	case strings.HasPrefix(ref, "/") || strings.HasSuffix(ref, "/"):
		return fmt.Errorf("%s name cannot start or end with '/'", kind)
	case strings.HasSuffix(ref, "."):
		return fmt.Errorf("%s name cannot end with '.'", kind)
	}
	for comp := range strings.SplitSeq(ref, "/") {
		switch {
		case comp == "":
			return fmt.Errorf("%s name has an empty path component", kind)
		case strings.HasPrefix(comp, "."):
			return fmt.Errorf("%s name component cannot start with '.'", kind)
		case strings.HasSuffix(comp, ".lock"):
			return fmt.Errorf("%s name component cannot end with '.lock'", kind)
		}
	}
	return nil
}
