package tui

import (
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
)

// TableOptions tunes the non-interactive (-I) renderer.
type TableOptions struct {
	// Density selects which columns show, mirroring the interactive view.
	Density Density
	// Header, when true, prints the column titles above the rows.
	Header bool
	// Marker, when true, prefixes the current row with "* " (and others with
	// "  "), matching git's own current-branch marker.
	Marker bool
}

// RenderTable writes items as an aligned plain-text table using the same Column
// definitions as the interactive List, so `-i` and `-I` never drift. This backs
// every command's `-I/--no-interactive` output.
func RenderTable(w io.Writer, columns []Column, items []Item, opts TableOptions) error {
	density := opts.Density

	// Select visible columns and remember their index in the full set.
	type vcol struct {
		col   Column
		index int
	}
	var cols []vcol
	for i, c := range columns {
		if c.visible(density) {
			cols = append(cols, vcol{col: c, index: i})
		}
	}

	widths := make([]int, len(cols))
	for i, vc := range cols {
		widths[i] = runewidth.StringWidth(vc.col.Title)
		for _, it := range items {
			if cw := runewidth.StringWidth(cell(it, vc.index)); cw > widths[i] {
				widths[i] = cw
			}
		}
		if vc.col.MaxWidth > 0 && widths[i] > vc.col.MaxWidth {
			widths[i] = vc.col.MaxWidth
		}
	}

	var b strings.Builder
	if opts.Header {
		if opts.Marker {
			b.WriteString("  ")
		}
		titles := make([]string, len(cols))
		for i, vc := range cols {
			titles[i] = fitCell(vc.col.Title, widths[i])
		}
		b.WriteString(strings.TrimRight(strings.Join(titles, colGap), " "))
		b.WriteByte('\n')
	}

	for _, it := range items {
		if opts.Marker {
			if it.Current() {
				b.WriteString("* ")
			} else {
				b.WriteString("  ")
			}
		}
		cells := make([]string, len(cols))
		for i, vc := range cols {
			cells[i] = fitCell(cell(it, vc.index), widths[i])
		}
		b.WriteString(strings.TrimRight(strings.Join(cells, colGap), " "))
		b.WriteByte('\n')
	}

	_, err := io.WriteString(w, b.String())
	return err
}
