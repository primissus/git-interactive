package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// stubRun prints what the real command would do once implemented. Phase 1
// only needs the Cobra skeleton; behavior lands in later phases.
func stubRun(name string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		flags := registeredFlags[cmd]
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "gint %s: not yet implemented (args=%v, interactive=%v, sort=%q, full=%v, short=%v)\n",
			name, args, flags.resolveInteractive(), flags.sort, flags.full, flags.short)
		return err
	}
}

// registeredFlags tracks each command's parsed common flags so stubRun can
// report them without every stub needing its own closure state.
var registeredFlags = map[*cobra.Command]*commonFlags{}

func attachCommonFlags(cmd *cobra.Command) {
	registeredFlags[cmd] = registerCommonFlags(cmd)
}

func newRebaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rebase",
		Aliases: []string{"reb"},
		Short:   "Interactive rebase with per-commit operations",
	}
	attachCommonFlags(cmd)
	cmd.RunE = stubRun("rebase")
	return cmd
}

func newMergeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "merge",
		Aliases: []string{"mrg"},
		Short:   "Merge branches",
	}
	attachCommonFlags(cmd)
	cmd.RunE = stubRun("merge")
	return cmd
}

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"st"},
		Short:   "Interactive status view",
	}
	attachCommonFlags(cmd)
	cmd.RunE = stubRun("status")
	return cmd
}

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Staging operations on the working tree",
	}
	attachCommonFlags(cmd)
	cmd.RunE = stubRun("add")
	return cmd
}

func newStashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stash",
		Aliases: []string{"sth"},
		Short:   "Interactive list of stashes",
	}
	attachCommonFlags(cmd)
	cmd.RunE = stubRun("stash")
	return cmd
}

func newCommitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "commit",
		Aliases: []string{"cm"},
		Short:   "Commit staged changes",
	}
	attachCommonFlags(cmd)
	cmd.RunE = stubRun("commit")
	return cmd
}
