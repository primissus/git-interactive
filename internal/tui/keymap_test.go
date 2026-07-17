package tui

import (
	"os"
	"path/filepath"
	"testing"
)

// withKeymapFile points XDG_CONFIG_HOME at a fresh temp dir, optionally
// writing gint/keymap.json with contents, and restores activeKeymap/the env
// var on cleanup so tests don't leak state into each other.
func withKeymapFile(t *testing.T, contents string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if contents != "" {
		gintDir := filepath.Join(dir, "gint")
		if err := os.MkdirAll(gintDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(gintDir, "keymap.json"), []byte(contents), 0o644); err != nil {
			t.Fatalf("write keymap.json: %v", err)
		}
	}
	saved := activeKeymap
	t.Cleanup(func() { activeKeymap = saved })
}

func TestLoadKeymapNoFileKeepsDefaults(t *testing.T) {
	withKeymapFile(t, "")
	want := defaultChrome()
	if err := LoadKeymap(); err != nil {
		t.Fatalf("LoadKeymap: %v", err)
	}
	if activeKeymap.Chrome.Footer != want.Footer {
		t.Errorf("Footer = %q, want default %q", activeKeymap.Chrome.Footer, want.Footer)
	}
	if len(activeKeymap.Operations) != 0 {
		t.Errorf("Operations = %v, want empty", activeKeymap.Operations)
	}
}

func TestLoadKeymapPartialOverride(t *testing.T) {
	withKeymapFile(t, `{
		"chrome": {"footer": "custom footer"},
		"operations": {"status.toggle stage": {"key": "x"}}
	}`)
	if err := LoadKeymap(); err != nil {
		t.Fatalf("LoadKeymap: %v", err)
	}

	if got, want := activeKeymap.Chrome.Footer, "custom footer"; got != want {
		t.Errorf("Footer = %q, want %q", got, want)
	}
	// Untouched chrome fields keep their default.
	want := defaultChrome()
	if got := activeKeymap.Chrome.SelectFooter; got != want.SelectFooter {
		t.Errorf("SelectFooter = %q, want default %q (should be untouched)", got, want.SelectFooter)
	}
	if got := activeKeymap.Chrome.Nav; len(got) != len(want.Nav) {
		t.Errorf("Nav = %v, want default (untouched, len %d)", got, len(want.Nav))
	}

	ov, ok := activeKeymap.Operations["status.toggle stage"]
	if !ok || ov.Key != "x" {
		t.Errorf("Operations[status.toggle stage] = %+v, ok=%v; want Key=x", ov, ok)
	}
}

func TestLoadKeymapMalformedFileReturnsErrorKeepsDefaults(t *testing.T) {
	withKeymapFile(t, `{ not valid json`)
	before := activeKeymap
	if err := LoadKeymap(); err == nil {
		t.Fatal("LoadKeymap: expected an error for malformed JSON, got nil")
	}
	if activeKeymap.Chrome.Footer != before.Chrome.Footer {
		t.Error("activeKeymap was mutated despite the malformed file")
	}
}

func TestApplyKeymapOverridesKeyAndLabel(t *testing.T) {
	withKeymapFile(t, `{"operations": {"demo.checkout": {"key": "z", "label": "check out"}}}`)
	if err := LoadKeymap(); err != nil {
		t.Fatalf("LoadKeymap: %v", err)
	}

	ops := []Operation{{Name: "checkout", Key: "C"}, {Name: "other", Key: "o"}}
	got := ApplyKeymap("demo", ops)

	if got[0].Key != "z" || got[0].Name != "check out" {
		t.Errorf("ops[0] = %+v, want Key=z Name=\"check out\"", got[0])
	}
	if got[1].Key != "o" || got[1].Name != "other" {
		t.Errorf("ops[1] (no override) = %+v, want unchanged", got[1])
	}
}

func TestApplyKeymapUsesIDOverName(t *testing.T) {
	withKeymapFile(t, `{"operations": {"conflict.take-ours": {"key": "1"}}}`)
	if err := LoadKeymap(); err != nil {
		t.Fatalf("LoadKeymap: %v", err)
	}

	ops := []Operation{{Name: "take ours (main)", ID: "take-ours", Key: "o"}}
	got := ApplyKeymap("conflict", ops)

	if got[0].Key != "1" {
		t.Errorf("Key = %q, want \"1\" (lookup should use ID, not the dynamic Name)", got[0].Key)
	}
	if got[0].Name != "take ours (main)" {
		t.Errorf("Name = %q, want unchanged (no label override set)", got[0].Name)
	}
}
