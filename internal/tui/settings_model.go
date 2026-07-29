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

// settingsModel is the `:settings` overlay: an Appearance toggle (System/Light/
// Dark) plus a scrollable theme list with live color swatches. Changes preview
// live — every cursor move or toggle fires setPalette + refreshes the owning
// List's *Styles, so the list behind the overlay repaints immediately. Esc
// reverts to the pre-overlay palette; s saves to disk and applies.
//
// It's a sibling of menuModel/confirmModel: the owning List routes messages to
// Update, reads state, and either reverts or commits.
type settingsModel struct {
	styles *Styles // ref to the owning List's styles — regenerated on preview

	appearance     string // current preview appearance ("system"|"light"|"dark")
	activeTheme    string // current preview theme name
	origAppearance string // snapshot for revert on esc
	origTheme      string // snapshot for revert on esc

	cursor int // 0 = appearance row, 1..len(themes) = theme rows
	state  settingsState

	saveErr error // surfaced via saveFailedMsg if SaveSettings failed
}

// newSettingsModel builds the overlay, snapshotting the current active palette
// so Esc can restore it bit-for-bit. Live preview starts from the current
// selection (no change until the user moves/toggles).
func newSettingsModel(styles *Styles) settingsModel {
	s := settingsModel{
		styles:         styles,
		appearance:     ActiveAppearance,
		activeTheme:    ActiveTheme.Name,
		origAppearance: ActiveAppearance,
		origTheme:      ActiveTheme.Name,
	}
	if s.appearance == "" {
		s.appearance = "system"
		s.origAppearance = "system"
	}
	if s.activeTheme == "" {
		s.activeTheme = "default"
		s.origTheme = "default"
	}
	// Cursor lands on the appearance row so left/right is immediately useful.
	s.cursor = 0
	return s
}

// Update advances the overlay. Returns nil (no commands) — settings has no
// blinking cursor. The owning List reads state after each call.
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
		if m.cursor < len(themes) {
			m.cursor++
		}
		return nil
	case "left", "h":
		if m.cursor == 0 {
			m.cycleAppearance(-1)
			m.preview()
		}
		return nil
	case "right", "l":
		if m.cursor == 0 {
			m.cycleAppearance(1)
			m.preview()
		}
		return nil
	case "enter":
		if m.cursor == 0 {
			// On appearance row, enter cycles too — same as ←/→ for discoverability.
			m.cycleAppearance(1)
			m.preview()
		} else {
			// On a theme row, enter selects it as the active preview theme.
			themeIdx := m.cursor - 1
			if themeIdx >= 0 && themeIdx < len(themes) {
				m.activeTheme = themes[themeIdx].Name
				m.preview()
			}
		}
		return nil
	case "s":
		// Save + apply + close. Failure surfaces via saveErr; the caller
		// shows it as a Status line but still closes the overlay (the live
		// preview is already in effect, just not persisted).
		m.saveErr = SaveSettings(&Settings{
			Appearance: m.appearance,
			Theme:      m.activeTheme,
		})
		m.state = settingsApplied
		return nil
	}
	return nil
}

// cycleAppearance advances the preview appearance by dir (±1) modulo the
// appearances cycle.
func (m *settingsModel) cycleAppearance(dir int) {
	idx := -1
	for i, a := range appearances {
		if a == m.appearance {
			idx = i
			break
		}
	}
	if idx < 0 {
		idx = 0 // unknown → start at "system"
	}
	n := len(appearances)
	m.appearance = appearances[(idx+dir+n)%n]
}

// preview refreshes the global palette + the owning List's Styles so the list
// behind the overlay repaints with the new colors. Called on every appearance
// toggle or theme selection.
func (m *settingsModel) preview() {
	setPalette(m.activeTheme, m.appearance)
	if m.styles != nil {
		*m.styles = StylesFromColors()
	}
}

// revert restores the snapshot palette taken at construction. Called on Esc.
func (m *settingsModel) revert() {
	setPalette(m.origTheme, m.origAppearance)
	if m.styles != nil {
		*m.styles = StylesFromColors()
	}
}

// View renders the settings overlay. It reads from m.styles so the preview uses
// whatever palette was last setPalette()'d, not the snapshot.
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

	if m.cursor == 0 {
		b.WriteString(m.renderAppearanceRow())
	} else {
		b.WriteString(m.renderAppearanceRowDim())
	}
	b.WriteByte('\n')
	b.WriteByte('\n')

	// Theme rows.
	b.WriteString(m.styles.SettingsSection.Render(chrome.Theme))
	b.WriteByte('\n')
	for i, t := range themes {
		row := t.Name
		if i == m.cursor-1 {
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

// renderAppearanceRow shows [◉ System] [○ Light] [○ Dark] with the active
// option highlighted (Cursor + colorCurrent), used when the appearance row
// itself is the cursor target.
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

// renderAppearanceRowDim is the same row but de-emphasized — shown when the
// cursor is on a theme row, so the appearance toggle reads as inactive.
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

// formatThemeRow builds one theme row: the cursor marker + name, padded, plus
// a 3-swatch preview rendered in the theme's resolved-appearance colors.
func (m settingsModel) formatThemeRow(name string, active bool) string {
	// Pad the name to align swatches across all rows.
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

	// Swatch palette uses the *preview* appearance, so toggling appearance
	// recolors every theme row's swatches instantly.
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

// chromeSettings returns the chrome hint strings for the settings overlay. Uses
// the same Chrome override pattern as every other view — fields are added to
// the activeKeymap.Chrome and overridable via keymap.json.
func chromeSettings() settingsChrome {
	c := chrome()
	return settingsChrome{
		Title:      orStr(c.SettingsTitle, "Settings"),
		Appearance: orStr(c.SettingsAppearance, "Appearance"),
		Theme:      orStr(c.SettingsTheme, "Theme"),
		Footer: orStr(c.SettingsFooter,
			"↑/↓ select · ←/→ toggle (on appearance) · enter select · s save · esc cancel"),
	}
}

// settingsChrome is the hint-string set for the settings overlay. Each field
// falls back to a default when the corresponding Chrome field is "" (which it
// isn't by default — defaultChrome() now populates them — but older keymap.json
// files won't have the new fields, so the fallback is defensive).
type settingsChrome struct {
	Title      string
	Appearance string
	Theme      string
	Footer     string
}

// orStr returns s when non-empty, else fallback.
func orStr(s, fallback string) string {
	if s != "" {
		return s
	}
	return fallback
}
