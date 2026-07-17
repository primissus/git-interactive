package cli

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

// add reuses status's data source and columns (PLAN.md → add: "reusing the
// status data source") with its own, staging-focused operation set.
func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Staging operations on the working tree",
	}
	attachCommonFlags(cmd)
	cmd.RunE = runAdd
	return cmd
}

func runAdd(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	runner := git.NewRunner("")
	flags := registeredFlags[cmd]

	items, err := loadStatusItems(ctx, runner)
	if err != nil {
		return err
	}

	if !flags.resolveInteractive() {
		return tui.RenderTable(cmd.OutOrStdout(), statusColumns(), items, tui.TableOptions{
			Density: densityFromFlags(flags),
			Header:  true,
		})
	}

	list := tui.New(tui.Config{
		Title:      "gint add",
		Columns:    statusColumns(),
		Items:      items,
		Operations: buildAddOperations(ctx, runner),
		Density:    densityFromFlags(flags),
		Sort:       flags.sort,
	})
	p := tea.NewProgram(list, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err = p.Run()
	return err
}

// buildAddOperations returns add's operation registry: stage, unstage, stage
// all, unstage all, restore file, clean untracked.
func buildAddOperations(ctx context.Context, r *git.Runner) []tui.Operation {
	refresh := func() tea.Cmd {
		items, err := loadStatusItems(ctx, r)
		if err != nil {
			return tui.Status(err.Error())
		}
		return tui.SetItems(items)
	}
	refreshWith := func(status string) tea.Cmd {
		return tea.Batch(tui.Status(status), refresh())
	}

	return tui.ApplyKeymap("add", []tui.Operation{
		{
			Name: "stage", Key: "s", Scope: tui.ScopeItem, Bulk: true,
			Run: func(c tui.OpContext) tea.Cmd {
				items := targetAddItems(c.Items)
				if len(items) == 0 {
					return tui.Status("select a file first")
				}
				for _, it := range items {
					if err := git.StageFile(ctx, r, it.e.Path); err != nil {
						return tui.Status(err.Error())
					}
				}
				return refresh()
			},
		},
		{
			Name: "unstage", Key: "u", Scope: tui.ScopeItem, Bulk: true,
			Run: func(c tui.OpContext) tea.Cmd {
				items := targetAddItems(c.Items)
				if len(items) == 0 {
					return tui.Status("select a file first")
				}
				for _, it := range items {
					if it.conflict {
						continue
					}
					if err := git.UnstageFile(ctx, r, it.e.Path); err != nil {
						return tui.Status(err.Error())
					}
				}
				return refresh()
			},
		},
		{
			Name: "stage all", Key: "S", Scope: tui.ScopeList,
			Run: func(c tui.OpContext) tea.Cmd {
				if err := git.StageAll(ctx, r); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("staged all changes")
			},
		},
		{
			Name: "unstage all", Key: "U", Scope: tui.ScopeList,
			Run: func(c tui.OpContext) tea.Cmd {
				if err := git.UnstageAll(ctx, r); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("unstaged all changes")
			},
		},
		{
			Name: "restore file", Key: "D", Scope: tui.ScopeItem, Bulk: true,
			Confirm: &tui.Confirm{Kind: tui.ConfirmTyped, Prompt: "Restore the selected file(s), discarding changes?", Phrase: "discard"},
			Run: func(c tui.OpContext) tea.Cmd {
				items := targetAddItems(c.Items)
				if len(items) == 0 {
					return tui.Status("select a file first")
				}
				for _, s := range items {
					if err := git.DiscardFile(ctx, r, s.e.Path, s.untracked); err != nil {
						return tui.Status(err.Error())
					}
				}
				if len(items) == 1 {
					return refreshWith("restored " + items[0].e.Path)
				}
				return refreshWith(fmt.Sprintf("restored %d files", len(items)))
			},
		},
		{
			Name: "clean untracked", Scope: tui.ScopeList,
			Confirm: &tui.Confirm{Kind: tui.ConfirmTyped, Prompt: "Delete every untracked file?", Phrase: "clean"},
			Run: func(c tui.OpContext) tea.Cmd {
				if err := git.CleanUntracked(ctx, r); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("cleaned untracked files")
			},
		},
	})
}

func targetAddItems(items []tui.Item) []statusItem {
	out := make([]statusItem, 0, len(items))
	for _, it := range items {
		if s, ok := it.(statusItem); ok {
			out = append(out, s)
		}
	}
	return out
}
