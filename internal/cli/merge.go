package cli

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

// mergeConfirmChoices is shared by the standalone `merge` command and
// branch/log/graph's "merge into current" operation, so every entry point
// offers identical options (PROMPT.md → merge).
func mergeConfirmChoices() []tui.Choice {
	return []tui.Choice{
		{Label: "no", Value: "", Key: "n"},
		{Label: "yes", Value: "yes", Key: "y"},
		{Label: "ff-only", Value: "ff-only", Key: "f"},
		{Label: "no-ff", Value: "no-ff", Key: "o"},
		{Label: "squash", Value: "squash", Key: "s"},
	}
}

// runMergeChoice executes one of mergeConfirmChoices' outcomes, merging
// branch into the current branch.
func runMergeChoice(ctx context.Context, r *git.Runner, branch, choice string) (string, error) {
	var mode git.MergeMode
	switch choice {
	case "yes":
		mode = git.MergeDefault
	case "ff-only":
		mode = git.MergeFFOnly
	case "no-ff":
		mode = git.MergeNoFF
	case "squash":
		mode = git.MergeSquash
	default:
		return "", nil
	}
	if err := git.MergeBranch(ctx, r, branch, mode); err != nil {
		return "", err
	}
	return "merged " + branch, nil
}

func newMergeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "merge [branch]",
		Aliases: []string{"mrg"},
		Short:   "Merge branches",
		Args:    cobra.MaximumNArgs(1),
	}
	attachCommonFlags(cmd)
	cmd.RunE = runMerge
	return cmd
}

func runMerge(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	r := git.NewRunner("")

	if state, err := git.DetectInProgress(ctx, r); err == nil && state != nil && state.Op == git.OpMerge {
		return runMergeInProgressFlow(cmd, r, state)
	}

	if len(args) != 1 {
		return fmt.Errorf("merge requires a branch name")
	}
	branch := args[0]

	flow := tui.NewFlow(tui.FlowConfig{
		Title:   "gint merge " + branch,
		Confirm: tui.Confirm{Kind: tui.ConfirmChoice, Prompt: "Merge " + branch + " into the current branch?", Choices: mergeConfirmChoices()},
		Run: func(_, choice string) (string, error) {
			return runMergeChoice(ctx, r, branch, choice)
		},
	})
	return runFlow(cmd, flow)
}

func runMergeInProgressFlow(cmd *cobra.Command, r *git.Runner, state *git.InProgressState) error {
	ctx := cmd.Context()
	flow := tui.NewFlow(tui.FlowConfig{
		Title: "gint merge (in progress)",
		Confirm: tui.Confirm{
			Kind:   tui.ConfirmChoice,
			Prompt: "A merge is already in progress. Continue or abort?",
			Choices: []tui.Choice{
				{Label: "no", Value: "", Key: "n"},
				{Label: "continue", Value: "continue", Key: "c"},
				{Label: "abort", Value: "abort", Key: "a"},
			},
		},
		Run: func(_, choice string) (string, error) {
			switch choice {
			case "continue":
				if err := state.Continue(ctx, r); err != nil {
					return "", err
				}
				return "continued merge", nil
			case "abort":
				if err := state.Abort(ctx, r); err != nil {
					return "", err
				}
				return "aborted merge", nil
			default:
				return "", nil
			}
		},
	})
	return runFlow(cmd, flow)
}

// runFlow runs a tui.Flow program and prints its outcome status.
func runFlow(cmd *cobra.Command, flow *tui.Flow) error {
	p := tea.NewProgram(flow, tea.WithContext(cmd.Context()))
	m, err := p.Run()
	if err != nil {
		return err
	}
	f := m.(*tui.Flow)
	if f.Err() != nil {
		return f.Err()
	}
	if f.Status() != "" {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), f.Status())
		return err
	}
	return nil
}
