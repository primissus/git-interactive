package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// inputState is the resolution status of an inputModel.
type inputState int

const (
	inputPending inputState = iota
	inputSubmitted
	inputCanceled
)

// inputModel is the single-line text prompt used for branch names, commit
// messages, and any operation with an InputSpec. It wraps bubbles/textinput.
type inputModel struct {
	spec   InputSpec
	input  textinput.Model
	styles *Styles
	state  inputState
}

func newInput(spec InputSpec, st *Styles) inputModel {
	ti := textinput.New()
	ti.Placeholder = spec.Placeholder
	ti.Prompt = "› "
	ti.SetValue(spec.Initial)
	ti.CursorEnd()
	ti.Focus()
	return inputModel{spec: spec, input: ti, styles: st}
}

// Update advances the prompt and returns any textinput command. Callers inspect
// state afterwards; an empty submission is rejected (stays pending).
func (m *inputModel) Update(msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c":
			m.state = inputCanceled
			return nil
		case "enter":
			if strings.TrimSpace(m.input.Value()) != "" {
				m.state = inputSubmitted
			}
			return nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}

func (m inputModel) value() string { return strings.TrimSpace(m.input.Value()) }

func (m inputModel) View() string {
	var b strings.Builder
	b.WriteString(m.styles.ConfirmPrompt.Render(m.spec.Prompt))
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	b.WriteString(m.styles.Help.Render("enter submit · esc cancel"))
	return m.styles.Overlay.Render(b.String())
}
