package cli

import (
	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

// branchItem adapts a git.Branch to tui.Item. Column order follows the
// git-br.py reference: name, last commit subject, relative date, author.
type branchItem struct {
	b git.Branch
}

func (i branchItem) Columns() []string {
	return []string{i.b.Name, i.b.Subject, i.b.CommitDate, i.b.AuthorName}
}
func (i branchItem) FilterValue() string { return i.b.Name }
func (i branchItem) Current() bool       { return i.b.Head }

func branchColumns() []tui.Column {
	return []tui.Column{
		{Title: "branch", MinWidth: 12, Flex: true, Density: tui.DensityShort},
		{Title: "last commit", MaxWidth: 50, Density: tui.DensityNormal},
		{Title: "date", MinWidth: 10, Density: tui.DensityNormal},
		{Title: "author", MinWidth: 10, Density: tui.DensityFull},
	}
}

func newBranchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "branch [branch-name]",
		Aliases: []string{"br"},
		Short:   "Interactive tabular list of branches",
	}
	attachCommonFlags(cmd)
	cmd.RunE = runBranch
	return cmd
}

// runBranch only wires up the -I smoke-test path for phase 1; the
// interactive view, filters/sort, and operations land in phase 3.
func runBranch(cmd *cobra.Command, args []string) error {
	flags := registeredFlags[cmd]
	if flags.resolveInteractive() {
		return stubRun("branch")(cmd, args)
	}

	runner := git.NewRunner("")
	branches, err := git.ListBranches(cmd.Context(), runner)
	if err != nil {
		return err
	}
	items := make([]tui.Item, len(branches))
	for i, b := range branches {
		items[i] = branchItem{b: b}
	}
	return tui.RenderTable(cmd.OutOrStdout(), branchColumns(), items, tui.TableOptions{
		Density: densityFromFlags(flags),
		Header:  true,
		Marker:  true,
	})
}
