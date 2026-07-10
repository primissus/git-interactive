package cli

import "github.com/spf13/cobra"

// commonFlags holds the shared flags every gint command accepts, per PROMPT.md's
// "Shared interaction model" (common flags table).
type commonFlags struct {
	interactive   bool
	noInteractive bool
	sort          string
	full          bool
	short         bool
}

// registerCommonFlags attaches the shared flag set to a command and returns a
// pointer whose fields are populated once cobra parses args.
func registerCommonFlags(cmd *cobra.Command) *commonFlags {
	f := &commonFlags{}
	cmd.Flags().BoolVarP(&f.interactive, "interactive", "i", true, "interactive mode (default)")
	cmd.Flags().BoolVarP(&f.noInteractive, "no-interactive", "I", false, "print the tabular list instead of the interactive view")
	cmd.Flags().StringVarP(&f.sort, "sort", "S", "", "sort order")
	cmd.Flags().BoolVarP(&f.full, "full", "F", false, "full view: complete commit message, date, author name")
	cmd.Flags().BoolVarP(&f.short, "short", "s", false, "minimal view (e.g. branch name only)")
	return f
}

// resolveInteractive applies the "-I wins" rule: --no-interactive always
// forces non-interactive output regardless of --interactive's value.
func (f *commonFlags) resolveInteractive() bool {
	if f.noInteractive {
		return false
	}
	return f.interactive
}
