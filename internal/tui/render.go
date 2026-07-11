package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const colGap = "  " // separator between columns

// View implements tea.Model. It draws the list body (title, header, rows) with
// the active overlay or footer underneath, so the list stays visible behind a
// menu/confirm/input prompt.
func (l *List) View() string {
	if l.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(l.styles.Title.Render(l.titleLine()))
	b.WriteByte('\n')

	widths := l.layout()
	b.WriteString(l.headerRow(widths))
	b.WriteByte('\n')
	b.WriteString(l.rows(widths))
	b.WriteString("\n\n")
	b.WriteString(l.footer())
	return b.String()
}

// titleLine is the title plus a row count and any active sort/filter context.
func (l *List) titleLine() string {
	parts := []string{l.title}
	parts = append(parts, fmt.Sprintf("(%d)", len(l.visible)))
	if l.sort != "" {
		parts = append(parts, "sort:"+l.sort)
	}
	return strings.Join(parts, " ")
}

// layout resolves a display width for each visible column: the max of the
// title, MinWidth, and widest cell (capped by MaxWidth), with flex columns
// absorbing any leftover terminal width.
func (l *List) layout() []int {
	cols := l.visibleColumns()
	widths := make([]int, len(cols))
	flex := []int{}

	for i, c := range cols {
		w := max(runewidth.StringWidth(c.Title), c.MinWidth)
		ci := l.columnIndex(c)
		for _, idx := range l.visible {
			if cw := runewidth.StringWidth(cell(l.items[idx], ci)); cw > w {
				w = cw
			}
		}
		if c.MaxWidth > 0 && w > c.MaxWidth {
			w = c.MaxWidth
		}
		widths[i] = w
		if c.Flex {
			flex = append(flex, i)
		}
	}

	if len(flex) > 0 {
		used := l.prefixWidth() + (len(cols)-1)*len(colGap)
		for _, w := range widths {
			used += w
		}
		if leftover := l.width - used; leftover > 0 {
			share := leftover / len(flex)
			for _, i := range flex {
				widths[i] += share
			}
		}
	}
	return widths
}

// columnIndex maps a (possibly density-filtered) column back to its index in
// the full column set, which is the index Item.Columns uses.
func (l *List) columnIndex(target Column) int {
	for i, c := range l.columns {
		if c.Title == target.Title {
			return i
		}
	}
	return 0
}

func (l *List) headerRow(widths []int) string {
	cols := l.visibleColumns()
	cells := make([]string, len(cols))
	for i, c := range cols {
		cells[i] = fitCell(c.Title, widths[i])
	}
	return l.styles.Header.Render(strings.Repeat(" ", l.prefixWidth()) + strings.Join(cells, colGap))
}

func (l *List) rows(widths []int) string {
	if len(l.visible) == 0 {
		return l.styles.Help.Render("  (no matching rows)")
	}
	cols := l.visibleColumns()
	rows := l.pageRows()

	// Pre-build per-column tint styles once for the whole page.
	colStyles := make([]lipgloss.Style, len(cols))
	for j, c := range cols {
		if c.Color != nil {
			colStyles[j] = lipgloss.NewStyle().Foreground(c.Color)
		}
	}

	var b strings.Builder
	for i := l.top; i < l.top+rows && i < len(l.visible); i++ {
		itemIdx := l.visible[i]
		it := l.items[itemIdx]
		highlighted := i == l.cursor
		current := it.Current()
		// A cursor or current row wears one whole-row style, so per-column tints
		// are suppressed to keep the highlight uniform and avoid SGR-reset cuts.
		plainRow := !highlighted && !current

		cells := make([]string, len(cols))
		for j, c := range cols {
			cv := fitCell(cell(it, l.columnIndex(c)), widths[j])
			if plainRow {
				switch {
				case c.Render != nil:
					cv = c.Render(cv)
				case c.Color != nil:
					cv = colStyles[j].Render(cv)
				}
			}
			cells[j] = cv
		}

		rowStyle := l.styles.Row
		switch {
		case highlighted:
			rowStyle = l.styles.RowSelected
		case current:
			rowStyle = l.styles.RowCurrent
		}

		body := strings.Join(cells, colGap)
		if pad := l.width - l.prefixWidth() - runewidth.StringWidth(body); pad > 0 {
			body += strings.Repeat(" ", pad)
		}

		line := l.rowPrefix(i+1, itemIdx, highlighted, rowStyle) + rowStyle.Render(body)
		b.WriteString(line)
		if i < l.top+rows-1 && i < len(l.visible)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// prefixWidth is the fixed width of the row-number, marker, and (in select
// mode) checkbox gutter.
func (l *List) prefixWidth() int {
	w := l.numWidth() + 1 // right-aligned row number + trailing space
	w += 2                // "● " current-item marker
	if l.selectMode {
		w += 4 // "[x] " checkbox
	}
	return w
}

// numWidth is the digit width needed to print every row's 1-indexed position
// — the number "Ng" jumps to — so the gutter stays aligned as the row count
// (or a search filter) changes how many digits the largest number needs.
func (l *List) numWidth() int {
	return len(strconv.Itoa(max(len(l.visible), 1)))
}

// rowPrefix renders the row-number gutter, the current-item dot marker, and,
// in select mode, the selection checkbox for the given row. pos is the row's
// 1-indexed position among l.visible — what "Ng" navigates to. Every glyph is
// rendered with rowStyle's own colors as its base (so the gutter's background
// matches the rest of the row, including on the highlighted/cursor row) with
// its foreground swapped to a distinct accent when active — each cell is a
// self-contained styled segment rather than nested inside a larger Render
// call, so an embedded reset can't cut the row's background short.
func (l *List) rowPrefix(pos, itemIdx int, highlighted bool, rowStyle lipgloss.Style) string {
	var numFg lipgloss.TerminalColor
	if !highlighted {
		// Faint on ordinary/current rows so the number reads as a gutter, not
		// data; on the highlighted row it just inherits rowStyle like the rest
		// of the line, since a dim foreground would fight the row's own bold
		// contrast color there.
		numFg = colorFaint
	}
	prefix := styledCell(rowStyle, numFg, false, fmt.Sprintf("%*d ", l.numWidth(), pos))

	current := l.items[itemIdx].Current()
	dot := "  "
	var dotFg lipgloss.TerminalColor
	if current {
		dot = "● "
		dotFg = colorCurrent
	}
	prefix += styledCell(rowStyle, dotFg, current, dot)

	if !l.selectMode {
		return prefix
	}

	box := "[ ] "
	var boxFg lipgloss.TerminalColor
	selected := l.selected[itemIdx]
	if selected {
		box = "[x] "
		boxFg = colorAccent
	}
	return prefix + styledCell(rowStyle, boxFg, selected, box)
}

// styledCell renders s using rowStyle's own foreground/background/bold as a
// base, swapping in fg as the foreground when non-nil and bolding when bold is
// true. Because the whole cell (including rowStyle's background) is re-stated
// in one Render call, it composes cleanly with adjacent styled segments — no
// gap in background color even though each call emits its own reset.
func styledCell(rowStyle lipgloss.Style, fg lipgloss.TerminalColor, bold bool, s string) string {
	st := rowStyle
	if fg != nil {
		st = st.Foreground(fg)
	}
	if bold {
		st = st.Bold(true)
	}
	return st.Render(s)
}

func (l *List) footer() string {
	switch l.mode {
	case modeSearch:
		return l.search.View()
	case modeMenu:
		return l.menu.View()
	case modeConfirm:
		return l.confirm.View()
	case modeInput:
		return l.input.View()
	case modeBatchPrompt:
		return l.batchPromptView()
	case modeHelp:
		return l.helpView()
	}

	help := "j/k move · u/d ½page · / search · enter menu · X select · ? help · q quit"
	if l.selectMode {
		help = fmt.Sprintf("space/x toggle · enter bulk ops · esc exit · %d selected", len(l.selected))
	}
	footer := l.styles.Help.Render(help)
	if l.status != "" {
		footer = l.styles.Status.Render(l.status) + "\n" + footer
	}
	return footer
}

// helpView renders the "?" overlay: the built-in navigation keys plus this
// view's own operation shortcuts, read live from its registry.
func (l *List) helpView() string {
	nav := [][2]string{
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
	}

	keyW := 0
	for _, r := range nav {
		if w := runewidth.StringWidth(r[0]); w > keyW {
			keyW = w
		}
	}
	for _, op := range l.ops {
		if op.Key != "" && runewidth.StringWidth(op.Key) > keyW {
			keyW = runewidth.StringWidth(op.Key)
		}
	}

	var b strings.Builder
	b.WriteString(l.styles.ConfirmPrompt.Render("Navigation"))
	b.WriteByte('\n')
	for _, r := range nav {
		fmt.Fprintf(&b, "%s  %s\n", l.styles.Status.Render(fitCell(r[0], keyW)), r[1])
	}

	var opLines []string
	for _, op := range l.ops {
		if op.Key == "" {
			continue
		}
		opLines = append(opLines, fmt.Sprintf("%s  %s", l.styles.Status.Render(fitCell(op.Key, keyW)), op.Name))
	}
	if len(opLines) > 0 {
		b.WriteByte('\n')
		b.WriteString(l.styles.ConfirmPrompt.Render("Operations"))
		b.WriteByte('\n')
		b.WriteString(strings.Join(opLines, "\n"))
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString(l.styles.Help.Render("any key closes"))
	return l.styles.Overlay.Render(b.String())
}

// fitCell pads s with spaces to width, or truncates it with an ellipsis when it
// is too long, measuring by display width so wide runes align.
func fitCell(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := runewidth.StringWidth(s)
	if w == width {
		return s
	}
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	if width == 1 {
		return "…"
	}
	return runewidth.Truncate(s, width, "…")
}
