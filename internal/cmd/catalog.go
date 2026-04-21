package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/fulmenhq/refbolt/internal/config"
	"github.com/fulmenhq/refbolt/internal/provider"
	"github.com/spf13/cobra"
)

// catalogCmd is the parent for the read-only browse subcommands. It
// intentionally does nothing on its own — `cmd.Help()` prints on invocation
// without a subcommand.
var catalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Browse the embedded provider catalog",
	Long: `Read-only view into refbolt's built-in provider catalog, enriched with
metadata from the embedded registry where available. No network or filesystem
dependency — everything is baked into the binary.

Subcommands:
  list    Table or JSON listing of all catalog providers
  show    Full detail for a single provider by slug
  topics  Topic summary with provider counts

The --config flag is silently ignored on these subcommands; they always read
from the embedded catalog and registry.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var (
	listTopic    string
	listStrategy string
	listJSON     bool
)

var catalogListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all providers in the embedded catalog",
	Long: `List catalog providers as a human-readable table (default) or JSON
(with --json). Optionally filter by topic or fetch strategy.`,
	RunE: runCatalogList,
}

var catalogShowCmd = &cobra.Command{
	Use:   "show <slug>",
	Short: "Show full detail for a single provider",
	Args:  cobra.ExactArgs(1),
	RunE:  runCatalogShow,
}

var catalogTopicsCmd = &cobra.Command{
	Use:   "topics",
	Short: "List catalog topics with provider counts",
	RunE:  runCatalogTopics,
}

func init() {
	catalogListCmd.Flags().StringVar(&listTopic, "topic", "", "Filter by topic slug")
	catalogListCmd.Flags().StringVar(&listStrategy, "strategy", "", "Filter by fetch strategy")
	catalogListCmd.Flags().BoolVar(&listJSON, "json", false, "Emit JSON to stdout (envelope + providers array)")

	catalogCmd.AddCommand(catalogListCmd)
	catalogCmd.AddCommand(catalogShowCmd)
	catalogCmd.AddCommand(catalogTopicsCmd)
	rootCmd.AddCommand(catalogCmd)
}

// resetCatalogFlags zeroes the command-local package globals. Cobra reuses
// the same flag variables across Execute calls (our tests drive the full
// root command), so a test setting --topic=llm-api will bleed into the next
// test unless we reset explicitly. Called at the top of every RunE.
func resetCatalogFlags() {
	// No-op for RunE itself (cobra has already populated the flags), but
	// useful for PersistentPostRunE-style cleanup. Tests register their own
	// t.Cleanup to reset between cases.
}

func runCatalogList(cmd *cobra.Command, _ []string) error {
	var entries []config.CatalogEntry
	var err error

	switch {
	case listTopic != "" && listStrategy != "":
		byTopic, terr := config.ProvidersByTopic(listTopic)
		if terr != nil {
			return terr
		}
		// Apply strategy filter in-memory.
		if !isValidStrategy(listStrategy) {
			return config.ErrUnknownStrategy{Name: listStrategy, Valid: validStrategyList()}
		}
		for _, e := range byTopic {
			if string(e.Provider.FetchStrategy) == listStrategy {
				entries = append(entries, e)
			}
		}
	case listTopic != "":
		entries, err = config.ProvidersByTopic(listTopic)
	case listStrategy != "":
		entries, err = config.ProvidersByStrategy(listStrategy)
	default:
		entries, err = config.CatalogEntries()
	}
	if err != nil {
		return err
	}

	if listJSON {
		return writeListJSON(cmd.OutOrStdout(), entries)
	}
	return writeListTable(cmd.OutOrStdout(), cmd.ErrOrStderr(), entries)
}

func runCatalogShow(cmd *cobra.Command, args []string) error {
	slug := args[0]
	entry, err := config.CatalogEntryBySlug(slug)
	if err != nil {
		return err
	}
	return writeShowDetail(cmd.OutOrStdout(), entry)
}

func runCatalogTopics(cmd *cobra.Command, _ []string) error {
	summaries, err := config.TopicSummaries()
	if err != nil {
		return err
	}
	return writeTopicsTable(cmd.OutOrStdout(), cmd.ErrOrStderr(), summaries)
}

// ─── Formatters ────────────────────────────────────────────────────────────

func writeListTable(stdout, stderr io.Writer, entries []config.CatalogEntry) error {
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SLUG\tTOPIC\tSTRATEGY\tCRED\tPAGES")
	for _, e := range entries {
		cred := formatCredentials(config.ProviderCredentials(e.Provider))
		pages := "—"
		if e.Registry != nil && e.Registry.EstimatedPages > 0 {
			pages = fmt.Sprintf("~%d", e.Registry.EstimatedPages)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			e.Provider.Slug,
			e.TopicSlug,
			string(e.Provider.FetchStrategy),
			cred,
			pages,
		)
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	// Hint line describes the rendered result set — not the full catalog —
	// so filtered invocations report their own totals, not the embedded
	// catalog's. `catalog topics` is the right command for catalog-wide
	// totals.
	topicCount := distinctTopics(entries)
	fmt.Fprintf(stderr, "\n%d %s across %d %s. Use `refbolt catalog show <slug>` for details.\n",
		len(entries), pluralize(len(entries), "provider", "providers"),
		topicCount, pluralize(topicCount, "topic", "topics"))
	return nil
}

// distinctTopics returns the number of unique topic slugs referenced by the
// given entries. Used for rendering result-set summaries on filtered output.
func distinctTopics(entries []config.CatalogEntry) int {
	seen := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		seen[e.TopicSlug] = struct{}{}
	}
	return len(seen)
}

// pluralize picks singular vs plural based on n. Simple helper — kept local
// because we use it in two places and a library would be overkill.
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

// providerJSON is the per-provider shape in `catalog list --json`. Kept
// flat and explicit (rather than reusing provider.ProviderConfig) so the
// JSON contract is stable independent of internal struct reshuffling.
type providerJSON struct {
	Slug           string   `json:"slug"`
	Name           string   `json:"name"`
	Topic          string   `json:"topic"`
	BaseURL        string   `json:"base_url"`
	FetchStrategy  string   `json:"fetch_strategy"`
	Credentials    []string `json:"credentials"`
	EstimatedPages *int     `json:"estimated_pages"` // null when registry absent
	Description    *string  `json:"description"`     // null when registry absent
	EnabledDefault bool     `json:"enabled_default"`
}

type listEnvelope struct {
	Version        string         `json:"version"`
	TopicsTotal    int            `json:"topics_total"`
	ProvidersTotal int            `json:"providers_total"`
	Providers      []providerJSON `json:"providers"`
}

func writeListJSON(stdout io.Writer, entries []config.CatalogEntry) error {
	// topics_total and providers_total describe the rendered result set.
	// Filtered invocations (--topic / --strategy) therefore report their
	// own totals, not the embedded catalog's. Consumers that want catalog-
	// wide totals should run `catalog topics` (or `catalog list` without
	// filters) — keeps each call's envelope self-describing.
	out := listEnvelope{
		Version:        currentVersion(),
		TopicsTotal:    distinctTopics(entries),
		ProvidersTotal: len(entries),
		Providers:      make([]providerJSON, 0, len(entries)),
	}

	for _, e := range entries {
		creds := config.ProviderCredentials(e.Provider)
		if creds == nil {
			creds = []string{} // encode as [] not null for stable schema
		}
		pj := providerJSON{
			Slug:           e.Provider.Slug,
			Name:           e.Provider.Name,
			Topic:          e.TopicSlug,
			BaseURL:        e.Provider.BaseURL,
			FetchStrategy:  string(e.Provider.FetchStrategy),
			Credentials:    creds,
			EnabledDefault: e.Provider.IsEnabled(),
		}
		if e.Registry != nil {
			p := e.Registry.EstimatedPages
			pj.EstimatedPages = &p
			d := e.Registry.Description
			pj.Description = &d
		}
		out.Providers = append(out.Providers, pj)
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeShowDetail(stdout io.Writer, e config.CatalogEntry) error {
	p := e.Provider

	displayName := p.Name
	if displayName == "" {
		displayName = p.Slug
	}
	fmt.Fprintf(stdout, "%s — %s\n\n", p.Slug, displayName)

	fmt.Fprintf(stdout, "  Topic:           %s\n", topicLabel(e))
	fmt.Fprintf(stdout, "  Base URL:        %s\n", p.BaseURL)
	fmt.Fprintf(stdout, "  Strategy:        %s\n", p.FetchStrategy)
	if p.LLMSTxtURL != "" {
		fmt.Fprintf(stdout, "  llms_txt URL:    %s\n", p.LLMSTxtURL)
	}
	if p.GitHubRepo != "" {
		fmt.Fprintf(stdout, "  GitHub repo:     %s\n", p.GitHubRepo)
	}
	fmt.Fprintf(stdout, "  Credentials:     %s\n", formatCredentialsLong(config.ProviderCredentials(p)))
	if e.Registry != nil && e.Registry.EstimatedPages > 0 {
		fmt.Fprintf(stdout, "  Estimated:       ~%d pages\n", e.Registry.EstimatedPages)
	} else {
		fmt.Fprintf(stdout, "  Estimated:       —\n")
	}
	status := "enabled by default"
	if !p.IsEnabled() {
		status = "disabled by default"
	}
	fmt.Fprintf(stdout, "  Status:          %s\n", status)

	if len(p.Paths) > 0 {
		fmt.Fprintln(stdout, "\nSample paths:")
		// Show the first handful to keep `show` scannable; full list lives
		// in the catalog source for anyone who needs it.
		limit := 6
		if len(p.Paths) < limit {
			limit = len(p.Paths)
		}
		for _, path := range p.Paths[:limit] {
			fmt.Fprintf(stdout, "  - %s\n", path)
		}
		if len(p.Paths) > limit {
			fmt.Fprintf(stdout, "  … %d more\n", len(p.Paths)-limit)
		}
	}

	if e.Registry != nil && e.Registry.Description != "" {
		fmt.Fprintln(stdout, "\nDescription:")
		fmt.Fprintf(stdout, "  %s\n", e.Registry.Description)
	}

	fmt.Fprintln(stdout, "\nArchive output:")
	archivePath := filepath.Join("<archive_root>", e.TopicSlug, p.Slug)
	fmt.Fprintf(stdout, "  %s/<YYYY-MM-DD>/…\n", archivePath)
	fmt.Fprintf(stdout, "  %s/latest → <latest-date>\n", archivePath)

	return nil
}

func writeTopicsTable(stdout, stderr io.Writer, summaries []config.TopicSummary) error {
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TOPIC\tPROVIDERS\tDESCRIPTION")
	for _, s := range summaries {
		desc := s.Description
		if desc == "" {
			desc = "—"
		}
		fmt.Fprintf(tw, "%s\t%d\t%s\n", s.Slug, s.ProviderCount, desc)
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintf(stderr, "\n%d topics. Use `refbolt catalog list --topic <slug>` for providers in a topic.\n", len(summaries))
	return nil
}

// ─── Rendering helpers ─────────────────────────────────────────────────────

func topicLabel(e config.CatalogEntry) string {
	if e.TopicName != "" {
		return fmt.Sprintf("%s (%s)", e.TopicSlug, e.TopicName)
	}
	return e.TopicSlug
}

// formatCredentials renders the CRED column for the table view. Compact
// labels for the common cases, falls back to the env var name otherwise.
// Empty input renders as the em-dash sentinel.
func formatCredentials(vars []string) string {
	if len(vars) == 0 {
		return "—"
	}
	labels := make([]string, 0, len(vars))
	for _, v := range vars {
		labels = append(labels, shortCredentialLabel(v))
	}
	sort.Strings(labels)
	return strings.Join(labels, ",")
}

// formatCredentialsLong is the `show` variant — lists full env var names and
// a human-readable "none required" when empty.
func formatCredentialsLong(vars []string) string {
	if len(vars) == 0 {
		return "none required"
	}
	return strings.Join(vars, ", ")
}

func shortCredentialLabel(envVar string) string {
	switch envVar {
	case "JINA_API_KEY":
		return "JINA"
	case "GITHUB_TOKEN":
		return "GITHUB"
	default:
		return envVar
	}
}

// isValidStrategy / validStrategyList mirror internal/config/catalog.go's
// validation but are defined here to keep command-layer imports tight and
// avoid exporting internals that aren't needed elsewhere.
func isValidStrategy(s string) bool {
	for _, v := range validStrategyList() {
		if v == s {
			return true
		}
	}
	return false
}

func validStrategyList() []string {
	return []string{
		string(provider.StrategyNative),
		string(provider.StrategyJina),
		string(provider.StrategyAuto),
		string(provider.StrategyGitHubRaw),
		string(provider.StrategyHierarchical),
	}
}

// currentVersion returns the value baked in at build time. Used for the
// `version` field of the JSON envelope so script consumers can key output by
// the refbolt release it came from.
func currentVersion() string {
	// appVersion is populated via SetVersionInfo from cmd/refbolt/main.go.
	return appVersion
}

// Ensure errors.As branches compile against the exported error types.
var (
	_ error = config.ErrUnknownSlug{}
	_ error = config.ErrUnknownTopic{}
	_ error = config.ErrUnknownStrategy{}
)

// errIsUnknownSlug is a small helper used by tests when they assert the
// "did you mean" path was taken.
func errIsUnknownSlug(err error) bool {
	var unk config.ErrUnknownSlug
	return errors.As(err, &unk)
}
