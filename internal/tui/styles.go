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

	// Overlays (menu, confirm, input, settings).
	Overlay           lipgloss.Style
	MenuItem          lipgloss.Style
	MenuActive        lipgloss.Style
	ConfirmPrompt     lipgloss.Style
	ConfirmOption     lipgloss.Style
	ConfirmActive     lipgloss.Style
	ConfirmPhrase     lipgloss.Style
	SearchPrompt      lipgloss.Style
	SettingsTitle     lipgloss.Style
	SettingsSection   lipgloss.Style
	SettingsRow       lipgloss.Style
	SettingsRowActive lipgloss.Style
	SettingsOption    lipgloss.Style
	SettingsOptionOn  lipgloss.Style // selected/toggled-on option
	SettingsOptionOff lipgloss.Style // dim option
	SettingsHelp      lipgloss.Style
	SettingsSwatch    lipgloss.Style // style wrapping a single swatch block
}

// Palette colors. Typed as lipgloss.TerminalColor (the interface) so they can
// hold either AdaptiveColor (auto-detecting, the pre-theming default) or a
// frozen lipgloss.Color from the active theme. setPalette in themes.go swaps
// these at startup and on settings changes.
var (
	colorAccent  lipgloss.TerminalColor
	colorCurrent lipgloss.TerminalColor
	colorFaint   lipgloss.TerminalColor
	colorDanger  lipgloss.TerminalColor
	colorInvert  lipgloss.TerminalColor
)

// Column tint colors, exported so command views can hand a hue to Column.Color.
// They reuse the base palette where it fits and add a few distinct accents so a
// wide table (sha · message · date · author · refs) stays scannable. Set by
// setPalette from the active theme; set once at startup so command constructors
// pick them up.
var (
	ColorName   lipgloss.TerminalColor
	ColorSHA    lipgloss.TerminalColor
	ColorDate   lipgloss.TerminalColor
	ColorAuthor lipgloss.TerminalColor
	ColorRef    lipgloss.TerminalColor
)

// graphLaneColors cycles across a graph's lanes so adjacent branches read as
// distinct strands rather than one monochrome mesh. Rebuilt by setPalette.
var graphLaneColors []lipgloss.TerminalColor

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

// StylesFromColors builds the full Styles struct from the active package-level
// palette vars. Called once by DefaultStyles at startup and again by the
// settings overlay whenever the user changes the theme/appearance (live
// preview + revert both route through here).
func StylesFromColors() Styles {
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

		// Settings overlay — uses the same palette but with a couple of extra
		// slots for the appearance-toggle row and theme-list rows.
		SettingsTitle:     lipgloss.NewStyle().Bold(true).Foreground(colorAccent),
		SettingsSection:   lipgloss.NewStyle().Bold(true).Foreground(colorFaint),
		SettingsRow:       lipgloss.NewStyle().Padding(0, 1),
		SettingsRowActive: lipgloss.NewStyle().Padding(0, 1).Bold(true).Foreground(colorInvert).Background(colorAccent),
		SettingsOption:    lipgloss.NewStyle().Padding(0, 1),
		SettingsOptionOn:  lipgloss.NewStyle().Padding(0, 1).Bold(true).Foreground(colorCurrent),
		SettingsOptionOff: lipgloss.NewStyle().Padding(0, 1).Foreground(colorFaint),
		SettingsHelp:      lipgloss.NewStyle().Foreground(colorFaint),
		SettingsSwatch:    lipgloss.NewStyle(),
	}
}

// DefaultStyles returns the standard gint theme, built from the active palette.
// Callers that want to tweak individual fields should copy the returned struct
// rather than call this repeatedly after a theme change.
func DefaultStyles() Styles {
	return StylesFromColors()
}
