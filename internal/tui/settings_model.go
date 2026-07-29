package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// settingsState is the resolution status of a settingsModel.
type settingsState int

const (
	settingsActive settingsState = iota
	settingsApplied
	settingsCanceled
)

// appearances is the cycle order for the ←/→ toggle and the list of options
// shown on the appearance row. "system" leads because it's the no-config default.
var appearances = []string{"system", "light", "dark"}

// dateFormats, branchFormats, authorFormats are the cycle orders for the
// display-format toggle rows.
var dateFormats = []string{"short", "long", "iso"}

var branchFormats = []string{"full", "short"}

var authorFormats = []string{"short", "initials", "full"}

// settingsModel is the `:settings` overlay: an Appearance toggle
// (System/Light/Dark), Date/Branch/Author format toggle rows, plus a
// scrollable theme list with live color swatches. Changes preview live — every
// cursor move or toggle fires setPalette + refreshes the owning List's *Styles,
// so the list behind the overlay repaints immediately. Esc reverts to the
// pre-overlay state; s saves to disk and applies.
type settingsModel struct {
	styles *Styles // ref to the owning List's styles — regenerated on preview

	appearance     string // current preview appearance ("system"|"light"|"dark")
	activeTheme    string // current preview theme name
	origAppearance string // snapshot for revert on esc
	origTheme      string // snapshot for revert on esc

	dateFormat       string // current preview date format
	branchFormat     string // current preview branch format
	authorFormat     string // current preview author format
	origDateFormat   string // snapshot for revert on esc
	origBranchFormat string
	origAuthorFormat string

	cursor int // 0 = appearance, 1 = date, 2 = branch, 3 = author, 4+ = themes
	state  settingsState

	saveErr error // surfaced via saveFailedMsg if SaveSettings failed
}

// newSettingsModel builds the overlay, snapshotting the current active state
// so Esc can restore it bit-for-bit.
func newSettingsModel(styles *Styles) settingsModel {
	s := settingsModel{
		styles:         styles,
		appearance:     ActiveAppearance,
		activeTheme:    ActiveTheme.Name,
		origAppearance: ActiveAppearance,
		origTheme:      ActiveTheme.Name,

		dateFormat:       activeDateFormat,
		branchFormat:     activeBranchFormat,
		authorFormat:     activeAuthorFormat,
		origDateFormat:   activeDateFormat,
		origBranchFormat: activeBranchFormat,
		origAuthorFormat: activeAuthorFormat,
	}
	if s.appearance == "" {
		s.appearance = "system"
		s.origAppearance = "system"
	}
	if s.activeTheme == "" {
		s.activeTheme = "default"
		s.origTheme = "default"
	}
	s.cursor = 0
	return s
}

// lastCursor returns the maximum valid cursor index.
func (m *settingsModel) lastCursor() int { return 3 + len(themes) }

// Update advances the overlay. Returns nil (no commands).
func (m *settingsModel) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch key.String() {
	case "esc", "ctrl+c":
		m.revert()
		m.state = settingsCanceled
		return nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return nil
	case "down", "j":
		if m.cursor < m.lastCursor() {
			m.cursor++
		}
		return nil
	case "left", "h":
		m.cycleCurrent(-1)
		return nil
	case "right", "l":
		m.cycleCurrent(1)
		return nil
	case "enter":
		if m.cursor == 0 {
			m.cycleCurrent(1)
		} else if m.cursor >= 4 {
			themeIdx := m.cursor - 4
			if themeIdx >= 0 && themeIdx < len(themes) {
				m.activeTheme = themes[themeIdx].Name
				m.preview()
			}
		} else {
			// date/branch/author: enter also cycles
			m.cycleCurrent(1)
		}
		return nil
	case "s":
		m.saveErr = SaveSettings(&Settings{
			Appearance:   m.appearance,
			Theme:        m.activeTheme,
			DateFormat:   m.dateFormat,
			BranchFormat: m.branchFormat,
			AuthorFormat: m.authorFormat,
		})
		m.state = settingsApplied
		return nil
	}
	return nil
}

// cycleCurrent advances the value at the current cursor row by dir (±1).
func (m *settingsModel) cycleCurrent(dir int) {
	var changed bool
	switch m.cursor {
	case 0:
		changed = m.cycleOption(&m.appearance, appearances, dir)
	case 1:
		changed = m.cycleOption(&m.dateFormat, dateFormats, dir)
	case 2:
		changed = m.cycleOption(&m.branchFormat, branchFormats, dir)
	case 3:
		changed = m.cycleOption(&m.authorFormat, authorFormats, dir)
	}
	if changed {
		m.preview()
	}
}

// cycleOption cycles *value through options by dir, returning true if changed.
func (m *settingsModel) cycleOption(value *string, options []string, dir int) bool {
	idx := -1
	for i, o := range options {
		if o == *value {
			idx = i
			break
		}
	}
	if idx < 0 {
		idx = 0
	}
	n := len(options)
	old := *value
	*value = options[(idx+dir+n)%n]
	return *value != old
}

// preview refreshes the global palette + format vars + the owning List's Styles
// so the list behind the overlay repaints instantly.
func (m *settingsModel) preview() {
	setPalette(m.activeTheme, m.appearance)
	activeDateFormat = m.dateFormat
	activeBranchFormat = m.branchFormat
	activeAuthorFormat = m.authorFormat
	if m.styles != nil {
		*m.styles = StylesFromColors()
	}
}

// revert restores the snapshot state taken at construction. Called on Esc.
func (m *settingsModel) revert() {
	setPalette(m.origTheme, m.origAppearance)
	activeDateFormat = m.origDateFormat
	activeBranchFormat = m.origBranchFormat
	activeAuthorFormat = m.origAuthorFormat
	if m.styles != nil {
		*m.styles = StylesFromColors()
	}
}

// settingsNumRows is the number of non-theme rows in the settings view.
const settingsNumRows = 4

// View renders the settings overlay.
func (m settingsModel) View() string {
	chrome := chromeSettings()
	var b strings.Builder

	b.WriteString(m.styles.SettingsTitle.Render(chrome.Title))
	b.WriteByte('\n')
	b.WriteByte('\n')

	// Appearance row.
	b.WriteString(m.styles.SettingsSection.Render(chrome.Appearance))
	resolved := ActiveResolvedAppearance
	if m.appearance == "system" && resolved != "" {
		b.WriteString(m.styles.SettingsHelp.Render("  (resolved: " + resolved + ")"))
	}
	b.WriteByte('\n')
	m.renderToggleRow(&b, m.cursor == 0, m.renderAppearanceRow, m.renderAppearanceRowDim)
	b.WriteByte('\n')
	b.WriteByte('\n')

	// Date format row.
	b.WriteString(m.styles.SettingsSection.Render(chrome.DateFormat))
	b.WriteByte('\n')
	m.renderToggleRow(&b, m.cursor == 1, func() string { return m.renderOptions(dateFormats, m.dateFormat) },
		func() string { return m.renderOptionsDim(dateFormats, m.dateFormat) })
	b.WriteByte('\n')

	// Branch format row.
	b.WriteString(m.styles.SettingsSection.Render(chrome.BranchFormat))
	b.WriteByte('\n')
	m.renderToggleRow(&b, m.cursor == 2, func() string { return m.renderOptions(branchFormats, m.branchFormat) },
		func() string { return m.renderOptionsDim(branchFormats, m.branchFormat) })
	b.WriteByte('\n')

	// Author format row.
	b.WriteString(m.styles.SettingsSection.Render(chrome.AuthorFormat))
	b.WriteByte('\n')
	m.renderToggleRow(&b, m.cursor == 3, func() string { return m.renderOptions(authorFormats, m.authorFormat) },
		func() string { return m.renderOptionsDim(authorFormats, m.authorFormat) })
	b.WriteByte('\n')
	b.WriteByte('\n')

	// Theme rows.
	b.WriteString(m.styles.SettingsSection.Render(chrome.Theme))
	b.WriteByte('\n')
	for i, t := range themes {
		row := t.Name
		if i == m.cursor-settingsNumRows {
			b.WriteString(m.styles.SettingsRowActive.Render(m.formatThemeRow(row, true)))
		} else {
			b.WriteString(m.styles.SettingsRow.Render(m.formatThemeRow(row, false)))
		}
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString(m.styles.SettingsHelp.Render(chrome.Footer))
	return m.styles.Overlay.Render(b.String())
}

// renderToggleRow calls activeRender or dimRender depending on whether this
// row is the cursor target.
func (m settingsModel) renderToggleRow(b *strings.Builder, active bool, activeRender, dimRender func() string) {
	if active {
		b.WriteString(activeRender())
	} else {
		b.WriteString(dimRender())
	}
}

// renderAppearanceRow shows [◉ System] [○ Light] [○ Dark] with the active
// option highlighted.
func (m settingsModel) renderAppearanceRow() string {
	parts := make([]string, len(appearances))
	for i, a := range appearances {
		marker := "○"
		style := m.styles.SettingsOptionOff
		if a == m.appearance {
			marker = "◉"
			style = m.styles.SettingsOptionOn
		}
		parts[i] = style.Render(marker + " " + labelForAppearance(a))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// renderAppearanceRowDim is the appearance row shown when cursor is elsewhere.
func (m settingsModel) renderAppearanceRowDim() string {
	parts := make([]string, len(appearances))
	for i, a := range appearances {
		marker := "○"
		if a == m.appearance {
			marker = "◉"
		}
		parts[i] = m.styles.SettingsOptionOff.Render(marker + " " + labelForAppearance(a))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// renderOptions renders a generic toggle row: [◉ optA] [○ optB] [○ optC] with
// the active option highlighted. Used by date, branch, author rows.
func (m settingsModel) renderOptions(opts []string, current string) string {
	parts := make([]string, len(opts))
	for i, o := range opts {
		marker := "○"
		style := m.styles.SettingsOptionOff
		if o == current {
			marker = "◉"
			style = m.styles.SettingsOptionOn
		}
		parts[i] = style.Render(marker + " " + labelForFormatOption(o))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// renderOptionsDim is the same row but all options de-emphasized.
func (m settingsModel) renderOptionsDim(opts []string, current string) string {
	parts := make([]string, len(opts))
	for i, o := range opts {
		marker := "○"
		if o == current {
			marker = "◉"
		}
		parts[i] = m.styles.SettingsOptionOff.Render(marker + " " + labelForFormatOption(o))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// labelForFormatOption returns a display label for a format option value.
func labelForFormatOption(o string) string {
	switch o {
	case "short":
		return "Short"
	case "long":
		return "Long"
	case "iso":
		return "ISO"
	case "full":
		return "Full"
	case "initials":
		return "Initials"
	}
	return o
}

// formatThemeRow builds one theme row: marker + name + swatches.
func (m settingsModel) formatThemeRow(name string, active bool) string {
	nameW := 0
	for _, th := range themes {
		if len(th.Name) > nameW {
			nameW = len(th.Name)
		}
	}
	marker := "  "
	if active {
		marker = "▸ "
	}
	nameStr := marker + name + strings.Repeat(" ", nameW-len(name)+3)
	pal := PaletteFor(name, m.appearance)
	swatch := func(c lipgloss.Color) string {
		return m.styles.SettingsSwatch.Foreground(c).Render("■")
	}
	s1 := swatch(pal.Accent)
	s2 := swatch(pal.Amber)
	s3 := swatch(pal.Teal)
	return nameStr + s1 + " " + s2 + " " + s3
}

// labelForAppearance turns "system" etc. into the display label.
func labelForAppearance(a string) string {
	switch a {
	case "system":
		return "System"
	case "light":
		return "Light"
	case "dark":
		return "Dark"
	}
	return a
}

// chromeSettings returns the chrome hint strings for the settings overlay.
func chromeSettings() settingsChrome {
	c := chrome()
	return settingsChrome{
		Title:        orStr(c.SettingsTitle, "Settings"),
		Appearance:   orStr(c.SettingsAppearance, "Appearance"),
		DateFormat:   orStr(c.SettingsDateFormat, "Date"),
		BranchFormat: orStr(c.SettingsBranchFormat, "Branch"),
		AuthorFormat: orStr(c.SettingsAuthorFormat, "Author"),
		Theme:        orStr(c.SettingsTheme, "Theme"),
		Footer: orStr(c.SettingsFooter,
			"↑/↓ select · ←/→ toggle · enter select · s save · esc cancel"),
	}
}

// settingsChrome is the hint-string set for the settings overlay.
type settingsChrome struct {
	Title        string
	Appearance   string
	DateFormat   string
	BranchFormat string
	AuthorFormat string
	Theme        string
	Footer       string
}

// orStr returns s when non-empty, else fallback.
func orStr(s, fallback string) string {
	if s != "" {
		return s
	}
	return fallback
}
