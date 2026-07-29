package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

// rebaseMarks holds the branch selector's two picks. They live outside the
// List so a declined confirmation can reopen the selector with the marks
// intact (PROMPT.md → rebase: "N … lets you reselect, don't clear selection").
type rebaseMarks struct {
	base   string
	target string
}

// role returns "base"/"target" when branch is marked, "" otherwise.
func (m *rebaseMarks) role(branch string) string {
	if branch == "" {
		return ""
	}
	switch branch {
	case m.base:
		return "base"
	case m.target:
		return "target"
	}
	return ""
}

// rebaseSelectItem is one branch row in the selector; the name cell carries
// the "(base)"/"(target)" mark prefix.
type rebaseSelectItem struct {
	b    git.Branch
	role string // "", "base", "target"
}

func (i rebaseSelectItem) Columns() []string {
	name := i.b.Name
	if i.role != "" {
		name = "(" + i.role + ") " + name
	} else {
		name = tui.FormatBranch(name)
	}
	return []string{name, i.b.Subject, tui.FormatDate(i.b.CommitUnix, i.b.CommitDate), tui.FormatAuthor(i.b.AuthorName)}
}
func (i rebaseSelectItem) FilterValue() string { return i.b.Name }
func (i rebaseSelectItem) Current() bool       { return i.b.Head }

// rebaseSelectItems wraps branches as selector rows, decorating marked ones.
func rebaseSelectItems(branches []git.Branch, marks *rebaseMarks) []tui.Item {
	items := make([]tui.Item, len(branches))
	for i, b := range branches {
		items[i] = rebaseSelectItem{b: b, role: marks.role(b.Name)}
	}
	return items
}

func targetRebaseBranch(items []tui.Item) (rebaseSelectItem, bool) {
	if len(items) != 1 {
		return rebaseSelectItem{}, false
	}
	it, ok := items[0].(rebaseSelectItem)
	return it, ok
}

// buildRebaseSelectOps returns the selector's operations: mark the highlighted
// branch as base/target (overwriting any previous mark, never equal to the
// other) and apply the selection.
func buildRebaseSelectOps(marks *rebaseMarks, branches []git.Branch, applied *bool) []tui.Operation {
	emit := func() tea.Cmd { return tui.SetItems(rebaseSelectItems(branches, marks)) }

	return tui.ApplyKeymap("rebase-select", []tui.Operation{
		{
			Name: "select base", Key: "B", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				it, ok := targetRebaseBranch(c.Items)
				if !ok {
					return tui.Status("select a branch first")
				}
				if marks.target == it.b.Name {
					return tui.Status("base and target must differ")
				}
				marks.base = it.b.Name
				return tea.Batch(tui.Status("base: "+it.b.Name), emit())
			},
		},
		{
			Name: "select target", Key: "T", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				it, ok := targetRebaseBranch(c.Items)
				if !ok {
					return tui.Status("select a branch first")
				}
				if marks.base == it.b.Name {
					return tui.Status("base and target must differ")
				}
				marks.target = it.b.Name
				return tea.Batch(tui.Status("target: "+it.b.Name), emit())
			},
		},
		{
			Name: "apply", Scope: tui.ScopeList,
			Run: func(tui.OpContext) tea.Cmd {
				if marks.base == "" || marks.target == "" {
					return tui.Status("select a base (B) and a target (T) first")
				}
				*applied = true
				return tea.Quit
			},
		},
	})
}

// runRebaseSelector runs the branch selector / confirmation loop: mark base
// and target, apply, confirm "Rebase <target> onto <base>?" — a declined
// confirm reopens the selector with the marks kept. In preview mode
// (--commits) apply returns the marks immediately, since nothing destructive
// needs confirming. It returns target == "" when the user quits without
// applying.
func runRebaseSelector(cmd *cobra.Command, r *git.Runner, preview bool) (target, base string, err error) {
	ctx := cmd.Context()
	branches, err := git.ListBranches(ctx, r)
	if err != nil {
		return "", "", err
	}
	marks := &rebaseMarks{}

	// Default the cursor to the current branch — the natural target candidate.
	cursor := 0
	for i, b := range branches {
		if b.Head {
			cursor = i
			break
		}
	}

	for {
		applied := false
		list := tui.New(tui.Config{
			Title:         "gint rebase — select base and target",
			Columns:       branchColumns(),
			Items:         rebaseSelectItems(branches, marks),
			Operations:    buildRebaseSelectOps(marks, branches, &applied),
			InitialCursor: cursor,
		})
		p := tea.NewProgram(list, tea.WithAltScreen(), tea.WithContext(ctx))
		if _, err := p.Run(); err != nil {
			return "", "", err
		}
		if !applied {
			return "", "", nil // quit without applying
		}
		if preview {
			return marks.target, marks.base, nil
		}
		ok, err := confirmRebaseMarks(cmd, marks)
		if err != nil {
			return "", "", err
		}
		if ok {
			return marks.target, marks.base, nil
		}
		// Declined: loop back into the selector with the marks kept.
	}
}

// confirmRebaseMarks asks "Rebase <target> onto <base>?" (default no) via the
// standalone Flow and reports whether the user accepted.
func confirmRebaseMarks(cmd *cobra.Command, marks *rebaseMarks) (bool, error) {
	confirmed := false
	flow := tui.NewFlow(tui.FlowConfig{
		Title: rebaseTitle(marks.target, marks.base),
		Confirm: tui.Confirm{
			Kind:   tui.ConfirmYesNo,
			Prompt: fmt.Sprintf("Rebase %s onto %s?", marks.target, marks.base),
		},
		Run: func(_, _ string) (string, error) {
			confirmed = true
			return "", nil
		},
	})
	p := tea.NewProgram(flow, tea.WithContext(cmd.Context()))
	if _, err := p.Run(); err != nil {
		return false, err
	}
	return confirmed, nil
}
