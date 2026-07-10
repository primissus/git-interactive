package cli

import (
	"github.com/spf13/cobra"
)

// registeredFlags tracks each command's parsed common flags so command RunE
// handlers can read them without every command threading its own closure state.
var registeredFlags = map[*cobra.Command]*commonFlags{}

func attachCommonFlags(cmd *cobra.Command) {
	registeredFlags[cmd] = registerCommonFlags(cmd)
}
