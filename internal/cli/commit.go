package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

// commitConfirmChoices is shared by the standalone `commit` command's Flow
// and status's inline commit/amend operations, so the two present identical
// options (PROMPT.md → commit).
func commitConfirmChoices() []tui.Choice {
	return []tui.Choice{
		{Label: "no", Value: "", Key: "n"},
		{Label: "yes", Value: "yes", Key: "y"},
		{Label: "amend", Value: "amend", Key: "a"},
		{Label: "stage all & commit", Value: "stage-all", Key: "s"},
		{Label: "commit & push", Value: "push", Key: "p"},
		{Label: "no-verify", Value: "no-verify", Key: "v"},
		{Label: "edit", Value: "edit", Key: "e"},
	}
}

// runCommitChoice executes one of commitConfirmChoices' outcomes against
// message. Shared by `gint commit` and status's inline commit/amend.
func runCommitChoice(ctx context.Context, r *git.Runner, message, choice string) (string, error) {
	switch choice {
	case "amend":
		pushed, err := amendWasPushed(ctx, r)
		if err != nil {
			return "", err
		}
		if err := git.AmendCommit(ctx, r, message, false); err != nil {
			return "", err
		}
		if pushed {
			return "amended (was already pushed — you'll need to force-push)", nil
		}
		return "amended", nil
	case "stage-all":
		if err := git.StageAll(ctx, r); err != nil {
			return "", err
		}
		if err := git.CommitStaged(ctx, r, message, false); err != nil {
			return "", err
		}
		return "staged all and committed", nil
	case "push":
		if err := git.CommitStaged(ctx, r, message, false); err != nil {
			return "", err
		}
		branch, err := git.CurrentBranch(ctx, r)
		if err != nil {
			return "", err
		}
		if err := git.PushBranch(ctx, r, branch); err != nil {
			return "", err
		}
		return "committed and pushed", nil
	case "no-verify":
		if err := git.CommitStaged(ctx, r, message, true); err != nil {
			return "", err
		}
		return "committed (no-verify)", nil
	case "yes":
		if err := git.CommitStaged(ctx, r, message, false); err != nil {
			return "", err
		}
		return "committed", nil
	default:
		return "", nil
	}
}

// amendWasPushed reports whether HEAD (the commit `amend` would rewrite) is
// already on a remote-tracking branch.
func amendWasPushed(ctx context.Context, r *git.Runner) (bool, error) {
	sha, err := git.RevParse(ctx, r, "HEAD")
	if err != nil {
		return false, err
	}
	return git.IsCommitPushed(ctx, r, sha)
}

func newCommitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "commit [message]",
		Aliases: []string{"cm"},
		Short:   "Commit staged changes",
		Args:    cobra.MaximumNArgs(1),
	}
	attachCommonFlags(cmd)
	cmd.Flags().BoolP("amend", "A", false, "amend the last commit")
	cmd.Flags().BoolP("no-verify", "V", false, "skip commit hooks")
	cmd.RunE = runCommit
	return cmd
}

func runCommit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	r := git.NewRunner("")
	amend, _ := cmd.Flags().GetBool("amend")
	noVerify, _ := cmd.Flags().GetBool("no-verify")
	flags := registeredFlags[cmd]

	// -A/-V are direct shortcuts (PROMPT.md → commit: "Shift+A"/"Shift+V …
	// also a direct shortcut"), bypassing the confirm flow entirely, same as
	// branch/worktree's flag-driven direct paths.
	if amend {
		message := ""
		if len(args) == 1 {
			message = args[0]
		}
		status, err := runCommitChoice(ctx, r, message, "amend")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), status)
		return err
	}

	if !flags.resolveInteractive() {
		if len(args) == 0 {
			return fmt.Errorf("commit -I requires a message")
		}
		choice := "yes"
		if noVerify {
			choice = "no-verify"
		}
		status, err := runCommitChoice(ctx, r, args[0], choice)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), status)
		return err
	}

	initial := ""
	if len(args) == 1 {
		initial = args[0]
	}
	flow := tui.NewFlow(tui.FlowConfig{
		Title:     "gint commit",
		Input:     &tui.InputSpec{Prompt: "Commit message", Initial: initial},
		Confirm:   tui.Confirm{Kind: tui.ConfirmChoice, Prompt: "Commit?", Choices: commitConfirmChoices()},
		EditValue: "edit",
		Run: func(message, choice string) (string, error) {
			return runCommitChoice(ctx, r, message, choice)
		},
	})
	return runFlow(cmd, flow)
}
