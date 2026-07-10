package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

// The conflict-resolution component. PROMPT.md requires one reusable
// implementation shared by `rebase`, the future `resolve-conflicts` command,
// and `status`. It is split into:
//
//   - fileResolutionOps: per-file take ours/theirs/both/$EDITOR, with the two
//     sides always labeled by branch/commit name (never raw ours/theirs, which
//     git inverts during a rebase);
//   - continueSkipAbortOps: the operation-level continue/skip/abort;
//   - runConflictResolver: a standalone List that hosts both, used when `rebase`
//     stops on a conflict.
//
// `status` composes the same fileResolutionOps into its own list (see
// buildStatusOperations), which is what "wire the component into status" means.

// conflictItem is one conflicted file in the standalone resolver list.
type conflictItem struct{ path string }

func (i conflictItem) Columns() []string   { return []string{i.path} }
func (i conflictItem) FilterValue() string { return i.path }
func (i conflictItem) Current() bool       { return false }

// conflictPath extracts the single targeted file path from a resolver row.
func conflictPath(items []tui.Item) (string, bool) {
	if len(items) != 1 {
		return "", false
	}
	c, ok := items[0].(conflictItem)
	if !ok {
		return "", false
	}
	return c.path, true
}

// fileResolutionOps builds the per-file conflict operations. pathOf resolves the
// target file from an operation's rows (so status and the standalone resolver
// can supply their own row types); refresh reloads the host list after a file
// is resolved. sides carries the branch/commit labels for the two conflict
// sides.
func fileResolutionOps(ctx context.Context, r *git.Runner, sides git.ConflictSides, pathOf func([]tui.Item) (string, bool), refresh func() tea.Cmd) []tui.Operation {
	refreshWith := func(status string) tea.Cmd {
		return tea.Batch(tui.Status(status), refresh())
	}
	resolve := func(items []tui.Item, do func(string) error, verb string) tea.Cmd {
		path, ok := pathOf(items)
		if !ok {
			return tui.Status("select a conflicted file first")
		}
		if err := do(path); err != nil {
			return tui.Status(err.Error())
		}
		return refreshWith(verb + " " + path)
	}

	return []tui.Operation{
		{
			Name: "take ours (" + sides.Ours + ")", Key: "o", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				return resolve(c.Items, func(p string) error { return git.TakeOurs(ctx, r, p) }, "took "+sides.Ours+" for")
			},
		},
		{
			Name: "take theirs (" + sides.Theirs + ")", Key: "t", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				return resolve(c.Items, func(p string) error { return git.TakeTheirs(ctx, r, p) }, "took "+sides.Theirs+" for")
			},
		},
		{
			Name: "take both", Key: "b", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				return resolve(c.Items, func(p string) error { return git.TakeBoth(ctx, r, p) }, "kept both sides of")
			},
		},
		{
			Name: "edit in $EDITOR", Key: "e", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				return editConflict(ctx, r, c.Items, pathOf, refreshWith)
			},
		},
	}
}

// editConflict opens the conflicted file in $EDITOR, then stages it and
// refreshes. It hands the terminal to the editor via tea.ExecProcess.
func editConflict(ctx context.Context, r *git.Runner, items []tui.Item, pathOf func([]tui.Item) (string, bool), refreshWith func(string) tea.Cmd) tea.Cmd {
	path, ok := pathOf(items)
	if !ok {
		return tui.Status("select a conflicted file first")
	}
	root, err := git.RepoRoot(ctx, r)
	if err != nil {
		return tui.Status(err.Error())
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command(editor, filepath.Join(root, path)) //nolint:gosec // $EDITOR is the user's own configured editor
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return tui.Status("editor: " + err.Error())()
		}
		if err := git.StageFile(ctx, r, path); err != nil {
			return tui.Status(err.Error())()
		}
		return refreshWith("edited " + path)()
	})
}

// conflictResult carries the resolver's outcome back to the CLI after its
// program exits, so a final status line can be printed to stdout.
type conflictResult struct {
	status string
	err    error
}

// continueSkipAbortOps builds the operation-level controls for an in-progress
// git operation: continue, skip (when supported), and abort. Completing or
// aborting the operation quits the resolver by recording the outcome in res and
// returning tea.Quit; a continue that stops on the next conflict refreshes the
// list instead.
func continueSkipAbortOps(ctx context.Context, r *git.Runner, state *git.InProgressState, res *conflictResult, refresh func() tea.Cmd) []tui.Operation {
	op := string(state.Op)

	advance := func(act func() error, done string) tea.Cmd {
		err := act()
		st, _ := git.DetectInProgress(ctx, r)
		if st == nil {
			res.status = done
			return tea.Quit
		}
		// Still in progress: git stopped on the next conflict (or an `edit`).
		msg := "stopped again — resolve the remaining conflicts"
		if err != nil {
			msg = err.Error()
		}
		return tea.Batch(tui.Status(msg), refresh())
	}

	ops := []tui.Operation{
		{
			Name: "continue " + op, Key: "c", Scope: tui.ScopeList,
			Run: func(tui.OpContext) tea.Cmd {
				return advance(func() error { return state.Continue(ctx, r) }, "continued "+op+" to completion")
			},
		},
	}
	if state.CanSkip() {
		ops = append(ops, tui.Operation{
			Name: "skip commit", Scope: tui.ScopeList,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Skip the current commit?"},
			Run: func(tui.OpContext) tea.Cmd {
				return advance(func() error { return state.Skip(ctx, r) }, "skipped to "+op+" completion")
			},
		})
	}
	ops = append(ops, tui.Operation{
		Name: "abort " + op, Scope: tui.ScopeList,
		Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Abort the in-progress " + op + "?"},
		Run: func(tui.OpContext) tea.Cmd {
			if err := state.Abort(ctx, r); err != nil {
				res.err = err
				return tea.Quit
			}
			res.status = "aborted " + op
			return tea.Quit
		},
	})
	return ops
}

// runConflictResolver runs the standalone conflict-resolution view over an
// in-progress operation: a list of conflicted files with per-file resolution
// plus continue/skip/abort. It is the view `rebase` enters when a rebase stops.
func runConflictResolver(cmd *cobra.Command, r *git.Runner, state *git.InProgressState) error {
	ctx := cmd.Context()
	sides := git.ResolveSides(ctx, r, state)

	load := func() ([]tui.Item, error) {
		files, err := git.ConflictedFiles(ctx, r)
		if err != nil {
			return nil, err
		}
		items := make([]tui.Item, len(files))
		for i, f := range files {
			items[i] = conflictItem{path: f}
		}
		return items, nil
	}

	items, err := load()
	if err != nil {
		return err
	}

	refresh := func() tea.Cmd {
		items, err := load()
		if err != nil {
			return tui.Status(err.Error())
		}
		return tui.SetItems(items)
	}

	res := &conflictResult{}
	ops := fileResolutionOps(ctx, r, sides, conflictPath, refresh)
	ops = append(ops, continueSkipAbortOps(ctx, r, state, res, refresh)...)

	list := tui.New(tui.Config{
		Title:      resolverTitle(ctx, r, state, sides),
		Columns:    []tui.Column{{Title: "conflicted file", MinWidth: 12, Flex: true, Density: tui.DensityShort}},
		Items:      items,
		Operations: ops,
	})
	p := tea.NewProgram(list, tea.WithAltScreen(), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		return err
	}

	if res.err != nil {
		return res.err
	}
	if res.status != "" {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), res.status)
		return err
	}
	return nil
}

// resolverTitle summarizes the stopped operation for the resolver header,
// including rebase branch/onto/progress when available.
func resolverTitle(ctx context.Context, r *git.Runner, state *git.InProgressState, sides git.ConflictSides) string {
	if state.Op == git.OpRebase {
		if p, err := git.ReadRebaseProgress(ctx, r); err == nil && p.Branch != "" {
			title := fmt.Sprintf("gint rebase — replaying %s onto %s", p.Branch, p.Onto)
			if p.Total > 0 {
				title += fmt.Sprintf(" (%d/%d)", p.Current, p.Total)
			}
			return title
		}
	}
	return fmt.Sprintf("gint %s — resolving conflicts (ours=%s, theirs=%s)", state.Op, sides.Ours, sides.Theirs)
}
