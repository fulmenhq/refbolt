package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	appVersion = "dev"
	appCommit  = "unknown"
	appBuild   = "unknown"
)

// SetVersionInfo is called from main to inject build-time values.
// Also wires cobra's built-in `--version` flag so `refbolt --version`
// works alongside the `refbolt version` subcommand (FA-111). Cobra's
// default `--version` template prints `<name> version <string>`; we
// override to match the subcommand's output for consistency.
func SetVersionInfo(version, commit, buildDate string) {
	appVersion = version
	appCommit = commit
	appBuild = buildDate

	rootCmd.Version = version
	rootCmd.SetVersionTemplate(
		fmt.Sprintf("refbolt %s (commit: %s, built: %s)\n", version, commit, buildDate),
	)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("refbolt %s (commit: %s, built: %s)\n", appVersion, appCommit, appBuild)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
