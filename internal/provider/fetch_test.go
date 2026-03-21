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

func TestHTTPFetcher_Pydantic_LLMSFullTxt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	cfg := ProviderConfig{
		Slug:       "pydantic",
		Name:       "Pydantic",
		BaseURL:    "https://docs.pydantic.dev/latest",
		LLMSTxtURL: "https://docs.pydantic.dev/latest/llms-full.txt",
		Paths: []string{
			"/concepts/models/index.md",
		},
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

	// Expect at least 2 pages: the raw llms-full.txt dump + the individual path.
	if len(pages) < 2 {
		t.Errorf("Expected at least 2 pages (raw dump + individual), got %d", len(pages))
	}

	// Verify the raw dump is archived under its real filename and is > 1MB.
	var foundRaw, foundSupplemental bool
	for _, p := range pages {
		switch p.Path {
		case "llms-full.txt":
			foundRaw = true
			if len(p.Content) < 1_000_000 {
				t.Errorf("Expected llms-full.txt content > 1MB, got %d bytes", len(p.Content))
			}
			t.Logf("Raw llms-full.txt: %d bytes", len(p.Content))
		case "concepts/models/index.md":
			foundSupplemental = true
			if len(p.Content) == 0 {
				t.Error("Supplemental page concepts/models/index.md has empty content")
			}
		}
	}
	if !foundRaw {
		t.Error("Expected page with Path \"llms-full.txt\", not found")
	}
	if !foundSupplemental {
		t.Error("Expected supplemental page \"concepts/models/index.md\", not found")
	}
}
