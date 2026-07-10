package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newShellInitCmd prints a shell wrapper function so that a worktree "checkout"
// actually changes the calling shell's directory. A gint subprocess cannot cd
// its parent, so the wrapper runs the real binary with a temp --cd-file, then
// cd's to whatever path the checkout wrote there. Non-worktree invocations pass
// straight through. Install with e.g. `eval "$(gint shell-init zsh)"`.
func newShellInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "shell-init [bash|zsh|fish]",
		Short:     "Print a shell wrapper so worktree checkout cd's your shell",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish"},
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := "bash"
			if len(args) == 1 {
				shell = args[0]
			}
			script, err := shellInitScript(shell)
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), script)
			return err
		},
	}
}

// shellInitScript returns the wrapper source for the given shell.
func shellInitScript(shell string) (string, error) {
	switch shell {
	case "bash", "zsh", "sh":
		return posixShellInit, nil
	case "fish":
		return fishShellInit, nil
	default:
		return "", fmt.Errorf("shell-init: unsupported shell %q (want bash, zsh, or fish)", shell)
	}
}

const posixShellInit = `# gint shell integration — add to your ~/.bashrc or ~/.zshrc:
#   eval "$(gint shell-init zsh)"
gint() {
	case "$1" in
	worktree | wt)
		local _gint_cd_file _gint_ret
		_gint_cd_file="$(mktemp "${TMPDIR:-/tmp}/gint-cd.XXXXXX")"
		command gint "$@" --cd-file "$_gint_cd_file"
		_gint_ret=$?
		if [ -s "$_gint_cd_file" ]; then
			cd "$(cat "$_gint_cd_file")" || _gint_ret=$?
		fi
		rm -f "$_gint_cd_file"
		return $_gint_ret
		;;
	*)
		command gint "$@"
		;;
	esac
}
`

const fishShellInit = `# gint shell integration — add to your ~/.config/fish/config.fish:
#   gint shell-init fish | source
function gint
	switch "$argv[1]"
	case worktree wt
		set -l _gint_cd_file (mktemp (test -n "$TMPDIR"; and echo $TMPDIR; or echo /tmp)/gint-cd.XXXXXX)
		command gint $argv --cd-file $_gint_cd_file
		set -l _gint_ret $status
		if test -s $_gint_cd_file
			cd (cat $_gint_cd_file)
		end
		rm -f $_gint_cd_file
		return $_gint_ret
	case '*'
		command gint $argv
	end
end
`
