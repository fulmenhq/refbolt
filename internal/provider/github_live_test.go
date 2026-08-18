package provider

import (
	"context"
	"testing"
)

func TestGitHubRawFetcher_SpaceXRocketsV4(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	cfg := ProviderConfig{
		Slug:           "spacex-rockets-v4",
		Name:           "SpaceX Rockets API (v4)",
		BaseURL:        "https://docs.spacexdata.com",
		FetchStrategy:  StrategyGitHubRaw,
		GitHubRepo:     "r-spacex/SpaceX-API",
		GitHubDocsPath: "docs/rockets/v4/",
		GitHubBranch:   "master",
		Paths:          []string{"**/*.md"},
	}
	f, err := NewGitHubRawFetcher(cfg)
	if err != nil {
		t.Fatal(err)
	}

	pages, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(pages) != 4 {
		t.Fatalf("len(pages) = %d, want 4", len(pages))
	}
	for _, p := range pages {
		if len(p.Content) == 0 {
			t.Fatalf("empty content for %s", p.Path)
		}
	}
	t.Logf("Fetched %d rocket API doc pages", len(pages))
}
