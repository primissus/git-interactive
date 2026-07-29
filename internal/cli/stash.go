package cli

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

// stashItem adapts a git.Stash to tui.Item. Columns: index, message, branch,
// relative date.
type stashItem struct{ s git.Stash }

func (i stashItem) Columns() []string {
	return []string{i.s.Ref, i.s.Message, tui.FormatBranch(i.s.Branch), tui.FormatDate(i.s.Unix, i.s.RelDate)}
}
func (i stashItem) FilterValue() string { return i.s.Message + " " + i.s.Branch }
func (i stashItem) Current() bool       { return false }

func stashColumns() []tui.Column {
	return []tui.Column{
		{Title: "index", MinWidth: 8, Density: tui.DensityShort, Color: tui.ColorSHA},
		{Title: "message", MinWidth: 12, Flex: true, Density: tui.DensityShort},
		{Title: "branch", MinWidth: 8, Density: tui.DensityNormal, Color: tui.ColorName},
		{Title: "date", MinWidth: 7, Density: tui.DensityNormal, Color: tui.ColorDate},
	}
}

func loadStashItems(ctx context.Context, r *git.Runner) ([]tui.Item, error) {
	stashes, err := git.ListStashes(ctx, r)
	if err != nil {
		return nil, err
	}
	items := make([]tui.Item, len(stashes))
	for i, s := range stashes {
		items[i] = stashItem{s: s}
	}
	return items, nil
}

func targetStash(items []tui.Item) (stashItem, bool) {
	if len(items) != 1 {
		return stashItem{}, false
	}
	s, ok := items[0].(stashItem)
	return s, ok
}

func newStashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stash",
		Aliases: []string{"sth"},
		Short:   "Interactive list of stashes",
	}
	attachCommonFlags(cmd)
	cmd.RunE = runStash
	return cmd
}

func runStash(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	runner := git.NewRunner("")
	flags := registeredFlags[cmd]

	items, err := loadStashItems(ctx, runner)
	if err != nil {
		return err
	}

	if !flags.resolveInteractive() {
		return tui.RenderTable(cmd.OutOrStdout(), stashColumns(), items, tui.TableOptions{
			Density: densityFromFlags(flags),
			Header:  true,
		})
	}

	list := tui.New(tui.Config{
		Title:      "gint stash",
		Columns:    stashColumns(),
		Items:      items,
		Operations: buildStashOperations(ctx, runner),
		Density:    densityFromFlags(flags),
		Sort:       flags.sort,
	})
	p := tea.NewProgram(list, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err = p.Run()
	return err
}

// buildStashOperations returns stash's operation registry: apply, pop, drop,
// show diff, create (with a message prompt), clear all.
func buildStashOperations(ctx context.Context, r *git.Runner) []tui.Operation {
	refresh := func() tea.Cmd {
		items, err := loadStashItems(ctx, r)
		if err != nil {
			return tui.Status(err.Error())
		}
		return tui.SetItems(items)
	}
	refreshWith := func(status string) tea.Cmd {
		return tea.Batch(tui.Status(status), refresh())
	}

	return tui.ApplyKeymap("stash", []tui.Operation{
		{
			Name: "apply", Key: "a", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				s, ok := targetStash(c.Items)
				if !ok {
					return tui.Status("select a stash first")
				}
				if err := git.StashApply(ctx, r, s.s.Ref); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("applied " + s.s.Ref)
			},
		},
		{
			Name: "pop", Key: "p", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				s, ok := targetStash(c.Items)
				if !ok {
					return tui.Status("select a stash first")
				}
				if err := git.StashPop(ctx, r, s.s.Ref); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("popped " + s.s.Ref)
			},
		},
		{
			Name: "drop", Key: "D", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Drop this stash?"},
			Run: func(c tui.OpContext) tea.Cmd {
				s, ok := targetStash(c.Items)
				if !ok {
					return tui.Status("select a stash first")
				}
				if err := git.StashDrop(ctx, r, s.s.Ref); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("dropped " + s.s.Ref)
			},
		},
		{
			Name: "diff", Key: "d", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				s, ok := targetStash(c.Items)
				if !ok {
					return tui.Status("select a stash first")
				}
				diff, err := git.StashDiff(ctx, r, s.s.Ref)
				if err != nil {
					return tui.Status(err.Error())
				}
				return showInPager(diff)
			},
		},
		{
			Name: "new", Key: "N", Scope: tui.ScopeList,
			Input: &tui.InputSpec{Prompt: "Stash message (optional)", AllowEmpty: true},
			Run: func(c tui.OpContext) tea.Cmd {
				if err := git.StashPush(ctx, r, c.Input); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("created stash")
			},
		},
		{
			Name: "clear all", Scope: tui.ScopeList,
			Confirm: &tui.Confirm{Kind: tui.ConfirmTyped, Prompt: "Drop every stash?", Phrase: "clear all"},
			Run: func(c tui.OpContext) tea.Cmd {
				if err := git.StashClear(ctx, r); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("cleared all stashes")
			},
		},
	})
}
