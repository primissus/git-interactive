package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newVersionCmd prints the tracked semver release, the exact commit that
// build came from, and the resolved binary path. `gint --version` alone
// reports the semver but not which binary answered or what commit it was
// built at — with multiple installs on $PATH (e.g. a `go install` under
// ~/go/bin shadowing a project-local build), that ambiguity is exactly what
// makes "I rebuilt but nothing changed" bugs hard to spot.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the gint version, build commit, and binary path that's running",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := os.Executable()
			if err != nil {
				path = "(unknown)"
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "gint %s (commit %s)\n  binary: %s\n", version, commit, path)
			return err
		},
	}
}
