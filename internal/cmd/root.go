package cmd

import (
	"github.com/fulmenhq/refbolt/internal/config"
	"github.com/spf13/cobra"
)

var (
	verbose    bool
	configFlag string
)

var rootCmd = &cobra.Command{
	Use:   "refbolt",
	Short: "Archive web docs into clean, versioned Markdown trees",
	Long: `refbolt snapshots documentation sites (especially LLM APIs)
into date-versioned Markdown + JSON archives for offline use.`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Commands that operate purely on embedded data skip user-config
		// loading entirely. init/version self-manage their lifecycle; the
		// catalog subcommands (`catalog list/show/topics`) are a read-only
		// view over the embedded catalog and registry, so a user config is
		// irrelevant — and the --config flag is silently ignored on them
		// (tested in internal/cmd/catalog_test.go) for ergonomic consistency.
		if skipsConfigLoad(cmd) {
			return nil
		}

		strict := cmd.Name() == "validate"
		resolved := config.ResolveConfigPath(configFlag)

		return config.Load(config.LoadOptions{
			ConfigPath:  resolved,
			Strict:      strict,
			UseEmbedded: resolved == "",
		})
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&configFlag, "config", "", "Path to providers config file")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// skipsConfigLoad returns true when the invoked command should NOT trigger
// user-config resolution + loading in PersistentPreRunE. Walks the command
// ancestry so subcommands (e.g. `refbolt catalog list`) are matched by
// their parent, not by cmd.Name() which reports the leaf.
func skipsConfigLoad(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "init", "version", "catalog":
			return true
		}
	}
	return false
}
