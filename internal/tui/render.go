package tui

import (
	"fmt"
	"strings"

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

	var b strings.Builder
	for i := l.top; i < l.top+rows && i < len(l.visible); i++ {
		itemIdx := l.visible[i]
		it := l.items[itemIdx]

		cells := make([]string, len(cols))
		for j, c := range cols {
			cells[j] = fitCell(cell(it, l.columnIndex(c)), widths[j])
		}
		line := l.rowPrefix(itemIdx) + strings.Join(cells, colGap)

		switch {
		case i == l.cursor:
			line = l.styles.RowSelected.Render(line)
		case it.Current():
			line = l.styles.RowCurrent.Render(line)
		default:
			line = l.styles.Row.Render(line)
		}
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
// selection checkbox for the given item index.
func (l *List) rowPrefix(itemIdx int) string {
	marker := "  "
	if l.items[itemIdx].Current() {
		marker = l.styles.Marker.Render("●") + " "
	}
	if l.selectMode {
		box := "[ ] "
		if l.selected[itemIdx] {
			box = l.styles.Checkbox.Render("[x]") + " "
		}
		return marker + box
	}
	return marker
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
	}

	help := "/ search · enter menu · shift+x select · q quit"
	if l.selectMode {
		help = fmt.Sprintf("space toggle · enter bulk ops · esc exit · %d selected", len(l.selected))
	}
	footer := l.styles.Help.Render(help)
	if l.status != "" {
		footer = l.styles.Status.Render(l.status) + "\n" + footer
	}
	return footer
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
