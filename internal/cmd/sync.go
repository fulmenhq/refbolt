package cmd

import (
	"fmt"

	"github.com/fulmenhq/fularchive/internal/archive"
	"github.com/fulmenhq/fularchive/internal/config"
	"github.com/fulmenhq/fularchive/internal/provider"
	"github.com/spf13/cobra"
)

var syncAll bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Run archive sync for configured providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		topics := config.Topics()
		if len(topics) == 0 {
			fmt.Println("No topics configured.")
			return nil
		}

		archiveRoot := config.ArchiveRoot()
		writer := archive.NewWriter(archiveRoot)

		for _, topic := range topics {
			fmt.Printf("Topic: %s\n", topic.Slug)
			for _, pc := range topic.Providers {
				if !pc.IsEnabled() {
					fmt.Printf("  %s: skipped (disabled)\n", pc.Slug)
					continue
				}
				fmt.Printf("  %s: fetching...\n", pc.Slug)

				fetcher, err := provider.NewFetcher(pc)
				if err != nil {
					fmt.Printf("  %s: error creating fetcher: %v\n", pc.Slug, err)
					continue
				}

				pages, err := fetcher.Fetch(cmd.Context())
				if err != nil {
					fmt.Printf("  %s: error fetching: %v\n", pc.Slug, err)
					continue
				}

				written, err := writer.Write(topic.Slug, pc.Slug, pages)
				if err != nil {
					fmt.Printf("  %s: error writing: %v\n", pc.Slug, err)
					continue
				}

				fmt.Printf("  %s: wrote %d files\n", pc.Slug, written)
			}
		}

		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncAll, "all", false, "Sync all configured providers")
	rootCmd.AddCommand(syncCmd)
}
