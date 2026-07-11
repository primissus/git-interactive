package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newVersionCmd prints the build-stamped version alongside the resolved
// binary path. `gint --version` alone reports the version but not which
// binary answered — with multiple installs on $PATH (e.g. a `go install`
// under ~/go/bin shadowing a project-local build), that ambiguity is exactly
// what makes "I rebuilt but nothing changed" bugs hard to spot.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the gint version and the binary path that's running",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := os.Executable()
			if err != nil {
				path = "(unknown)"
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "gint %s\n  binary: %s\n", version, path)
			return err
		},
	}
}
