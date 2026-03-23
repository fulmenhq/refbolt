package provider

import (
	"context"
	"testing"
)

func TestHierarchicalFetcher_AWSGlue(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	cfg := ProviderConfig{
		Slug:          "aws-glue-dg",
		Name:          "AWS Glue Developer Guide",
		BaseURL:       "https://docs.aws.amazon.com/glue/latest/dg",
		FetchStrategy: StrategyHierarchical,
		LLMSTxtURL:    "https://docs.aws.amazon.com/llms.txt",
	}
	f, err := NewHierarchicalFetcher(cfg)
	if err != nil {
		t.Fatal(err)
	}

	pages, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Got %d pages from AWS Glue", len(pages))

	// AWS service llms.txt is a TOC index (links to HTML pages), not a content
	// dump with split delimiters. The raw llms.txt itself is the archived artifact.
	if len(pages) < 1 {
		t.Fatal("Expected at least 1 page (raw llms.txt)")
	}
	if pages[0].Path != "llms.txt" {
		t.Errorf("First page path = %q, want llms.txt", pages[0].Path)
	}
	if len(pages[0].Content) < 10000 {
		t.Errorf("Glue llms.txt suspiciously small: %d bytes", len(pages[0].Content))
	}
	t.Logf("Raw llms.txt: %d bytes", len(pages[0].Content))
}

func TestHierarchicalFetcher_AWSBedrockUserguide(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	cfg := ProviderConfig{
		Slug:          "aws-bedrock-userguide",
		Name:          "AWS Bedrock User Guide",
		BaseURL:       "https://docs.aws.amazon.com/bedrock/latest/userguide",
		FetchStrategy: StrategyHierarchical,
		LLMSTxtURL:    "https://docs.aws.amazon.com/llms.txt",
	}
	f, err := NewHierarchicalFetcher(cfg)
	if err != nil {
		t.Fatal(err)
	}

	pages, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Got %d pages from AWS Bedrock User Guide", len(pages))

	if len(pages) < 1 {
		t.Fatal("Expected at least 1 page (raw llms.txt)")
	}
	if len(pages[0].Content) < 10000 {
		t.Errorf("Bedrock userguide llms.txt suspiciously small: %d bytes", len(pages[0].Content))
	}
	t.Logf("Raw llms.txt: %d bytes", len(pages[0].Content))
}
