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
