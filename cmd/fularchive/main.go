package main

import (
	"fmt"
	"os"

	"github.com/fulmenhq/fularchive/internal/cmd"
)

// Set via -ldflags at build time (see Makefile).
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, buildDate)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
