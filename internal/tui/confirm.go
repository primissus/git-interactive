package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmKind selects one of the three confirmation shapes.
type ConfirmKind int

const (
	// ConfirmYesNo is a plain yes/no prompt. It resolves with choice "yes".
	ConfirmYesNo ConfirmKind = iota
	// ConfirmTyped requires the user to type an exact phrase (e.g. "force",
	// "delete all") before it can be accepted. It resolves with choice "yes".
	ConfirmTyped
	// ConfirmChoice offers several labelled options (e.g. no / soft / mixed /
	// hard). It resolves with the selected Choice.Value.
	ConfirmChoice
)

// Choice is one option in a ConfirmChoice prompt.
type Choice struct {
	// Label is the option text, e.g. "hard".
	Label string
	// Value is returned in OpContext.Choice when this option is chosen. An empty
	// Value marks the decline option (e.g. "no"): choosing it cancels the flow
	// rather than running the operation.
	Value string
	// Key is an optional single key that selects this option directly, e.g. "h".
	Key string
	// Phrase, when set, escalates this option to a typed confirmation: the user
	// must type Phrase exactly before the option is accepted (e.g. choosing
	// "hard" requires typing "reset hard").
	Phrase string
}

// Confirm parameterizes a confirmation flow. One component renders all three
// kinds so no command hand-rolls a prompt.
type Confirm struct {
	Kind ConfirmKind
	// Prompt is the question shown above the options, e.g. "Delete branch foo?".
	Prompt string
	// Phrase is the required text for ConfirmTyped.
	Phrase string
	// Choices are the options for ConfirmChoice.
	Choices []Choice
}

// confirmState is the resolution status of a confirmModel.
type confirmState int

const (
	confirmPending confirmState = iota
	confirmAccepted
	confirmCanceled
)

// confirmModel drives one Confirm to resolution. The owning List routes key
// events to Update, then reads state/choice once it leaves confirmPending.
type confirmModel struct {
	spec    Confirm
	choices []Choice // normalized options (yes/no synthesized for ConfirmYesNo)
	cursor  int
	styles  *Styles

	// typing is active while a typed phrase is required — either a ConfirmTyped
	// prompt or an escalated ConfirmChoice option.
	typing bool
	phrase string // phrase currently required while typing
	input  textinput.Model

	state  confirmState
	choice string // resolved value once accepted
}

func newConfirm(spec Confirm, st *Styles) confirmModel {
	m := confirmModel{spec: spec, styles: st}

	switch spec.Kind {
	case ConfirmYesNo:
		m.choices = []Choice{{Label: "yes", Value: "yes", Key: "y"}, {Label: "no", Value: "", Key: "n"}}
		m.cursor = 1 // default to the safe option
	case ConfirmChoice:
		m.choices = spec.Choices
	case ConfirmTyped:
		m.beginTyping(spec.Phrase, "yes")
	}
	return m
}

// beginTyping switches the model into typed-phrase entry, remembering the value
// to resolve with once the phrase matches.
func (m *confirmModel) beginTyping(phrase, value string) {
	ti := textinput.New()
	ti.Placeholder = phrase
	ti.Prompt = chrome().ConfirmPrompt
	ti.Focus()
	m.typing = true
	m.phrase = phrase
	m.choice = value
	m.input = ti
}

// Update advances the confirmation and returns any textinput command (cursor
// blink). Callers inspect state afterwards.
func (m *confirmModel) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		if m.typing {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return cmd
		}
		return nil
	}

	if m.typing {
		return m.updateTyping(key)
	}
	return m.updateChoosing(key)
}

func (m *confirmModel) updateTyping(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "esc", "ctrl+c":
		m.state = confirmCanceled
		return nil
	case "enter":
		if strings.TrimSpace(m.input.Value()) == m.phrase {
			m.state = confirmAccepted
		}
		return nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(key)
	return cmd
}

func (m *confirmModel) updateChoosing(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "esc", "ctrl+c":
		m.state = confirmCanceled
		return nil
	case "left", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return nil
	case "right", "down":
		if m.cursor < len(m.choices)-1 {
			m.cursor++
		}
		return nil
	case "enter":
		return m.pick(m.choices[m.cursor])
	}
	// Direct key selection (y/n, or a choice's Key). Letter keys are matched
	// here rather than as navigation so a choice can bind h/j/k/l without
	// colliding with vim motions.
	for _, c := range m.choices {
		if c.Key != "" && key.String() == c.Key {
			return m.pick(c)
		}
	}
	return nil
}

// pick resolves the given choice, escalating to typed entry when it carries a
// required phrase.
func (m *confirmModel) pick(c Choice) tea.Cmd {
	if c.Phrase != "" {
		m.beginTyping(c.Phrase, c.Value)
		return textinput.Blink
	}
	// An empty Value is the decline option (yes/no "no", or a choice-menu
	// "no"): cancel rather than accept an empty choice.
	if c.Value == "" {
		m.state = confirmCanceled
		return nil
	}
	m.choice = c.Value
	m.state = confirmAccepted
	return nil
}

func (m confirmModel) View() string {
	var b strings.Builder
	b.WriteString(m.styles.ConfirmPrompt.Render(m.spec.Prompt))
	b.WriteString("\n\n")

	if m.typing {
		b.WriteString("type ")
		b.WriteString(m.styles.ConfirmPhrase.Render(m.phrase))
		b.WriteString(" to confirm:\n")
		b.WriteString(m.input.View())
		b.WriteString("\n\n")
		b.WriteString(m.styles.Help.Render(chrome().ConfirmYesNoFooter))
		return m.styles.Overlay.Render(b.String())
	}

	opts := make([]string, len(m.choices))
	for i, c := range m.choices {
		label := c.Label
		if c.Key != "" {
			label = "[" + c.Key + "] " + label
		}
		if c.Phrase != "" {
			label += " …"
		}
		if i == m.cursor {
			opts[i] = m.styles.ConfirmActive.Render(label)
		} else {
			opts[i] = m.styles.ConfirmOption.Render(label)
		}
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, opts...))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Help.Render(chrome().ConfirmChoiceFooter))
	return m.styles.Overlay.Render(b.String())
}
