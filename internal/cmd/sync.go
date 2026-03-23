package cmd

import (
	"fmt"

	"github.com/fulmenhq/refbolt/internal/archive"
	"github.com/fulmenhq/refbolt/internal/config"
	gitpkg "github.com/fulmenhq/refbolt/internal/git"
	"github.com/fulmenhq/refbolt/internal/provider"
	"github.com/spf13/cobra"
)

var (
	syncAll     bool
	gitCommit   bool
	gitPush     bool
	gitBranch   string
	gitTrailers []string
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Run archive sync for configured providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		if gitPush && !gitCommit {
			return fmt.Errorf("--git-push requires --git-commit")
		}

		topics := config.Topics()
		if len(topics) == 0 {
			fmt.Println("No topics configured.")
			return nil
		}

		archiveRoot := config.ArchiveRoot()
		writer := archive.NewWriter(archiveRoot)

		// Early git pre-flight: validate client and reject pre-existing dirt
		// before the sync writes anything. This ensures the commit message
		// accurately describes only changes produced by this sync invocation.
		var gc *gitpkg.Client
		if gitCommit {
			var err error
			gc, err = gitpkg.NewClient(archiveRoot)
			if err != nil {
				return fmt.Errorf("git pre-flight failed: %w", err)
			}

			dirt, err := gc.DirtyLines()
			if err != nil {
				return err
			}
			if dirt != "" {
				return fmt.Errorf("archive has pre-existing uncommitted changes; commit or stash them first so the sync commit message accurately reflects this run's changes:\n%s", dirt)
			}
		}

		var syncResults []gitpkg.SyncResult

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

				if written > 0 {
					syncResults = append(syncResults, gitpkg.SyncResult{
						TopicSlug:    topic.Slug,
						ProviderSlug: pc.Slug,
						FilesWritten: written,
					})
				}
			}
		}

		// Git operations (opt-in via --git-commit).
		// Pre-flight already ran above; gc is non-nil.
		if gitCommit && gc != nil {
			has, err := gc.HasChanges()
			if err != nil {
				return err
			}
			if !has {
				fmt.Println("Git: no changes in archive, skipping commit.")
				return nil
			}

			if err := gc.StageArchive(); err != nil {
				return err
			}

			msg := gitpkg.BuildCommitMessage(syncResults, archiveRoot, gitTrailers)
			if err := gc.Commit(msg); err != nil {
				return err
			}
			fmt.Println("Git: committed archive changes.")

			if gitPush {
				if err := gc.Push(gitBranch); err != nil {
					return err
				}
				if gitBranch != "" {
					fmt.Printf("Git: pushed to branch %s.\n", gitBranch)
				} else {
					fmt.Println("Git: pushed to remote.")
				}
			}
		}

		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncAll, "all", false, "Sync all configured providers")
	syncCmd.Flags().BoolVar(&gitCommit, "git-commit", false, "Stage archive changes and commit after sync")
	syncCmd.Flags().BoolVar(&gitPush, "git-push", false, "Push after commit (requires --git-commit)")
	syncCmd.Flags().StringVar(&gitBranch, "git-branch", "", "Remote branch to push to (default: current branch)")
	syncCmd.Flags().StringArrayVar(&gitTrailers, "git-trailer", nil, "Trailer line(s) to append to commit message (repeatable)")
	rootCmd.AddCommand(syncCmd)
}
