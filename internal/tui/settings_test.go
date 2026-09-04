package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withSettingsFile points XDG_CONFIG_HOME at a fresh temp dir, optionally
// writing gint/settings.json with contents, and restores env on cleanup.
func withSettingsFile(t *testing.T, contents string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if contents != "" {
		gintDir := filepath.Join(dir, "gint")
		if err := os.MkdirAll(gintDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		path := filepath.Join(gintDir, "settings.json")
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write settings.json: %v", err)
		}
	}
}

func TestLoadSettingsMissingFile(t *testing.T) {
	withSettingsFile(t, "")
	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: unexpected error %v", err)
	}
	if s != nil {
		t.Errorf("LoadSettings() = %+v, want nil for missing file", s)
	}
}

func TestLoadSettingsDefaults(t *testing.T) {
	withSettingsFile(t, `{}`)
	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if s == nil {
		t.Fatal("LoadSettings() = nil, want non-nil for empty {}")
	}
	if s.Appearance != "system" {
		t.Errorf("Appearance = %q, want %q (default)", s.Appearance, "system")
	}
	if s.Theme != "default" {
		t.Errorf("Theme = %q, want %q (default)", s.Theme, "default")
	}
}

func TestLoadSettingsRoundTrip(t *testing.T) {
	withSettingsFile(t, `{"appearance":"dark","theme":"gruvbox"}`)
	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if s.Appearance != "dark" {
		t.Errorf("Appearance = %q, want %q", s.Appearance, "dark")
	}
	if s.Theme != "gruvbox" {
		t.Errorf("Theme = %q, want %q", s.Theme, "gruvbox")
	}
}

// TestLoadSettingsNewFields verifies the phase-14 fields round-trip and that
// unknown values fall back to their defaults.
func TestLoadSettingsNewFields(t *testing.T) {
	withSettingsFile(t, `{
		"branchFormat":"ultra-short",
		"worktreePathFormat":"relative",
		"branchHiddenColumns":["last commit","worktree"],
		"logHiddenColumns":["sha"]
	}`)
	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if s.BranchFormat != "ultra-short" {
		t.Errorf("BranchFormat = %q, want %q", s.BranchFormat, "ultra-short")
	}
	if s.WorktreePathFormat != "relative" {
		t.Errorf("WorktreePathFormat = %q, want %q", s.WorktreePathFormat, "relative")
	}
	if len(s.BranchHiddenColumns) != 2 {
		t.Errorf("BranchHiddenColumns = %v, want 2 entries", s.BranchHiddenColumns)
	}
	if len(s.LogHiddenColumns) != 1 || s.LogHiddenColumns[0] != "sha" {
		t.Errorf("LogHiddenColumns = %v, want [sha]", s.LogHiddenColumns)
	}
}

func TestLoadSettingsNewFieldFallbacks(t *testing.T) {
	withSettingsFile(t, `{"branchFormat":"bogus","worktreePathFormat":"nope"}`)
	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if s.BranchFormat != "full" {
		t.Errorf("BranchFormat = %q, want fallback %q", s.BranchFormat, "full")
	}
	if s.WorktreePathFormat != "shortest" {
		t.Errorf("WorktreePathFormat = %q, want fallback %q", s.WorktreePathFormat, "shortest")
	}
}

// TestLoadSettingsUnknownThemeNormalizes verifies a corrupted settings.json
// (unknown theme) doesn't propagate the invalid name — it falls back to
// "default" so setPalette never sees an unknown theme downstream.
func TestLoadSettingsUnknownThemeNormalizes(t *testing.T) {
	withSettingsFile(t, `{"appearance":"light","theme":"notarealtheme"}`)
	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if s.Theme != "default" {
		t.Errorf("Theme = %q, want fallback %q", s.Theme, "default")
	}
}

// TestLoadSettingsMalformed warns-but-continues — same pattern as LoadKeymap:
// a malformed file returns an error so the caller can warn, but a missing
// file is not an error.
func TestLoadSettingsMalformed(t *testing.T) {
	withSettingsFile(t, `{not json`)
	if _, err := LoadSettings(); err == nil {
		t.Error("LoadSettings(malformed) = nil err, want error")
	}
}

func TestSaveSettingsRoundTrip(t *testing.T) {
	withSettingsFile(t, "")
	in := &Settings{Appearance: "light", Theme: "solarized"}
	if err := SaveSettings(in); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	out, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings after save: %v", err)
	}
	if out == nil {
		t.Fatal("LoadSettings after save = nil")
	}
	if out.Appearance != in.Appearance {
		t.Errorf("Appearance round-trip = %q, want %q", out.Appearance, in.Appearance)
	}
	if out.Theme != in.Theme {
		t.Errorf("Theme round-trip = %q, want %q", out.Theme, in.Theme)
	}
}

func TestSaveSettingsNilIsNoop(t *testing.T) {
	withSettingsFile(t, "")
	if err := SaveSettings(nil); err != nil {
		t.Errorf("SaveSettings(nil) = %v, want nil", err)
	}
	// No file should have been created.
	if _, err := LoadSettings(); err != nil {
		t.Errorf("LoadSettings after nil save = %v, want nil", err)
	}
}

// TestSaveSettingsErrorSurfaced verifies a write to a read-only dir returns an
// error (the caller's Status() message surfaces it to the user). We don't
// assert the exact error text — just that one is returned.
func TestSaveSettingsErrorSurfaced(t *testing.T) {
	// Point XDG_CONFIG_HOME at a read-only dir so the settings subdir can't
	// be created.
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := SaveSettings(&Settings{Appearance: "dark", Theme: "default"})
	if err == nil {
		t.Error("SaveSettings(read-only dir) = nil, want error")
	}
}

// TestApplySettingsUpdatesActiveState confirms ApplySettings propagates the
// Settings fields into the package-level ActiveTheme/Appearance globals.
func TestApplySettingsUpdatesActiveState(t *testing.T) {
	defer ApplySettings(&Settings{Appearance: "dark", Theme: "default"})

	ApplySettings(&Settings{Appearance: "light", Theme: "catppuccin"})
	if ActiveTheme.Name != "catppuccin" {
		t.Errorf("ActiveTheme.Name = %q, want %q", ActiveTheme.Name, "catppuccin")
	}
	if ActiveAppearance != "light" {
		t.Errorf("ActiveAppearance = %q, want %q", ActiveAppearance, "light")
	}
	if ActiveResolvedAppearance != "light" {
		t.Errorf("ActiveResolvedAppearance = %q, want %q",
			ActiveResolvedAppearance, "light")
	}
}

// TestApplySettingsNilRestoresDefaults verifies a nil pointer resets to the
// safe built-in defaults (system + default theme) — used by cli.Execute when
// LoadSettings returns an error so we never run with a half-set palette.
func TestApplySettingsNilRestoresDefaults(t *testing.T) {
	defer ApplySettings(&Settings{Appearance: "dark", Theme: "default"})

	ApplySettings(nil)
	if ActiveTheme.Name != "default" {
		t.Errorf("ActiveTheme.Name = %q, want %q", ActiveTheme.Name, "default")
	}
	// ApplySettings(nil) sets appearance="" per setPalette internals — normalize
	// via CurrentSettings for the assertion.
	cs := CurrentSettings()
	if cs.Theme != "default" {
		t.Errorf("CurrentSettings().Theme = %q, want %q", cs.Theme, "default")
	}
}

// TestCurrentSettingsSnapshotsActive confirms CurrentSettings reads the live
// active state — used by the settings overlay to detect drift between preview
// and saved state.
func TestCurrentSettingsSnapshotsActive(t *testing.T) {
	defer ApplySettings(&Settings{Appearance: "dark", Theme: "default"})
	ApplySettings(&Settings{Appearance: "dark", Theme: "nord"})
	cs := CurrentSettings()
	if cs.Appearance != "dark" {
		t.Errorf("Appearance = %q, want %q", cs.Appearance, "dark")
	}
	if cs.Theme != "nord" {
		t.Errorf("Theme = %q, want %q", cs.Theme, "nord")
	}
}

// branchDemoColumns mirrors gint br's column set (incl. the worktree and pr
// columns).
func branchDemoColumns() []Column {
	return []Column{
		{Title: "branch", MinWidth: 12, Flex: true, Density: DensityShort},
		{Title: "last commit", MaxWidth: 40, Density: DensityNormal},
		{Title: "date", MinWidth: 7, Density: DensityNormal},
		{Title: "author", MinWidth: 10, Density: DensityNormal},
		{Title: "worktree", MinWidth: 8, Density: DensityNormal},
		{Title: "pr", MinWidth: 6, Density: DensityNormal},
	}
}

func logDemoColumns() []Column {
	return []Column{
		{Title: "sha", MinWidth: 7, Density: DensityShort},
		{Title: "message", MinWidth: 12, MaxWidth: 40, Flex: true, Density: DensityShort},
		{Title: "date", MinWidth: 7, Density: DensityNormal},
		{Title: "author", MinWidth: 8, Density: DensityNormal},
		{Title: "branches", MinWidth: 8, Density: DensityNormal},
		{Title: "worktree", MinWidth: 8, Density: DensityFull},
	}
}

// newSettingsList builds a List configured for the given view with a live
// hidden-column predicate driven by the active hidden maps. DensityFull so
// every column passes the density gate and the tests exercise only the
// hidden-toggle behavior.
func newSettingsList(view string, columns []Column) *List {
	l := New(Config{
		Title:   "gint " + view,
		Columns: columns,
		Items:   DemoItems(),
		Density: DensityFull,
		View:    view,
	})
	switch view {
	case "branch":
		l.hiddenColumns = func() map[string]bool { return activeBranchHidden }
	case "log":
		l.hiddenColumns = func() map[string]bool { return activeLogHidden }
	}
	return l
}

func TestSettingsModelBranchViewSections(t *testing.T) {
	defer ApplySettings(&Settings{Appearance: "dark", Theme: "default"})
	ApplySettings(&Settings{Appearance: "dark", Theme: "default"})

	l := newSettingsList("branch", branchDemoColumns())
	l.openSettings()
	m := l.settings

	// rows: appearance, date, 6 display toggles (incl. pr), worktree-path cycle, 7 themes.
	if len(m.rows) != 16 {
		t.Fatalf("branch view has %d rows, want 16", len(m.rows))
	}
	if m.rows[0].kind != settingsRowCycle || m.rows[0].section != "Appearance" {
		t.Errorf("rows[0] = %+v, want Appearance cycle row", m.rows[0])
	}
	for i, title := range branchColumnTitles {
		r := m.rows[2+i]
		if r.kind != settingsRowToggle || r.toggleTitle != title || r.section != "Display" {
			t.Errorf("rows[2+%d] = %+v, want Display toggle %q", i, r, title)
		}
	}
	wt := m.rows[8]
	if wt.kind != settingsRowCycle || wt.section != "WorktreePath" || len(wt.options) != 3 {
		t.Errorf("rows[8] = %+v, want WorktreePath cycle with 3 options", wt)
	}
	for i := 9; i < 16; i++ {
		if m.rows[i].kind != settingsRowTheme {
			t.Errorf("rows[%d] = %+v, want theme row", i, m.rows[i])
		}
	}
}

func TestSettingsModelLogViewSections(t *testing.T) {
	defer ApplySettings(&Settings{Appearance: "dark", Theme: "default"})
	ApplySettings(&Settings{Appearance: "dark", Theme: "default"})

	l := newSettingsList("log", logDemoColumns())
	l.openSettings()
	m := l.settings

	// rows: appearance, date, 6 display toggles, author, branch, 7 themes.
	if len(m.rows) != 17 {
		t.Fatalf("log view has %d rows, want 17", len(m.rows))
	}
	for i, title := range logColumnTitles {
		r := m.rows[2+i]
		if r.kind != settingsRowToggle || r.toggleTitle != title || r.section != "Display" {
			t.Errorf("rows[2+%d] = %+v, want Display toggle %q", i, r, title)
		}
	}
	if m.rows[8].kind != settingsRowCycle || m.rows[8].section != "Author" {
		t.Errorf("rows[8] = %+v, want Author cycle row", m.rows[8])
	}
	if m.rows[9].kind != settingsRowCycle || m.rows[9].section != "Branch" {
		t.Errorf("rows[9] = %+v, want Branch cycle row", m.rows[9])
	}
	if len(m.rows[9].options) != 3 {
		t.Errorf("branch format options = %v, want 3 (incl. ultra-short)", m.rows[9].options)
	}
}

func TestSettingsDisplayToggleCheckedWhenShown(t *testing.T) {
	defer ApplySettings(&Settings{Appearance: "dark", Theme: "default"})
	ApplySettings(&Settings{Appearance: "dark", Theme: "default"})

	l := newSettingsList("branch", branchDemoColumns())
	l.openSettings()
	view := l.settings.View()
	for _, title := range branchColumnTitles {
		want := "[x] " + title
		if !strings.Contains(view, want) {
			t.Errorf("settings view missing %q (shown columns should be checked):\n%s", want, view)
		}
	}

	l.settings.cursor = 2
	l.settings.activate(2, 1) // hide "branch"
	view = l.settings.View()
	if !strings.Contains(view, "[ ] branch") {
		t.Errorf("hidden column should render as [ ] branch:\n%s", view)
	}
	if strings.Contains(view, "[x] branch") {
		t.Errorf("hidden branch still rendered as [x]:\n%s", view)
	}
}

func TestAuthorVisibleAtNormalDensityWhenNotHidden(t *testing.T) {
	l := New(Config{
		Columns:       branchDemoColumns(),
		Items:         DemoItems(),
		Density:       DensityNormal,
		HiddenColumns: func() map[string]bool { return map[string]bool{} },
	})
	got := map[string]bool{}
	for _, c := range l.visibleColumns() {
		got[c.Title] = true
	}
	if !got["author"] {
		t.Error("author missing at DensityNormal when not hidden")
	}
}

func TestSettingsModelToggleHiddenColumnLiveAndRevert(t *testing.T) {
	defer ApplySettings(&Settings{Appearance: "dark", Theme: "default"})
	ApplySettings(&Settings{Appearance: "dark", Theme: "default"})

	l := newSettingsList("branch", branchDemoColumns())
	if len(l.visibleColumns()) != 6 {
		t.Fatalf("initial visible columns = %d, want 6", len(l.visibleColumns()))
	}
	l.openSettings()
	m := l.settings

	// Cursor starts at appearance (0); j j → first Display toggle ("branch").
	m.cursor = 2
	m.activate(2, 1) // enter on "branch" toggle
	if !activeBranchHidden["branch"] {
		t.Errorf("activeBranchHidden after toggle = %v, want 'branch' hidden", activeBranchHidden)
	}
	if len(l.visibleColumns()) != 5 {
		t.Errorf("live visible columns = %d, want 5 (branch hidden)", len(l.visibleColumns()))
	}

	// Second toggle: "last commit".
	m.cursor = 3
	m.activate(3, 1)
	if !activeBranchHidden["last commit"] {
		t.Errorf("activeBranchHidden after 2nd toggle = %v, want 'last commit' hidden", activeBranchHidden)
	}
	if len(l.visibleColumns()) != 4 {
		t.Errorf("live visible columns = %d, want 4", len(l.visibleColumns()))
	}

	// Esc reverts everything live.
	m.revert()
	if len(activeBranchHidden) != 0 {
		t.Errorf("activeBranchHidden after revert = %v, want empty", activeBranchHidden)
	}
	if len(l.visibleColumns()) != 6 {
		t.Errorf("visible columns after revert = %d, want 6", len(l.visibleColumns()))
	}
}

// TestHiddenColumnsKeepCellAlignment is the regression test for the p14 bug
// where hiding columns filtered l.columns itself, so columnIndex returned
// positions within the filtered set and rows rendered the wrong cells (branch
// names under the "last commit" header). Cells must keep mapping to their
// full-set indices regardless of what is hidden.
func TestHiddenColumnsKeepCellAlignment(t *testing.T) {
	l := New(Config{
		Title:   "gint branch",
		Columns: branchDemoColumns(),
		Items:   DemoItems(),
		HiddenColumns: func() map[string]bool {
			// Hide everything except "last commit" — the buggy screenshot's
			// exact state.
			return map[string]bool{"branch": true, "date": true, "author": true, "worktree": true, "pr": true}
		},
	})

	cols := l.visibleColumns()
	if len(cols) != 1 || cols[0].Title != "last commit" {
		t.Fatalf("visibleColumns = %v, want only 'last commit'", cols)
	}
	// columnIndex must resolve against the FULL set: "last commit" is cell 1,
	// not 0. Cell 0 is the branch name — rendering it under "last commit" was
	// the bug.
	ci := l.columnIndex(cols[0])
	if ci != 1 {
		t.Fatalf("columnIndex('last commit') = %d, want 1 (full-set position)", ci)
	}
	for _, it := range DemoItems() {
		if cell(it, ci) != it.Columns()[1] {
			t.Errorf("cell(item, %d) = %q, want %q", ci, cell(it, ci), it.Columns()[1])
		}
	}
}

func TestSettingsModelWorktreePathCycleLiveAndRevert(t *testing.T) {
	defer ApplySettings(&Settings{Appearance: "dark", Theme: "default"})
	ApplySettings(&Settings{Appearance: "dark", Theme: "default"})

	l := newSettingsList("branch", branchDemoColumns())
	l.openSettings()
	m := l.settings

	if m.worktreePathFormat != "shortest" || activeWorktreePathFormat != "shortest" {
		t.Fatalf("initial worktree format = %q (active %q), want shortest", m.worktreePathFormat, activeWorktreePathFormat)
	}
	m.cursor = 8 // worktree-path cycle row
	m.activate(8, 1)
	if m.worktreePathFormat != "relative" || activeWorktreePathFormat != "relative" {
		t.Errorf("after right-cycle = %q (active %q), want relative", m.worktreePathFormat, activeWorktreePathFormat)
	}

	m.revert()
	if m.worktreePathFormat != "shortest" || activeWorktreePathFormat != "shortest" {
		t.Errorf("after revert = %q (active %q), want shortest", m.worktreePathFormat, activeWorktreePathFormat)
	}
}

// TestSettingsModelSavePersistsNewFields verifies `s` in the overlay writes the
// hidden-column lists + worktree-path format to settings.json, alongside the
// existing appearance/theme/formats.
func TestSettingsModelSavePersistsNewFields(t *testing.T) {
	defer ApplySettings(&Settings{Appearance: "dark", Theme: "default"})
	ApplySettings(&Settings{Appearance: "dark", Theme: "default"})
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	l := newSettingsList("branch", branchDemoColumns())
	l.openSettings()
	m := l.settings

	m.cursor = 2 // "branch" toggle
	m.activate(2, 1)
	m.cursor = 3 // "last commit" toggle
	m.activate(3, 1)
	m.cursor = 8     // worktree-path cycle
	m.activate(8, 1) // → relative

	m.Update(keyRunes('s'))
	if m.state != settingsApplied {
		t.Fatalf("state after save = %v, want settingsApplied", m.state)
	}

	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings after save: %v", err)
	}
	if s == nil {
		t.Fatal("LoadSettings after save = nil")
	}
	if s.WorktreePathFormat != "relative" {
		t.Errorf("persisted WorktreePathFormat = %q, want %q", s.WorktreePathFormat, "relative")
	}
	if len(s.BranchHiddenColumns) != 2 {
		t.Errorf("persisted BranchHiddenColumns = %v, want 2 entries", s.BranchHiddenColumns)
	}
	got := map[string]bool{}
	for _, title := range s.BranchHiddenColumns {
		got[title] = true
	}
	if !got["branch"] || !got["last commit"] {
		t.Errorf("persisted BranchHiddenColumns = %v, want branch + last commit", s.BranchHiddenColumns)
	}
}
