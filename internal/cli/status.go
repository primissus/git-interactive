package cli

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

// statusItem adapts a git.StatusEntry to tui.Item.
//
// grouped marks a row built by loadGroupedStatusItems (the interactive,
// sectioned view) as opposed to loadStatusItems (the flat, script-friendly
// non-interactive view) — it gates whether Columns shows a section-scoped
// single status letter or git's raw two-character XY code.
//
// staged marks a row's membership in the Staged bucket (as opposed to
// Unstaged) — a file with both staged and further unstaged changes ("MM")
// gets one row in each bucket, and staged distinguishes which side a given
// row represents. Only meaningful when grouped is true.
type statusItem struct {
	e         git.StatusEntry
	conflict  bool
	grouped   bool
	staged    bool
	untracked bool
}

// Columns returns git's raw two-character XY code for flat (non-interactive)
// rows, unchanged from before grouping existed. For grouped (interactive)
// rows it instead returns a single status letter scoped to the row's section
// (the index/staged letter under Staged, the worktree/unstaged letter under
// Unstaged), so no "no change on this side" placeholder "." ever needs to be
// displayed. Conflict ("UU") and untracked ("??") rows always show their code
// as-is in both modes.
func (i statusItem) Columns() []string { return []string{i.displayCode(), i.e.Path} }

func (i statusItem) displayCode() string {
	switch {
	case !i.grouped, i.conflict, i.untracked, len(i.e.Code) != 2:
		return i.e.Code
	case i.staged:
		return string(i.e.Code[0])
	default:
		return string(i.e.Code[1])
	}
}

func (i statusItem) FilterValue() string { return i.e.Path }
func (i statusItem) Current() bool       { return false }

func statusColumns() []tui.Column {
	return []tui.Column{
		{Title: "status", MinWidth: 6, Density: tui.DensityShort},
		{Title: "path", MinWidth: 12, Flex: true, Density: tui.DensityShort},
	}
}

// loadStatusItems fetches working-tree status and adapts it to tui.Items for
// the non-interactive/plain output path: conflicts first, then staged/
// unstaged paths (deduped — a path with both staged and unstaged changes
// appears once), then untracked paths. This flat, script-friendly shape (and
// the raw XY status code) is unchanged from before grouping was added to the
// interactive view.
func loadStatusItems(ctx context.Context, r *git.Runner) ([]tui.Item, error) {
	st, err := git.GetStatus(ctx, r)
	if err != nil {
		return nil, err
	}

	var items []tui.Item
	for _, e := range st.Conflicts {
		items = append(items, statusItem{e: e, conflict: true})
	}
	seen := map[string]bool{}
	for _, e := range append(append([]git.StatusEntry{}, st.Staged...), st.Unstaged...) {
		if seen[e.Path] {
			continue
		}
		seen[e.Path] = true
		items = append(items, statusItem{e: e})
	}
	for _, e := range st.Untracked {
		items = append(items, statusItem{e: e, untracked: true})
	}
	return items, nil
}

// loadGroupedStatusItems fetches working-tree status and adapts it to
// tui.Items for the interactive view: rows grouped under section headers
// ("Conflicts"/"Staged"/"Unstaged"/"Untracked", each shown only when
// non-empty), matching plain `git status`. Staged and Unstaged are
// independent buckets from git.Status, so a path with both staged and further
// unstaged changes ("MM") appears once in each section, not deduped.
func loadGroupedStatusItems(ctx context.Context, r *git.Runner) ([]tui.Item, error) {
	st, err := git.GetStatus(ctx, r)
	if err != nil {
		return nil, err
	}

	var items []tui.Item
	if len(st.Conflicts) > 0 {
		items = append(items, tui.HeaderItem{Label: "Conflicts"})
		for _, e := range st.Conflicts {
			items = append(items, statusItem{e: e, grouped: true, conflict: true})
		}
	}
	if len(st.Staged) > 0 {
		items = append(items, tui.HeaderItem{Label: "Staged"})
		for _, e := range st.Staged {
			items = append(items, statusItem{e: e, grouped: true, staged: true})
		}
	}
	if len(st.Unstaged) > 0 {
		items = append(items, tui.HeaderItem{Label: "Unstaged"})
		for _, e := range st.Unstaged {
			items = append(items, statusItem{e: e, grouped: true})
		}
	}
	if len(st.Untracked) > 0 {
		items = append(items, tui.HeaderItem{Label: "Untracked"})
		for _, e := range st.Untracked {
			items = append(items, statusItem{e: e, grouped: true, untracked: true})
		}
	}
	return items, nil
}

func targetStatusEntry(items []tui.Item) (statusItem, bool) {
	if len(items) != 1 {
		return statusItem{}, false
	}
	s, ok := items[0].(statusItem)
	return s, ok
}

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"st"},
		Short:   "Interactive status view",
	}
	attachCommonFlags(cmd)
	cmd.RunE = runStatus
	return cmd
}

func runStatus(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	runner := git.NewRunner("")
	flags := registeredFlags[cmd]

	if !flags.resolveInteractive() {
		items, err := loadStatusItems(ctx, runner)
		if err != nil {
			return err
		}
		return tui.RenderTable(cmd.OutOrStdout(), statusColumns(), items, tui.TableOptions{
			Density: densityFromFlags(flags),
			Header:  true,
		})
	}

	items, err := loadGroupedStatusItems(ctx, runner)
	if err != nil {
		return err
	}

	list := tui.New(tui.Config{
		Title:      "gint status",
		Columns:    statusColumns(),
		Items:      items,
		Operations: buildStatusOperations(ctx, runner),
		Density:    densityFromFlags(flags),
		Sort:       flags.sort,
	})
	p := tea.NewProgram(list, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err = p.Run()
	return err
}

// buildStatusOperations returns status's operation registry: stage/unstage
// toggle, diff, discard, per-file stash, commit/amend (delegating to the same
// choice logic as the standalone `commit` command), and conflict
// continue/abort via git.InProgressState — the interface phase 7's
// resolve-conflicts command will share.
func buildStatusOperations(ctx context.Context, r *git.Runner) []tui.Operation {
	refresh := func() tea.Cmd {
		items, err := loadGroupedStatusItems(ctx, r)
		if err != nil {
			return tui.Status(err.Error())
		}
		return tui.SetItems(items)
	}
	refreshWith := func(status string) tea.Cmd {
		return tea.Batch(tui.Status(status), refresh())
	}

	ops := []tui.Operation{
		{
			// PROMPT.md says Enter toggles stage/unstage directly, but the
			// shared List always uses Enter to open the context menu (every
			// view relies on that). "toggle stage" is listed first in the
			// menu (reachable via Enter → Enter) and also bound to a direct
			// shortcut key; see .context/decisions.md.
			Name: "toggle stage", Key: "a", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				s, ok := targetStatusEntry(c.Items)
				if !ok {
					return tui.Status("select a file first")
				}
				if s.conflict {
					return tui.Status("resolve the conflict before staging")
				}
				var err error
				if s.staged {
					err = git.UnstageFile(ctx, r, s.e.Path)
				} else {
					err = git.StageFile(ctx, r, s.e.Path)
				}
				if err != nil {
					return tui.Status(err.Error())
				}
				return refresh()
			},
		},
		{
			Name: "diff", Key: "d", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				s, ok := targetStatusEntry(c.Items)
				if !ok {
					return tui.Status("select a file first")
				}
				diff, err := git.DiffFile(ctx, r, s.e.Path, s.staged)
				if err != nil {
					return tui.Status(err.Error())
				}
				if strings.TrimSpace(diff) == "" {
					return tui.Status("no diff (try staging/unstaging first)")
				}
				return showInPager(diff)
			},
		},
		{
			Name: "discard", Key: "D", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{Kind: tui.ConfirmTyped, Prompt: "Discard changes to this file?", Phrase: "discard"},
			Run: func(c tui.OpContext) tea.Cmd {
				s, ok := targetStatusEntry(c.Items)
				if !ok {
					return tui.Status("select a file first")
				}
				if err := git.DiscardFile(ctx, r, s.e.Path, s.untracked); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("discarded " + s.e.Path)
			},
		},
		{
			Name: "stash file", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Stash this file?"},
			Run: func(c tui.OpContext) tea.Cmd {
				s, ok := targetStatusEntry(c.Items)
				if !ok {
					return tui.Status("select a file first")
				}
				if err := git.StashPush(ctx, r, "", s.e.Path); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("stashed " + s.e.Path)
			},
		},
		{
			// Delegates to the same choice logic as the standalone `commit`
			// command (PLAN.md → status task 2). The shared List's Confirm
			// has no "return to input" transition like the standalone
			// command's Flow does, so choosing "edit" here just cancels —
			// press 'c' again to retype the message.
			Name: "commit", Key: "c", Scope: tui.ScopeList,
			Input:   &tui.InputSpec{Prompt: "Commit message"},
			Confirm: &tui.Confirm{Kind: tui.ConfirmChoice, Prompt: "Commit?", Choices: commitConfirmChoices()},
			Run: func(c tui.OpContext) tea.Cmd {
				status, err := runCommitChoice(ctx, r, c.Input, c.Choice)
				if err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith(status)
			},
		},
		{
			Name: "amend", Key: "A", Scope: tui.ScopeList,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Amend the last commit? (keeps its message)"},
			Run: func(c tui.OpContext) tea.Cmd {
				status, err := runCommitChoice(ctx, r, "", "amend")
				if err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith(status)
			},
		},
	}

	// When an operation is stopped on conflicts, layer in the shared
	// conflict-resolution component: per-file take ours/theirs/both/$EDITOR
	// (labels resolved to branch/commit names) plus continue/abort. This is the
	// same component the standalone `rebase` resolver uses (see conflict.go).
	if state, err := git.DetectInProgress(ctx, r); err == nil && state != nil {
		sides := git.ResolveSides(ctx, r, state)
		ops = append(ops, fileResolutionOps(ctx, r, sides, statusConflictPath, refresh)...)
		ops = append(ops,
			tui.Operation{
				Name: "continue " + string(state.Op), ID: "continue", Scope: tui.ScopeList,
				Run: func(c tui.OpContext) tea.Cmd {
					if err := state.Continue(ctx, r); err != nil {
						return tui.Status(err.Error())
					}
					return refreshWith("continued " + string(state.Op))
				},
			},
			tui.Operation{
				Name: "abort " + string(state.Op), ID: "abort", Scope: tui.ScopeList,
				Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Abort the in-progress " + string(state.Op) + "?"},
				Run: func(c tui.OpContext) tea.Cmd {
					if err := state.Abort(ctx, r); err != nil {
						return tui.Status(err.Error())
					}
					return refreshWith("aborted " + string(state.Op))
				},
			},
		)
	}

	return tui.ApplyKeymap("status", ops)
}

// statusConflictPath resolves a status row to a conflicted file path for the
// shared file-resolution operations, rejecting non-conflict rows.
func statusConflictPath(items []tui.Item) (string, bool) {
	s, ok := targetStatusEntry(items)
	if !ok || !s.conflict {
		return "", false
	}
	return s.e.Path, true
}
