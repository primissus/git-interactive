// Package cli wires the Cobra command tree for gint.
package cli

import (
	"github.com/spf13/cobra"
)

// Execute builds the command tree and runs it against os.Args.
func Execute() error {
	root := newRootCmd()
	return root.Execute()
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "gint",
		Short:         "gint wraps common git operations in interactive TUI views",
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	root.AddCommand(
		newBranchCmd(),
		newWorktreeCmd(),
		newGraphBranchCmd(),
		newLogCmd(),
		newGraphCmd(),
		newRebaseCmd(),
		newMergeCmd(),
		newStatusCmd(),
		newAddCmd(),
		newStashCmd(),
		newCommitCmd(),
		newDemoCmd(),
	)

	return root
}
