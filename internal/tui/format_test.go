package tui

import (
	"testing"
	"time"
)

func TestShortDate(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		unix int64
		want string
	}{
		{now.Unix(), "now"},
		{now.Unix() - 30, "now"},
		{now.Unix() - 59, "now"},
		{now.Unix() - 60, "1 min"},
		{now.Unix() - 3599, "59 min"},
		{now.Unix() - 3600, "1 hr"},
		{now.Unix() - 86399, "23 hr"},
		{now.Unix() - 86400, "1 day"},
		{now.Unix() - 86400*29, "29 day"},
		{now.Unix() - 86400*30, "1 mth"},
		{now.Unix() - 86400*364, "12 mth"},
		{now.Unix() - 86400*365, "1 yr"},
		{now.Unix() - 86400*365*3, "3 yr"},
		{0, ""},
		{now.Unix() + 3600, "now"}, // future → "now"
	}

	for _, tc := range tests {
		got := ShortDate(tc.unix, now)
		if got != tc.want {
			t.Errorf("ShortDate(%d, now) = %q, want %q", tc.unix, got, tc.want)
		}
	}
}

func TestShortBranch(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"main", "main"},
		{"feature/login", "f/login"},
		{"domain1/domain2/branch-name", "d/d/branch-name"},
		{"a/b/c/d", "a/b/c/d"},
		{"features/Fix-bug/UI_Tweak", "f/f/UI_Tweak"},
		{"(detached)", "(detached)"},
		{"", ""},
		{"feature/", "f/"},
		{"/leading-slash", "/leading-slash"},
	}

	for _, tc := range tests {
		got := ShortBranch(tc.name)
		if got != tc.want {
			t.Errorf("ShortBranch(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestShortAuthor(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Test User", "Test U."},
		{"Ada Lovelace", "Ada L."},
		{"Cher", "Cher"},
		{"Mary Jane Watson", "Mary Jane W."},
		{"", ""},
		{"SingleName", "SingleName"},
		{"  spaces  ", "  spaces  "},
	}

	for _, tc := range tests {
		got := ShortAuthor(tc.name)
		if got != tc.want {
			t.Errorf("ShortAuthor(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestInitialsAuthor(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Test User", "TU"},
		{"Ada Lovelace", "AL"},
		{"Cher", "C"},
		{"Mary Jane Watson", "MJW"},
		{"", ""},
		{"  spaces  ", "S"},
		{"josé garcía", "JG"},
	}

	for _, tc := range tests {
		got := InitialsAuthor(tc.name)
		if got != tc.want {
			t.Errorf("InitialsAuthor(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	savedNow := nowFunc
	nowFunc = func() time.Time { return now }
	defer func() { nowFunc = savedNow }()

	unix := now.Unix() - 7200 // 2 hours ago
	rel := "2 hours ago"

	// short (default)
	activeDateFormat = "short"
	if got := FormatDate(unix, rel); got != "2 hr" {
		t.Errorf("FormatDate(short) = %q, want %q", got, "2 hr")
	}

	// long
	activeDateFormat = "long"
	if got := FormatDate(unix, rel); got != rel {
		t.Errorf("FormatDate(long) = %q, want %q", got, rel)
	}

	// fallback when rel empty → short
	if got := FormatDate(unix, ""); got != "2 hr" {
		t.Errorf("FormatDate(long, empty rel) = %q, want %q", got, "2 hr")
	}

	// iso (UTC)
	activeDateFormat = "iso"
	wantISO := "2026-01-15 10:00"
	if got := FormatDate(unix, rel); got != wantISO {
		t.Errorf("FormatDate(iso) = %q, want %q", got, wantISO)
	}

	// iso with zero unix returns rel
	if got := FormatDate(0, rel); got != rel {
		t.Errorf("FormatDate(iso, 0 unix) = %q, want %q", got, rel)
	}

	activeDateFormat = "short" // restore
}

func TestFormatBranch(t *testing.T) {
	activeBranchFormat = "full"
	if got := FormatBranch("domain1/domain2/name"); got != "domain1/domain2/name" {
		t.Errorf("FormatBranch(full) = %q, want %q", got, "domain1/domain2/name")
	}

	activeBranchFormat = "short"
	if got := FormatBranch("domain1/domain2/name"); got != "d/d/name" {
		t.Errorf("FormatBranch(short) = %q, want %q", got, "d/d/name")
	}

	activeBranchFormat = "ultra-short"
	if got := FormatBranch("feat/auth/login-form"); got != "lgn-frm" {
		t.Errorf("FormatBranch(ultra-short) = %q, want %q", got, "lgn-frm")
	}

	activeBranchFormat = "full" // restore
}

func TestUltraBranch(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"feat/auth/login-form", "lgn-frm"},
		{"main", "mn"},
		{"feature/UI_Tweak", "_Twk"},
		{"a/e/i/o/u", ""}, // vowel-only last segment
		{"deps/2024.3-r1", "2024.3-r1"},
		{"(detached)", "(dtchd)"},
		{"", ""},
		{"single", "sngl"},
		{"Mixed/Case-Version", "Cs-Vrsn"},
		{"trailing/", ""},
		{"no-slash-here", "n-slsh-hr"},
	}

	for _, tc := range tests {
		got := UltraBranch(tc.name)
		if got != tc.want {
			t.Errorf("UltraBranch(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestFormatAuthor(t *testing.T) {
	activeAuthorFormat = "short"
	if got := FormatAuthor("Test User"); got != "Test U." {
		t.Errorf("FormatAuthor(short) = %q, want %q", got, "Test U.")
	}

	activeAuthorFormat = "initials"
	if got := FormatAuthor("Test User"); got != "TU" {
		t.Errorf("FormatAuthor(initials) = %q, want %q", got, "TU")
	}

	activeAuthorFormat = "full"
	if got := FormatAuthor("Test User"); got != "Test User" {
		t.Errorf("FormatAuthor(full) = %q, want %q", got, "Test User")
	}

	activeAuthorFormat = "short" // restore
}

func TestFreezeFormats(t *testing.T) {
	// nil → defaults
	freezeFormats(nil)
	if activeDateFormat != "short" || activeBranchFormat != "full" || activeAuthorFormat != "short" {
		t.Errorf("freezeFormats(nil) = %s/%s/%s, want short/full/short",
			activeDateFormat, activeBranchFormat, activeAuthorFormat)
	}
	if activeWorktreePathFormat != "shortest" {
		t.Errorf("freezeFormats(nil) worktreePathFormat = %q, want %q", activeWorktreePathFormat, "shortest")
	}
	if len(activeBranchHidden) != 0 || len(activeLogHidden) != 0 {
		t.Errorf("freezeFormats(nil) hidden maps non-empty: %v / %v", activeBranchHidden, activeLogHidden)
	}

	// empty strings → defaults
	freezeFormats(&Settings{DateFormat: "", BranchFormat: "", AuthorFormat: ""})
	if activeDateFormat != "short" || activeBranchFormat != "full" || activeAuthorFormat != "short" {
		t.Errorf("freezeFormats(empty) = %s/%s/%s, want short/full/short",
			activeDateFormat, activeBranchFormat, activeAuthorFormat)
	}

	// explicit values
	freezeFormats(&Settings{
		DateFormat: "iso", BranchFormat: "short", AuthorFormat: "initials",
		WorktreePathFormat:  "relative",
		BranchHiddenColumns: []string{"last commit", ""},
		LogHiddenColumns:    []string{"worktree"},
	})
	if activeDateFormat != "iso" || activeBranchFormat != "short" || activeAuthorFormat != "initials" {
		t.Errorf("freezeFormats(explicit) = %s/%s/%s, want iso/short/initials",
			activeDateFormat, activeBranchFormat, activeAuthorFormat)
	}
	if activeWorktreePathFormat != "relative" {
		t.Errorf("freezeFormats(explicit) worktreePathFormat = %q, want %q", activeWorktreePathFormat, "relative")
	}
	if !activeBranchHidden["last commit"] {
		t.Errorf("freezeFormats(explicit) branchHidden = %v, want 'last commit' hidden", activeBranchHidden)
	}
	if len(activeBranchHidden) != 1 {
		t.Errorf("freezeFormats(explicit) branchHidden = %v, want exactly 1 entry (empty title dropped)", activeBranchHidden)
	}
	if !activeLogHidden["worktree"] {
		t.Errorf("freezeFormats(explicit) logHidden = %v, want 'worktree' hidden", activeLogHidden)
	}
}

func TestFormatDateEdgeCases(t *testing.T) {
	activeDateFormat = "short"
	// 0 unix, empty rel
	if got := FormatDate(0, ""); got != "" {
		t.Errorf("FormatDate(0, \"\") = %q, want empty string", got)
	}
	// zero unix but rel present → returns rel in long mode
	activeDateFormat = "long"
	if got := FormatDate(0, "some time"); got != "some time" {
		t.Errorf("FormatDate(0, 'some time') long = %q, want rel", got)
	}
	activeDateFormat = "short"
}

func TestFormatWorktreePath(t *testing.T) {
	const cwd = "/home/user/proj"
	tests := []struct {
		format string
		path   string
		want   string
	}{
		{"shortest", "/home/user/proj/sub", "sub"},
		{"shortest", "/home/user/proj", "."},
		{"shortest", "/other/repo", "/other/repo"}, // escapes cwd and home → absolute
		{"relative", "/home/user/proj/sub", "sub"},
		{"relative", "/home/user/proj", "."},
		{"relative", "/other/repo", "/other/repo"}, // escapes cwd → absolute
		{"absolute", "/home/user/proj/sub", "/home/user/proj/sub"},
	}
	for _, tc := range tests {
		activeWorktreePathFormat = tc.format
		if got := FormatWorktreePath(tc.path, cwd); got != tc.want {
			t.Errorf("FormatWorktreePath(%q, %q) [%s] = %q, want %q", tc.path, cwd, tc.format, got, tc.want)
		}
	}
	activeWorktreePathFormat = "shortest" // restore
}

func TestHiddenSetHelpers(t *testing.T) {
	set := toHiddenSet([]string{"branch", "", "date", "branch"})
	if !set["branch"] || !set["date"] {
		t.Errorf("toHiddenSet = %v, want branch+date present", set)
	}
	if len(set) != 2 {
		t.Errorf("toHiddenSet = %v, want 2 unique entries (empty+dup dropped)", set)
	}

	list := hiddenToList(set)
	if len(list) != 2 {
		t.Errorf("hiddenToList = %v, want 2 entries", list)
	}
	for _, title := range list {
		if !set[title] {
			t.Errorf("hiddenToList = %v, entry %q missing from set", list, title)
		}
	}
}
