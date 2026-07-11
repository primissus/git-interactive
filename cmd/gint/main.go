package main

import (
	"fmt"
	"os"

	"git-interact/internal/cli"
)

// version and commit are stamped at build time via -ldflags "-X main.version=...
// -X main.commit=...". See the Makefile's build/install targets. version is the
// tracked semver release (from the VERSION file); commit pins the exact source
// state, distinguishing builds that share a version.
var (
	version = "0.0.0-dev"
	commit  = "unknown"
)

func main() {
	cli.SetVersion(version)
	cli.SetCommit(commit)
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
