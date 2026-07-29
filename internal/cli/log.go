package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

// logItem adapts a git.Commit to tui.Item. Columns follow the git-lg.py
// reference: sha, message, date, author, local branches, worktree dir.
type logItem struct {
	c      git.Commit
	full   bool
	wtDir  string // "" if no worktree has this commit's branch checked out
	isHead bool
}

func (i logItem) Columns() []string {
	if i.full {
		return []string{i.c.ShortSHA, i.c.Subject, i.c.AbsDate, i.c.AuthorName, strings.Join(i.c.Refs, ", "), i.wtDir}
	}
	return []string{i.c.ShortSHA, i.c.Subject, tui.FormatDate(i.c.CommitUnix, i.c.RelDate), tui.FormatAuthor(i.c.AuthorName), strings.Join(i.c.Refs, ", "), i.wtDir}
}
func (i logItem) FilterValue() string { return i.c.Subject + " " + strings.Join(i.c.Refs, " ") }
func (i logItem) Current() bool       { return i.isHead }

func logColumns(full bool) []tui.Column {
	msgMax := 60
	if full {
		msgMax = 0
	}
	return []tui.Column{
		{Title: "sha", MinWidth: 7, Density: tui.DensityShort, Color: tui.ColorSHA},
		{Title: "message", MinWidth: 12, MaxWidth: msgMax, Flex: true, Density: tui.DensityShort},
		{Title: "date", MinWidth: 7, Density: tui.DensityNormal, Color: tui.ColorDate},
		{Title: "author", MinWidth: 8, Density: tui.DensityNormal, Color: tui.ColorAuthor},
		{Title: "branches", MinWidth: 8, Density: tui.DensityNormal, Color: tui.ColorRef},
		{Title: "worktree", MinWidth: 8, Density: tui.DensityFull, Color: tui.ColorDate},
	}
}

// loadLogItems fetches commit history and adapts it to tui.Items, resolving
// each commit's worktree column from the current worktree list.
func loadLogItems(ctx context.Context, r *git.Runner, full bool) ([]tui.Item, error) {
	commits, err := git.ListCommits(ctx, r)
	if err != nil {
		return nil, err
	}
	return loadCommitItems(ctx, r, commits, full)
}

// loadCommitRangeItems adapts the commits in revRange (e.g. "base..target") to
// tui.Items — the data source for `gint rebase --commits`.
func loadCommitRangeItems(ctx context.Context, r *git.Runner, revRange string, full bool) ([]tui.Item, error) {
	commits, err := git.ListCommitsRange(ctx, r, revRange)
	if err != nil {
		return nil, err
	}
	return loadCommitItems(ctx, r, commits, full)
}

// loadCommitItems adapts commits (newest first) to tui.Items, resolving each
// commit's worktree column from the current worktree list. The row matching
// HEAD gets the current-item marker — in a range view that may be no row.
func loadCommitItems(ctx context.Context, r *git.Runner, commits []git.Commit, full bool) ([]tui.Item, error) {
	branchWT, err := worktreeByBranch(ctx, r)
	if err != nil {
		return nil, err
	}
	headSHA, _ := git.RevParse(ctx, r, "HEAD") // "" on an unborn branch; no row matches

	items := make([]tui.Item, len(commits))
	for i, c := range commits {
		wtDir := ""
		for _, ref := range c.Refs {
			if dir, ok := branchWT[ref]; ok {
				wtDir = dir
				break
			}
		}
		items[i] = logItem{c: c, full: full, wtDir: wtDir, isHead: c.SHA == headSHA}
	}
	return items, nil
}

// worktreeByBranch maps each checked-out branch to its worktree's shortest
// display path.
func worktreeByBranch(ctx context.Context, r *git.Runner) (map[string]string, error) {
	worktrees, err := git.ListWorktrees(ctx, r)
	if err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	out := map[string]string{}
	for _, w := range worktrees {
		if w.Branch != "" {
			out[w.Branch] = shortestPath(w.Path, cwd)
		}
	}
	return out, nil
}

func targetCommit(items []tui.Item) (logItem, bool) {
	if len(items) != 1 {
		return logItem{}, false
	}
	c, ok := items[0].(logItem)
	return c, ok
}

func targetCommits(items []tui.Item) []logItem {
	out := make([]logItem, 0, len(items))
	for _, it := range items {
		if c, ok := it.(logItem); ok {
			out = append(out, c)
		}
	}
	return out
}

func newLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "log",
		Aliases: []string{"lg"},
		Short:   "Interactive commit list",
	}
	attachCommonFlags(cmd)
	cmd.RunE = runLog
	return cmd
}

func runLog(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	runner := git.NewRunner("")
	flags := registeredFlags[cmd]

	items, err := loadLogItems(ctx, runner, flags.full)
	if err != nil {
		return err
	}

	if !flags.resolveInteractive() {
		return tui.RenderTable(cmd.OutOrStdout(), logColumns(flags.full), items, tui.TableOptions{
			Density: densityFromFlags(flags),
			Header:  true,
			Marker:  true,
		})
	}

	list := tui.New(tui.Config{
		Title:      "gint log",
		Columns:    logColumns(flags.full),
		Items:      items,
		Operations: buildLogOperations(ctx, runner, flags.full),
		Density:    densityFromFlags(flags),
		Sort:       flags.sort,
	})
	p := tea.NewProgram(list, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err = p.Run()
	return err
}

// buildLogOperations returns the log view's operation registry: checkout,
// copy sha, and the commit-specific operations (cherry-pick, squash, reset,
// merge-into-current stub). Branch's delete/rename/pull/push don't have a
// sensible meaning for a commit, so they are intentionally not offered here
// — see .context/decisions.md.
func buildLogOperations(ctx context.Context, r *git.Runner, full bool) []tui.Operation {
	refresh := func() tea.Cmd {
		items, err := loadLogItems(ctx, r, full)
		if err != nil {
			return tui.Status(err.Error())
		}
		return tui.SetItems(items)
	}
	refreshWith := func(status string) tea.Cmd {
		return tea.Batch(tui.Status(status), refresh())
	}

	return tui.ApplyKeymap("log", []tui.Operation{
		{
			Name: "checkout", Key: "C", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Checkout this commit? (detaches HEAD)"},
			Run: func(c tui.OpContext) tea.Cmd {
				commit, ok := targetCommit(c.Items)
				if !ok {
					return tui.Status("select a commit first")
				}
				if err := git.CheckoutCommit(ctx, r, commit.c.SHA); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("checked out " + commit.c.ShortSHA)
			},
		},
		{
			Name: "copy sha", Key: "y", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				commit, ok := targetCommit(c.Items)
				if !ok {
					return tui.Status("select a commit first")
				}
				if err := copyToClipboard(commit.c.SHA); err != nil {
					return tui.Status("sha " + commit.c.SHA + " (clipboard unavailable: " + err.Error() + ")")
				}
				return tui.Status("copied sha " + commit.c.ShortSHA)
			},
		},
		{
			Name: "cherry-pick", Key: "c", Scope: tui.ScopeItem, Bulk: true,
			Confirm: &tui.Confirm{
				Kind:   tui.ConfirmChoice,
				Prompt: "Cherry-pick the selected commit(s) onto the current branch?",
				Choices: []tui.Choice{
					{Label: "no", Value: "", Key: "n"},
					{Label: "yes", Value: "yes", Key: "y"},
					{Label: "no-commit", Value: "no-commit", Key: "c"},
				},
			},
			Run: func(c tui.OpContext) tea.Cmd {
				commits := targetCommits(c.Items)
				if len(commits) == 0 {
					return tui.Status("select a commit first")
				}
				shas := make([]string, len(commits))
				for i, cm := range commits {
					shas[len(commits)-1-i] = cm.c.SHA // oldest first, matching selection order reversed from newest-first log
				}
				if err := git.CherryPick(ctx, r, shas, c.Choice == "no-commit"); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("cherry-picked " + plural2(len(commits)))
			},
		},
		{
			Name: "squash", Key: "S", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Squash this commit into its parent?"},
			Run: func(c tui.OpContext) tea.Cmd {
				commit, ok := targetCommit(c.Items)
				if !ok {
					return tui.Status("select a commit first")
				}
				if !commit.isHead {
					return tui.Status("squash currently only supports the HEAD commit (full rebase support arrives in phase 7)")
				}
				if err := git.SquashHead(ctx, r); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("squashed " + commit.c.ShortSHA + " into its parent")
			},
		},
		{
			// No shortcut key: reset has no key in PROMPT.md's log table, and
			// "X" is reserved framework-wide for select mode (tui/list.go), so
			// binding it here would be a dead shortcut. Reachable via the menu.
			Name: "reset", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{
				Kind:   tui.ConfirmChoice,
				Prompt: "Reset the current branch to this commit?",
				Choices: []tui.Choice{
					{Label: "no", Value: "", Key: "n"},
					{Label: "soft", Value: "soft", Key: "s"},
					{Label: "mixed", Value: "mixed", Key: "m"},
					{Label: "hard", Value: "hard", Key: "h", Phrase: "reset hard"},
				},
			},
			Run: func(c tui.OpContext) tea.Cmd {
				commit, ok := targetCommit(c.Items)
				if !ok {
					return tui.Status("select a commit first")
				}
				if err := git.ResetTo(ctx, r, commit.c.SHA, git.ResetMode(c.Choice)); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith(c.Choice + "-reset to " + commit.c.ShortSHA)
			},
		},
		{
			Name: "merge into current", Key: "M", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{Kind: tui.ConfirmChoice, Prompt: "Merge the branch at this commit into the current branch?", Choices: mergeConfirmChoices()},
			Run: func(c tui.OpContext) tea.Cmd {
				commit, ok := targetCommit(c.Items)
				if !ok {
					return tui.Status("select a commit first")
				}
				branch, ok := firstRef(commit.c.Refs)
				if !ok {
					return tui.Status("no local branch points at this commit")
				}
				if _, err := runMergeChoice(ctx, r, branch, c.Choice); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("merged " + branch)
			},
		},
	})
}

// firstRef returns a commit's first local branch ref, if any.
func firstRef(refs []string) (string, bool) {
	if len(refs) == 0 {
		return "", false
	}
	return refs[0], true
}

// plural2 renders a commit count as "1 commit" / "N commits".
func plural2(n int) string {
	if n == 1 {
		return "1 commit"
	}
	return fmt.Sprintf("%d commits", n)
}
