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
	errMsg string // last validation error, shown under the field
}

func newInput(spec InputSpec, st *Styles) inputModel {
	ti := textinput.New()
	ti.Placeholder = spec.Placeholder
	ti.Prompt = chrome().InputPrompt
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
			val := strings.TrimSpace(m.input.Value())
			if val == "" {
				if m.spec.AllowEmpty {
					m.state = inputSubmitted
				}
				return nil
			}
			if m.spec.Validate != nil {
				if err := m.spec.Validate(val); err != nil {
					m.errMsg = err.Error()
					return nil
				}
			}
			m.errMsg = ""
			m.state = inputSubmitted
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
	if m.errMsg != "" {
		b.WriteString(m.styles.Status.Render(m.errMsg))
		b.WriteString("\n\n")
	}
	b.WriteString(m.styles.Help.Render(chrome().InputFooter))
	return m.styles.Overlay.Render(b.String())
}
