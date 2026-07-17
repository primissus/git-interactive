package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles holds every Lip Gloss style the shared views use. DefaultStyles
// returns a ready theme; commands may tweak individual fields.
type Styles struct {
	// List chrome.
	Title         lipgloss.Style
	Header        lipgloss.Style // column header row
	SectionHeader lipgloss.Style // grouped-list section divider (e.g. "Staged")
	Help          lipgloss.Style // footer key hints
	Status        lipgloss.Style // footer status/result line
	Paginator     lipgloss.Style // "page 1/3" indicator

	// Rows.
	Row         lipgloss.Style // ordinary row
	RowSelected lipgloss.Style // cursor row
	RowCurrent  lipgloss.Style // the "current" item (current branch/HEAD)
	Marker      lipgloss.Style // dot marker for the current item
	Checkbox    lipgloss.Style // select-mode marker

	// Overlays (menu, confirm, input).
	Overlay       lipgloss.Style
	MenuItem      lipgloss.Style
	MenuActive    lipgloss.Style
	ConfirmPrompt lipgloss.Style
	ConfirmOption lipgloss.Style
	ConfirmActive lipgloss.Style
	ConfirmPhrase lipgloss.Style
	SearchPrompt  lipgloss.Style
}

// Palette colors — adaptive so they read on light and dark terminals.
var (
	colorAccent  = lipgloss.AdaptiveColor{Light: "#7D56F4", Dark: "#A78BFA"}
	colorCurrent = lipgloss.AdaptiveColor{Light: "#1A7F37", Dark: "#3FB950"}
	colorFaint   = lipgloss.AdaptiveColor{Light: "#6E7781", Dark: "#8B949E"}
	colorDanger  = lipgloss.AdaptiveColor{Light: "#CF222E", Dark: "#F85149"}
	colorInvert  = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#0D1117"}
)

// Column tint colors, exported so command views can hand a hue to Column.Color.
// They reuse the base palette where it fits and add a few distinct accents so a
// wide table (sha · message · date · author · refs) stays scannable.
var (
	ColorName   = colorAccent                                               // branch / primary identifier
	ColorSHA    = lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#E3B341"} // amber
	ColorDate   = colorFaint                                                // de-emphasized
	ColorAuthor = lipgloss.AdaptiveColor{Light: "#0550AE", Dark: "#58A6FF"} // blue
	ColorRef    = colorCurrent                                              // branch/ref decorations
)

// graphLaneColors cycles across a graph's lanes so adjacent branches read as
// distinct strands rather than one monochrome mesh.
var graphLaneColors = []lipgloss.TerminalColor{
	colorAccent,
	lipgloss.AdaptiveColor{Light: "#0550AE", Dark: "#58A6FF"}, // blue
	colorCurrent, // green
	lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#E3B341"}, // amber
	lipgloss.AdaptiveColor{Light: "#CF222E", Dark: "#F85149"}, // red
	lipgloss.AdaptiveColor{Light: "#8250DF", Dark: "#D2A8FF"}, // violet
	lipgloss.AdaptiveColor{Light: "#1B7C83", Dark: "#39C5CF"}, // teal
}

// ColorizeGraphPrefix tints a `git log --graph` glyph run lane by lane: each
// visible column position cycles through graphLaneColors, so vertical strands
// keep a stable hue down the view. It preserves display width (every rune is
// wrapped in its own SGR pair), which the table renderer relies on. Suitable as
// a Column.Render.
func ColorizeGraphPrefix(s string) string {
	var b strings.Builder
	col := 0 // visual column, so a lane keeps one hue straight down the view
	for _, r := range s {
		switch r {
		case '|', '*', '/', '\\', '_', '.', '-', '+', '<', '>':
			style := lipgloss.NewStyle().Foreground(graphLaneColors[col%len(graphLaneColors)])
			b.WriteString(style.Render(string(r)))
		default:
			b.WriteRune(r)
		}
		col++
	}
	return b.String()
}

// DefaultStyles returns the standard gint theme.
func DefaultStyles() Styles {
	return Styles{
		Title:         lipgloss.NewStyle().Bold(true).Foreground(colorAccent),
		Header:        lipgloss.NewStyle().Bold(true).Foreground(colorFaint),
		SectionHeader: lipgloss.NewStyle().Bold(true).Foreground(colorFaint),
		Help:          lipgloss.NewStyle().Foreground(colorFaint),
		Status:        lipgloss.NewStyle().Foreground(colorAccent),
		Paginator:     lipgloss.NewStyle().Foreground(colorFaint),

		Row:         lipgloss.NewStyle(),
		RowSelected: lipgloss.NewStyle().Bold(true).Foreground(colorInvert).Background(colorAccent),
		RowCurrent:  lipgloss.NewStyle().Foreground(colorCurrent),
		Marker:      lipgloss.NewStyle().Foreground(colorCurrent).Bold(true),
		Checkbox:    lipgloss.NewStyle().Foreground(colorAccent).Bold(true),

		Overlay: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 2),
		MenuItem:      lipgloss.NewStyle().Padding(0, 1),
		MenuActive:    lipgloss.NewStyle().Padding(0, 1).Bold(true).Foreground(colorInvert).Background(colorAccent),
		ConfirmPrompt: lipgloss.NewStyle().Bold(true),
		ConfirmOption: lipgloss.NewStyle().Padding(0, 1).Foreground(colorFaint),
		ConfirmActive: lipgloss.NewStyle().Padding(0, 1).Bold(true).Foreground(colorInvert).Background(colorAccent),
		ConfirmPhrase: lipgloss.NewStyle().Bold(true).Foreground(colorDanger),
		SearchPrompt:  lipgloss.NewStyle().Foreground(colorAccent),
	}
}
