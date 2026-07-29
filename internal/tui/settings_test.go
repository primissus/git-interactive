package tui

import (
	"os"
	"path/filepath"
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
