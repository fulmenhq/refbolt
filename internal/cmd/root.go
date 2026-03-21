package cmd

import (
	"github.com/fulmenhq/fularchive/internal/config"
	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "fularchive",
	Short: "Archive web docs into clean, versioned Markdown trees",
	Long: `fularchive snapshots documentation sites (especially LLM APIs)
into date-versioned Markdown + JSON archives for offline use.`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return config.Load()
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
