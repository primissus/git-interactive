package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFlowInputThenConfirmRuns(t *testing.T) {
	var gotInput, gotChoice string
	f := NewFlow(FlowConfig{
		Title: "test",
		Input: &InputSpec{Prompt: "Message"},
		Confirm: Confirm{
			Kind: ConfirmChoice,
			Choices: []Choice{
				{Label: "no", Value: "", Key: "n"},
				{Label: "yes", Value: "yes", Key: "y"},
			},
		},
		Run: func(input, choice string) (string, error) {
			gotInput, gotChoice = input, choice
			return "done", nil
		},
	})

	typeString(f.updateWrap, "hello")
	f.updateWrap(keyType(tea.KeyEnter)) // submit input
	f.updateWrap(keyRunes('y'))         // choose "yes"

	if gotInput != "hello" || gotChoice != "yes" {
		t.Fatalf("Run got input=%q choice=%q, want hello/yes", gotInput, gotChoice)
	}
	if f.Status() != "done" {
		t.Fatalf("Status() = %q, want %q", f.Status(), "done")
	}
}

func TestFlowEditReturnsToInput(t *testing.T) {
	runs := 0
	f := NewFlow(FlowConfig{
		Title: "test",
		Input: &InputSpec{Prompt: "Message"},
		Confirm: Confirm{
			Kind: ConfirmChoice,
			Choices: []Choice{
				{Label: "edit", Value: "edit", Key: "e"},
				{Label: "yes", Value: "yes", Key: "y"},
			},
		},
		EditValue: "edit",
		Run: func(input, choice string) (string, error) {
			runs++
			return "", nil
		},
	})

	typeString(f.updateWrap, "first")
	f.updateWrap(keyType(tea.KeyEnter))
	f.updateWrap(keyRunes('e')) // edit: back to input

	if f.step != flowInput {
		t.Fatalf("step after edit = %v, want flowInput", f.step)
	}
	if runs != 0 {
		t.Fatalf("Run called %d times before re-submitting, want 0", runs)
	}

	typeString(f.updateWrap, " edited")
	f.updateWrap(keyType(tea.KeyEnter))
	f.updateWrap(keyRunes('y'))

	if runs != 1 {
		t.Fatalf("Run called %d times, want 1", runs)
	}
	if f.message != "first edited" {
		t.Fatalf("message = %q, want %q", f.message, "first edited")
	}
}

// updateWrap adapts Flow.Update's (tea.Model, tea.Cmd) signature to the
// update(tea.Msg) tea.Cmd shape typeString expects.
func (f *Flow) updateWrap(msg tea.Msg) tea.Cmd {
	_, cmd := f.Update(msg)
	return cmd
}
