package tui

import tea "github.com/charmbracelet/bubbletea"

// Scope says what an Operation acts on.
type Scope int

const (
	// ScopeItem operates on the highlighted row; in select mode it operates on
	// the whole selection instead.
	ScopeItem Scope = iota
	// ScopeList operates on the view as a whole and needs no row — e.g. "create
	// branch". List operations still appear in the menu and can bind shortcuts.
	ScopeList
)

// Operation is one action a command exposes: a menu entry, an optional
// shortcut, and the flow that runs it. Commands register operations on a List;
// the framework drives the (optional) input prompt and confirmation before
// calling Run. The same Operation backs both the context menu and its shortcut,
// so a key and the menu always trigger the identical flow.
type Operation struct {
	// Name is the label shown in the context menu and matched by menu fuzzy
	// search, e.g. "checkout", "force delete".
	Name string
	// Key is an optional shortcut. It is matched against a key event's string
	// form (bubbletea's KeyMsg.String()), e.g. "C" for Shift+C, "p" for pull,
	// "shift+x". Empty means menu-only.
	Key string
	// Scope selects the operation's target (row vs. whole view).
	Scope Scope
	// Bulk, when true, offers the operation over a multi-row selection (the
	// select-mode menu) in addition to acting on a single row.
	Bulk bool
	// BulkOnly restricts the operation to select mode: it appears only in the
	// bulk menu, never the single-row menu. Use it for genuinely collective
	// actions (archive, "delete all"). BulkOnly implies Bulk.
	BulkOnly bool
	// Input, when non-nil, prompts for a line of text before confirming/running
	// (branch name, commit message…). The text arrives in OpContext.Input.
	Input *InputSpec
	// Confirm, when non-nil, gates the operation behind a confirmation flow.
	// The chosen value (for multi-choice) arrives in OpContext.Choice.
	Confirm *Confirm
	// Batch, when non-nil, runs the operation over its target items one at a
	// time with per-failure recovery (see BatchSpec), instead of Run. Use it for
	// destructive bulk actions so one failing item does not strand the rest.
	// When both are set Batch wins.
	Batch *BatchSpec
	// Run performs the operation and returns a tea.Cmd (often Status(...) or
	// SetItems(...)). It runs only after any Input and Confirm steps succeed.
	Run func(OpContext) tea.Cmd
}

// OpContext carries everything an Operation.Run needs: its target rows plus any
// text and choice gathered by the preceding input/confirm steps.
type OpContext struct {
	// Items are the operation's targets. For ScopeItem it is the highlighted
	// row, or every selected row in select mode; for ScopeList it is empty.
	Items []Item
	// Input is the submitted text when Operation.Input was set, else "".
	Input string
	// Choice is the selected value when the confirmation was multi-choice, the
	// literal "yes" for a yes/no confirm, or "" otherwise.
	Choice string
}

// InputSpec configures the single-line text prompt shown before an operation.
type InputSpec struct {
	// Prompt is the label shown to the left of the input, e.g. "New branch".
	Prompt string
	// Placeholder is faint helper text shown while the field is empty.
	Placeholder string
	// Initial pre-fills the field (e.g. an existing name for rename).
	Initial string
	// AllowEmpty lets a blank submission through (for genuinely optional inputs
	// like a stash message or lock reason). By default an empty field cannot be
	// submitted, so required inputs need no extra guard.
	AllowEmpty bool
	// Validate, when non-nil, checks the trimmed value on submit. A non-nil
	// error keeps the prompt open and shows the message; nil accepts the value.
	// It is not called for a blank AllowEmpty submission.
	Validate func(string) error
}

// statusMsg sets the List's footer status line. Emit it from an Operation.Run
// with Status to report an outcome.
type statusMsg string

// Status returns a command that sets the list's footer status text.
func Status(format string) tea.Cmd {
	return func() tea.Msg { return statusMsg(format) }
}

// itemsMsg replaces the List's data set. Emit it with SetItems after an
// operation mutates the underlying data (delete, rename, refetch…).
type itemsMsg []Item

// SetItems returns a command that replaces the list's rows, preserving the
// cursor position where possible and clearing any active selection.
func SetItems(items []Item) tea.Cmd {
	return func() tea.Msg { return itemsMsg(items) }
}
