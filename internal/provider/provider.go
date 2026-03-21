// Package provider defines the interface and types for documentation fetchers.
package provider

import (
	"context"
	"fmt"
)

// FetchStrategy controls how content is retrieved from a provider.
type FetchStrategy string

const (
	StrategyNative    FetchStrategy = "native"     // Direct .md or known Markdown endpoints.
	StrategyJina      FetchStrategy = "jina"       // Jina Reader HTML-to-Markdown conversion.
	StrategyAuto      FetchStrategy = "auto"       // Try native first, fall back to Jina.
	StrategyGitHubRaw FetchStrategy = "github-raw" // GitHub tree discovery + raw Markdown fetch.
)

// RateLimitConfig controls request pacing for a provider.
type RateLimitConfig struct {
	RequestsPerSecond float64 `yaml:"requests_per_second,omitempty"`
	Burst             int     `yaml:"burst,omitempty"`
}

// Page represents a single fetched documentation page.
type Page struct {
	// SourceURL is the original URL this page was fetched from.
	SourceURL string
	// Path is the relative path within the provider archive tree.
	// Example: "docs/guides/working-with-text.md"
	Path string
	// Content is the Markdown content of the page.
	Content []byte
}

// ProviderConfig holds the configuration for a single documentation provider,
// matching the schema in schemas/providers/v0/providers.schema.yaml.
type ProviderConfig struct {
	Slug           string           `yaml:"slug"`
	Name           string           `yaml:"name"`
	BaseURL        string           `yaml:"base_url"`
	Paths          []string         `yaml:"paths"`
	FetchStrategy  FetchStrategy    `yaml:"fetch_strategy"`
	LLMSTxtURL     string           `yaml:"llms_txt_url,omitempty"`
	OpenAPIURL     string           `yaml:"openapi_url,omitempty"`
	GitHubRepo     string           `yaml:"github_repo,omitempty"`
	GitHubDocsPath string           `yaml:"github_docs_path,omitempty"`
	GitHubBranch   string           `yaml:"github_branch,omitempty"`
	RateLimit      *RateLimitConfig `yaml:"rate_limit,omitempty"`
	AuthEnvVar     string           `yaml:"auth_env_var,omitempty"`
	Enabled        *bool            `yaml:"enabled,omitempty"`
}

// IsEnabled returns whether this provider is active (defaults to true).
func (c *ProviderConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// Fetcher retrieves documentation pages from a provider.
type Fetcher interface {
	// Name returns the human-readable provider name.
	Name() string

	// Fetch retrieves all pages matching the provider's path patterns.
	// Returns the fetched pages and any error encountered.
	Fetch(ctx context.Context) ([]Page, error)
}

// Registry maps provider slugs to their fetcher constructors.
var registry = map[string]func(cfg ProviderConfig) (Fetcher, error){}

// Register adds a fetcher constructor for a provider slug.
func Register(slug string, constructor func(cfg ProviderConfig) (Fetcher, error)) {
	registry[slug] = constructor
}

// NewFetcher creates a Fetcher for the given provider config.
// If a slug-specific constructor is registered, it is used; otherwise
// the default HTTP fetcher is returned.
func NewFetcher(cfg ProviderConfig) (Fetcher, error) {
	if cfg.FetchStrategy == StrategyGitHubRaw {
		return NewGitHubRawFetcher(cfg)
	}
	if constructor, ok := registry[cfg.Slug]; ok {
		return constructor(cfg)
	}
	return NewHTTPFetcher(cfg)
}

// FetchAll runs Fetch on each enabled provider config and returns all pages
// grouped by provider slug.
func FetchAll(ctx context.Context, configs []ProviderConfig) (map[string][]Page, error) {
	results := make(map[string][]Page)
	for _, cfg := range configs {
		if !cfg.IsEnabled() {
			continue
		}
		fetcher, err := NewFetcher(cfg)
		if err != nil {
			return nil, fmt.Errorf("creating fetcher for %s: %w", cfg.Slug, err)
		}
		pages, err := fetcher.Fetch(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching %s: %w", cfg.Slug, err)
		}
		results[cfg.Slug] = pages
	}
	return results, nil
}
