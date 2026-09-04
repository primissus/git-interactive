package tui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func newTestModel(t *testing.T) *teatest.TestModel {
	t.Helper()
	return teatest.NewTestModel(t, Demo(DensityNormal), teatest.WithInitialTermSize(100, 24))
}

// sendKeys posts a sequence of key messages in order.
func sendKeys(tm *teatest.TestModel, keys ...tea.KeyMsg) {
	for _, k := range keys {
		tm.Send(k)
	}
}

// finish quits the program and returns the final List for state inspection.
func finish(t *testing.T, tm *teatest.TestModel) *List {
	t.Helper()
	tm.Send(keyRunes('q'))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	return tm.FinalModel(t).(*List)
}

func waitForText(t *testing.T, tm *teatest.TestModel, want string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte(want))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(25*time.Millisecond))
}

func TestNavigationMovesCursor(t *testing.T) {
	tm := newTestModel(t)
	// down, down, down, up → row index 2.
	sendKeys(tm,
		keyRunes('j'), keyRunes('j'), keyRunes('j'), keyRunes('k'),
	)
	l := finish(t, tm)
	if l.cursor != 2 {
		t.Fatalf("cursor after j j j k: got %d, want 2", l.cursor)
	}

	// g/G jump to the ends.
	tm = newTestModel(t)
	sendKeys(tm, keyRunes('G'))
	l = finish(t, tm)
	if l.cursor != len(l.visible)-1 {
		t.Fatalf("cursor after G: got %d, want %d", l.cursor, len(l.visible)-1)
	}
}

func TestSearchFiltersRows(t *testing.T) {
	tm := newTestModel(t)
	sendKeys(tm, keyRunes('/'))
	tm.Type("branch")
	sendKeys(tm, keyType(tea.KeyEnter))
	l := finish(t, tm)

	if len(l.visible) != 1 {
		t.Fatalf("search 'branch' matched %d rows, want 1", len(l.visible))
	}
	if got := l.items[l.visible[0]].FilterValue(); got != "feat/branch-view" {
		t.Fatalf("search 'branch' matched %q, want feat/branch-view", got)
	}
}

func TestSearchEscRestoresRows(t *testing.T) {
	tm := newTestModel(t)
	sendKeys(tm, keyRunes('/'))
	tm.Type("branch")
	sendKeys(tm, keyType(tea.KeyEsc))
	l := finish(t, tm)

	if len(l.visible) != len(DemoItems()) {
		t.Fatalf("esc did not restore rows: got %d, want %d", len(l.visible), len(DemoItems()))
	}
	if l.mode != modeList {
		t.Fatalf("esc did not return to list mode: got %v", l.mode)
	}
}

func TestMenuDispatchesOperation(t *testing.T) {
	tm := newTestModel(t)
	// Enter opens the menu (cursor on "checkout"); Enter selects it; the
	// yes/no confirm accepts on 'y'.
	sendKeys(tm, keyType(tea.KeyEnter), keyType(tea.KeyEnter), keyRunes('y'))
	waitForText(t, tm, "checked out main")
	finish(t, tm)
}

// defaultActionItem is a minimal DefaultActioner for TestEnterOnDefaultActionerSkipsMenu.
type defaultActionItem struct{ op string }

func (defaultActionItem) Columns() []string   { return []string{"+ create"} }
func (defaultActionItem) FilterValue() string { return "" }
func (defaultActionItem) Current() bool       { return false }
func (i defaultActionItem) DefaultOp() string { return i.op }

func TestEnterOnDefaultActionerSkipsMenu(t *testing.T) {
	l := New(Config{
		Title:   "test",
		Columns: []Column{{Title: "col", MinWidth: 4, Flex: true}},
		Items:   []Item{defaultActionItem{op: "new"}},
		Operations: []Operation{
			{
				Name: "new", Key: "N", Scope: ScopeList,
				Input: &InputSpec{Prompt: "New name", Placeholder: "name"},
				Run: func(c OpContext) tea.Cmd {
					return Status("created " + c.Input)
				},
			},
		},
	})
	tm := teatest.NewTestModel(t, l, teatest.WithInitialTermSize(100, 24))
	// Enter on a DefaultActioner row must jump straight to its named
	// operation's input prompt, not the (here, single-entry) context menu.
	sendKeys(tm, keyType(tea.KeyEnter))
	tm.Type("foo")
	sendKeys(tm, keyType(tea.KeyEnter))
	waitForText(t, tm, "created foo")
	finish(t, tm)
}

func TestMenuFuzzyMatchesOperation(t *testing.T) {
	tm := newTestModel(t)
	// Open the menu and type "pu" to disambiguate to "push", then run it.
	sendKeys(tm, keyType(tea.KeyEnter))
	tm.Type("push")
	sendKeys(tm, keyType(tea.KeyEnter), keyRunes('y'))
	waitForText(t, tm, "pushed main")
	finish(t, tm)
}

func TestShortcutTypedConfirm(t *testing.T) {
	tm := newTestModel(t)
	// 'D' deletes the current row; choose "force" (escalates to a typed phrase).
	sendKeys(tm, keyRunes('D'), keyRunes('f'))
	tm.Type("force")
	sendKeys(tm, keyType(tea.KeyEnter))
	waitForText(t, tm, "deleted main (force)")
	finish(t, tm)
}

func TestInputFlow(t *testing.T) {
	tm := newTestModel(t)
	// 'R' renames the current row via a text prompt.
	sendKeys(tm, keyRunes('R'))
	tm.Type("renamed-main")
	sendKeys(tm, keyType(tea.KeyEnter))
	waitForText(t, tm, "renamed main to renamed-main")
	finish(t, tm)
}

func TestSelectModeBulkTypedConfirm(t *testing.T) {
	tm := newTestModel(t)
	// Enter select mode, select the first two rows, open the bulk menu, choose
	// "delete all" and type its confirmation phrase.
	sendKeys(tm,
		keyRunes('X'),                              // select mode
		tea.KeyMsg{Type: tea.KeySpace},             // select main
		keyRunes('j'),                              // move down
		tea.KeyMsg{Type: tea.KeySpace},             // select feat/tui-framework
		keyType(tea.KeyEnter),                      // open bulk menu
		keyType(tea.KeyDown), keyType(tea.KeyDown), // cursor to "delete all"
		keyType(tea.KeyEnter), // select it
	)
	tm.Type("delete all")
	sendKeys(tm, keyType(tea.KeyEnter))
	waitForText(t, tm, "deleted 2 branches")
	finish(t, tm)
}

func TestResizeUpdatesDimensions(t *testing.T) {
	tm := newTestModel(t)
	tm.Send(tea.WindowSizeMsg{Width: 40, Height: 10})
	// Navigate after the resize to exercise the reflowed viewport without panic.
	sendKeys(tm, keyRunes('G'), keyRunes('g'))
	l := finish(t, tm)
	if l.width != 40 || l.height != 10 {
		t.Fatalf("resize: got %dx%d, want 40x10", l.width, l.height)
	}
}

func TestCountPrefixJump(t *testing.T) {
	tm := newTestModel(t)
	// "3j" moves the cursor down three rows in one motion.
	sendKeys(tm, keyRunes('3'), keyRunes('j'))
	l := finish(t, tm)
	if l.cursor != 3 {
		t.Fatalf("cursor after 3j: got %d, want 3", l.cursor)
	}
}

func TestHalfPageJumpClampsToEnd(t *testing.T) {
	tm := newTestModel(t)
	// 'd' is a half-page jump when the view binds no "d" operation; on the short
	// demo list it clamps to the last row.
	sendKeys(tm, keyRunes('d'))
	l := finish(t, tm)
	if l.cursor != len(l.visible)-1 {
		t.Fatalf("cursor after d (half-page): got %d, want %d", l.cursor, len(l.visible)-1)
	}
}

func TestGotoRowJumpsToNumberedRow(t *testing.T) {
	tm := newTestModel(t)
	// "3g" jumps to the row numbered 3 in the gutter (1-indexed) — cursor 2.
	sendKeys(tm, keyRunes('3'), keyRunes('g'))
	l := finish(t, tm)
	if l.cursor != 2 {
		t.Fatalf("cursor after 3g: got %d, want 2", l.cursor)
	}

	// Bare "g" (no count) still goes to the top row, same as before "Ng" existed.
	tm = newTestModel(t)
	sendKeys(tm, keyRunes('G'), keyRunes('g'))
	l = finish(t, tm)
	if l.cursor != 0 {
		t.Fatalf("cursor after G g: got %d, want 0", l.cursor)
	}
}

func TestAltArrowsHalfPageJump(t *testing.T) {
	tm := newTestModel(t)
	// Alt+Down should jump the same as 'd'/ctrl+d: clamps to the last row on
	// the short demo list.
	sendKeys(tm, tea.KeyMsg{Type: tea.KeyDown, Alt: true})
	l := finish(t, tm)
	if l.cursor != len(l.visible)-1 {
		t.Fatalf("cursor after alt+down: got %d, want %d", l.cursor, len(l.visible)-1)
	}

	// Alt+Up from there should bring it back toward the top.
	tm = newTestModel(t)
	sendKeys(tm, tea.KeyMsg{Type: tea.KeyDown, Alt: true}, tea.KeyMsg{Type: tea.KeyUp, Alt: true})
	l = finish(t, tm)
	if l.cursor != 0 {
		t.Fatalf("cursor after alt+down alt+up: got %d, want 0", l.cursor)
	}
}

// headerDemoRow is a minimal selectable Item for the header-skipping test.
type headerDemoRow struct{ name string }

func (r headerDemoRow) Columns() []string   { return []string{r.name} }
func (r headerDemoRow) FilterValue() string { return r.name }
func (r headerDemoRow) Current() bool       { return false }

// newHeaderTestModel builds a list with a leading header and a header mid-list:
// Header("A"), row0, row1, Header("B"), row2.
func newHeaderTestModel(t *testing.T) *teatest.TestModel {
	t.Helper()
	l := New(Config{
		Title:   "headers",
		Columns: []Column{{Title: "name", MinWidth: 8}},
		Items: []Item{
			HeaderItem{Label: "A"},
			headerDemoRow{"row0"},
			headerDemoRow{"row1"},
			HeaderItem{Label: "B"},
			headerDemoRow{"row2"},
		},
	})
	return teatest.NewTestModel(t, l, teatest.WithInitialTermSize(100, 24))
}

func TestHeaderRowsSkippedByNavigation(t *testing.T) {
	// A list opening with a leading header lands the cursor on the first real
	// row, not the header.
	tm := newHeaderTestModel(t)
	l := finish(t, tm)
	if got := l.items[l.visible[l.cursor]]; got != (headerDemoRow{"row0"}) {
		t.Fatalf("initial cursor item = %v, want row0", got)
	}

	// j/k never rest on the "B" header between row1 and row2.
	tm = newHeaderTestModel(t)
	sendKeys(tm, keyRunes('j'), keyRunes('j'))
	l = finish(t, tm)
	if got := l.items[l.visible[l.cursor]]; got != (headerDemoRow{"row2"}) {
		t.Fatalf("cursor after jj = %v, want row2 (should skip the B header)", got)
	}

	// G lands on the last real row, not a trailing header (none here, but
	// exercises the skip-backward path).
	tm = newHeaderTestModel(t)
	sendKeys(tm, keyRunes('G'))
	l = finish(t, tm)
	if got := l.items[l.visible[l.cursor]]; got != (headerDemoRow{"row2"}) {
		t.Fatalf("cursor after G = %v, want row2", got)
	}

	// Ng addresses data rows only: row numbers 1,2,3 map to row0,row1,row2 —
	// headers consume no number.
	tm = newHeaderTestModel(t)
	sendKeys(tm, keyRunes('3'), keyRunes('g'))
	l = finish(t, tm)
	if got := l.items[l.visible[l.cursor]]; got != (headerDemoRow{"row2"}) {
		t.Fatalf("cursor after 3g = %v, want row2", got)
	}
}

func TestHelpOverlayOpensAndCloses(t *testing.T) {
	tm := newTestModel(t)
	sendKeys(tm, keyRunes('?'))
	waitForText(t, tm, "Navigation")
	// Any key dismisses it and returns to list mode.
	sendKeys(tm, keyType(tea.KeyEsc))
	l := finish(t, tm)
	if l.mode != modeList {
		t.Fatalf("help overlay did not close: mode=%v", l.mode)
	}
}

func TestSelectToggleWithX(t *testing.T) {
	tm := newTestModel(t)
	// In select mode "x" toggles the row like space does.
	sendKeys(tm, keyRunes('X'), keyRunes('x'), keyRunes('j'), keyRunes('x'))
	l := finish(t, tm)
	if got := len(l.SelectedItems()); got != 2 {
		t.Fatalf("x-toggle selected %d rows, want 2", got)
	}
}

func TestBatchContinuesPastFailure(t *testing.T) {
	tm := newTestModel(t)
	// Select main (deletes ok) and fix/pagination (fails), run resilient delete.
	sendKeys(tm,
		keyRunes('X'),
		tea.KeyMsg{Type: tea.KeySpace}, // select main
		keyRunes('j'), keyRunes('j'), keyRunes('j'),
		tea.KeyMsg{Type: tea.KeySpace}, // select fix/pagination
		keyType(tea.KeyEnter),          // bulk menu
	)
	tm.Type("resilient")
	sendKeys(tm, keyType(tea.KeyEnter), keyRunes('y')) // pick op, confirm yes/no
	// The failed branch pauses with a continue prompt.
	waitForText(t, tm, "deleted failed: fix/pagination")
	// Continue past it; the run finishes with a summary.
	sendKeys(tm, keyRunes('y'))
	waitForText(t, tm, "deleted 1 · failed 1")
	finish(t, tm)
}

func TestBatchStopOnFailure(t *testing.T) {
	tm := newTestModel(t)
	sendKeys(tm,
		keyRunes('X'),
		keyRunes('j'), keyRunes('j'), keyRunes('j'),
		tea.KeyMsg{Type: tea.KeySpace}, // select fix/pagination (fails first)
		keyType(tea.KeyEnter),
	)
	tm.Type("resilient")
	sendKeys(tm, keyType(tea.KeyEnter), keyRunes('y'))
	waitForText(t, tm, "deleted failed: fix/pagination")
	// 'n' stops the run; the summary reports zero deleted, one failed.
	sendKeys(tm, keyRunes('n'))
	waitForText(t, tm, "deleted 0 · failed 1")
	finish(t, tm)
}

func TestSelectModeTracksSelection(t *testing.T) {
	tm := newTestModel(t)
	sendKeys(tm,
		keyRunes('X'),
		tea.KeyMsg{Type: tea.KeySpace},
		keyRunes('j'),
		tea.KeyMsg{Type: tea.KeySpace},
	)
	l := finish(t, tm)
	if got := len(l.SelectedItems()); got != 2 {
		t.Fatalf("selected %d rows, want 2", got)
	}
}

// TestSettingsOpensViaCommandPalette exercises the `:settings` flow end-to-end:
// `:` opens the palette, typing "settings" narrows it, Enter opens the settings
// overlay (modeSettings), Esc cancels and returns to modeList with the original
// palette restored.
func TestSettingsOpensViaCommandPalette(t *testing.T) {
	// Pin to default/dark so the test isn't sensitive to host OS appearance.
	defer setPalette("default", "dark")
	setPalette("default", "dark")

	tm := newTestModel(t)
	// `:` opens the palette.
	sendKeys(tm, keyRunes(':'))
	// Type "settings" to narrow the fuzzy filter to the builtin entry.
	tm.Type("settings")
	sendKeys(tm, keyType(tea.KeyEnter)) // pick "settings"

	// The settings overlay should now be visible (Title renders as "Settings").
	waitForText(t, tm, "Settings")

	// Esc cancels — back to list mode, ActiveTheme reset to default.
	sendKeys(tm, keyType(tea.KeyEsc))
	waitForText(t, tm, "j/k move") // footer chrome returns once overlay closes
	finish(t, tm)

	if ActiveTheme.Name != "default" {
		t.Errorf("ActiveTheme.Name after esc = %q, want %q", ActiveTheme.Name, "default")
	}
}

// TestSettingsAppliesThemeAndSaves verifies j/k navigation + Enter selects a
// different theme (live preview), and `s` saves to a temp settings.json.
func TestSettingsAppliesThemeAndSaves(t *testing.T) {
	defer setPalette("default", "dark")
	setPalette("default", "dark")
	// Redirect config dir to a temp location for SaveSettings — without this,
	// SaveSettings would write to the user's real ~/.config/gint/.
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	tm := newTestModel(t)
	// Open palette, type "settings", enter.
	sendKeys(tm, keyRunes(':'))
	tm.Type("settings")
	sendKeys(tm, keyType(tea.KeyEnter))
	waitForText(t, tm, "Settings")

	// Move to gruvbox theme row. With the added format rows the layout is:
	// cursor 0=appearance, 1=date, 2=branch, 3=author, 4=default(theme), 5=gruvbox.
	sendKeys(tm, keyRunes('j'), keyRunes('j'), keyRunes('j'), keyRunes('j'), keyRunes('j'))
	sendKeys(tm, keyType(tea.KeyEnter)) // preview-select gruvbox
	// The gruvbox row should now be the active ▸ row in the visible output.
	waitForText(t, tm, "gruvbox")
	if ActiveTheme.Name != "gruvbox" {
		t.Errorf("ActiveTheme.Name after preview = %q, want %q", ActiveTheme.Name, "gruvbox")
	}

	// Save + close — 's' triggers settingsApplied → modeList + status set.
	sendKeys(tm, keyRunes('s'))
	// After saving, the overlay closes and the saved-confirmation status shows.
	waitForText(t, tm, "saved")
	finish(t, tm)

	// Verify settings.json was actually written to the temp config dir.
	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings after save: %v", err)
	}
	if s == nil {
		t.Fatal("LoadSettings after save = nil, want file present")
	}
	if s.Theme != "gruvbox" {
		t.Errorf("persisted Theme = %q, want %q", s.Theme, "gruvbox")
	}
}

// --- ConfirmFrom (phase 15) ------------------------------------------------

func TestConfirmFromFiresWhenNonNil(t *testing.T) {
	l := New(Config{Columns: DemoColumns(), Items: DemoItems()})
	op := Operation{
		Name:        "maybe",
		ConfirmFrom: func(items []Item) *Confirm { return &Confirm{Kind: ConfirmYesNo, Prompt: "sure?"} },
		Run:         func(c OpContext) tea.Cmd { return Status("ran") },
	}
	l.runConfirm(op, l.targetItems(), "")
	if l.mode != modeConfirm {
		t.Fatalf("mode = %v, want modeConfirm when ConfirmFrom returns non-nil", l.mode)
	}
}

func TestConfirmFromNilSkipsPrompt(t *testing.T) {
	l := New(Config{Columns: DemoColumns(), Items: DemoItems()})
	ran := false
	op := Operation{
		Name:        "always",
		ConfirmFrom: func(items []Item) *Confirm { return nil },
		Run:         func(c OpContext) tea.Cmd { ran = true; return nil },
	}
	l.runConfirm(op, l.targetItems(), "")
	if l.mode == modeConfirm {
		t.Fatalf("mode = modeConfirm, want no prompt when ConfirmFrom returns nil")
	}
	if !ran {
		t.Fatalf("Run was not called when ConfirmFrom returns nil")
	}
}

func TestConfirmFromOverridesStaticConfirm(t *testing.T) {
	l := New(Config{Columns: DemoColumns(), Items: DemoItems()})
	ran := false
	op := Operation{
		Name:        "both",
		Confirm:     &Confirm{Kind: ConfirmYesNo, Prompt: "static"},
		ConfirmFrom: func(items []Item) *Confirm { return nil }, // overrides Confirm entirely
		Run:         func(c OpContext) tea.Cmd { ran = true; return nil },
	}
	l.runConfirm(op, l.targetItems(), "")
	if !ran {
		t.Fatalf("ConfirmFrom returning nil should override a static Confirm and run immediately")
	}
}

// --- InitCmd (phase 15) ----------------------------------------------------

func TestInitReturnsNilWithNoInitCmdOrOpenMenu(t *testing.T) {
	l := New(Config{Columns: DemoColumns(), Items: DemoItems()})
	if cmd := l.Init(); cmd != nil {
		t.Errorf("Init() = non-nil, want nil when neither OpenMenuOnStart nor InitCmd is set")
	}
}

func TestInitRunsInitCmdAlone(t *testing.T) {
	l := New(Config{
		Columns: DemoColumns(),
		Items:   DemoItems(),
		InitCmd: func() tea.Msg { return statusMsg("loaded") },
	})
	cmd := l.Init()
	if cmd == nil {
		t.Fatal("Init() = nil, want InitCmd's cmd")
	}
	if msg := cmd(); msg != statusMsg("loaded") {
		t.Errorf("Init() cmd produced %v, want statusMsg(\"loaded\")", msg)
	}
}

func TestInitBatchesInitCmdWithOpenMenuBlink(t *testing.T) {
	l := New(Config{
		Columns:         DemoColumns(),
		Items:           DemoItems(),
		OpenMenuOnStart: true,
		InitCmd:         func() tea.Msg { return statusMsg("from-init") },
	})
	cmd := l.Init()
	if cmd == nil {
		t.Fatal("Init() = nil, want a batched cmd")
	}
	// Direct-menu mode must still open regardless of InitCmd.
	if l.mode != modeMenu {
		t.Errorf("mode = %v, want modeMenu (direct-menu mode unaffected by InitCmd)", l.mode)
	}
}

// TestInitCmdMessageReachesSetItems verifies a message produced by InitCmd —
// the shape the branch/worktree views' background PR fetch uses via
// tui.SetItems(items)() — flows through List.Update like any other itemsMsg.
func TestInitCmdMessageReachesSetItems(t *testing.T) {
	want := []Item{demoRow{name: "from-init"}}
	l := New(Config{
		Columns: DemoColumns(),
		Items:   DemoItems(),
		InitCmd: SetItems(want),
	})
	cmd := l.Init()
	if cmd == nil {
		t.Fatal("Init() = nil, want InitCmd's cmd")
	}
	l.Update(cmd())
	if len(l.items) != 1 || l.items[0].FilterValue() != "from-init" {
		t.Fatalf("items after InitCmd = %v, want [from-init]", l.items)
	}
}
