package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

// menuState is the resolution status of a menuModel.
type menuState int

const (
	menuPending menuState = iota
	menuChosen
	menuCanceled
)

// menuModel is the context menu opened with Enter (or by `gint branch <name>`
// with a pre-filled filter). It fuzzy-matches operation names so a user can
// type to disambiguate — e.g. "pu" narrows to pull/push. Its options are the
// operations the command registered for the current context.
type menuModel struct {
	ops     []Operation
	matches []int // indexes into ops, in match order
	cursor  int
	filter  textinput.Model
	styles  *Styles

	state  menuState
	chosen Operation
}

// newMenu builds a menu over ops. initial pre-fills the fuzzy filter (used by
// `gint branch <name>` disambiguation); pass "" for an empty filter.
func newMenu(ops []Operation, initial string, st *Styles) menuModel {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = "filter operations"
	ti.SetValue(initial)
	ti.CursorEnd()
	ti.Focus()

	m := menuModel{ops: ops, filter: ti, styles: st}
	m.recompute()
	return m
}

// names returns the operation names in registration order for fuzzy matching.
func (m menuModel) names() []string {
	out := make([]string, len(m.ops))
	for i, op := range m.ops {
		out[i] = op.Name
	}
	return out
}

// recompute refreshes the match set for the current filter text, keeping the
// cursor in range.
func (m *menuModel) recompute() {
	q := strings.TrimSpace(m.filter.Value())
	m.matches = m.matches[:0]
	if q == "" {
		for i := range m.ops {
			m.matches = append(m.matches, i)
		}
	} else {
		for _, res := range fuzzy.Find(q, m.names()) {
			m.matches = append(m.matches, res.Index)
		}
	}
	if m.cursor >= len(m.matches) {
		m.cursor = max(0, len(m.matches)-1)
	}
}

// Update advances the menu and returns any textinput command. Callers inspect
// state afterwards.
func (m *menuModel) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		return cmd
	}

	switch key.String() {
	case "esc", "ctrl+c":
		m.state = menuCanceled
		return nil
	case "up", "ctrl+p":
		if m.cursor > 0 {
			m.cursor--
		}
		return nil
	case "down", "ctrl+n":
		if m.cursor < len(m.matches)-1 {
			m.cursor++
		}
		return nil
	case "enter":
		if len(m.matches) > 0 {
			m.chosen = m.ops[m.matches[m.cursor]]
			m.state = menuChosen
		}
		return nil
	}

	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(key)
	m.recompute()
	return cmd
}

func (m menuModel) View() string {
	var b strings.Builder
	b.WriteString(m.filter.View())
	b.WriteString("\n\n")

	if len(m.matches) == 0 {
		b.WriteString(m.styles.Help.Render("no matching operation"))
	}
	for i, idx := range m.matches {
		op := m.ops[idx]
		label := op.Name
		if op.Key != "" {
			label += "  (" + op.Key + ")"
		}
		if i == m.cursor {
			b.WriteString(m.styles.MenuActive.Render(label))
		} else {
			b.WriteString(m.styles.MenuItem.Render(label))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("type to filter · ↑/↓ move · enter run · esc cancel"))
	return m.styles.Overlay.Render(b.String())
}
