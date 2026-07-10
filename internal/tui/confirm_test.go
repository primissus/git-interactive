package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// keyRunes builds a KeyMsg for a printable key press.
func keyRunes(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

// keyType builds a KeyMsg for a special key (enter, esc, arrows…).
func keyType(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

func typeString(update func(tea.Msg) tea.Cmd, s string) {
	for _, r := range s {
		update(keyRunes(r))
	}
}

func TestConfirmYesNoDefaultsToSafe(t *testing.T) {
	st := DefaultStyles()

	// Enter on the default cursor accepts the safe "no" option → cancel.
	m := newConfirm(Confirm{Kind: ConfirmYesNo, Prompt: "?"}, &st)
	m.Update(keyType(tea.KeyEnter))
	if m.state != confirmCanceled {
		t.Fatalf("yes/no enter on default: got state %v, want canceled", m.state)
	}

	// Pressing 'y' accepts with choice "yes".
	m = newConfirm(Confirm{Kind: ConfirmYesNo, Prompt: "?"}, &st)
	m.Update(keyRunes('y'))
	if m.state != confirmAccepted || m.choice != "yes" {
		t.Fatalf("yes/no 'y': state=%v choice=%q, want accepted/yes", m.state, m.choice)
	}
}

func TestConfirmTypedRequiresExactPhrase(t *testing.T) {
	st := DefaultStyles()
	m := newConfirm(Confirm{Kind: ConfirmTyped, Prompt: "?", Phrase: "force"}, &st)

	// Wrong text: enter must not accept.
	typeString(m.Update, "forc")
	m.Update(keyType(tea.KeyEnter))
	if m.state != confirmPending {
		t.Fatalf("typed confirm accepted a wrong phrase: state=%v", m.state)
	}

	// Complete the phrase and accept.
	typeString(m.Update, "e")
	m.Update(keyType(tea.KeyEnter))
	if m.state != confirmAccepted {
		t.Fatalf("typed confirm rejected the exact phrase: state=%v", m.state)
	}
}

func TestConfirmChoiceEscalatesToTyped(t *testing.T) {
	st := DefaultStyles()
	m := newConfirm(Confirm{
		Kind:   ConfirmChoice,
		Prompt: "?",
		Choices: []Choice{
			{Label: "no", Value: "", Key: "n"},
			{Label: "hard", Value: "hard", Key: "h", Phrase: "reset hard"},
		},
	}, &st)

	// Selecting the escalated option should not accept immediately.
	m.Update(keyRunes('h'))
	if m.state != confirmPending || !m.typing {
		t.Fatalf("escalated choice: state=%v typing=%v, want pending/typing", m.state, m.typing)
	}

	// Typing the required phrase accepts with the choice's value.
	typeString(m.Update, "reset hard")
	m.Update(keyType(tea.KeyEnter))
	if m.state != confirmAccepted || m.choice != "hard" {
		t.Fatalf("escalated choice: state=%v choice=%q, want accepted/hard", m.state, m.choice)
	}
}

func TestConfirmChoiceNoCancels(t *testing.T) {
	st := DefaultStyles()
	m := newConfirm(Confirm{
		Kind:   ConfirmChoice,
		Prompt: "?",
		Choices: []Choice{
			{Label: "no", Value: "", Key: "n"},
			{Label: "soft", Value: "soft", Key: "s"},
		},
	}, &st)

	// Choosing the empty-value "no" option cancels rather than accepting.
	m.Update(keyRunes('n'))
	if m.state != confirmCanceled {
		t.Fatalf("choice 'no': state=%v, want canceled", m.state)
	}
}

func TestInputRejectsEmptySubmit(t *testing.T) {
	st := DefaultStyles()
	m := newInput(InputSpec{Prompt: "Name"}, &st)

	m.Update(keyType(tea.KeyEnter))
	if m.state != inputPending {
		t.Fatalf("empty input accepted: state=%v", m.state)
	}

	typeString(m.Update, "feature")
	m.Update(keyType(tea.KeyEnter))
	if m.state != inputSubmitted || m.value() != "feature" {
		t.Fatalf("input: state=%v value=%q, want submitted/feature", m.state, m.value())
	}
}
