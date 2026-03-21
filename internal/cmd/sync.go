package cmd

import (
	"fmt"

	"github.com/fulmenhq/fularchive/internal/config"
	"github.com/spf13/cobra"
)

var syncAll bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Run archive sync for configured providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		providers := config.Providers()
		if len(providers) == 0 {
			fmt.Println("No providers configured.")
			return nil
		}
		fmt.Println("Syncing providers:", providers)
		// TODO: implement provider fetch loop
		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncAll, "all", false, "Sync all configured providers")
	rootCmd.AddCommand(syncCmd)
}
