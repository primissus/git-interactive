package cli

import (
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"git-interact/internal/tui"
)

// showInPager suspends the TUI and pipes content into $PAGER (less -R if
// unset), resuming the list once the pager exits. Backs every "paged diff
// view" operation (status/stash's `d`/"show diff").
func showInPager(content string) tea.Cmd {
	pagerCmd := os.Getenv("PAGER")
	if pagerCmd == "" {
		pagerCmd = "less -R"
	}
	fields := strings.Fields(pagerCmd)
	c := exec.Command(fields[0], fields[1:]...)
	c.Stdin = strings.NewReader(content)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return tui.Status(err.Error())()
		}
		return nil
	})
}
