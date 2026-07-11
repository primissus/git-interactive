package tui

import (
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// mode is the List's current interaction mode. Exactly one overlay is active at
// a time; select mode is an orthogonal flag layered on modeList.
type mode int

const (
	modeList mode = iota
	modeSearch
	modeMenu
	modeConfirm
	modeInput
	modeBatchPrompt
	modeHelp
)

// List is gint's shared interactive view: a paginated, navigable, fuzzy-
// searchable table with a context menu, confirmation flows, select mode, and a
// shortcut registry. Command views construct one via New with their columns,
// rows, and operations; the List owns every interaction from PROMPT.md's
// "Shared interaction model".
type List struct {
	title   string
	columns []Column
	items   []Item
	density Density
	sort    string // -S hint; applied by the command, surfaced here for display

	visible []int // item indices after the active search filter, in match order
	cursor  int   // index into visible
	top     int   // first visible row (scroll offset), index into visible
	width   int
	height  int

	ops       []Operation
	shortcuts map[string]Operation // Key -> Operation

	selectMode bool
	selected   map[int]bool // item indices currently selected

	mode     mode
	search   textinput.Model
	menu     menuModel
	confirm  confirmModel
	input    inputModel
	pending  *pending  // operation awaiting input/confirmation
	batch    *batchRun // in-flight resilient bulk operation
	countBuf string    // digits typed as a motion count prefix (e.g. "10" in 10j)

	styles        Styles
	status        string
	openMenu      string // pre-filled fuzzy filter for the on-start item menu
	openMenuStart bool   // open the item menu on Init (direct-menu mode)
	quitting      bool
}

// pending records an operation partway through its input/confirm flow.
type pending struct {
	op    Operation
	items []Item
	input string
}

// Config parameterizes a List. Only Columns and Items are strictly required.
type Config struct {
	Title      string
	Columns    []Column
	Items      []Item
	Operations []Operation
	// Density selects per-column visibility (-F/-s). Defaults to DensityNormal.
	Density Density
	// Sort is the -S hint; the command sorts Items itself and passes the label
	// here for display. The List does not reorder rows.
	Sort string
	// Styles overrides the theme; nil uses DefaultStyles.
	Styles *Styles
	// InitialCursor positions the cursor on this item index on start, instead
	// of the first row. Used e.g. by a create-entry row pinned above the list,
	// so default focus lands on the first real item past it.
	InitialCursor int
	// OpenMenuOnStart, when true, opens the context menu for the current row
	// immediately on Init — the mechanism behind `gint branch <name>` direct-menu
	// mode. OpenMenu pre-fills its fuzzy operation filter (usually "").
	OpenMenuOnStart bool
	OpenMenu        string
}

// New builds a List from cfg.
func New(cfg Config) *List {
	styles := DefaultStyles()
	if cfg.Styles != nil {
		styles = *cfg.Styles
	}

	si := textinput.New()
	si.Prompt = "/ "
	si.Placeholder = "fuzzy search"

	l := &List{
		title:         cfg.Title,
		columns:       cfg.Columns,
		items:         cfg.Items,
		density:       cfg.Density,
		sort:          cfg.Sort,
		ops:           cfg.Operations,
		shortcuts:     map[string]Operation{},
		selected:      map[int]bool{},
		mode:          modeList,
		cursor:        cfg.InitialCursor,
		search:        si,
		styles:        styles,
		openMenu:      cfg.OpenMenu,
		openMenuStart: cfg.OpenMenuOnStart,
		width:         80,
		height:        24,
	}
	for _, op := range cfg.Operations {
		if op.Key != "" {
			l.shortcuts[op.Key] = op
		}
	}
	l.applyFilter()
	return l
}

// Init implements tea.Model.
func (l *List) Init() tea.Cmd {
	if l.openMenuStart && len(l.visible) > 0 {
		l.openItemMenu(l.openMenu)
		return textinput.Blink
	}
	return nil
}

// SelectedItems returns the currently selected rows in list order. Bulk
// operations receive the same slice via OpContext.Items; a future two-item diff
// operation reads it and acts when exactly two rows are selected.
func (l *List) SelectedItems() []Item {
	var out []Item
	for _, idx := range l.visible {
		if l.selected[idx] {
			out = append(out, l.items[idx])
		}
	}
	return out
}

// Update implements tea.Model.
func (l *List) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		l.width, l.height = msg.Width, msg.Height
		l.clampScroll()
		return l, nil
	case statusMsg:
		l.status = string(msg)
		return l, nil
	case itemsMsg:
		l.setItems([]Item(msg))
		return l, nil
	}

	switch l.mode {
	case modeSearch:
		return l, l.updateSearch(msg)
	case modeMenu:
		return l, l.updateMenu(msg)
	case modeConfirm:
		return l, l.updateConfirm(msg)
	case modeInput:
		return l, l.updateInput(msg)
	case modeBatchPrompt:
		return l, l.updateBatchPrompt(msg)
	case modeHelp:
		l.updateHelp(msg)
		return l, nil
	default:
		return l.updateList(msg)
	}
}

func (l *List) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return l, nil
	}
	ks := key.String()

	// A digit starts/extends a motion count prefix (e.g. "10j"). A leading "0"
	// is not a count (reserved for a future line-start motion); it falls through.
	if len(ks) == 1 && ks[0] >= '1' && ks[0] <= '9' || (l.countBuf != "" && ks == "0") {
		l.countBuf += ks
		return l, nil
	}
	n := l.takeCount() // count prefix (default 1), cleared for the coming motion

	switch ks {
	case "q", "ctrl+c":
		l.quitting = true
		return l, tea.Quit
	case "up", "k":
		l.moveCursor(-n)
	case "down", "j":
		l.moveCursor(n)
	case "left", "h", "pgup":
		l.page(-n)
	case "right", "l", "pgdown":
		l.page(n)
	case "ctrl+u", "alt+up":
		l.moveCursor(-n * l.halfPage())
	case "ctrl+d", "alt+down":
		l.moveCursor(n * l.halfPage())
	case "u", "d":
		// u/d are half-page jumps unless the current view binds them to an
		// operation (e.g. unstage/diff), which takes precedence.
		if op, ok := l.shortcuts[ks]; ok {
			return l, l.startOp(op)
		}
		dir := 1
		if ks == "u" {
			dir = -1
		}
		l.moveCursor(dir * n * l.halfPage())
	case "g", "home":
		// Bare "g" goes to row 1 (the top); a count prefix goes to that row
		// number instead — e.g. "12g" jumps to the row numbered 12 in the gutter.
		l.gotoRow(n)
	case "G", "end":
		l.cursor = max(0, len(l.visible)-1)
		l.clampScroll()
	case "/":
		l.mode = modeSearch
		l.search.Focus()
		return l, textinput.Blink
	case "?":
		l.mode = modeHelp
	case "X":
		l.toggleSelectMode()
	case " ":
		l.toggleSelection()
	case "x":
		// In select mode "x" toggles the row (matching the [x] checkbox);
		// otherwise it falls through to any view shortcut bound to "x".
		if l.selectMode {
			l.toggleSelection()
		} else if op, ok := l.shortcuts["x"]; ok {
			return l, l.startOp(op)
		}
	case "enter":
		return l, l.openMenuForContext()
	case "esc":
		if l.selectMode {
			l.exitSelectMode()
		} else {
			l.status = ""
		}
	default:
		if op, ok := l.shortcuts[ks]; ok {
			return l, l.startOp(op)
		}
	}
	return l, nil
}

// takeCount returns the pending motion count (default 1) and clears the buffer.
func (l *List) takeCount() int {
	if l.countBuf == "" {
		return 1
	}
	n, err := strconv.Atoi(l.countBuf)
	l.countBuf = ""
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// halfPage is half the viewport height, the distance ctrl+u / ctrl+d (and the
// unbound u / d) jump.
func (l *List) halfPage() int {
	return max(1, l.pageRows()/2)
}

// updateHelp dismisses the shortcut overlay on any key.
func (l *List) updateHelp(msg tea.Msg) {
	if _, ok := msg.(tea.KeyMsg); ok {
		l.mode = modeList
	}
}

// --- navigation -----------------------------------------------------------

func (l *List) moveCursor(delta int) {
	if len(l.visible) == 0 {
		return
	}
	l.cursor = clamp(l.cursor+delta, 0, len(l.visible)-1)
	l.clampScroll()
}

func (l *List) page(dir int) {
	if len(l.visible) == 0 {
		return
	}
	l.cursor = clamp(l.cursor+dir*l.pageRows(), 0, len(l.visible)-1)
	l.clampScroll()
}

// gotoRow jumps the cursor to the row numbered n (1-indexed, matching the
// gutter drawn by rowPrefix). n=1 — "g" with no count prefix — lands on the
// top row, same as before "Ng" existed.
func (l *List) gotoRow(n int) {
	if len(l.visible) == 0 {
		return
	}
	l.cursor = clamp(n-1, 0, len(l.visible)-1)
	l.clampScroll()
}

// pageRows is how many rows fit in the viewport between the header and footer.
func (l *List) pageRows() int {
	// title(1) + header(1) + blank(1) + footer(1) reserved.
	return max(1, l.height-4)
}

// clampScroll keeps the cursor row within the visible window.
func (l *List) clampScroll() {
	rows := l.pageRows()
	if l.cursor < l.top {
		l.top = l.cursor
	}
	if l.cursor >= l.top+rows {
		l.top = l.cursor - rows + 1
	}
	if l.top < 0 {
		l.top = 0
	}
}

// --- select mode ----------------------------------------------------------

func (l *List) toggleSelectMode() {
	if l.selectMode {
		l.exitSelectMode()
		return
	}
	l.selectMode = true
	l.status = "select mode: space to toggle, enter for bulk operations"
}

func (l *List) exitSelectMode() {
	l.selectMode = false
	l.selected = map[int]bool{}
	l.status = ""
}

func (l *List) toggleSelection() {
	if !l.selectMode || len(l.visible) == 0 {
		return
	}
	idx := l.visible[l.cursor]
	if l.selected[idx] {
		delete(l.selected, idx)
	} else {
		l.selected[idx] = true
	}
}

// --- operation flow -------------------------------------------------------

// openMenuForContext opens the context menu appropriate to the current mode:
// bulk operations when a selection is active, otherwise the current row's
// operations.
func (l *List) openMenuForContext() tea.Cmd {
	if l.selectMode {
		if len(l.selected) == 0 {
			l.status = "no rows selected"
			return nil
		}
		return l.openMenuWith(l.bulkOps())
	}
	if len(l.visible) == 0 {
		return nil
	}
	it := l.items[l.visible[l.cursor]]
	if da, ok := it.(DefaultActioner); ok {
		if op, ok := l.opByName(da.DefaultOp()); ok {
			return l.startOp(op)
		}
	}
	return l.openMenuWith(l.itemOps())
}

// opByName finds a registered operation by Operation.Name.
func (l *List) opByName(name string) (Operation, bool) {
	for _, op := range l.ops {
		if op.Name == name {
			return op, true
		}
	}
	return Operation{}, false
}

func (l *List) openItemMenu(filter string) {
	l.menu = newMenu(l.itemOps(), filter, &l.styles)
	l.mode = modeMenu
}

func (l *List) openMenuWith(ops []Operation) tea.Cmd {
	if len(ops) == 0 {
		l.status = "no operations available"
		return nil
	}
	l.menu = newMenu(ops, "", &l.styles)
	l.mode = modeMenu
	return textinput.Blink
}

// itemOps are the operations offered for a single row: item- and list-scoped
// operations, excluding bulk-only ones.
func (l *List) itemOps() []Operation {
	var out []Operation
	for _, op := range l.ops {
		if op.BulkOnly {
			continue
		}
		if op.Scope == ScopeItem || op.Scope == ScopeList {
			out = append(out, op)
		}
	}
	return out
}

// bulkOps are the operations offered over a multi-row selection: item-scoped
// operations marked Bulk/BulkOnly, plus view-wide (ScopeList) operations, which
// stay applicable regardless of the selection (e.g. "stage all", "clean").
func (l *List) bulkOps() []Operation {
	var out []Operation
	for _, op := range l.ops {
		switch {
		case op.Scope == ScopeList:
			out = append(out, op)
		case op.Scope == ScopeItem && (op.Bulk || op.BulkOnly):
			out = append(out, op)
		}
	}
	return out
}

// targetItems resolves the rows an item-scoped operation acts on: the whole
// selection in select mode, otherwise the current row.
func (l *List) targetItems() []Item {
	if l.selectMode && len(l.selected) > 0 {
		return l.SelectedItems()
	}
	if len(l.visible) == 0 {
		return nil
	}
	return []Item{l.items[l.visible[l.cursor]]}
}

// startOp begins an operation's flow: resolve its targets, then run any input
// step, confirmation step, and finally the operation itself.
func (l *List) startOp(op Operation) tea.Cmd {
	var items []Item
	if op.Scope == ScopeItem {
		items = l.targetItems()
		if len(items) == 0 {
			l.status = "no row to act on"
			return nil
		}
	}
	return l.runInput(op, items)
}

func (l *List) runInput(op Operation, items []Item) tea.Cmd {
	if op.Input != nil {
		l.pending = &pending{op: op, items: items}
		l.input = newInput(*op.Input, &l.styles)
		l.mode = modeInput
		return textinput.Blink
	}
	return l.runConfirm(op, items, "")
}

func (l *List) runConfirm(op Operation, items []Item, input string) tea.Cmd {
	if op.Confirm != nil {
		l.pending = &pending{op: op, items: items, input: input}
		l.confirm = newConfirm(*op.Confirm, &l.styles)
		l.mode = modeConfirm
		return textinput.Blink
	}
	return l.exec(op, items, input, "")
}

// exec runs the operation and returns to the list, clearing any selection that
// a bulk operation consumed.
func (l *List) exec(op Operation, items []Item, input, choice string) tea.Cmd {
	l.mode = modeList
	l.pending = nil
	if op.Batch != nil {
		return l.startBatch(op, items)
	}
	if l.selectMode {
		l.exitSelectMode()
	}
	if op.Run == nil {
		return nil
	}
	return op.Run(OpContext{Items: items, Input: input, Choice: choice})
}

// --- overlay update routing ----------------------------------------------

func (l *List) updateSearch(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		l.search, cmd = l.search.Update(msg)
		return cmd
	}
	switch key.String() {
	case "enter":
		l.mode = modeList
		l.search.Blur()
		return nil
	case "esc", "ctrl+c":
		l.mode = modeList
		l.search.Blur()
		l.search.SetValue("")
		l.applyFilter()
		return nil
	case "up", "ctrl+p":
		l.moveCursor(-1)
		return nil
	case "down", "ctrl+n":
		l.moveCursor(1)
		return nil
	}
	var cmd tea.Cmd
	l.search, cmd = l.search.Update(key)
	l.cursor, l.top = 0, 0 // a changed query re-anchors the cursor to the top
	l.applyFilter()
	return cmd
}

func (l *List) updateMenu(msg tea.Msg) tea.Cmd {
	cmd := l.menu.Update(msg)
	switch l.menu.state {
	case menuChosen:
		l.mode = modeList
		return l.startOp(l.menu.chosen)
	case menuCanceled:
		l.mode = modeList
		return nil
	}
	return cmd
}

func (l *List) updateConfirm(msg tea.Msg) tea.Cmd {
	cmd := l.confirm.Update(msg)
	switch l.confirm.state {
	case confirmAccepted:
		p := l.pending
		return l.exec(p.op, p.items, p.input, l.confirm.choice)
	case confirmCanceled:
		l.mode = modeList
		l.pending = nil
		l.status = "canceled"
		return nil
	}
	return cmd
}

func (l *List) updateInput(msg tea.Msg) tea.Cmd {
	cmd := l.input.Update(msg)
	switch l.input.state {
	case inputSubmitted:
		p := l.pending
		return l.runConfirm(p.op, p.items, l.input.value())
	case inputCanceled:
		l.mode = modeList
		l.pending = nil
		l.status = "canceled"
		return nil
	}
	return cmd
}

// --- data -----------------------------------------------------------------

func (l *List) setItems(items []Item) {
	l.items = items
	l.selected = map[int]bool{}
	l.applyFilter()
}

// applyFilter recomputes the visible index set from the current search query
// and clamps the cursor/scroll into range.
func (l *List) applyFilter() {
	l.visible = filterItems(l.search.Value(), l.items)
	if l.cursor >= len(l.visible) {
		l.cursor = max(0, len(l.visible)-1)
	}
	l.clampScroll()
}

// --- helpers --------------------------------------------------------------

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// visibleColumns returns the columns shown at the current density.
func (l *List) visibleColumns() []Column {
	var out []Column
	for _, c := range l.columns {
		if c.visible(l.density) {
			out = append(out, c)
		}
	}
	return out
}
