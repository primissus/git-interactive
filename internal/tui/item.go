// Package tui is gint's shared Bubble Tea interaction layer. It implements
// PROMPT.md's "Shared interaction model" once — list navigation, fuzzy search,
// a context menu, confirmation flows, select mode, and a shortcut registry — so
// each command view only has to supply its data (Items) and operations
// (Operations); it never reimplements the interactions.
package tui

// Item is a single row in a list view. Command packages implement it on their
// own row types (a branch, a commit, a stash…).
type Item interface {
	// Columns returns one cell value per Column defined for the list, in order.
	// A view may return fewer cells than there are columns; missing trailing
	// cells render empty.
	Columns() []string
	// FilterValue is the text that fuzzy search (/) and menu disambiguation
	// match against. Usually the row's primary identifier (branch name, sha…).
	FilterValue() string
	// Current reports whether this row is the "current" item — the checked-out
	// branch, HEAD commit, active worktree — which is drawn with a highlight
	// color and a dot marker.
	Current() bool
}

// Column describes one column of a list or tabular view. The same Column set
// drives both the interactive List and the non-interactive RenderTable output
// so the two never drift.
type Column struct {
	// Title is the header label.
	Title string
	// MinWidth is the minimum content width in cells. 0 means "fit the title".
	MinWidth int
	// MaxWidth caps the content width; 0 means uncapped. Values longer than the
	// resolved width are truncated with an ellipsis.
	MaxWidth int
	// Flex, when true, lets this column absorb leftover horizontal space after
	// fixed columns are laid out. Multiple flex columns share the space evenly.
	Flex bool
	// Density is the lowest view density at which this column appears. The
	// default (the DensityNormal zero value) shows the column in the normal and
	// full views but hides it in the short view; DensityShort keeps it always
	// visible; DensityFull restricts it to the full view.
	Density Density
}

// Density selects how much each row shows, driven by the -F/--full and
// -s/--short flags. It gates per-column visibility via Column.Density.
// DensityNormal is the zero value so an unconfigured view renders normally.
type Density int

const (
	// DensityShort is the minimal view (-s): only always-on columns.
	DensityShort Density = -1
	// DensityNormal is the default view.
	DensityNormal Density = 0
	// DensityFull is the full view (-F): every column, including verbose ones.
	DensityFull Density = 1
)

// visible reports whether a column shows at the given view density.
func (c Column) visible(d Density) bool {
	return d >= c.Density
}

// cell returns item's value for column index i, or "" when the item supplies
// fewer cells than there are columns.
func cell(it Item, i int) string {
	cols := it.Columns()
	if i < 0 || i >= len(cols) {
		return ""
	}
	return cols[i]
}
