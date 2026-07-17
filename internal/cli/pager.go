package cli

import (
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"git-interact/internal/tui"
)

// showInPager suspends the TUI and pipes content into $PAGER (bat if
// installed and PAGER is unset, else less -R), resuming the list once the
// pager exits. Backs every "paged diff view" operation (status/stash's
// `d`/"show diff").
func showInPager(content string) tea.Cmd {
	pagerCmd := os.Getenv("PAGER")
	if pagerCmd == "" {
		pagerCmd = defaultPager()
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

// defaultPager picks the pager used when $PAGER is unset: bat (or its Debian/
// Ubuntu rename, batcat) when installed, syntax-highlighting the diff, else
// the previous less -R.
func defaultPager() string {
	for _, bin := range []string{"bat", "batcat"} {
		if _, err := exec.LookPath(bin); err == nil {
			return bin + " --paging=always --color=always --language=diff"
		}
	}
	return "less -R"
}
