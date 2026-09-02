package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
	"git-interact/internal/validate"
)

// copyToClipboard is a package var so tests can stub it out; the real
// clipboard is unavailable in CI/headless environments.
var copyToClipboard = clipboard.WriteAll

// branchItem adapts a git.Branch to tui.Item. Column order follows the
// git-br.py reference (and phase 1's -I output): name, last commit subject,
// relative date, author, worktree.
type branchItem struct {
	b           git.Branch
	merged      bool
	displayName string // leaf segment + indent when tree view is active
	wtPath      string // absolute path of the worktree holding this branch ("" = none)
	cwd         string // process cwd, for live worktree-path formatting
}

func (i branchItem) Columns() []string {
	name := i.b.Name
	if i.displayName != "" {
		name = i.displayName
	} else {
		name = tui.FormatBranch(name)
	}
	return []string{name, i.b.Subject, tui.FormatDate(i.b.CommitUnix, i.b.CommitDate), tui.FormatAuthor(i.b.AuthorName), tui.FormatWorktreePath(i.wtPath, i.cwd)}
}
func (i branchItem) FilterValue() string { return i.b.Name }
func (i branchItem) Current() bool       { return i.b.Head }

// createBranchItem is the pinned "create a branch" row shown above the list;
// see PROMPT.md branch → "Create: an entry above the list (default focus
// stays on first branch)". It participates in the same list so the "new"
// operation is reachable both from Shift+N and by pressing enter on its row —
// DefaultOp skips the regular item-ops menu (which doesn't apply to a
// placeholder row) and goes straight to the name prompt.
type createBranchItem struct{}

func (createBranchItem) Columns() []string   { return []string{"+ new branch (Shift+N)", "", "", "", ""} }
func (createBranchItem) FilterValue() string { return "" }
func (createBranchItem) Current() bool       { return false }
func (createBranchItem) DefaultOp() string   { return "new" }

func branchColumns() []tui.Column {
	return []tui.Column{
		{Title: "branch", MinWidth: 12, Flex: true, Density: tui.DensityShort, Color: tui.ColorName},
		{Title: "last commit", MaxWidth: 50, Flex: true, Density: tui.DensityNormal},
		{Title: "date", MinWidth: 7, Density: tui.DensityNormal, Color: tui.ColorDate},
		{Title: "author", MinWidth: 10, Density: tui.DensityNormal, Color: tui.ColorAuthor},
		// Worktree column: visible in the default and -F views, hidden in -s
		// (DensityNormal), matching the intent of "visible in default and -F".
		{Title: "worktree", MinWidth: 8, Density: tui.DensityNormal, Color: tui.ColorDate},
	}
}

// branchFilters is the parsed form of branch's (and worktree's) filter flags.
type branchFilters struct {
	author    string
	since     string // "" | 1d | 3d | 1w | 1m | ytd | 1y
	merged    bool
	notMerged bool
	gone      bool
}

func (f branchFilters) validate() error {
	if f.merged && f.notMerged {
		return fmt.Errorf("--merged and --not-merged are mutually exclusive")
	}
	switch f.since {
	case "", "1d", "3d", "1w", "1m", "ytd", "1y":
	default:
		return fmt.Errorf("--since: unknown bucket %q (want 1d, 3d, 1w, 1m, ytd, or 1y)", f.since)
	}
	return nil
}

// sinceCutoff returns the unix cutoff for a --since bucket; the zero value
// (bucket == "") means "no cutoff".
func sinceCutoff(bucket string) int64 {
	now := time.Now()
	switch bucket {
	case "1d":
		return now.AddDate(0, 0, -1).Unix()
	case "3d":
		return now.AddDate(0, 0, -3).Unix()
	case "1w":
		return now.AddDate(0, 0, -7).Unix()
	case "1m":
		return now.AddDate(0, -1, 0).Unix()
	case "ytd":
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()).Unix()
	case "1y":
		return now.AddDate(-1, 0, 0).Unix()
	default:
		return 0
	}
}

// filterBranches applies branchFilters over branches, given the set of branch
// names merged into the current branch.
func filterBranches(branches []git.Branch, f branchFilters, merged map[string]bool) []git.Branch {
	cutoff := sinceCutoff(f.since)
	out := make([]git.Branch, 0, len(branches))
	for _, b := range branches {
		if f.author != "" && !strings.Contains(strings.ToLower(b.AuthorName), strings.ToLower(f.author)) {
			continue
		}
		if f.since != "" && b.CommitUnix < cutoff {
			continue
		}
		if f.merged && !merged[b.Name] {
			continue
		}
		if f.notMerged && merged[b.Name] {
			continue
		}
		if f.gone && !b.Gone() {
			continue
		}
		out = append(out, b)
	}
	return out
}

// sortModeFromFlag normalizes the -S value to one of the branch view's sort
// modes, defaulting to "last-commit".
func sortModeFromFlag(s string) string {
	switch s {
	case "created", "author", "off":
		return s
	default:
		return "last-commit"
	}
}

// sortModes is the cycle order for the runtime sort toggle (S).
var sortModes = []string{"last-commit", "created", "author", "off"}

// nextSortMode returns the mode that follows current in sortModes.
func nextSortMode(current string) string {
	for i, m := range sortModes {
		if m == current {
			return sortModes[(i+1)%len(sortModes)]
		}
	}
	return sortModes[0]
}

// sortBranches reorders branches in place per mode. "off" leaves ListBranches'
// own order (most-recent-commit-first) untouched.
func sortBranches(branches []git.Branch, mode string) {
	switch mode {
	case "created":
		// Git has no stored branch-creation timestamp; the tip commit's
		// author date (unlike committer date, unaffected by rebase/amend) is
		// the closest single-call heuristic, and matches the common
		// `git branch --sort=-creatordate` convention. See .context/decisions.md.
		sort.SliceStable(branches, func(i, j int) bool { return branches[i].AuthorUnix > branches[j].AuthorUnix })
	case "author":
		sort.SliceStable(branches, func(i, j int) bool {
			return strings.ToLower(branches[i].AuthorName) < strings.ToLower(branches[j].AuthorName)
		})
	case "off":
		// no-op: keep git's own ordering.
	default: // "last-commit"
		sort.SliceStable(branches, func(i, j int) bool { return branches[i].CommitUnix > branches[j].CommitUnix })
	}
}

// loadBranchItems fetches, filters, and sorts branches, and adapts them to
// tui.Items; includeCreateRow pins the create-branch row at index 0 (used for
// the interactive list, not for -I output or direct-menu mode). Each row's
// worktree cell comes from the git worktree list map (raw absolute paths,
// formatted at render time), which covers the main worktree — %(worktreepath)
// alone is empty there.
func loadBranchItems(ctx context.Context, r *git.Runner, f branchFilters, sortMode string, includeCreateRow bool) ([]tui.Item, error) {
	branches, err := git.ListBranches(ctx, r)
	if err != nil {
		return nil, err
	}
	merged, err := git.MergedBranches(ctx, r)
	if err != nil {
		return nil, err
	}
	branchWT, err := worktreeByBranch(ctx, r)
	if err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	branches = filterBranches(branches, f, merged)
	sortBranches(branches, sortMode)

	items := make([]tui.Item, 0, len(branches)+1)
	if includeCreateRow {
		items = append(items, createBranchItem{})
	}
	for _, b := range branches {
		items = append(items, branchItem{b: b, merged: merged[b.Name], wtPath: branchWT[b.Name], cwd: cwd})
	}
	return items, nil
}

// targetBranch resolves a single-branch operation's target, rejecting the
// create-row placeholder (or an empty/mixed selection).
func targetBranch(items []tui.Item) (branchItem, bool) {
	if len(items) != 1 {
		return branchItem{}, false
	}
	b, ok := items[0].(branchItem)
	return b, ok
}

// targetBranches resolves a bulk operation's targets, dropping the create-row
// placeholder if it was somehow included in the selection.
func targetBranches(items []tui.Item) []branchItem {
	out := make([]branchItem, 0, len(items))
	for _, it := range items {
		if b, ok := it.(branchItem); ok {
			out = append(out, b)
		}
	}
	return out
}

func newBranchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "branch [branch-name]",
		Aliases: []string{"br"},
		Short:   "Interactive tabular list of branches",
		Args:    cobra.MaximumNArgs(1),
	}
	attachCommonFlags(cmd)
	cmd.Flags().StringP("new", "b", "", "create branch")
	cmd.Flags().BoolP("delete", "D", false, "delete branch (force)")
	cmd.Flags().StringP("rename", "m", "", "rename branch to this name")
	cmd.Flags().String("author", "", "filter: branches by this author")
	cmd.Flags().String("since", "", "filter: last commit within 1d|3d|1w|1m|ytd|1y")
	cmd.Flags().Bool("merged", false, "filter: only branches merged into the current branch")
	cmd.Flags().Bool("not-merged", false, "filter: only branches not merged into the current branch")
	cmd.Flags().Bool("gone", false, "filter: only branches whose upstream was deleted")
	cmd.RunE = runBranch
	return cmd
}

func runBranch(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	runner := git.NewRunner("")

	newName, _ := cmd.Flags().GetString("new")
	del, _ := cmd.Flags().GetBool("delete")
	rename, _ := cmd.Flags().GetString("rename")

	switch {
	case newName != "":
		if err := git.CreateBranch(ctx, runner, newName, ""); err != nil {
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "created branch %s\n", newName)
		return err
	case del:
		if len(args) != 1 {
			return fmt.Errorf("branch -D/--delete requires exactly one branch name")
		}
		if err := git.DeleteBranch(ctx, runner, args[0], true); err != nil {
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "deleted branch %s\n", args[0])
		return err
	case rename != "":
		old := ""
		switch len(args) {
		case 0:
			cur, err := git.CurrentBranch(ctx, runner)
			if err != nil {
				return err
			}
			old = cur
		case 1:
			old = args[0]
		}
		if err := git.RenameBranch(ctx, runner, old, rename); err != nil {
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "renamed branch %s to %s\n", old, rename)
		return err
	}

	f, err := branchFiltersFromFlags(cmd)
	if err != nil {
		return err
	}
	flags := registeredFlags[cmd]
	sortMode := sortModeFromFlag(flags.sort)
	state := &branchViewState{sort: sortMode}

	if len(args) == 1 {
		return runBranchDirectMenu(cmd, runner, args[0], f, state)
	}

	interactive := flags.resolveInteractive()
	if !interactive {
		items, err := loadBranchItems(ctx, runner, f, state.sort, false)
		if err != nil {
			return err
		}
		return tui.RenderTable(cmd.OutOrStdout(), branchColumns(), items, tui.TableOptions{
			Density: densityFromFlags(flags),
			Header:  true,
			Marker:  true,
		})
	}

	for {
		items, err := loadBranchItems(ctx, runner, f, state.sort, true)
		if err != nil {
			return err
		}
		if state.grouped {
			items = applyGrouping(items, state.collapsed)
		}
		list := tui.New(tui.Config{
			Title:         "gint branch",
			Columns:       branchColumns(),
			Items:         items,
			Operations:    buildBranchOperations(ctx, runner, f, state, true),
			Density:       densityFromFlags(flags),
			Sort:          state.sortLabel(),
			InitialCursor: 1, // focus stays on the first real branch, past the create row
			View:          "branch",
			// Live hidden-column set from the :settings overlay. The List
			// re-reads it per frame; -I output keeps the full column set.
			HiddenColumns: tui.ActiveBranchHidden,
		})
		p := tea.NewProgram(list, tea.WithAltScreen(), tea.WithContext(ctx))
		if _, err := p.Run(); err != nil {
			return err
		}
		reentered, err := runBranchRebaseHandoff(cmd, runner, state, flags)
		if err != nil {
			return err
		}
		if !reentered {
			return nil
		}
	}
}

func branchFiltersFromFlags(cmd *cobra.Command) (branchFilters, error) {
	author, _ := cmd.Flags().GetString("author")
	since, _ := cmd.Flags().GetString("since")
	merged, _ := cmd.Flags().GetBool("merged")
	notMerged, _ := cmd.Flags().GetBool("not-merged")
	gone, _ := cmd.Flags().GetBool("gone")
	f := branchFilters{author: author, since: since, merged: merged, notMerged: notMerged, gone: gone}
	if err := f.validate(); err != nil {
		return branchFilters{}, err
	}
	return f, nil
}

// runBranchDirectMenu implements `gint branch <name>`: skip the list, open the
// operations menu for that branch directly, with fuzzy op matching (PROMPT.md
// → branch, .context/glossary.md → "Direct-menu mode").
func runBranchDirectMenu(cmd *cobra.Command, r *git.Runner, name string, f branchFilters, state *branchViewState) error {
	ctx := cmd.Context()
	flags := registeredFlags[cmd]
	for {
		items, err := loadBranchItems(ctx, r, f, state.sort, false)
		if err != nil {
			return err
		}
		idx := -1
		for i, it := range items {
			if b, ok := it.(branchItem); ok && b.b.Name == name {
				idx = i
				break
			}
		}
		if idx == -1 {
			return fmt.Errorf("branch: no such branch %q", name)
		}

		list := tui.New(tui.Config{
			Title:           "gint branch " + name,
			Columns:         branchColumns(),
			Items:           items,
			Operations:      buildBranchOperations(ctx, r, f, state, false),
			InitialCursor:   idx,
			OpenMenuOnStart: true,
		})
		p := tea.NewProgram(list, tea.WithAltScreen(), tea.WithContext(ctx))
		if _, err := p.Run(); err != nil {
			return err
		}
		reentered, err := runBranchRebaseHandoff(cmd, r, state, flags)
		if err != nil {
			return err
		}
		if !reentered {
			return nil
		}
	}
}

// runBranchRebaseHandoff runs the rebase flow when the branch view quit for
// the "rebase onto this branch" operation. It reports (false, nil) when no
// rebase is pending (normal exit). On a pending rebase it either resumes an
// in-progress rebase via the conflict resolver or starts the interactive plan
// view via runRebaseInteractive. Returns (true, nil) after the flow so the
// caller re-enters the refreshed branch list.
func runBranchRebaseHandoff(cmd *cobra.Command, r *git.Runner, state *branchViewState, flags *commonFlags) (reenter bool, err error) {
	if state.rebaseBase == "" {
		return false, nil
	}
	base := state.rebaseBase
	state.rebaseBase = ""
	ctx := cmd.Context()

	if st, err := git.DetectInProgress(ctx, r); err == nil && st != nil && st.Op == git.OpRebase {
		return true, runConflictResolver(cmd, r, st)
	}
	cur, err := git.CurrentBranch(ctx, r)
	if err != nil || cur == "" {
		cur = "HEAD"
	}
	return true, runRebaseInteractive(cmd, r, cur, base, flags)
}

// branchViewState holds the mutable runtime state shared between the ops
// closure and the refresh function inside buildBranchOperations.
type branchViewState struct {
	sort       string
	grouped    bool
	collapsed  map[string]bool
	rebaseBase string // set by the rebase op to hand off to runRebaseInteractive; reset after handoff
}

func (s *branchViewState) sortLabel() string {
	l := s.sort
	if s.grouped {
		l += " · tree"
	}
	return l
}

// buildBranchOperations returns the branch view's operation registry.
// includeCreateRow must match the value used to build the list's initial
// items so a post-mutation refresh keeps the same shape.
func buildBranchOperations(ctx context.Context, r *git.Runner, f branchFilters, state *branchViewState, includeCreateRow bool) []tui.Operation {
	refresh := func() tea.Cmd {
		items, err := loadBranchItems(ctx, r, f, state.sort, includeCreateRow)
		if err != nil {
			return tui.Status(err.Error())
		}
		if state.grouped {
			items = applyGrouping(items, state.collapsed)
		}
		return tea.Batch(tui.SetSortLabel(state.sortLabel()), tui.SetItems(items))
	}
	refreshWith := func(status string) tea.Cmd {
		return tea.Batch(tui.Status(status), refresh())
	}

	// oldestFirst orders bulk-delete targets from the oldest last-commit to the
	// newest, so a resilient delete run peels stale branches off first.
	oldestFirst := func(items []tui.Item) []tui.Item {
		out := append([]tui.Item(nil), items...)
		sort.SliceStable(out, func(i, j int) bool {
			bi, iok := out[i].(branchItem)
			bj, jok := out[j].(branchItem)
			if !iok || !jok {
				return iok && !jok
			}
			return bi.b.CommitUnix < bj.b.CommitUnix
		})
		return out
	}

	return tui.ApplyKeymap("branch", []tui.Operation{
		{
			Name: "checkout", Key: "C", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				b, ok := targetBranch(c.Items)
				if !ok {
					return tui.Status("select a branch first")
				}
				if err := git.CheckoutBranch(ctx, r, b.b.Name); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("checked out " + b.b.Name)
			},
		},
		{
			Name: "delete", Key: "D", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{
				Kind:   tui.ConfirmChoice,
				Prompt: "Delete this branch?",
				Choices: []tui.Choice{
					{Label: "no", Value: "", Key: "n"},
					{Label: "delete", Value: "delete", Key: "d"},
					{Label: "force", Value: "force", Key: "f", Phrase: "force"},
				},
			},
			Run: func(c tui.OpContext) tea.Cmd {
				b, ok := targetBranch(c.Items)
				if !ok {
					return tui.Status("select a branch first")
				}
				if err := git.DeleteBranch(ctx, r, b.b.Name, c.Choice == "force"); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("deleted " + b.b.Name)
			},
		},
		{
			Name: "pull", Key: "p", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				b, ok := targetBranch(c.Items)
				if !ok {
					return tui.Status("select a branch first")
				}
				if err := git.PullBranch(ctx, r, b.b.Name, b.b.Head); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("pulled " + b.b.Name)
			},
		},
		{
			Name: "push", Key: "P", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Push this branch to origin?"},
			Run: func(c tui.OpContext) tea.Cmd {
				b, ok := targetBranch(c.Items)
				if !ok {
					return tui.Status("select a branch first")
				}
				if err := git.PushBranch(ctx, r, b.b.Name); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("pushed " + b.b.Name)
			},
		},
		{
			Name: "rename", Key: "R", Scope: tui.ScopeItem,
			Input: &tui.InputSpec{
				Prompt: "New name", Placeholder: "branch name", Validate: validate.BranchName,
				InitialFrom: func(items []tui.Item) string {
					if b, ok := items[0].(branchItem); ok {
						return b.b.Name
					}
					return ""
				},
			},
			Run: func(c tui.OpContext) tea.Cmd {
				b, ok := targetBranch(c.Items)
				if !ok {
					return tui.Status("select a branch first")
				}
				if err := git.RenameBranch(ctx, r, b.b.Name, c.Input); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("renamed " + b.b.Name + " to " + c.Input)
			},
		},
		{
			Name: "copy sha", Key: "y", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				b, ok := targetBranch(c.Items)
				if !ok {
					return tui.Status("select a branch first")
				}
				sha, err := git.RevParse(ctx, r, b.b.Name)
				if err != nil {
					return tui.Status(err.Error())
				}
				if err := copyToClipboard(sha); err != nil {
					return tui.Status("sha " + sha + " (clipboard unavailable: " + err.Error() + ")")
				}
				return tui.Status("copied sha " + sha + " (" + b.b.Name + ")")
			},
		},
		{
			Name: "copy name", Key: "c", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				b, ok := targetBranch(c.Items)
				if !ok {
					return tui.Status("select a branch first")
				}
				if err := copyToClipboard(b.b.Name); err != nil {
					return tui.Status(b.b.Name + " (clipboard unavailable: " + err.Error() + ")")
				}
				return tui.Status("copied " + b.b.Name)
			},
		},
		{
			// PROMPT.md branch → Shift+M merge selected into current, reusing
			// the `merge` command's confirmation flow (mergeConfirmChoices).
			Name: "merge into current", Key: "M", Scope: tui.ScopeItem, Bulk: true,
			Confirm: &tui.Confirm{Kind: tui.ConfirmChoice, Prompt: "Merge the selected branch(es) into the current branch?", Choices: mergeConfirmChoices()},
			Run: func(c tui.OpContext) tea.Cmd {
				targets := targetBranches(c.Items)
				if len(targets) == 0 {
					return tui.Status("select a branch first")
				}
				for _, b := range targets {
					if _, err := runMergeChoice(ctx, r, b.b.Name, c.Choice); err != nil {
						return tui.Status(err.Error())
					}
				}
				names := make([]string, len(targets))
				for i, b := range targets {
					names[i] = b.b.Name
				}
				return refreshWith("merged " + strings.Join(names, ", "))
			},
		},
		{
			Name: "rebase onto this branch", Key: "B", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				b, ok := targetBranch(c.Items)
				if !ok {
					return tui.Status("select a branch first")
				}
				cur, err := git.CurrentBranch(ctx, r)
				if err != nil {
					return tui.Status(err.Error())
				}
				if b.b.Name == cur {
					return tui.Status("already on " + cur + " — nothing to rebase onto")
				}
				state.rebaseBase = b.b.Name
				return tea.Quit
			},
		},
		{
			Name: "new", Key: "N", Scope: tui.ScopeList,
			Input: &tui.InputSpec{Prompt: "New branch", Placeholder: "branch name", Validate: validate.BranchName},
			Run: func(c tui.OpContext) tea.Cmd {
				if err := git.CreateBranch(ctx, r, c.Input, ""); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("created branch " + c.Input)
			},
		},
		{
			Name: "sort", Key: "S", Scope: tui.ScopeList,
			Run: func(c tui.OpContext) tea.Cmd {
				state.sort = nextSortMode(state.sort)
				return tea.Batch(tui.Status("sorted by "+state.sort), refresh())
			},
		},
		{
			Name: "group", Key: "T", Scope: tui.ScopeList,
			Run: func(c tui.OpContext) tea.Cmd {
				state.grouped = !state.grouped
				if !state.grouped {
					state.collapsed = nil
					return refreshWith("tree view off")
				}
				if state.collapsed == nil {
					state.collapsed = map[string]bool{}
				}
				return refreshWith("tree view on")
			},
		},
		{
			Name: "toggle group", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				if len(c.Items) != 1 {
					return tui.Status("select a group first")
				}
				g, ok := c.Items[0].(branchGroupItem)
				if !ok {
					return tui.Status("not a group")
				}
				state.collapsed[g.prefix] = !g.collapsed
				return refresh()
			},
		},
		{
			// Bulk archive/delete run as resilient batches (BatchSpec): a branch
			// that fails to delete pauses to ask whether to continue instead of
			// aborting the whole run, and targets are peeled oldest-first.
			Name: "archive", Scope: tui.ScopeItem, BulkOnly: true,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Archive the selected branch(es)? (tag archive/<name> and delete)"},
			Batch: &tui.BatchSpec{
				Verb:  "archived",
				Order: oldestFirst,
				Step: func(it tui.Item) error {
					b := it.(branchItem)
					sha, err := git.RevParse(ctx, r, b.b.Name)
					if err != nil {
						return err
					}
					if err := git.TagRef(ctx, r, "archive/"+b.b.Name, sha); err != nil {
						return err
					}
					return git.DeleteBranch(ctx, r, b.b.Name, true)
				},
				Refresh: refresh,
			},
		},
		{
			Name: "delete", Scope: tui.ScopeItem, BulkOnly: true,
			Confirm: &tui.Confirm{Kind: tui.ConfirmTyped, Prompt: "Delete the selected branches?", Phrase: "delete all"},
			Batch: &tui.BatchSpec{
				Verb:  "deleted",
				Order: oldestFirst,
				Step: func(it tui.Item) error {
					return git.DeleteBranch(ctx, r, it.(branchItem).b.Name, false)
				},
				Refresh: refresh,
			},
		},
		{
			Name: "force delete", Scope: tui.ScopeItem, BulkOnly: true,
			Confirm: &tui.Confirm{Kind: tui.ConfirmTyped, Prompt: "Force-delete the selected branches?", Phrase: "force delete"},
			Batch: &tui.BatchSpec{
				Verb:  "force-deleted",
				Order: oldestFirst,
				Step: func(it tui.Item) error {
					return git.DeleteBranch(ctx, r, it.(branchItem).b.Name, true)
				},
				Refresh: refresh,
			},
		},
	})
}
