package tui

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
)

// Chrome holds every generic (non-command-specific) hint string the shared
// views render: footers, prompts, placeholders, and the "?" overlay's
// navigation table. defaultChrome mirrors today's hardcoded values exactly;
// an optional ~/.config/gint/keymap.json overrides individual fields.
type Chrome struct {
	Footer       string
	SelectFooter string // has one %d verb (selection count)
	SelectStatus string

	SearchPrompt      string
	SearchPlaceholder string

	MenuPrompt      string
	MenuPlaceholder string
	MenuFooter      string

	InputPrompt string
	InputFooter string

	ConfirmPrompt       string
	ConfirmYesNoFooter  string
	ConfirmChoiceFooter string

	BatchFooter         string
	BatchContinuePrompt string // has one %d verb (remaining count)
	BatchSummary        string // has %s (verb) and %d (count) verbs
	BatchFailSuffix     string // has %d (count) and %s (reasons) verbs
	Nav                 [][2]string
}

// defaultChrome returns gint's built-in hint text — the literal values every
// call site used before keymap.json existed. This is the sole source of
// truth for defaults; keymap.json only needs to list what it overrides.
func defaultChrome() Chrome {
	return Chrome{
		Footer:       "j/k move · u/d ½page · / search · enter menu · X select · ? help · q quit",
		SelectFooter: "space/x toggle · enter bulk ops · esc exit · %d selected",
		SelectStatus: "select mode: space to toggle, enter for bulk operations",

		SearchPrompt:      "/ ",
		SearchPlaceholder: "fuzzy search",

		MenuPrompt:      "/ ",
		MenuPlaceholder: "filter operations",
		MenuFooter:      "type to filter · ↑/↓ move · enter run · esc cancel",

		InputPrompt: "› ",
		InputFooter: "enter submit · esc cancel",

		ConfirmPrompt:       "› ",
		ConfirmYesNoFooter:  "enter confirm · esc cancel",
		ConfirmChoiceFooter: "←/→ move · enter select · esc cancel",

		BatchFooter:         "[y] yes · [a] all · [n] stop",
		BatchContinuePrompt: "continue? %d left  ",
		BatchSummary:        "%s %d",
		BatchFailSuffix:     " · failed %d (%s)",

		Nav: [][2]string{
			{"j / k", "move down / up"},
			{"u / d / ⌥↑ / ⌥↓", "half-page down / up"},
			{"h / l", "page back / forward"},
			{"g / G", "jump to top / bottom (12g → row 12)"},
			{"10j", "repeat a motion N times"},
			{"/", "fuzzy search"},
			{"enter", "open menu"},
			{"X", "select mode"},
			{"space / x", "toggle selection"},
			{"?", "this help"},
			{"q", "quit"},
		},
	}
}

// OperationOverride replaces an Operation's Key and/or Name. Either field may
// be left empty (omitted from the JSON) to keep that side's default.
type OperationOverride struct {
	Key   string `json:"key,omitempty"`
	Label string `json:"label,omitempty"`
}

// Keymap is the resolved (defaults + user overrides) hint configuration.
type Keymap struct {
	Chrome     Chrome
	Operations map[string]OperationOverride // "<namespace>.<op id>" -> override
}

// chromeOverride mirrors Chrome with every field optional, so a keymap.json
// only needs to name the fields it actually overrides.
type chromeOverride struct {
	Footer       *string `json:"footer"`
	SelectFooter *string `json:"select_footer"`
	SelectStatus *string `json:"select_status"`

	SearchPrompt      *string `json:"search_prompt"`
	SearchPlaceholder *string `json:"search_placeholder"`

	MenuPrompt      *string `json:"menu_prompt"`
	MenuPlaceholder *string `json:"menu_placeholder"`
	MenuFooter      *string `json:"menu_footer"`

	InputPrompt *string `json:"input_prompt"`
	InputFooter *string `json:"input_footer"`

	ConfirmPrompt       *string `json:"confirm_prompt"`
	ConfirmYesNoFooter  *string `json:"confirm_yes_no_footer"`
	ConfirmChoiceFooter *string `json:"confirm_choice_footer"`

	BatchFooter         *string      `json:"batch_footer"`
	BatchContinuePrompt *string      `json:"batch_continue_prompt"`
	BatchSummary        *string      `json:"batch_summary"`
	BatchFailSuffix     *string      `json:"batch_fail_suffix"`
	Nav                 *[][2]string `json:"nav"`
}

// keymapFile is keymap.json's on-disk shape.
type keymapFile struct {
	Chrome     *chromeOverride              `json:"chrome,omitempty"`
	Operations map[string]OperationOverride `json:"operations,omitempty"`
}

// activeKeymap is the process-wide resolved keymap, set once by LoadKeymap
// (or left at defaults if it's never called / no override file exists).
var activeKeymap = Keymap{Chrome: defaultChrome(), Operations: map[string]OperationOverride{}}

// keymapPath resolves ~/.config/gint/keymap.json, honoring $XDG_CONFIG_HOME
// when set. Deliberately XDG-style on every OS (not os.UserConfigDir, which
// resolves to "Application Support" on macOS) — gint is a terminal tool and
// this keeps the path predictable and grep-able across platforms.
func keymapPath() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "gint", "keymap.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gint", "keymap.json"), nil
}

// LoadKeymap reads the user's keymap.json (if any) and merges it over the
// built-in defaults, replacing the active keymap for the process. Call once
// at startup, before any command builds its Operations or renders. A missing
// file is not an error — defaults stay active. A present-but-malformed file
// returns an error; the caller should warn and continue (never fatal over a
// config typo), which is why defaults are only replaced on full success.
func LoadKeymap() error {
	path, err := keymapPath()
	if err != nil || path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("keymap.json: %w", err)
	}

	var kf keymapFile
	if err := json.Unmarshal(data, &kf); err != nil {
		return fmt.Errorf("keymap.json: %w", err)
	}

	km := Keymap{Chrome: defaultChrome(), Operations: map[string]OperationOverride{}}
	applyChromeOverride(&km.Chrome, kf.Chrome)
	maps.Copy(km.Operations, kf.Operations)
	activeKeymap = km
	return nil
}

// applyChromeOverride copies every non-nil field of ov onto c.
func applyChromeOverride(c *Chrome, ov *chromeOverride) {
	if ov == nil {
		return
	}
	set := func(dst *string, src *string) {
		if src != nil {
			*dst = *src
		}
	}
	set(&c.Footer, ov.Footer)
	set(&c.SelectFooter, ov.SelectFooter)
	set(&c.SelectStatus, ov.SelectStatus)
	set(&c.SearchPrompt, ov.SearchPrompt)
	set(&c.SearchPlaceholder, ov.SearchPlaceholder)
	set(&c.MenuPrompt, ov.MenuPrompt)
	set(&c.MenuPlaceholder, ov.MenuPlaceholder)
	set(&c.MenuFooter, ov.MenuFooter)
	set(&c.InputPrompt, ov.InputPrompt)
	set(&c.InputFooter, ov.InputFooter)
	set(&c.ConfirmPrompt, ov.ConfirmPrompt)
	set(&c.ConfirmYesNoFooter, ov.ConfirmYesNoFooter)
	set(&c.ConfirmChoiceFooter, ov.ConfirmChoiceFooter)
	set(&c.BatchFooter, ov.BatchFooter)
	set(&c.BatchContinuePrompt, ov.BatchContinuePrompt)
	set(&c.BatchSummary, ov.BatchSummary)
	set(&c.BatchFailSuffix, ov.BatchFailSuffix)
	if ov.Nav != nil {
		c.Nav = *ov.Nav
	}
}

// chrome returns the active chrome hint set. Render call sites read it
// directly — no parsing happens here, just a struct field read.
func chrome() Chrome { return activeKeymap.Chrome }

// id returns op's keymap-lookup identifier: ID when set, else Name.
func (op Operation) id() string {
	if op.ID != "" {
		return op.ID
	}
	return op.Name
}

// ApplyKeymap overrides Key/Name on ops that have a configured override under
// "<namespace>.<op id>", leaving Scope/Confirm/Input/Batch/Run untouched. Call
// once per command, right after building its []Operation and before handing
// it to tui.New.
func ApplyKeymap(namespace string, ops []Operation) []Operation {
	for i := range ops {
		ov, ok := activeKeymap.Operations[namespace+"."+ops[i].id()]
		if !ok {
			continue
		}
		if ov.Key != "" {
			ops[i].Key = ov.Key
		}
		if ov.Label != "" {
			ops[i].Name = ov.Label
		}
	}
	return ops
}
