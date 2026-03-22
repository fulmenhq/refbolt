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

func TestHTTPFetcher_Anthropic_LLMSFullTxt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	cfg := ProviderConfig{
		Slug:       "anthropic",
		Name:       "Anthropic",
		BaseURL:    "https://platform.claude.com/docs",
		LLMSTxtURL: "https://platform.claude.com/llms-full.txt",
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

	// Anthropic llms-full.txt has ~488 URL-delimited sections + 1 raw file.
	if len(pages) < 400 {
		t.Errorf("Expected 400+ pages from Anthropic llms-full.txt, got %d", len(pages))
	}

	// Check for key pages by path.
	wantPages := []string{
		"llms-full.txt", // raw dump
		"en/agents-and-tools/tool-use/overview.md", // key split page
	}
	found := make(map[string]bool)
	for _, p := range pages {
		found[p.Path] = true
	}
	for _, want := range wantPages {
		if !found[want] {
			t.Errorf("Expected page %q, not found in %d pages", want, len(pages))
		}
	}
}

func TestHTTPFetcher_OpenAI_JinaWithOpenAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	cfg := ProviderConfig{
		Slug:          "openai",
		Name:          "OpenAI",
		BaseURL:       "https://platform.openai.com",
		FetchStrategy: StrategyJina,
		Paths: []string{
			"/docs/api-reference/chat",
			"/docs/api-reference/responses",
			"/docs/api-reference/assistants",
		},
		OpenAPIURL: "https://raw.githubusercontent.com/openai/openai-openapi/refs/heads/manual_spec/openapi.yaml",
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

	// Expect 4: chat + responses + assistants via Jina + OpenAPI spec.
	if len(pages) < 4 {
		t.Fatalf("Expected at least 4 pages (3 Jina + OpenAPI), got %d", len(pages))
	}

	found := make(map[string]bool)
	for _, p := range pages {
		found[p.Path] = true
		t.Logf("  %s: %d bytes", p.Path, len(p.Content))
	}

	for _, want := range []string{
		"docs/api-reference/chat.md",
		"docs/api-reference/responses.md",
		"docs/api-reference/assistants.md",
		"openapi.yaml",
	} {
		if !found[want] {
			t.Errorf("Missing expected page: %s", want)
		}
	}

	// Verify Jina content is clean Markdown, not HTML.
	for _, p := range pages {
		if p.Path != "openapi.yaml" && looksLikeHTML(p.Content) {
			t.Errorf("%s content is HTML; expected Markdown from Jina", p.Path)
		}
	}
}
