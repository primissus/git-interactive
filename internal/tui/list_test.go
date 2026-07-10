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
