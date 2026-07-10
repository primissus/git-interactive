package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

// worktreeItem adapts a git.Worktree to tui.Item. Columns: shortest path,
// branch, short commit, relative date (PLAN.md → worktree).
type worktreeItem struct {
	w        git.Worktree
	cwd      string
	relDate  string
	isMainWT bool
}

func (i worktreeItem) Columns() []string {
	branch := i.w.Branch
	if i.w.Detached {
		branch = "(detached)"
	}
	return []string{shortestPath(i.w.Path, i.cwd), branch, shortSHA(i.w.Head), i.relDate}
}
func (i worktreeItem) FilterValue() string { return i.w.Path + " " + i.w.Branch }
func (i worktreeItem) Current() bool       { return i.isMainWT }

func worktreeColumns() []tui.Column {
	return []tui.Column{
		{Title: "path", MinWidth: 12, Flex: true, Density: tui.DensityShort},
		{Title: "branch", MinWidth: 10, Density: tui.DensityShort},
		{Title: "commit", MinWidth: 7, Density: tui.DensityNormal},
		{Title: "date", MinWidth: 10, Density: tui.DensityNormal},
	}
}

// shortestPath renders path relative to cwd, or "~"-abbreviated from the home
// directory when that is shorter than the relative form (PLAN.md → worktree
// "shortest path (relative, or ~-absolute)").
func shortestPath(path, cwd string) string {
	if rel, err := filepath.Rel(cwd, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	if home, err := os.UserHomeDir(); err == nil {
		if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.Join("~", rel)
		}
	}
	return path
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// loadWorktreeItems fetches worktrees and adapts them to tui.Items, resolving
// each row's commit relative-date with a lightweight extra call (worktree
// counts are small, so a per-row call is simpler than batching).
func loadWorktreeItems(ctx context.Context, r *git.Runner) ([]tui.Item, error) {
	worktrees, err := git.ListWorktrees(ctx, r)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(worktrees, func(i, j int) bool { return worktrees[i].Path < worktrees[j].Path })

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	items := make([]tui.Item, 0, len(worktrees))
	for _, w := range worktrees {
		date, err := commitRelDate(ctx, r, w.Head)
		if err != nil {
			date = ""
		}
		items = append(items, worktreeItem{w: w, cwd: cwd, relDate: date, isMainWT: w.Path == worktrees[0].Path})
	}
	return items, nil
}

// commitRelDate returns sha's committer date in relative form, e.g. "3 days ago".
func commitRelDate(ctx context.Context, r *git.Runner, sha string) (string, error) {
	if sha == "" {
		return "", nil
	}
	out, err := r.Run(ctx, "show", "-s", "--format=%cr", sha)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func targetWorktree(items []tui.Item) (worktreeItem, bool) {
	if len(items) != 1 {
		return worktreeItem{}, false
	}
	w, ok := items[0].(worktreeItem)
	return w, ok
}

func targetWorktrees(items []tui.Item) []worktreeItem {
	out := make([]worktreeItem, 0, len(items))
	for _, it := range items {
		if w, ok := it.(worktreeItem); ok {
			out = append(out, w)
		}
	}
	return out
}

func newWorktreeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "worktree",
		Aliases: []string{"wt"},
		Short:   "Interactive list of worktrees",
		Args:    cobra.MaximumNArgs(1),
	}
	attachCommonFlags(cmd)
	cmd.Flags().StringP("new", "b", "", "create a worktree at this path")
	cmd.Flags().String("branch", "", "branch for --new (defaults to the path's base name); with an existing branch, checks it out instead of creating one")
	cmd.Flags().BoolP("delete", "D", false, "remove worktree (force)")
	cmd.Flags().StringP("rename", "m", "", "rename the worktree's branch to this name")
	cmd.RunE = runWorktree
	return cmd
}

func runWorktree(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	runner := git.NewRunner("")

	newPath, _ := cmd.Flags().GetString("new")
	branch, _ := cmd.Flags().GetString("branch")
	del, _ := cmd.Flags().GetBool("delete")
	rename, _ := cmd.Flags().GetString("rename")

	switch {
	case newPath != "":
		if branch == "" {
			branch = filepath.Base(newPath)
		}
		exists, err := git.BranchExists(ctx, runner, branch)
		if err != nil {
			return err
		}
		if err := git.AddWorktree(ctx, runner, newPath, branch, !exists, ""); err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "created worktree %s (%s)\n", newPath, branch)
		return err
	case del:
		if len(args) != 1 {
			return fmt.Errorf("worktree -D/--delete requires exactly one path")
		}
		if err := git.RemoveWorktree(ctx, runner, args[0], true); err != nil {
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "removed worktree %s\n", args[0])
		return err
	case rename != "":
		if len(args) != 1 {
			return fmt.Errorf("worktree -m/--rename requires exactly one path")
		}
		old, err := worktreeBranch(ctx, runner, args[0])
		if err != nil {
			return err
		}
		if err := git.RenameBranch(ctx, runner, old, rename); err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "renamed branch %s to %s (worktree %s)\n", old, rename, args[0])
		return err
	}

	flags := registeredFlags[cmd]
	interactive := flags.resolveInteractive()
	items, err := loadWorktreeItems(ctx, runner)
	if err != nil {
		return err
	}

	if !interactive {
		return tui.RenderTable(cmd.OutOrStdout(), worktreeColumns(), items, tui.TableOptions{
			Density: densityFromFlags(flags),
			Header:  true,
			Marker:  true,
		})
	}

	// checkoutPath receives the selected worktree's path from the "checkout"
	// operation; runWorktree prints it after the program exits so a shell
	// wrapper can `cd "$(gint worktree ...)"` (PLAN.md → worktree "cd-on-
	// checkout mechanism"; see .context/decisions.md).
	var checkoutPath string
	list := tui.New(tui.Config{
		Title:      "gint worktree",
		Columns:    worktreeColumns(),
		Items:      items,
		Operations: buildWorktreeOperations(ctx, runner, &checkoutPath),
		Density:    densityFromFlags(flags),
		Sort:       flags.sort,
	})
	p := tea.NewProgram(list, tea.WithAltScreen(), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		return err
	}
	if checkoutPath != "" {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), checkoutPath)
		return err
	}
	return nil
}

// worktreeBranch resolves the branch checked out at path.
func worktreeBranch(ctx context.Context, r *git.Runner, path string) (string, error) {
	worktrees, err := git.ListWorktrees(ctx, r)
	if err != nil {
		return "", err
	}
	for _, w := range worktrees {
		if w.Path == path {
			return w.Branch, nil
		}
	}
	return "", fmt.Errorf("worktree: no such worktree %q", path)
}

// buildWorktreeOperations returns the worktree view's operation registry.
// checkoutPath is written by the "checkout" operation and read by runWorktree
// once the program exits, since a subprocess cannot cd its parent shell.
func buildWorktreeOperations(ctx context.Context, r *git.Runner, checkoutPath *string) []tui.Operation {
	refresh := func() tea.Cmd {
		items, err := loadWorktreeItems(ctx, r)
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
			Run: func(c tui.OpContext) tea.Cmd {
				w, ok := targetWorktree(c.Items)
				if !ok {
					return tui.Status("select a worktree first")
				}
				*checkoutPath = w.w.Path
				return tea.Quit
			},
		},
		{
			Name: "delete", Key: "D", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{
				Kind:   tui.ConfirmChoice,
				Prompt: "Remove this worktree?",
				Choices: []tui.Choice{
					{Label: "no", Value: "", Key: "n"},
					{Label: "remove", Value: "remove", Key: "d"},
					{Label: "force", Value: "force", Key: "f", Phrase: "force"},
				},
			},
			Run: func(c tui.OpContext) tea.Cmd {
				w, ok := targetWorktree(c.Items)
				if !ok {
					return tui.Status("select a worktree first")
				}
				if err := git.RemoveWorktree(ctx, r, w.w.Path, c.Choice == "force"); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("removed " + w.w.Path)
			},
		},
		{
			Name: "fetch", Key: "f", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				w, ok := targetWorktree(c.Items)
				if !ok {
					return tui.Status("select a worktree first")
				}
				if _, err := git.NewRunner(w.w.Path).Run(ctx, "fetch"); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("fetched " + w.w.Path)
			},
		},
		{
			Name: "pull", Key: "p", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				w, ok := targetWorktree(c.Items)
				if !ok {
					return tui.Status("select a worktree first")
				}
				if _, err := git.NewRunner(w.w.Path).Run(ctx, "pull"); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("pulled " + w.w.Path)
			},
		},
		{
			Name: "push", Key: "P", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Push this worktree's branch to origin?"},
			Run: func(c tui.OpContext) tea.Cmd {
				w, ok := targetWorktree(c.Items)
				if !ok {
					return tui.Status("select a worktree first")
				}
				if _, err := git.NewRunner(w.w.Path).Run(ctx, "push", "-u", "origin", w.w.Branch); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("pushed " + w.w.Path)
			},
		},
		{
			Name: "rename branch", Key: "R", Scope: tui.ScopeItem,
			Input: &tui.InputSpec{Prompt: "New branch name", Placeholder: "branch name"},
			Run: func(c tui.OpContext) tea.Cmd {
				w, ok := targetWorktree(c.Items)
				if !ok {
					return tui.Status("select a worktree first")
				}
				if err := git.RenameBranch(ctx, r, w.w.Branch, c.Input); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("renamed " + w.w.Branch + " to " + c.Input)
			},
		},
		{
			Name: "copy path", Key: "y", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				w, ok := targetWorktree(c.Items)
				if !ok {
					return tui.Status("select a worktree first")
				}
				if err := copyToClipboard(w.w.Path); err != nil {
					return tui.Status(w.w.Path + " (clipboard unavailable: " + err.Error() + ")")
				}
				return tui.Status("copied path " + w.w.Path)
			},
		},
		{
			Name: "lock", Scope: tui.ScopeItem,
			Input: &tui.InputSpec{Prompt: "Lock reason (optional)"},
			Run: func(c tui.OpContext) tea.Cmd {
				w, ok := targetWorktree(c.Items)
				if !ok {
					return tui.Status("select a worktree first")
				}
				if err := git.LockWorktree(ctx, r, w.w.Path, c.Input); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("locked " + w.w.Path)
			},
		},
		{
			Name: "unlock", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				w, ok := targetWorktree(c.Items)
				if !ok {
					return tui.Status("select a worktree first")
				}
				if err := git.UnlockWorktree(ctx, r, w.w.Path); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("unlocked " + w.w.Path)
			},
		},
		{
			Name: "new", Key: "N", Scope: tui.ScopeList,
			Input: &tui.InputSpec{Prompt: "New worktree path", Placeholder: "path [branch]"},
			Run: func(c tui.OpContext) tea.Cmd {
				fields := strings.Fields(c.Input)
				if len(fields) == 0 {
					return tui.Status("a path is required")
				}
				path := fields[0]
				branch := filepath.Base(path)
				if len(fields) > 1 {
					branch = fields[1]
				}
				exists, err := git.BranchExists(ctx, r, branch)
				if err != nil {
					return tui.Status(err.Error())
				}
				if err := git.AddWorktree(ctx, r, path, branch, !exists, ""); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("created worktree " + path)
			},
		},
		{
			Name: "delete", Scope: tui.ScopeItem, BulkOnly: true,
			Confirm: &tui.Confirm{Kind: tui.ConfirmTyped, Prompt: "Remove the selected worktrees?", Phrase: "delete all"},
			Run: func(c tui.OpContext) tea.Cmd {
				targets := targetWorktrees(c.Items)
				for _, w := range targets {
					if err := git.RemoveWorktree(ctx, r, w.w.Path, false); err != nil {
						return tui.Status(err.Error())
					}
				}
				return refreshWith(fmt.Sprintf("removed %d worktrees", len(targets)))
			},
		},
		{
			Name: "prune", Scope: tui.ScopeList,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Prune stale worktree admin files?"},
			Run: func(c tui.OpContext) tea.Cmd {
				if err := git.PruneWorktrees(ctx, r); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("pruned stale worktrees")
			},
		},
	}
}
