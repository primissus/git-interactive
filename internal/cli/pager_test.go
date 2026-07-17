package cli

import (
	"strings"
	"testing"
)

func TestDefaultPager(t *testing.T) {
	got := defaultPager()
	if got != "less -R" &&
		!strings.HasPrefix(got, "bat --paging=always --color=always --language=diff") &&
		!strings.HasPrefix(got, "batcat --paging=always --color=always --language=diff") {
		t.Errorf("defaultPager() = %q, want less -R or a bat/batcat command", got)
	}
}
