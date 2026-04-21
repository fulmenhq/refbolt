package main

import (
	"fmt"
	"os"

	"github.com/fulmenhq/refbolt/assets"
	"github.com/fulmenhq/refbolt/internal/cmd"
	"github.com/fulmenhq/refbolt/internal/config"
)

// Set via -ldflags at build time (see Makefile).
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, buildDate)
	config.SetEmbeddedAssets(assets.Catalog, assets.Schema)
	config.SetEmbeddedRegistry(assets.Registry)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
