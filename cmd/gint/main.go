package main

import (
	"fmt"
	"os"

	"git-interact/internal/cli"
)

// version is stamped at build time via -ldflags "-X main.version=...".
// See the Makefile's build/install targets.
var version = "dev"

func main() {
	cli.SetVersion(version)
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
