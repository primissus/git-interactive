package tui

import (
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// Format settings: frozen by ApplySettings and read live on every render.
var (
	activeDateFormat   = "short" // "short" | "long" | "iso"
	activeBranchFormat = "full"  // "full" | "short" | "ultra-short"
	activeAuthorFormat = "short" // "short" | "initials" | "full"

	// activeWorktreePathFormat selects how worktree paths render in the branch
	// and log views' worktree columns: "shortest" | "relative" | "absolute".
	activeWorktreePathFormat = "shortest"

	// activeBranchHidden / activeLogHidden hold the per-view column titles the
	// user has hidden via the view's settings overlay. The interactive List
	// filters its columns against these; -I output never does.
	activeBranchHidden = map[string]bool{}
	activeLogHidden    = map[string]bool{}
)

// nowFunc is time.Now by default; stubbed in tests.
var nowFunc = time.Now

// freezeFormats copies the format fields from Settings into the active
// package-level vars. Called by ApplySettings.
func freezeFormats(s *Settings) {
	if s == nil {
		activeDateFormat = "short"
		activeBranchFormat = "full"
		activeAuthorFormat = "short"
		activeWorktreePathFormat = "shortest"
		activeBranchHidden = map[string]bool{}
		activeLogHidden = map[string]bool{}
		return
	}
	activeDateFormat = s.DateFormat
	if activeDateFormat == "" {
		activeDateFormat = "short"
	}
	activeBranchFormat = s.BranchFormat
	if activeBranchFormat == "" {
		activeBranchFormat = "full"
	}
	activeAuthorFormat = s.AuthorFormat
	if activeAuthorFormat == "" {
		activeAuthorFormat = "short"
	}
	activeWorktreePathFormat = s.WorktreePathFormat
	if activeWorktreePathFormat == "" {
		activeWorktreePathFormat = "shortest"
	}
	activeBranchHidden = toHiddenSet(s.BranchHiddenColumns)
	activeLogHidden = toHiddenSet(s.LogHiddenColumns)
}

// toHiddenSet converts a persisted hidden-column title list into the lookup
// map the renderer checks on every frame.
func toHiddenSet(list []string) map[string]bool {
	set := make(map[string]bool, len(list))
	for _, title := range list {
		if title != "" {
			set[title] = true
		}
	}
	return set
}

// ActiveBranchHidden returns the branch view's hidden-column set.
func ActiveBranchHidden() map[string]bool { return activeBranchHidden }

// ActiveLogHidden returns the log view's hidden-column set.
func ActiveLogHidden() map[string]bool { return activeLogHidden }

// ActiveWorktreePathFormat returns the active worktree-path display format.
func ActiveWorktreePathFormat() string { return activeWorktreePathFormat }

// ---------------------------------------------------------------------------
// Date formatting
// ---------------------------------------------------------------------------

// FormatDate returns the date string per activeDateFormat.
func FormatDate(unix int64, rel string) string {
	switch activeDateFormat {
	case "long":
		if rel != "" {
			return rel
		}
		// fallback: compute short when no git-relative string is available
		return ShortDate(unix, nowFunc())
	case "iso":
		if unix == 0 {
			if rel != "" {
				return rel
			}
			return ""
		}
		return time.Unix(unix, 0).UTC().Format("2006-01-02 15:04")
	default: // "short"
		return ShortDate(unix, nowFunc())
	}
}

// ShortDate renders a unix timestamp as a compact relative string.
// Bucket boundaries are wall-clock approximations (<60 min → min, <24h → hr,
// <30d → day, <365d → mth, else yr).
func ShortDate(unix int64, now time.Time) string {
	if unix == 0 {
		return ""
	}
	diff := now.Unix() - unix
	if diff < 0 {
		diff = 0 // don't display future dates as "now"
	}
	switch {
	case diff < 60:
		return "now"
	case diff < 3600:
		return itoa(diff/60) + " min"
	case diff < 86400:
		return itoa(diff/3600) + " hr"
	case diff < 2592000: // 30 days
		return itoa(diff/86400) + " day"
	case diff < 31536000: // 365 days
		return itoa(diff/2592000) + " mth"
	default:
		return itoa(diff/31536000) + " yr"
	}
}

// itoa is a small int → string conversion. Avoids fmt import in format.go.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 12)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

// ---------------------------------------------------------------------------
// Branch name formatting
// ---------------------------------------------------------------------------

// FormatBranch returns the branch name per activeBranchFormat.
func FormatBranch(name string) string {
	switch activeBranchFormat {
	case "short":
		return ShortBranch(name)
	case "ultra-short":
		return UltraBranch(name)
	default: // "full"
		return name
	}
}

// ShortBranch compresses "domain1/domain2/branch-name" into "d/d/branch-name"
// by taking the first rune of every segment except the last.
func ShortBranch(name string) string {
	if !strings.ContainsRune(name, '/') {
		return name
	}
	parts := strings.Split(name, "/")
	for i := 0; i < len(parts)-1; i++ {
		parts[i] = firstRune(parts[i])
	}
	return strings.Join(parts, "/")
}

// firstRune returns the lowercased first character of s as a string, or "".
func firstRune(s string) string {
	for _, r := range s {
		return string(unicode.ToLower(r))
	}
	return ""
}

// UltraBranch compresses a branch name to the segment after its last "/" with
// every vowel stripped — "feat/auth/login-form" → "lgn-frm". Digits, "-", "_"
// and any non-vowel characters survive, so the result stays a recognizable
// slug even when the branch's own segment is vowel-heavy.
func UltraBranch(name string) string {
	seg := name
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		seg = name[i+1:]
	}
	var b strings.Builder
	for _, r := range seg {
		switch r {
		case 'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U':
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Author name formatting
// ---------------------------------------------------------------------------

// FormatAuthor returns the author name per activeAuthorFormat.
func FormatAuthor(name string) string {
	switch activeAuthorFormat {
	case "full":
		return name
	case "initials":
		return InitialsAuthor(name)
	default: // "short"
		return ShortAuthor(name)
	}
}

// ShortAuthor renders "Test User" as "Test U." (first name + last-name
// initial). A single-word name passes through unchanged.
func ShortAuthor(name string) string {
	if name == "" {
		return ""
	}
	fields := strings.Fields(name)
	if len(fields) < 2 {
		return name
	}
	last := fields[len(fields)-1]
	runes := []rune(last)
	if len(runes) == 0 {
		return strings.Join(fields, " ")
	}
	return strings.Join(fields[:len(fields)-1], " ") + " " + string(runes[0]) + "."
}

// InitialsAuthor renders "Test User" as "TU" (first rune of each word,
// uppercased).
func InitialsAuthor(name string) string {
	if name == "" {
		return ""
	}
	fields := strings.Fields(name)
	var b strings.Builder
	for _, f := range fields {
		for _, r := range f {
			b.WriteRune(unicode.ToUpper(r))
			break
		}
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Worktree path formatting
// ---------------------------------------------------------------------------

// FormatWorktreePath renders a worktree's path per activeWorktreePathFormat:
// "absolute" keeps it as-is, "relative" makes it relative to cwd (falling back
// to the absolute path when it escapes cwd), and "shortest" — the default —
// keeps the historical behavior: relative to cwd, else "~"-abbreviated from
// the home directory, else absolute.
func FormatWorktreePath(path, cwd string) string {
	switch activeWorktreePathFormat {
	case "absolute":
		return path
	case "relative":
		if rel, err := filepath.Rel(cwd, path); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
		return path
	default: // "shortest"
		if rel, err := filepath.Rel(cwd, path); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
		if home, err := os.UserHomeDir(); err == nil {
			if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
				return filepath.Join("~", rel)
			}
		}
		return path
	}
}
