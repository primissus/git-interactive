package cli

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

// rebaseStepItem adapts a *git.RebaseStep to tui.Item. It holds a pointer so
// the op-setting operations mutate the shared plan in place; re-emitting the
// items then re-renders the changed op. Columns: chosen op, sha, subject.
type rebaseStepItem struct {
	step   *git.RebaseStep
	isHead bool
}

func (i rebaseStepItem) Columns() []string {
	subject := i.step.Commit.Subject
	if i.step.Op == git.RebaseReword && i.step.Message != "" {
		subject = i.step.Message + "  (was: " + i.step.Commit.Subject + ")"
	}
	return []string{string(i.step.Op), i.step.Commit.ShortSHA, subject}
}
func (i rebaseStepItem) FilterValue() string { return i.step.Commit.Subject }
func (i rebaseStepItem) Current() bool       { return i.isHead }

func rebaseColumns() []tui.Column {
	return []tui.Column{
		{Title: "op", MinWidth: 6, Density: tui.DensityShort, Color: tui.ColorName},
		{Title: "sha", MinWidth: 7, Density: tui.DensityShort, Color: tui.ColorSHA},
		{Title: "message", MinWidth: 12, Flex: true, Density: tui.DensityShort},
	}
}

// stepItems wraps the plan's steps as list rows; the first (newest) commit is
// HEAD and gets the current-item marker.
func stepItems(steps []*git.RebaseStep) []tui.Item {
	items := make([]tui.Item, len(steps))
	for i, s := range steps {
		items[i] = rebaseStepItem{step: s, isHead: i == 0}
	}
	return items
}

func targetStep(items []tui.Item) (*git.RebaseStep, bool) {
	if len(items) != 1 {
		return nil, false
	}
	it, ok := items[0].(rebaseStepItem)
	if !ok {
		return nil, false
	}
	return it.step, true
}

// rebaseOutcome records what the plan view did after it exits, so runRebase can
// route to the conflict resolver, report an error, or print success.
type rebaseOutcome struct {
	submitted bool
	err       error
}

func newRebaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rebase [target] [base]",
		Aliases: []string{"reb"},
		Short:   "Interactive rebase with per-commit operations",
		Args:    cobra.MaximumNArgs(2),
	}
	attachCommonFlags(cmd)
	cmd.Flags().Bool("commits", false, "preview the commits in the rebase range (like `gint log`), then exit")
	cmd.RunE = runRebase
	return cmd
}

// rebaseTitle names the rebase in views and prompts: target first, then base.
func rebaseTitle(target, base string) string {
	return fmt.Sprintf("gint rebase %s onto %s", target, base)
}

// resolveDefaultBase returns the base for the one-arg form: the current branch
// name, or HEAD's sha when detached.
func resolveDefaultBase(ctx context.Context, r *git.Runner) (string, error) {
	if b, err := git.CurrentBranch(ctx, r); err == nil && b != "" {
		return b, nil
	}
	return git.RevParse(ctx, r, "HEAD")
}

// validateRebaseBranch checks that name is an existing local branch.
func validateRebaseBranch(ctx context.Context, r *git.Runner, name string) error {
	exists, err := git.BranchExists(ctx, r, name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("rebase: no such branch %q", name)
	}
	return nil
}

func runRebase(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	r := git.NewRunner("")

	// Never start a second rebase: if one is already in progress, resume it.
	if state, err := git.DetectInProgress(ctx, r); err == nil && state != nil && state.Op == git.OpRebase {
		return runConflictResolver(cmd, r, state)
	}

	flags := registeredFlags[cmd]
	preview, _ := cmd.Flags().GetBool("commits")

	// Resolve target and base: explicit args, or the branch selector. The arg
	// order is target-first (the inverse of `git rebase <base> <branch>`); a
	// single target rebases onto the current branch.
	var target, base string
	switch len(args) {
	case 1, 2:
		target = args[0]
		if err := validateRebaseBranch(ctx, r, target); err != nil {
			return err
		}
		if len(args) == 2 {
			base = args[1]
			if err := validateRebaseBranch(ctx, r, base); err != nil {
				return err
			}
		} else {
			var err error
			base, err = resolveDefaultBase(ctx, r)
			if err != nil {
				return err
			}
		}
		if target == base {
			return fmt.Errorf("rebase: base and target must differ (both resolve to %q)", target)
		}
	default:
		if !flags.resolveInteractive() {
			return fmt.Errorf("rebase -I requires a target branch")
		}
		t, b, err := runRebaseSelector(cmd, r, preview)
		if err != nil {
			return err
		}
		if t == "" {
			return nil // selector aborted before applying
		}
		target, base = t, b
	}

	if preview {
		return runCommitsPreview(cmd, r, target, base, flags)
	}

	startBranch, _ := git.CurrentBranch(ctx, r)
	plan, err := git.PlanRebaseRange(ctx, r, target, base)
	if err != nil {
		return err
	}
	if len(plan.Commits) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "nothing to rebase")
		return err
	}

	steps := make([]*git.RebaseStep, len(plan.Commits))
	for i, c := range plan.Commits {
		steps[i] = &git.RebaseStep{Commit: c, Op: git.RebasePick}
	}

	if !flags.resolveInteractive() {
		return tui.RenderTable(cmd.OutOrStdout(), rebaseColumns(), stepItems(steps), tui.TableOptions{
			Density: densityFromFlags(flags),
			Header:  true,
			Marker:  true,
		})
	}

	outcome := &rebaseOutcome{}
	list := tui.New(tui.Config{
		Title:      rebaseTitle(target, base),
		Columns:    rebaseColumns(),
		Items:      stepItems(steps),
		Operations: buildRebasePlanOps(ctx, r, plan, steps, outcome),
		Density:    densityFromFlags(flags),
	})
	p := tea.NewProgram(list, tea.WithAltScreen(), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		return err
	}

	if !outcome.submitted {
		return nil // quit without submitting
	}
	// The rebase ran. If it stopped (conflict or an `edit`), hand off to the
	// shared resolver; otherwise report the outcome.
	if state, err := git.DetectInProgress(ctx, r); err == nil && state != nil && state.Op == git.OpRebase {
		return runConflictResolver(cmd, r, state)
	}
	if outcome.err != nil {
		return outcome.err
	}
	msg := "rebased " + plural2(len(steps))
	if target != startBranch {
		msg += " — now on " + target
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), msg)
	return err
}

// runCommitsPreview shows the commits in base..target exactly like `gint log`
// — an interactive read-only list, or a plain table under -I — then exits.
func runCommitsPreview(cmd *cobra.Command, r *git.Runner, target, base string, flags *commonFlags) error {
	ctx := cmd.Context()
	items, err := loadCommitRangeItems(ctx, r, base+".."+target, flags.full)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "nothing to rebase")
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
		Title:   rebaseTitle(target, base) + " (preview)",
		Columns: logColumns(flags.full),
		Items:   items,
		Density: densityFromFlags(flags),
	})
	p := tea.NewProgram(list, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err = p.Run()
	return err
}

// buildRebasePlanOps returns the planning view's operations: one per rebase op
// (pick/reword/edit/squash/fixup/drop) that mutates the highlighted commit's
// choice, plus a submit that confirms then runs the whole plan.
func buildRebasePlanOps(ctx context.Context, r *git.Runner, plan git.RebasePlan, steps []*git.RebaseStep, outcome *rebaseOutcome) []tui.Operation {
	setOp := func(op git.RebaseOp) func(tui.OpContext) tea.Cmd {
		return func(c tui.OpContext) tea.Cmd {
			step, ok := targetStep(c.Items)
			if !ok {
				return tui.Status("select a commit first")
			}
			step.Op = op
			step.Message = ""
			return tui.SetItems(stepItems(steps))
		}
	}

	ops := []tui.Operation{
		{Name: "pick", Key: "p", Scope: tui.ScopeItem, Run: setOp(git.RebasePick)},
		{
			Name: "reword", Key: "r", Scope: tui.ScopeItem,
			Input: &tui.InputSpec{Prompt: "New commit subject", Placeholder: "new subject line (pick keeps the original)"},
			Run: func(c tui.OpContext) tea.Cmd {
				step, ok := targetStep(c.Items)
				if !ok {
					return tui.Status("select a commit first")
				}
				step.Op = git.RebaseReword
				step.Message = c.Input
				return tui.SetItems(stepItems(steps))
			},
		},
		{Name: "edit", Key: "e", Scope: tui.ScopeItem, Run: setOp(git.RebaseEdit)},
		{Name: "squash", Key: "s", Scope: tui.ScopeItem, Run: setOp(git.RebaseSquash)},
		{Name: "fixup", Key: "f", Scope: tui.ScopeItem, Run: setOp(git.RebaseFixup)},
		{Name: "drop", Key: "x", Scope: tui.ScopeItem, Run: setOp(git.RebaseDrop)},
		{
			Name: "submit", Key: "S", Scope: tui.ScopeList,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Run the rebase with the chosen operations?"},
			Run: func(tui.OpContext) tea.Cmd {
				concrete := make([]git.RebaseStep, len(steps))
				for i, s := range steps {
					concrete[i] = *s
				}
				if err := git.ValidateRebaseSteps(concrete); err != nil {
					return tui.Status(err.Error())
				}
				outcome.submitted = true
				outcome.err = git.RunRebasePlan(ctx, r, plan, concrete)
				return tea.Quit
			},
		},
	}
	return tui.ApplyKeymap("rebase", ops)
}
