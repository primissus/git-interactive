package cli

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"git-interact/internal/tui"
)

// newDemoCmd builds the hidden `gint _demo` command: the phase-2 acceptance
// harness that drives the shared TUI framework end to end with sample data. It
// is not part of the public command set.
func newDemoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "_demo",
		Short:  "Exercise the shared TUI framework (development harness)",
		Hidden: true,
	}
	attachCommonFlags(cmd)
	cmd.RunE = runDemo
	return cmd
}

func runDemo(cmd *cobra.Command, _ []string) error {
	flags := registeredFlags[cmd]
	density := densityFromFlags(flags)

	if !flags.resolveInteractive() {
		return tui.RenderTable(cmd.OutOrStdout(), tui.DemoColumns(), tui.DemoItems(), tui.TableOptions{
			Density: density,
			Header:  true,
			Marker:  true,
		})
	}

	model := tui.Demo(density)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(cmd.Context()))
	_, err := p.Run()
	return err
}

// densityFromFlags maps the -F/-s view flags onto a tui.Density. -F (full)
// wins over -s (short); neither yields the normal view.
func densityFromFlags(f *commonFlags) tui.Density {
	switch {
	case f.full:
		return tui.DensityFull
	case f.short:
		return tui.DensityShort
	default:
		return tui.DensityNormal
	}
}
