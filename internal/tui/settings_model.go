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

// dateFormats, branchFormats, authorFormats, worktreePathFormats are the cycle
// orders for the display-format toggle rows.
var dateFormats = []string{"short", "long", "iso"}

var branchFormats = []string{"full", "short", "ultra-short"}

var authorFormats = []string{"short", "initials", "full"}

var worktreePathFormats = []string{"shortest", "relative", "absolute"}

// branchColumnTitles / logColumnTitles are the column titles the branch and
// log views' Display sections can hide, in column order. They must match the
// Column.Title values the commands build, since hiding filters by title.
var branchColumnTitles = []string{"branch", "last commit", "date", "author", "worktree"}

var logColumnTitles = []string{"sha", "message", "date", "author", "branches", "worktree"}

// settingsRowKind classifies a selectable settings row.
type settingsRowKind int

const (
	settingsRowCycle  settingsRowKind = iota // ←/→ (and enter) cycle options
	settingsRowToggle                        // enter/space flips a bool
	settingsRowTheme                         // enter selects the theme
)

// settingsRow is one selectable row in the settings overlay. section is a
// chrome key ("Appearance", "Date", "Branch", "Author", "Display",
// "WorktreePath", "Theme", …) labeling a header rendered above the first row
// that introduces it; rows that share a section render under that one header.
type settingsRow struct {
	kind    settingsRowKind
	section string

	// cycle rows (settingsRowCycle):
	options []string
	get     func() string
	set     func(string)
	label   func(string) string
	// appearanceRow marks the appearance cycle row so its "resolved: …" hint
	// can render under the section header (system mode only).
	appearanceRow bool

	// toggle rows (settingsRowToggle):
	toggleTitle string
	hidden      func() bool
	setHidden   func(bool)

	// theme rows (settingsRowTheme):
	themeIndex int
}

// settingsModel is the `:settings` overlay. It is view-aware: the generic
// sections (appearance, date format, themes) appear in every view; the branch
// view adds a Display column-toggle section + a worktree-path format row, and
// the log view adds a Display section + author/branch format rows. Changes
// preview live — every cursor move or toggle re-runs setPalette, refreshes the
// owning List's *Styles, and re-filters its columns, so the list behind the
// overlay repaints immediately. Esc reverts to the pre-overlay state; s saves
// to disk and applies.
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

	worktreePathFormat     string // current preview worktree-path format
	origWorktreePathFormat string // snapshot for revert on esc

	// Hidden-column toggles mutate the active package maps directly; the
	// owning List re-reads them via its HiddenColumns predicate on every
	// render, so toggles preview live. These are the pre-overlay snapshots
	// restored by Esc.
	origBranchHidden map[string]bool
	origLogHidden    map[string]bool

	rows   []settingsRow
	cursor int
	state  settingsState

	saveErr error // surfaced via saveFailedMsg if SaveSettings failed
}

// newSettingsModel builds the overlay for the given view, snapshotting the
// current active state so Esc can restore it bit-for-bit. It returns a pointer:
// the row closures read/write fields through it, so the overlay instance stored
// on the List is exactly the one the closures see.
func newSettingsModel(l *List, view string) *settingsModel {
	s := &settingsModel{
		styles:         &l.styles,
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

		worktreePathFormat:     activeWorktreePathFormat,
		origWorktreePathFormat: activeWorktreePathFormat,

		origBranchHidden: cloneHidden(activeBranchHidden),
		origLogHidden:    cloneHidden(activeLogHidden),

		cursor: 0,
	}
	if s.appearance == "" {
		s.appearance = "system"
		s.origAppearance = "system"
	}
	if s.activeTheme == "" {
		s.activeTheme = "default"
		s.origTheme = "default"
	}
	s.rows = s.buildRows(view)
	return s
}

// buildRows assembles the overlay's row list for a view. The generic rows
// (appearance, date format) lead in every view; branch and log add their own
// Display/format sections between date and the theme list.
func (m *settingsModel) buildRows(view string) []settingsRow {
	var rows []settingsRow

	addCycle := func(section string, opts []string, get func() string, set func(string), label func(string) string) {
		if label == nil {
			label = labelForFormatOption
		}
		rows = append(rows, settingsRow{kind: settingsRowCycle, section: section, options: opts, get: get, set: set, label: label})
	}
	addToggle := func(section, title string, hidden func() bool, setHidden func(bool)) {
		rows = append(rows, settingsRow{kind: settingsRowToggle, section: section, toggleTitle: title, hidden: hidden, setHidden: setHidden})
	}

	appearanceRow := settingsRow{
		kind: settingsRowCycle, section: "Appearance", options: appearances,
		get:   func() string { return m.appearance },
		set:   func(v string) { m.appearance = v },
		label: labelForAppearance, appearanceRow: true,
	}
	rows = append(rows, appearanceRow)
	addCycle("Date", dateFormats, func() string { return m.dateFormat }, func(v string) { m.dateFormat = v }, labelForFormatOption)

	switch view {
	case "branch":
		for _, title := range branchColumnTitles {
			addToggle("Display", title,
				func() bool { return activeBranchHidden[title] },
				func(v bool) { setHiddenVal(activeBranchHidden, title, v) })
		}
		addCycle("WorktreePath", worktreePathFormats,
			func() string { return m.worktreePathFormat },
			func(v string) { m.worktreePathFormat = v }, labelForFormatOption)
	case "log":
		for _, title := range logColumnTitles {
			addToggle("Display", title,
				func() bool { return activeLogHidden[title] },
				func(v bool) { setHiddenVal(activeLogHidden, title, v) })
		}
		addCycle("Author", authorFormats, func() string { return m.authorFormat }, func(v string) { m.authorFormat = v }, labelForFormatOption)
		addCycle("Branch", branchFormats, func() string { return m.branchFormat }, func(v string) { m.branchFormat = v }, labelForFormatOption)
	default:
		addCycle("Branch", branchFormats, func() string { return m.branchFormat }, func(v string) { m.branchFormat = v }, labelForFormatOption)
		addCycle("Author", authorFormats, func() string { return m.authorFormat }, func(v string) { m.authorFormat = v }, labelForFormatOption)
	}

	for i := range themes {
		rows = append(rows, settingsRow{kind: settingsRowTheme, section: "Theme", themeIndex: i})
	}
	return rows
}

// cloneHidden snapshots a hidden-column set so Esc can restore it.
func cloneHidden(set map[string]bool) map[string]bool {
	out := make(map[string]bool, len(set))
	for k, v := range set {
		out[k] = v
	}
	return out
}

// setHiddenVal flips one title in a hidden-column set.
func setHiddenVal(set map[string]bool, title string, hidden bool) {
	if hidden {
		set[title] = true
	} else {
		delete(set, title)
	}
}

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
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
		return nil
	case "left", "h":
		return m.activate(m.cursor, -1)
	case "right", "l":
		return m.activate(m.cursor, 1)
	case "enter", " ":
		return m.activate(m.cursor, 1)
	case "s":
		m.saveErr = SaveSettings(&Settings{
			Appearance:          m.appearance,
			Theme:               m.activeTheme,
			DateFormat:          m.dateFormat,
			BranchFormat:        m.branchFormat,
			AuthorFormat:        m.authorFormat,
			WorktreePathFormat:  m.worktreePathFormat,
			BranchHiddenColumns: hiddenToList(activeBranchHidden),
			LogHiddenColumns:    hiddenToList(activeLogHidden),
		})
		m.state = settingsApplied
		return nil
	}
	return nil
}

// activate drives the row at cursor in direction dir: cycle rows cycle their
// options, toggle rows flip their bool (direction-agnostic), theme rows select
// the theme.
func (m *settingsModel) activate(rowIdx, dir int) tea.Cmd {
	row := m.rows[rowIdx]
	switch row.kind {
	case settingsRowCycle:
		m.cycleRow(row, dir)
	case settingsRowToggle:
		row.setHidden(!row.hidden())
		m.preview()
	case settingsRowTheme:
		m.activeTheme = themes[row.themeIndex].Name
		m.preview()
	}
	return nil
}

// cycleRow advances a cycle row's option by dir, previewing on change.
func (m *settingsModel) cycleRow(row settingsRow, dir int) {
	old := row.get()
	idx := -1
	for i, o := range row.options {
		if o == old {
			idx = i
			break
		}
	}
	if idx < 0 {
		idx = 0
	}
	row.set(row.options[(idx+dir+len(row.options))%len(row.options)])
	if row.get() != old {
		m.preview()
	}
}

// preview refreshes the global palette + format vars and the owning List's
// Styles so the list behind the overlay repaints instantly. Hidden-column
// toggles need no extra hook here: they mutate the active maps, which the
// List's HiddenColumns predicate re-reads on every render.
func (m *settingsModel) preview() {
	setPalette(m.activeTheme, m.appearance)
	activeDateFormat = m.dateFormat
	activeBranchFormat = m.branchFormat
	activeAuthorFormat = m.authorFormat
	activeWorktreePathFormat = m.worktreePathFormat
	if m.styles != nil {
		*m.styles = StylesFromColors()
	}
}

// revert restores the snapshot state taken at construction. Called on Esc.
func (m *settingsModel) revert() {
	m.appearance = m.origAppearance
	m.activeTheme = m.origTheme
	m.dateFormat = m.origDateFormat
	m.branchFormat = m.origBranchFormat
	m.authorFormat = m.origAuthorFormat
	m.worktreePathFormat = m.origWorktreePathFormat

	setPalette(m.origTheme, m.origAppearance)
	activeDateFormat = m.origDateFormat
	activeBranchFormat = m.origBranchFormat
	activeAuthorFormat = m.origAuthorFormat
	activeWorktreePathFormat = m.origWorktreePathFormat
	activeBranchHidden = cloneHidden(m.origBranchHidden)
	activeLogHidden = cloneHidden(m.origLogHidden)
	if m.styles != nil {
		*m.styles = StylesFromColors()
	}
}

// View renders the settings overlay: each section header followed by its rows,
// with the cursor row highlighted.
func (m settingsModel) View() string {
	chrome := chromeSettings()
	sectionLabel := map[string]string{
		"Appearance":   chrome.Appearance,
		"Date":         chrome.DateFormat,
		"Branch":       chrome.BranchFormat,
		"Author":       chrome.AuthorFormat,
		"Display":      chrome.Display,
		"WorktreePath": chrome.WorktreePath,
		"Theme":        chrome.Theme,
	}
	var b strings.Builder

	b.WriteString(m.styles.SettingsTitle.Render(chrome.Title))
	b.WriteByte('\n')
	b.WriteByte('\n')

	section := ""
	for i, row := range m.rows {
		if row.section != "" && row.section != section {
			// Blank line only where the original layout had one: after the
			// appearance section and before the theme list. Format rows and
			// display toggles stay tightly grouped so the overlay fits short
			// terminals.
			if prev := section; prev != "" && (prev == "Appearance" || row.section == "Theme") {
				b.WriteByte('\n')
			}
			b.WriteString(m.styles.SettingsSection.Render(sectionLabel[row.section]))
			b.WriteByte('\n')
			section = row.section
		}
		if row.appearanceRow && m.appearance == "system" && ActiveResolvedAppearance != "" {
			b.WriteString(m.styles.SettingsHelp.Render("  (resolved: " + ActiveResolvedAppearance + ")"))
			b.WriteByte('\n')
		}

		active := i == m.cursor
		switch row.kind {
		case settingsRowCycle:
			if active {
				b.WriteString(m.renderOptions(row))
			} else {
				b.WriteString(m.renderOptionsDim(row))
			}
		case settingsRowToggle:
			b.WriteString(m.renderToggle(row, active))
		case settingsRowTheme:
			if active {
				b.WriteString(m.styles.SettingsRowActive.Render(m.formatThemeRow(row.themeIndex)))
			} else {
				b.WriteString(m.styles.SettingsRow.Render(m.formatThemeRow(row.themeIndex)))
			}
		}
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString(m.styles.SettingsHelp.Render(chrome.Footer))
	return m.styles.Overlay.Render(b.String())
}

// renderOptions renders a cycle row: [◉ optA] [○ optB] [○ optC] with the
// active option highlighted when the row is the cursor target.
func (m settingsModel) renderOptions(row settingsRow) string {
	parts := make([]string, len(row.options))
	for i, o := range row.options {
		marker := "○"
		style := m.styles.SettingsOptionOff
		if o == row.get() {
			marker = "◉"
			style = m.styles.SettingsOptionOn
		}
		parts[i] = style.Render(marker + " " + row.label(o))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// renderOptionsDim is the same row but every option de-emphasized.
func (m settingsModel) renderOptionsDim(row settingsRow) string {
	parts := make([]string, len(row.options))
	for i, o := range row.options {
		marker := "○"
		if o == row.get() {
			marker = "◉"
		}
		parts[i] = m.styles.SettingsOptionOff.Render(marker + " " + row.label(o))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// renderToggle renders a Display toggle row: "[x] <column>" when the column is
// shown, "[ ] <column>" when hidden. The cursor row is highlighted.
func (m settingsModel) renderToggle(row settingsRow, active bool) string {
	marker := "x"
	if row.hidden() {
		marker = " "
	}
	text := "[" + marker + "] " + row.toggleTitle
	if active {
		return m.styles.SettingsRowActive.Render(text)
	}
	return m.styles.SettingsRow.Render(text)
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
	case "ultra-short":
		return "Ultra-short"
	case "shortest":
		return "Shortest"
	case "relative":
		return "Relative"
	case "absolute":
		return "Absolute"
	}
	return o
}

// formatThemeRow builds one theme row: marker + name + swatches.
func (m settingsModel) formatThemeRow(idx int) string {
	name := themes[idx].Name
	nameW := 0
	for _, th := range themes {
		if len(th.Name) > nameW {
			nameW = len(th.Name)
		}
	}
	marker := "  "
	if themes[idx].Name == m.activeTheme {
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
		Display:      orStr(c.SettingsDisplay, "Display"),
		WorktreePath: orStr(c.SettingsWorktreePath, "Worktree path"),
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
	Display      string
	WorktreePath string
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
