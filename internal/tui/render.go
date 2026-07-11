package tui

import (
	"fmt"
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

		line := l.rowPrefix(itemIdx, rowStyle) + rowStyle.Render(body)
		b.WriteString(line)
		if i < l.top+rows-1 && i < len(l.visible)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// prefixWidth is the fixed width of the marker (and select checkbox) gutter.
func (l *List) prefixWidth() int {
	w := 2 // "● " current-item marker
	if l.selectMode {
		w += 4 // "[x] " checkbox
	}
	return w
}

// rowPrefix renders the current-item dot marker and, in select mode, the
// selection checkbox for the given item index. Both glyphs are rendered with
// rowStyle's own colors as their base (so the gutter's background matches the
// rest of the row, including on the highlighted/cursor row) with their
// foreground swapped to a contrasting accent when active — each cell is a
// self-contained styled segment rather than being nested inside a larger
// Render call, so an embedded reset can't cut the row's background short.
func (l *List) rowPrefix(itemIdx int, rowStyle lipgloss.Style) string {
	dot := "  "
	if l.items[itemIdx].Current() {
		dot = "● "
	}
	prefix := styledCell(rowStyle, colorCurrent, l.items[itemIdx].Current(), dot)

	if !l.selectMode {
		return prefix
	}

	box := "[ ] "
	selected := l.selected[itemIdx]
	if selected {
		box = "[x] "
	}
	return prefix + styledCell(rowStyle, colorAccent, selected, box)
}

// styledCell renders s using rowStyle's own foreground/background/bold as a
// base, swapping in accent as the foreground when active is true. Because the
// whole cell (including rowStyle's background) is re-stated in one Render
// call, it composes cleanly with adjacent styled segments — no gap in
// background color even though each call emits its own reset.
func styledCell(rowStyle lipgloss.Style, accent lipgloss.TerminalColor, active bool, s string) string {
	st := rowStyle
	if active {
		st = st.Foreground(accent).Bold(true)
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
		{"u / d", "half-page down / up"},
		{"h / l", "page back / forward"},
		{"g / G", "jump to top / bottom"},
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
