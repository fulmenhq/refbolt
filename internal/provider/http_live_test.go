package provider

import (
	"context"
	"testing"
)

func TestHTTPFetcher_StarlinkGuides_GettingStarted(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	cfg := ProviderConfig{
		Slug:          "starlink-api-v2-guides",
		Name:          "Starlink API v2 Guides",
		BaseURL:       "https://starlink.readme.io/docs",
		FetchStrategy: StrategyNative,
		Paths:         []string{"getting-started.md"},
	}
	f, err := NewHTTPFetcher(cfg)
	if err != nil {
		t.Fatal(err)
	}

	pages, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("len(pages) = %d, want 1", len(pages))
	}
	if len(pages[0].Content) < 100 {
		t.Fatalf("getting-started content suspiciously small: %d bytes", len(pages[0].Content))
	}
	t.Logf("Fetched getting-started.md: %d bytes", len(pages[0].Content))
}
