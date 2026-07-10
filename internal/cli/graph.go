package cli

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

// graphItem adapts a git.GraphRow to tui.Item. The graph glyph prefix is its
// own column so it never enters fuzzy search (PLAN.md → graph: "graph glyph
// column is not part of fuzzy search").
type graphItem struct {
	row    git.GraphRow
	full   bool
	isHead bool
}

func (i graphItem) Columns() []string {
	if !i.row.HasCommit {
		return []string{i.row.Prefix, "", "", "", ""}
	}
	author := i.row.Commit.AuthorName
	date := i.row.Commit.RelDate
	if i.full {
		date = i.row.Commit.AbsDate
	} else {
		author = authorInitial(i.row.Commit.AuthorName)
	}
	return []string{i.row.Prefix + i.row.Commit.ShortSHA, i.row.Commit.Subject, date, author, strings.Join(i.row.Commit.Refs, ", ")}
}

// FilterValue excludes the graph glyphs so `/` search matches commit content,
// not ASCII art; connector-only rows (no commit) never match a query.
func (i graphItem) FilterValue() string {
	if !i.row.HasCommit {
		return ""
	}
	return i.row.Commit.Subject + " " + strings.Join(i.row.Commit.Refs, " ")
}
func (i graphItem) Current() bool { return i.isHead }

func graphColumns() []tui.Column {
	return []tui.Column{
		{Title: "graph", MinWidth: 4, Density: tui.DensityShort},
		{Title: "message", MinWidth: 12, MaxWidth: 60, Flex: true, Density: tui.DensityShort},
		{Title: "date", MinWidth: 10, Density: tui.DensityNormal},
		{Title: "author", MinWidth: 8, Density: tui.DensityNormal},
		{Title: "branches", MinWidth: 8, Density: tui.DensityFull},
	}
}

// loadGraphItems fetches the commit graph (all branches, or just HEAD's
// ancestry when notAll is set) and adapts it to tui.Items. simplify limits
// the graph to decorated commits — graph-branch's "each branch's last commit".
func loadGraphItems(ctx context.Context, r *git.Runner, notAll, simplify, full bool) ([]tui.Item, error) {
	rows, err := git.ListCommitGraph(ctx, r, !notAll, simplify)
	if err != nil {
		return nil, err
	}
	items := make([]tui.Item, len(rows))
	headSeen := false
	for i, row := range rows {
		isHead := row.HasCommit && !headSeen
		if isHead {
			headSeen = true
		}
		items[i] = graphItem{row: row, full: full, isHead: isHead}
	}
	return items, nil
}

func newGraphBranchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "graph-branch",
		Aliases: []string{"grb"},
		Short:   "Graph view of branches (last commit only)",
	}
	attachCommonFlags(cmd)
	cmd.Flags().BoolP("not-all", "A", false, "use the current branch as base instead of all branches")
	cmd.RunE = runGraph(true)
	return cmd
}

func newGraphCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "graph",
		Aliases: []string{"gr"},
		Short:   "Graph view of commits",
	}
	attachCommonFlags(cmd)
	cmd.Flags().BoolP("not-all", "A", false, "use the current branch as base instead of all branches")
	cmd.RunE = runGraph(false)
	return cmd
}

// runGraph builds the RunE for graph (simplify=false) and graph-branch
// (simplify=true); the two commands share every other behavior.
func runGraph(simplify bool) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		runner := git.NewRunner("")
		flags := registeredFlags[cmd]
		notAll, _ := cmd.Flags().GetBool("not-all")

		items, err := loadGraphItems(ctx, runner, notAll, simplify, flags.full)
		if err != nil {
			return err
		}

		if !flags.resolveInteractive() {
			return tui.RenderTable(cmd.OutOrStdout(), graphColumns(), items, tui.TableOptions{
				Density: densityFromFlags(flags),
				Header:  true,
			})
		}

		title := "gint graph"
		if simplify {
			title = "gint graph-branch"
		}
		list := tui.New(tui.Config{
			Title:      title,
			Columns:    graphColumns(),
			Items:      items,
			Operations: buildGraphOperations(ctx, runner, notAll, simplify, flags.full),
			Density:    densityFromFlags(flags),
			Sort:       flags.sort,
		})
		p := tea.NewProgram(list, tea.WithAltScreen(), tea.WithContext(ctx))
		_, err = p.Run()
		return err
	}
}

func targetGraphCommit(items []tui.Item) (graphItem, bool) {
	if len(items) != 1 {
		return graphItem{}, false
	}
	g, ok := items[0].(graphItem)
	if !ok || !g.row.HasCommit {
		return graphItem{}, false
	}
	return g, ok
}

// buildGraphOperations mirrors log's operations (checkout, copy sha, merge
// stub), adapted to a graph row's possibly-connector-only selection.
func buildGraphOperations(ctx context.Context, r *git.Runner, notAll, simplify, full bool) []tui.Operation {
	refresh := func() tea.Cmd {
		items, err := loadGraphItems(ctx, r, notAll, simplify, full)
		if err != nil {
			return tui.Status(err.Error())
		}
		return tui.SetItems(items)
	}
	refreshWith := func(status string) tea.Cmd {
		return tea.Batch(tui.Status(status), refresh())
	}

	return []tui.Operation{
		{
			Name: "checkout", Key: "C", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Checkout this commit? (detaches HEAD)"},
			Run: func(c tui.OpContext) tea.Cmd {
				g, ok := targetGraphCommit(c.Items)
				if !ok {
					return tui.Status("select a commit first")
				}
				if err := git.CheckoutCommit(ctx, r, g.row.Commit.SHA); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("checked out " + g.row.Commit.ShortSHA)
			},
		},
		{
			Name: "copy sha", Key: "y", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				g, ok := targetGraphCommit(c.Items)
				if !ok {
					return tui.Status("select a commit first")
				}
				if err := copyToClipboard(g.row.Commit.SHA); err != nil {
					return tui.Status("sha " + g.row.Commit.SHA + " (clipboard unavailable: " + err.Error() + ")")
				}
				return tui.Status("copied sha " + g.row.Commit.ShortSHA)
			},
		},
		{
			// Stubbed until phase 6, same seam as branch/log.
			Name: "merge into current", Key: "M", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Merge the branch at this commit into the current branch?"},
			Run: func(c tui.OpContext) tea.Cmd {
				return tui.Status("merge is not yet implemented (arrives in phase 6)")
			},
		},
	}
}
