package provider

import (
	"context"
	"testing"
)

func TestHTTPFetcher_XAI_LLMSTxt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	cfg := ProviderConfig{
		Slug:       "xai",
		Name:       "xAI",
		BaseURL:    "https://docs.x.ai",
		LLMSTxtURL: "https://docs.x.ai/llms.txt",
	}
	f, err := NewHTTPFetcher(cfg)
	if err != nil {
		t.Fatal(err)
	}

	pages, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Got %d pages", len(pages))
	if len(pages) < 10 {
		t.Errorf("Expected many pages from llms.txt, got %d", len(pages))
	}
}
