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
func SetVersionInfo(version, commit, buildDate string) {
	appVersion = version
	appCommit = commit
	appBuild = buildDate
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("fularchive %s (commit: %s, built: %s)\n", appVersion, appCommit, appBuild)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
