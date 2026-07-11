// Package cli wires the Cobra command tree for gint.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"git-interact/internal/git"
)

// version is the tracked semver release (from the repo's VERSION file); commit
// is the git ref that build came from, pinning a build down when two builds
// share a version. Both are overridden via -ldflags at build time (see
// Makefile); they default to placeholders for `go run`/plain builds.
var (
	version = "0.0.0-dev"
	commit  = "unknown"
)

// SetVersion overrides the reported version. main passes the ldflags-stamped
// value through here so the cli package owns the --version wiring.
func SetVersion(v string) {
	if v != "" {
		version = v
	}
}

// SetCommit overrides the reported build commit, same wiring as SetVersion.
func SetCommit(c string) {
	if c != "" {
		commit = c
	}
}

// Execute builds the command tree and runs it against os.Args.
func Execute() error {
	root := newRootCmd()
	return root.Execute()
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "gint",
		Short:   "gint wraps common git operations in interactive TUI views",
		Version: version,
		// main prints the returned error and sets the exit code, so cobra must
		// not also print it — otherwise every failure shows up twice.
		SilenceUsage:  true,
		SilenceErrors: true,
		// Every gint command operates on a repository; fail early with a clear
		// message when run outside one, instead of surfacing a raw git error
		// from deep in a command. The demo command uses canned data, so it is
		// exempt.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Name() == "demo" || cmd.Name() == "shell-init" || cmd.Name() == "version" {
				return nil
			}
			if _, err := git.NewRunner("").Run(cmd.Context(), "rev-parse", "--git-dir"); err != nil {
				return fmt.Errorf("not a git repository — run gint inside a git repo")
			}
			return nil
		},
	}
	root.SetVersionTemplate("gint {{.Version}}\n")

	root.AddCommand(
		newBranchCmd(),
		newTagsCmd(),
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
		newShellInitCmd(),
		newVersionCmd(),
	)

	return root
}
