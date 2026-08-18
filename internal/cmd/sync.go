package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/fulmenhq/refbolt/internal/archive"
	"github.com/fulmenhq/refbolt/internal/config"
	gitpkg "github.com/fulmenhq/refbolt/internal/git"
	"github.com/fulmenhq/refbolt/internal/provider"
	syncpkg "github.com/fulmenhq/refbolt/internal/sync"
	"github.com/spf13/cobra"
)

var (
	syncAll          bool
	syncForce        bool
	providerSlugs    []string
	topicSlugs       []string
	excludeProviders []string
	gitCommit        bool
	gitPush          bool
	gitBranch        string
	gitTrailers      []string
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

		// Resolve which providers to sync.
		selected, err := resolveProviders(topics, syncAll, providerSlugs, topicSlugs, excludeProviders)
		if err != nil {
			return err
		}

		archiveRoot := config.ArchiveRoot()
		writer := archive.NewWriter(archiveRoot)

		// Early git pre-flight: validate client and reject pre-existing dirt
		// before the sync writes anything.
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

		for _, sp := range selected {
			fmt.Printf("Topic: %s\n", sp.topicSlug)

			// Compute config hash for this provider.
			cfgFields := syncpkg.ProviderConfigFields(
				sp.cfg.Slug, sp.cfg.BaseURL, string(sp.cfg.FetchStrategy),
				sp.cfg.LLMSTxtURL, sp.cfg.GitHubRepo, sp.cfg.GitHubDocsPath,
				sp.cfg.GitHubBranch, sp.cfg.Paths,
			)
			cfgHash := syncpkg.ConfigHash(cfgFields)

			metaPath := syncpkg.MetaPath(archiveRoot, sp.topicSlug, sp.cfg.Slug)
			existingMeta, metaErr := syncpkg.Read(metaPath)

			// Check for incremental skip (unless --force).
			if !syncForce {
				if metaErr != nil {
					fmt.Printf("  %s: warning reading metadata: %v\n", sp.cfg.Slug, metaErr)
				}

				if syncpkg.ShouldSkip(existingMeta, cfgHash) {
					// Config unchanged — check strategy-specific hints.
					skipped := false

					fetcher, fErr := provider.NewFetcher(sp.cfg)
					if fErr == nil {
						if hc, ok := fetcher.(provider.HintChecker); ok {
							hint, hErr := hc.CheckHints(cmd.Context())
							if hErr == nil {
								skipped = hintsMatch(existingMeta, hint)
							}
						}
					}

					if skipped {
						elapsed := time.Since(existingMeta.LastSync).Truncate(time.Minute)
						fmt.Printf("  %s: skipped (unchanged, last sync %s ago)\n", sp.cfg.Slug, elapsed)
						continue
					}
				}
			}

			fmt.Printf("  %s: fetching...\n", sp.cfg.Slug)

			fetcher, err := provider.NewFetcher(sp.cfg)
			if err != nil {
				fmt.Printf("  %s: error creating fetcher: %v\n", sp.cfg.Slug, err)
				continue
			}

			if inc, ok := fetcher.(provider.IncrementalFetcher); ok {
				latestDir := filepath.Join(archiveRoot, sp.topicSlug, sp.cfg.Slug, "latest")
				var prior provider.FetchHint
				if existingMeta != nil {
					prior = providerHintFromSync(existingMeta.Hint)
				}
				inc.SetIncrementalContext(latestDir, prior)
			}

			pages, err := fetcher.Fetch(cmd.Context())
			if err != nil {
				fmt.Printf("  %s: error fetching: %v\n", sp.cfg.Slug, err)
				continue
			}

			// Compute content hash from raw pages for metadata.
			contentHash := computeContentHash(pages)

			stat, err := writer.Write(sp.topicSlug, sp.cfg.Slug, pages)
			if err != nil {
				fmt.Printf("  %s: error writing: %v\n", sp.cfg.Slug, err)
				continue
			}

			// Console output with dedup stats.
			if stat.Skipped > 0 {
				fmt.Printf("  %s: %d files (%d unchanged, %d updated)\n",
					sp.cfg.Slug, stat.Total, stat.Skipped, stat.Written)
			} else {
				fmt.Printf("  %s: wrote %d files\n", sp.cfg.Slug, stat.Total)
			}

			// Write sync metadata when content changed, on cold start, or when
			// per-path hints were captured (enables conditional GET on next sync).
			if existingMeta == nil {
				existingMeta, _ = syncpkg.Read(metaPath)
			}
			isColdStart := existingMeta == nil

			capturedHint := captureFetchHint(cmd.Context(), fetcher)

			priorHint := provider.FetchHint{}
			if existingMeta != nil {
				priorHint = providerHintFromSync(existingMeta.Hint)
			}
			pathHintsChanged := len(capturedHint.PathHints) > 0 &&
				!provider.PathHintsMatch(priorHint.PathHints, capturedHint.PathHints)

			if stat.Written > 0 || isColdStart || pathHintsChanged {
				newMeta := &syncpkg.SyncMeta{
					Provider:    sp.cfg.Slug,
					Strategy:    string(sp.cfg.FetchStrategy),
					ConfigHash:  cfgHash,
					LastSync:    time.Now().UTC(),
					ContentHash: contentHash,
					FileCount:   stat.Total,
					Hint:        providerHintToSync(capturedHint),
				}
				if wErr := syncpkg.Write(metaPath, newMeta); wErr != nil {
					fmt.Printf("  %s: warning writing metadata: %v\n", sp.cfg.Slug, wErr)
				}
			}

			if stat.Written > 0 {
				syncResults = append(syncResults, gitpkg.SyncResult{
					TopicSlug:    sp.topicSlug,
					ProviderSlug: sp.cfg.Slug,
					FilesWritten: stat.Written,
				})
			}
		}

		// Git operations (opt-in via --git-commit).
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

// hintsMatch compares stored metadata hints against current upstream hints.
func hintsMatch(meta *syncpkg.SyncMeta, hint provider.FetchHint) bool {
	if len(hint.PathHints) > 0 {
		return provider.PathHintsMatch(providerHintFromSync(meta.Hint).PathHints, hint.PathHints)
	}
	// Tree SHA match (github-raw).
	if hint.TreeSHA != "" && meta.Hint.TreeSHA != "" {
		return hint.TreeSHA == meta.Hint.TreeSHA
	}
	// ETag match.
	if hint.ETag != "" && meta.Hint.ETag != "" {
		return hint.ETag == meta.Hint.ETag
	}
	// Content-Length + Last-Modified match (both must be present).
	if hint.ContentLength > 0 && meta.Hint.ContentLength > 0 &&
		hint.LastModified != "" && meta.Hint.LastModified != "" {
		return hint.ContentLength == meta.Hint.ContentLength &&
			hint.LastModified == meta.Hint.LastModified
	}
	// No comparable hints — cannot skip.
	return false
}

func captureFetchHint(ctx context.Context, fetcher provider.Fetcher) provider.FetchHint {
	var hint provider.FetchHint
	if hr, ok := fetcher.(provider.HintRecorder); ok {
		hint = hr.LastFetchHint()
	}
	if hc, ok := fetcher.(provider.HintChecker); ok {
		checked, err := hc.CheckHints(ctx)
		if err != nil {
			if hint.ETag != "" || len(hint.PathHints) > 0 {
				return hint
			}
			return provider.FetchHint{}
		}
		// Native providers with llms_txt_url plus supplemental paths record per-path
		// hints during Fetch but rely on the bulk llms ETag for provider-level skip.
		if len(hint.PathHints) > 0 && checked.ETag != "" && len(checked.PathHints) == 0 {
			hint.ETag = checked.ETag
			hint.LastModified = checked.LastModified
			hint.ContentLength = checked.ContentLength
			return hint
		}
		if len(hint.PathHints) > 0 || hint.ETag != "" {
			return hint
		}
		return checked
	}
	return hint
}

func providerHintFromSync(h syncpkg.FetchHint) provider.FetchHint {
	out := provider.FetchHint{
		ETag:          h.ETag,
		LastModified:  h.LastModified,
		ContentLength: h.ContentLength,
		TreeSHA:       h.TreeSHA,
	}
	if len(h.PathHints) > 0 {
		out.PathHints = make(map[string]provider.FetchHint, len(h.PathHints))
		for k, v := range h.PathHints {
			out.PathHints[k] = provider.FetchHint{
				ETag:          v.ETag,
				LastModified:  v.LastModified,
				ContentLength: v.ContentLength,
			}
		}
	}
	return out
}

func providerHintToSync(h provider.FetchHint) syncpkg.FetchHint {
	out := syncpkg.FetchHint{
		ETag:          h.ETag,
		LastModified:  h.LastModified,
		ContentLength: h.ContentLength,
		TreeSHA:       h.TreeSHA,
	}
	if len(h.PathHints) > 0 {
		out.PathHints = make(map[string]syncpkg.FetchHint, len(h.PathHints))
		for k, v := range h.PathHints {
			out.PathHints[k] = syncpkg.FetchHint{
				ETag:          v.ETag,
				LastModified:  v.LastModified,
				ContentLength: v.ContentLength,
			}
		}
	}
	return out
}

// computeContentHash hashes the concatenated content of all pages.
func computeContentHash(pages []provider.Page) string {
	var total int
	for _, p := range pages {
		total += len(p.Content)
	}
	buf := make([]byte, 0, total)
	for _, p := range pages {
		buf = append(buf, p.Content...)
	}
	return syncpkg.ContentHash(buf)
}

// selectedProvider pairs a provider config with its parent topic slug.
type selectedProvider struct {
	topicSlug string
	cfg       provider.ProviderConfig
}

// resolveProviders applies the selection semantics defined in FA-081:
//  1. Require at least one positive selector: --all, --topic, or --provider
//  2. Union positive selectors
//  3. enabled: false applies to --all and --topic only; --provider overrides it
//  4. --exclude-provider removes from the resolved set
//  5. Error on unknown slugs, conflicts, or empty result
func resolveProviders(
	topics []config.Topic,
	all bool,
	providerFlags, topicFlags, excludeFlags []string,
) ([]selectedProvider, error) {
	if !all && len(providerFlags) == 0 && len(topicFlags) == 0 {
		// Multi-line error with concrete next-step hints. The first line
		// is prefixed with "Error: " by cmd/refbolt/main.go; subsequent
		// lines (the hint list) indent with two spaces and render cleanly
		// in the terminal (FA-111 item #4).
		return nil, fmt.Errorf(`no providers selected. Choose one of:
  refbolt sync --all             # use built-in catalog
  refbolt sync --topic <slug>    # enable one topic
  refbolt sync --provider <slug> # enable specific providers
  refbolt catalog list           # browse available topics and providers`)
	}

	// Build lookup indexes.
	allProviderSlugs := make(map[string]bool)
	allTopicSlugs := make(map[string]bool)
	for _, t := range topics {
		allTopicSlugs[t.Slug] = true
		for _, p := range t.Providers {
			allProviderSlugs[p.Slug] = true
		}
	}

	// Validate unknown slugs.
	for _, s := range providerFlags {
		if !allProviderSlugs[s] {
			return nil, fmt.Errorf("unknown provider slug: %q", s)
		}
	}
	for _, s := range topicFlags {
		if !allTopicSlugs[s] {
			return nil, fmt.Errorf("unknown topic slug: %q", s)
		}
	}
	for _, s := range excludeFlags {
		if !allProviderSlugs[s] {
			return nil, fmt.Errorf("unknown provider slug in --exclude-provider: %q", s)
		}
	}

	// Check for conflicts: same slug in --provider and --exclude-provider.
	explicitSet := make(map[string]bool)
	for _, s := range providerFlags {
		explicitSet[s] = true
	}
	for _, s := range excludeFlags {
		if explicitSet[s] {
			return nil, fmt.Errorf("provider %q appears in both --provider and --exclude-provider", s)
		}
	}

	// Build the selected set.
	topicSet := make(map[string]bool)
	for _, s := range topicFlags {
		topicSet[s] = true
	}
	excludeSet := make(map[string]bool)
	for _, s := range excludeFlags {
		excludeSet[s] = true
	}

	seen := make(map[string]bool)
	var result []selectedProvider

	for _, t := range topics {
		for _, p := range t.Providers {
			if seen[p.Slug] {
				continue
			}

			selected := false

			// Explicit --provider always selects, ignoring enabled flag.
			if explicitSet[p.Slug] {
				selected = true
			}

			// --all or matching --topic selects if enabled.
			if !selected && (all || topicSet[t.Slug]) {
				if !p.IsEnabled() {
					continue
				}
				selected = true
			}

			if !selected {
				continue
			}

			// Apply exclusions.
			if excludeSet[p.Slug] {
				continue
			}

			seen[p.Slug] = true
			result = append(result, selectedProvider{
				topicSlug: t.Slug,
				cfg:       p,
			})
		}
	}

	if len(result) == 0 {
		var parts []string
		if all {
			parts = append(parts, "--all")
		}
		for _, s := range topicFlags {
			parts = append(parts, "--topic "+s)
		}
		for _, s := range providerFlags {
			parts = append(parts, "--provider "+s)
		}
		for _, s := range excludeFlags {
			parts = append(parts, "--exclude-provider "+s)
		}
		return nil, fmt.Errorf("no providers matched after filtering (%s)", strings.Join(parts, ", "))
	}

	return result, nil
}

func init() {
	syncCmd.Flags().BoolVar(&syncAll, "all", false, "Sync all configured providers")
	syncCmd.Flags().BoolVar(&syncForce, "force", false, "Force re-fetch all providers, ignoring cached metadata")
	syncCmd.Flags().StringArrayVar(&providerSlugs, "provider", nil, "Sync specific provider(s) by slug (repeatable)")
	syncCmd.Flags().StringArrayVar(&topicSlugs, "topic", nil, "Sync all providers in topic(s) (repeatable)")
	syncCmd.Flags().StringArrayVar(&excludeProviders, "exclude-provider", nil, "Exclude provider(s) from sync (repeatable)")
	syncCmd.Flags().BoolVar(&gitCommit, "git-commit", false, "Stage archive changes and commit after sync")
	syncCmd.Flags().BoolVar(&gitPush, "git-push", false, "Push after commit (requires --git-commit)")
	syncCmd.Flags().StringVar(&gitBranch, "git-branch", "", "Remote branch to push to (default: current branch)")
	syncCmd.Flags().StringArrayVar(&gitTrailers, "git-trailer", nil, "Trailer line(s) to append to commit message (repeatable)")
	rootCmd.AddCommand(syncCmd)
}
