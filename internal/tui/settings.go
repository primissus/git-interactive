package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Settings is the on-disk shape of ~/.config/gint/settings.json. Both fields are
// optional — empty/missing means "use the defaults" (System appearance, default
// theme).
type Settings struct {
	// Appearance is one of "system", "light", "dark", or "" (→ system).
	Appearance string `json:"appearance,omitempty"`
	// Theme is a name from ThemeNames(); "" or unknown falls back to "default".
	Theme string `json:"theme,omitempty"`
}

// settingsPath resolves ~/.config/gint/settings.json, honoring $XDG_CONFIG_HOME
// when set. Mirrors keymapPath()'s logic — deliberately XDG-style on every OS
// so the path stays predictable and grep-able across platforms.
func settingsPath() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "gint", "settings.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gint", "settings.json"), nil
}

// configDir returns the directory containing gint's config files (keymap.json,
// settings.json), creating it on demand. Shared with keymapPath()'s resolution.
func configDir() (string, error) {
	path, err := settingsPath()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("settings.json: %w", err)
	}
	return dir, nil
}

// LoadSettings reads settings.json. A missing file is not an error: it returns
// (nil, nil), and the caller applies the built-in defaults (System + default).
// A present-but-malformed file returns an error; the caller warns and continues
// with defaults — same pattern as LoadKeymap.
func LoadSettings() (*Settings, error) {
	path, err := settingsPath()
	if err != nil || path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("settings.json: %w", err)
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("settings.json: %w", err)
	}
	if s.Appearance == "" {
		s.Appearance = "system"
	}
	if s.Theme == "" {
		s.Theme = "default"
	}
	// Unknown theme: warn-and-fall-back. The caller decides whether to surface
	// a warning; here we just normalize so the rest of the app never sees an
	// invalid theme name.
	if _, ok := ThemeByName(s.Theme); !ok {
		s.Theme = "default"
	}
	return &s, nil
}

// SaveSettings writes settings.json, creating the config dir if needed. The
// caller surfaces any write failure via a SettingsSaveFailed status message —
// never crash the TUI over a disk error.
func SaveSettings(s *Settings) error {
	if s == nil {
		return nil
	}
	dir, err := configDir()
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".settings-*.json")
	if err != nil {
		return fmt.Errorf("settings.json: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // cleanup on any failure path
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("settings.json: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("settings.json: %w", err)
	}
	// Atomic rename — never leave a half-written settings.json on disk.
	return os.Rename(tmpPath, filepath.Join(dir, "settings.json"))
}

// ApplySettings sets the active theme + appearance from a loaded Settings, then
// runs setPalette so all the package-level color vars reflect the user's choice.
// Called by cli.Execute before any command builds its columns/operations, and
// again by the settings overlay's "save" action when the user confirms a change.
func ApplySettings(s *Settings) {
	if s == nil {
		setPalette("default", "system")
		return
	}
	setPalette(s.Theme, s.Appearance)
}

// CurrentSettings snapshots the active theme + appearance so the settings
// overlay can pre-fill with them. Messy values are normalized to the persisted
// shape ("" → "system", unknown theme → "default").
func CurrentSettings() *Settings {
	appearance := ActiveAppearance
	if appearance == "" {
		appearance = "system"
	}
	theme := ActiveTheme.Name
	if theme == "" {
		theme = "default"
	}
	return &Settings{Appearance: appearance, Theme: theme}
}
