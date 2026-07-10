package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type flowStep int

const (
	flowInput flowStep = iota
	flowConfirm
)

// FlowConfig configures a Flow.
type FlowConfig struct {
	Title string
	// Input, when non-nil, prompts for a line of text before Confirm (e.g.
	// commit's message). Nil skips straight to Confirm (e.g. merge).
	Input   *InputSpec
	Confirm Confirm
	// EditValue, when a Confirm choice resolves with this Value, returns to
	// the Input step (pre-filled with the previous text) instead of running —
	// commit's "edit" option, which loops back to the message prompt.
	EditValue string
	Styles    *Styles
	// Run executes the flow's outcome once a non-edit choice is confirmed.
	// status is printed by the command after the program exits.
	Run func(input, choice string) (status string, err error)
}

// Flow is a minimal standalone wizard — an optional message input, then a
// multi-choice confirmation — for commands with no list to select from
// (commit, merge). It reuses List's input/confirm components so every
// command shares the same interaction primitives (PROMPT.md's "Shared
// interaction model"), just without row navigation.
type Flow struct {
	cfg     FlowConfig
	step    flowStep
	input   inputModel
	confirm confirmModel
	styles  Styles

	message  string
	status   string
	err      error
	quitting bool
}

// NewFlow builds a Flow from cfg.
func NewFlow(cfg FlowConfig) *Flow {
	styles := DefaultStyles()
	if cfg.Styles != nil {
		styles = *cfg.Styles
	}
	f := &Flow{cfg: cfg, styles: styles}
	if cfg.Input != nil {
		f.input = newInput(*cfg.Input, &f.styles)
		f.step = flowInput
	} else {
		f.confirm = newConfirm(cfg.Confirm, &f.styles)
		f.step = flowConfirm
	}
	return f
}

// Init implements tea.Model.
func (f *Flow) Init() tea.Cmd { return textinput.Blink }

// Update implements tea.Model.
func (f *Flow) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		return f, nil
	}
	switch f.step {
	case flowInput:
		return f.updateInput(msg)
	default:
		return f.updateConfirm(msg)
	}
}

func (f *Flow) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := f.input.Update(msg)
	switch f.input.state {
	case inputSubmitted:
		f.message = f.input.value()
		f.confirm = newConfirm(f.cfg.Confirm, &f.styles)
		f.step = flowConfirm
		return f, textinput.Blink
	case inputCanceled:
		f.quitting = true
		return f, tea.Quit
	}
	return f, cmd
}

func (f *Flow) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := f.confirm.Update(msg)
	switch f.confirm.state {
	case confirmAccepted:
		if f.cfg.EditValue != "" && f.confirm.choice == f.cfg.EditValue {
			spec := InputSpec{}
			if f.cfg.Input != nil {
				spec = *f.cfg.Input
			}
			spec.Initial = f.message
			f.input = newInput(spec, &f.styles)
			f.step = flowInput
			return f, textinput.Blink
		}
		if f.cfg.Run != nil {
			f.status, f.err = f.cfg.Run(f.message, f.confirm.choice)
		}
		f.quitting = true
		return f, tea.Quit
	case confirmCanceled:
		f.status = "canceled"
		f.quitting = true
		return f, tea.Quit
	}
	return f, cmd
}

// View implements tea.Model.
func (f *Flow) View() string {
	if f.quitting {
		return ""
	}
	title := f.styles.Title.Render(f.cfg.Title)
	switch f.step {
	case flowInput:
		return title + "\n\n" + f.input.View()
	default:
		return title + "\n\n" + f.confirm.View()
	}
}

// Status reports the outcome status line once the program exits.
func (f *Flow) Status() string { return f.status }

// Err reports the outcome error, if any, once the program exits.
func (f *Flow) Err() error { return f.err }
