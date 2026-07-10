package cli

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

// statusItem adapts a git.StatusEntry to tui.Item. Columns: status code
// (XY, e.g. "M.", ".M", "??", "UU"), path.
type statusItem struct {
	e         git.StatusEntry
	conflict  bool
	untracked bool
}

func (i statusItem) Columns() []string   { return []string{i.e.Code, i.e.Path} }
func (i statusItem) FilterValue() string { return i.e.Path }
func (i statusItem) Current() bool       { return false }

func statusColumns() []tui.Column {
	return []tui.Column{
		{Title: "status", MinWidth: 6, Density: tui.DensityShort},
		{Title: "path", MinWidth: 12, Flex: true, Density: tui.DensityShort},
	}
}

// loadStatusItems fetches working-tree status and adapts it to tui.Items:
// conflicts first, then staged/unstaged paths (deduped — StageFile parses
// git's combined XY code, so a path with both staged and unstaged changes
// appears once), then untracked paths.
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
// toggle, diff, discard, per-file stash, commit/amend stubs (real flow
// arrives in phase 6), and conflict continue/abort via git.InProgressState —
// the interface phase 7's resolve-conflicts command will share.
func buildStatusOperations(ctx context.Context, r *git.Runner) []tui.Operation {
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

	ops := []tui.Operation{
		{
			// PROMPT.md says Enter toggles stage/unstage directly, but the
			// shared List always uses Enter to open the context menu (every
			// view relies on that). "toggle stage" is listed first in the
			// menu (reachable via Enter → Enter) and also bound to a direct
			// shortcut key; see .context/decisions.md.
			Name: "toggle stage", Key: "t", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				s, ok := targetStatusEntry(c.Items)
				if !ok {
					return tui.Status("select a file first")
				}
				if s.conflict {
					return tui.Status("resolve the conflict before staging")
				}
				var err error
				if fullyStaged(s.e.Code) {
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
				diff, err := git.DiffFile(ctx, r, s.e.Path, fullyStaged(s.e.Code))
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
			// Stubbed until the real `commit` flow lands in phase 6
			// (PLAN.md → status task 2); status then delegates to it.
			Name: "commit", Key: "c", Scope: tui.ScopeList,
			Run: func(c tui.OpContext) tea.Cmd {
				return tui.Status("commit is not yet implemented (arrives in phase 6)")
			},
		},
		{
			Name: "amend", Key: "A", Scope: tui.ScopeList,
			Run: func(c tui.OpContext) tea.Cmd {
				return tui.Status("amend is not yet implemented (arrives in phase 6)")
			},
		},
	}

	if state, err := git.DetectInProgress(ctx, r); err == nil && state != nil {
		ops = append(ops,
			tui.Operation{
				Name: "continue " + string(state.Op), Scope: tui.ScopeList,
				Run: func(c tui.OpContext) tea.Cmd {
					if err := state.Continue(ctx, r); err != nil {
						return tui.Status(err.Error())
					}
					return refreshWith("continued " + string(state.Op))
				},
			},
			tui.Operation{
				Name: "abort " + string(state.Op), Scope: tui.ScopeList,
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

	return ops
}

// fullyStaged reports whether a status XY code has no remaining unstaged
// component, e.g. "M." (staged, clean working tree) vs "MM" (staged AND
// further modified).
func fullyStaged(code string) bool {
	return len(code) == 2 && code[0] != '.' && code[1] == '.'
}
