package config

import (
	"errors"
	"fmt"
	"sort"

	"github.com/fulmenhq/refbolt/internal/provider"
)

// CatalogEntry is a provider from the embedded catalog, joined with its
// containing topic and any matching entry from the embedded registry.
// Registry is nil when no registry line shares the same slug — command code
// must treat this as "enrichment unavailable", not an error.
type CatalogEntry struct {
	Provider  provider.ProviderConfig
	TopicSlug string
	TopicName string // falls back to TopicSlug at display time if empty
	Registry  *RegistryEntry
}

// ErrUnknownSlug is returned when a slug isn't in the catalog. Suggested
// carries the closest-match catalog slug (empty when no good suggestion
// exists) so the CLI can print "Did you mean ...?" without re-computing.
type ErrUnknownSlug struct {
	Slug      string
	Suggested string
}

func (e ErrUnknownSlug) Error() string {
	if e.Suggested != "" {
		return fmt.Sprintf("unknown provider %q (did you mean %q?)", e.Slug, e.Suggested)
	}
	return fmt.Sprintf("unknown provider %q", e.Slug)
}

// ErrUnknownTopic mirrors ErrUnknownSlug but for topic filters.
type ErrUnknownTopic struct {
	Slug  string
	Valid []string
}

func (e ErrUnknownTopic) Error() string {
	return fmt.Sprintf("unknown topic %q (valid: %v)", e.Slug, e.Valid)
}

// ErrUnknownStrategy surfaces bad --strategy filter values.
type ErrUnknownStrategy struct {
	Name  string
	Valid []string
}

func (e ErrUnknownStrategy) Error() string {
	return fmt.Sprintf("unknown strategy %q (valid: %v)", e.Name, e.Valid)
}

// validStrategies lists the fetch strategies the catalog command recognizes
// for --strategy filtering. Kept in sync with the provider package's
// FetchStrategy enum (internal/provider/provider.go).
func validStrategies() []string {
	return []string{
		string(provider.StrategyNative),
		string(provider.StrategyJina),
		string(provider.StrategyAuto),
		string(provider.StrategyGitHubRaw),
		string(provider.StrategyHierarchical),
	}
}

// CatalogEntries loads the embedded catalog and registry and returns every
// catalog provider as a CatalogEntry, sorted alphabetically by slug. Registry
// lookup failures are downgraded to "no enrichment available" — a corrupted
// registry should never prevent the command from listing the catalog.
func CatalogEntries() ([]CatalogEntry, error) {
	topics, err := CatalogTopics()
	if err != nil {
		return nil, err
	}
	registry, _ := LoadRegistry() // graceful: empty map on error

	var out []CatalogEntry
	for _, t := range topics {
		for _, p := range t.Providers {
			entry := CatalogEntry{
				Provider:  p,
				TopicSlug: t.Slug,
				TopicName: t.Name,
			}
			if r, ok := registry[p.Slug]; ok {
				rCopy := r
				entry.Registry = &rCopy
			}
			out = append(out, entry)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Provider.Slug < out[j].Provider.Slug
	})
	return out, nil
}

// CatalogEntryBySlug returns a single entry, or ErrUnknownSlug (with a
// best-match suggestion) when the slug isn't in the catalog.
func CatalogEntryBySlug(slug string) (CatalogEntry, error) {
	entries, err := CatalogEntries()
	if err != nil {
		return CatalogEntry{}, err
	}
	for _, e := range entries {
		if e.Provider.Slug == slug {
			return e, nil
		}
	}

	// Build suggestion from catalog slugs.
	slugs := make([]string, 0, len(entries))
	for _, e := range entries {
		slugs = append(slugs, e.Provider.Slug)
	}
	return CatalogEntry{}, ErrUnknownSlug{Slug: slug, Suggested: suggestSlug(slug, slugs)}
}

// ProvidersByTopic returns catalog entries in the given topic. Unknown topics
// return ErrUnknownTopic with the list of valid topic slugs — keeps the CLI
// error message honest about what's actually available.
func ProvidersByTopic(topicSlug string) ([]CatalogEntry, error) {
	entries, err := CatalogEntries()
	if err != nil {
		return nil, err
	}

	// Collect the set of valid topic slugs as we go; if we don't find a
	// match, we emit ErrUnknownTopic carrying the valid list.
	seen := map[string]struct{}{}
	var filtered []CatalogEntry
	for _, e := range entries {
		seen[e.TopicSlug] = struct{}{}
		if e.TopicSlug == topicSlug {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == 0 {
		valid := make([]string, 0, len(seen))
		for s := range seen {
			valid = append(valid, s)
		}
		sort.Strings(valid)
		return nil, ErrUnknownTopic{Slug: topicSlug, Valid: valid}
	}
	return filtered, nil
}

// ProvidersByStrategy filters catalog entries by fetch_strategy. Unknown
// strategies return ErrUnknownStrategy.
func ProvidersByStrategy(name string) ([]CatalogEntry, error) {
	valid := validStrategies()
	known := false
	for _, v := range valid {
		if v == name {
			known = true
			break
		}
	}
	if !known {
		return nil, ErrUnknownStrategy{Name: name, Valid: valid}
	}

	entries, err := CatalogEntries()
	if err != nil {
		return nil, err
	}
	var out []CatalogEntry
	for _, e := range entries {
		if string(e.Provider.FetchStrategy) == name {
			out = append(out, e)
		}
	}
	return out, nil
}

// TopicSummary carries everything `refbolt catalog topics` needs to render
// a single row.
type TopicSummary struct {
	Slug          string
	Name          string
	Description   string
	ProviderCount int
}

// TopicSummaries returns the catalog's topics sorted alphabetically by slug.
// Provider counts reflect the catalog (not the registry).
func TopicSummaries() ([]TopicSummary, error) {
	topics, err := CatalogTopics()
	if err != nil {
		return nil, err
	}
	out := make([]TopicSummary, 0, len(topics))
	for _, t := range topics {
		out = append(out, TopicSummary{
			Slug:          t.Slug,
			Name:          t.Name,
			Description:   t.Description,
			ProviderCount: len(t.Providers),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Slug < out[j].Slug
	})
	return out, nil
}

// suggestSlug returns the closest-match slug from `candidates`, using a small
// Levenshtein-distance search with a conservative threshold. Empty when no
// candidate is close enough to be worth suggesting.
func suggestSlug(input string, candidates []string) string {
	if input == "" || len(candidates) == 0 {
		return ""
	}
	// Prefer prefix match when it's unambiguous (e.g., "anthr" → "anthropic").
	var prefixHits []string
	for _, c := range candidates {
		if len(input) > 0 && len(c) > len(input) && c[:len(input)] == input {
			prefixHits = append(prefixHits, c)
		}
	}
	if len(prefixHits) == 1 {
		return prefixHits[0]
	}

	best := ""
	bestDist := -1
	// Threshold scales lightly with input length; stays small so we don't
	// suggest wildly unrelated slugs.
	threshold := 2
	if len(input) > 8 {
		threshold = 3
	}
	for _, c := range candidates {
		d := levenshtein(input, c)
		if d <= threshold && (bestDist == -1 || d < bestDist) {
			best = c
			bestDist = d
		}
	}
	return best
}

// levenshtein computes edit distance between two strings. Iterative DP;
// small enough for our ~25-provider catalog that allocations don't matter.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min3(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

// Sanity check at package init: the strategy enum list in validStrategies()
// should match the provider package's declared strategies. If a future
// strategy is added without updating this file, fail loudly at startup rather
// than silently rejecting a valid --strategy flag.
func init() {
	known := map[string]bool{}
	for _, s := range validStrategies() {
		known[s] = true
	}
	for _, declared := range []provider.FetchStrategy{
		provider.StrategyNative,
		provider.StrategyJina,
		provider.StrategyAuto,
		provider.StrategyGitHubRaw,
		provider.StrategyHierarchical,
	} {
		if !known[string(declared)] {
			panic(errors.New("config.validStrategies() is out of sync with provider.FetchStrategy"))
		}
	}
}
