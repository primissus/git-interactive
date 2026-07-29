package tui

import (
	"strings"
	"time"
	"unicode"
)

// Format settings: frozen by ApplySettings and read live on every render.
var (
	activeDateFormat   = "short" // "short" | "long" | "iso"
	activeBranchFormat = "full"  // "full" | "short"
	activeAuthorFormat = "short" // "short" | "initials" | "full"
)

// nowFunc is time.Now by default; stubbed in tests.
var nowFunc = time.Now

// freezeFormats copies the three format fields from Settings into the active
// package-level vars. Called by ApplySettings.
func freezeFormats(s *Settings) {
	if s == nil {
		activeDateFormat = "short"
		activeBranchFormat = "full"
		activeAuthorFormat = "short"
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
}

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
	if activeBranchFormat != "short" {
		return name
	}
	return ShortBranch(name)
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
