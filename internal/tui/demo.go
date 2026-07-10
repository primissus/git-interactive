package tui

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// This file provides a self-contained demo view that exercises every shared
// interaction. It backs the hidden `gint _demo` command (the phase-2 acceptance
// harness) and the teatest suite, so both drive the exact same configuration.

// demoRow is a sample list row standing in for a branch.
type demoRow struct {
	name   string
	commit string
	author string
	date   string
	head   bool
}

func (r demoRow) Columns() []string   { return []string{r.name, r.commit, r.author, r.date} }
func (r demoRow) FilterValue() string { return r.name }
func (r demoRow) Current() bool       { return r.head }

// DemoColumns are the demo view's columns (mirroring the future `branch` view).
func DemoColumns() []Column {
	return []Column{
		{Title: "branch", MinWidth: 12, Flex: true, Density: DensityShort},
		{Title: "last commit", MaxWidth: 40, Density: DensityNormal},
		{Title: "author", MinWidth: 8, Density: DensityFull},
		{Title: "date", MinWidth: 6, Density: DensityNormal},
	}
}

// DemoItems returns a fixed set of sample rows.
func DemoItems() []Item {
	return []Item{
		demoRow{"main", "Init the module layout", "Ada Lovelace", "2 days ago", true},
		demoRow{"feat/tui-framework", "Add shared list model", "Grace Hopper", "1 hour ago", false},
		demoRow{"feat/branch-view", "Wire branch columns", "Alan Turing", "3 hours ago", false},
		demoRow{"fix/pagination", "Clamp scroll offset", "Ada Lovelace", "yesterday", false},
		demoRow{"chore/lint", "Enable golangci-lint", "Grace Hopper", "5 days ago", false},
		demoRow{"experiment/graph", "Sketch commit graph", "Alan Turing", "2 weeks ago", false},
	}
}

// DemoOperations returns operations covering every confirmation shape and
// operation flow: a plain action, a yes/no confirm, a text-input flow, a
// multi-choice with typed escalation, and bulk actions with typed confirmation.
func DemoOperations() []Operation {
	return []Operation{
		{
			Name: "checkout", Key: "C", Scope: ScopeItem,
			Confirm: &Confirm{Kind: ConfirmYesNo, Prompt: "Check out this branch?"},
			Run: func(c OpContext) tea.Cmd {
				return Status("checked out " + c.Items[0].FilterValue())
			},
		},
		{
			Name: "copy sha", Key: "y", Scope: ScopeItem,
			Run: func(c OpContext) tea.Cmd {
				return Status("copied sha of " + c.Items[0].FilterValue())
			},
		},
		{
			Name: "rename", Key: "R", Scope: ScopeItem,
			Input: &InputSpec{Prompt: "New name", Placeholder: "branch name"},
			Run: func(c OpContext) tea.Cmd {
				return Status("renamed " + c.Items[0].FilterValue() + " to " + c.Input)
			},
		},
		{
			Name: "push", Key: "P", Scope: ScopeItem,
			Confirm: &Confirm{Kind: ConfirmYesNo, Prompt: "Push to upstream?"},
			Run: func(c OpContext) tea.Cmd {
				return Status("pushed " + c.Items[0].FilterValue())
			},
		},
		{
			Name: "delete", Key: "D", Scope: ScopeItem,
			Confirm: &Confirm{
				Kind:   ConfirmChoice,
				Prompt: "Delete this branch?",
				Choices: []Choice{
					{Label: "no", Value: "", Key: "n"},
					{Label: "delete", Value: "delete", Key: "d"},
					{Label: "force", Value: "force", Key: "f", Phrase: "force"},
				},
			},
			Run: func(c OpContext) tea.Cmd {
				return Status("deleted " + c.Items[0].FilterValue() + " (" + c.Choice + ")")
			},
		},
		{
			Name: "reset", Scope: ScopeItem,
			Confirm: &Confirm{
				Kind:   ConfirmChoice,
				Prompt: "Reset to this commit?",
				Choices: []Choice{
					{Label: "no", Value: "", Key: "n"},
					{Label: "soft", Value: "soft", Key: "s"},
					{Label: "mixed", Value: "mixed", Key: "m"},
					{Label: "hard", Value: "hard", Key: "h", Phrase: "reset hard"},
				},
			},
			Run: func(c OpContext) tea.Cmd {
				return Status("reset " + c.Choice)
			},
		},
		{
			Name: "new", Key: "N", Scope: ScopeList,
			Input: &InputSpec{Prompt: "New branch", Placeholder: "branch name"},
			Run: func(c OpContext) tea.Cmd {
				return Status("created branch " + c.Input)
			},
		},
		{
			Name: "archive", Scope: ScopeItem, BulkOnly: true,
			Confirm: &Confirm{Kind: ConfirmYesNo, Prompt: "Archive and delete the selected branches?"},
			Run: func(c OpContext) tea.Cmd {
				return Status(plural("archived", len(c.Items)))
			},
		},
		{
			Name: "delete all", Scope: ScopeItem, BulkOnly: true,
			Confirm: &Confirm{Kind: ConfirmTyped, Prompt: "Delete the selected branches?", Phrase: "delete all"},
			Run: func(c OpContext) tea.Cmd {
				return Status(plural("deleted", len(c.Items)))
			},
		},
	}
}

// Demo builds the demo List at the given density.
func Demo(density Density) *List {
	return New(Config{
		Title:      "gint _demo",
		Columns:    DemoColumns(),
		Items:      DemoItems(),
		Operations: DemoOperations(),
		Density:    density,
	})
}

func plural(verb string, n int) string {
	unit := " branch"
	if n != 1 {
		unit = " branches"
	}
	return verb + " " + strconv.Itoa(n) + unit
}
