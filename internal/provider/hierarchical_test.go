package provider

import (
	"testing"
)

// Sample index content mimicking the AWS llms.txt format.
const testIndex = `# AWS Documentation
> Index of service documentation.

## Guides

- [AWS Glue User Guide](https://docs.aws.amazon.com/glue/latest/dg/what-is-glue.html): AWS Glue. [llms.txt](https://docs.aws.amazon.com/glue/latest/dg/llms.txt)

- [AWS Glue Web API Reference](https://docs.aws.amazon.com/glue/latest/webapi/Welcome.html): AWS Glue API. [llms.txt](https://docs.aws.amazon.com/glue/latest/webapi/llms.txt)

- [Amazon Bedrock User Guide](https://docs.aws.amazon.com/bedrock/latest/userguide/what-is-bedrock.html): Bedrock. [llms.txt](https://docs.aws.amazon.com/bedrock/latest/userguide/llms.txt)

- [Amazon Bedrock API Reference](https://docs.aws.amazon.com/bedrock/latest/APIReference/Welcome.html): Bedrock API. [llms.txt](https://docs.aws.amazon.com/bedrock/latest/APIReference/llms.txt)

- [Amazon Bedrock AgentCore User Guide](https://docs.aws.amazon.com/bedrock-agentcore/latest/userguide/what-is.html): Bedrock AgentCore. [llms.txt](https://docs.aws.amazon.com/bedrock-agentcore/latest/userguide/llms.txt)

- [Amazon S3 User Guide](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html): S3. [llms.txt](https://docs.aws.amazon.com/AmazonS3/latest/userguide/llms.txt)
`

func TestParseIndexLLMSTxtURLs(t *testing.T) {
	urls := parseIndexLLMSTxtURLs([]byte(testIndex))

	// Should find 6 llms.txt URLs.
	if len(urls) != 6 {
		t.Fatalf("parseIndexLLMSTxtURLs() returned %d URLs, want 6", len(urls))
	}

	// Spot check.
	wantFirst := "https://docs.aws.amazon.com/glue/latest/dg/llms.txt"
	if urls[0] != wantFirst {
		t.Errorf("first URL = %q, want %q", urls[0], wantFirst)
	}
}

func TestServicePrefix(t *testing.T) {
	tests := []struct {
		baseURL string
		want    string
	}{
		{"https://docs.aws.amazon.com/glue/latest", "/glue/latest/"},
		{"https://docs.aws.amazon.com/bedrock/latest/userguide", "/bedrock/latest/userguide/"},
		{"https://docs.aws.amazon.com/bedrock/latest/APIReference", "/bedrock/latest/APIReference/"},
		{"https://docs.aws.amazon.com/glue/latest/", "/glue/latest/"},
		{"https://example.com", "/"},
	}
	for _, tt := range tests {
		t.Run(tt.baseURL, func(t *testing.T) {
			got := servicePrefix(tt.baseURL)
			if got != tt.want {
				t.Errorf("servicePrefix(%q) = %q, want %q", tt.baseURL, got, tt.want)
			}
		})
	}
}

func TestMatchServiceURLs_ExactPrefix(t *testing.T) {
	allURLs := parseIndexLLMSTxtURLs([]byte(testIndex))

	// Glue Developer Guide with guide-specific base_url should match exactly one.
	glueDGPrefix := servicePrefix("https://docs.aws.amazon.com/glue/latest/dg")
	glueDGMatches := matchServiceURLs(allURLs, glueDGPrefix)
	if len(glueDGMatches) != 1 {
		t.Errorf("glue dg matches: got %d, want 1: %v", len(glueDGMatches), glueDGMatches)
	}

	// Broad /glue/latest prefix would match both dg and webapi — this is why
	// provider config should use guide-specific prefixes for multi-guide services.
	glueBroadPrefix := servicePrefix("https://docs.aws.amazon.com/glue/latest")
	glueBroadMatches := matchServiceURLs(allURLs, glueBroadPrefix)
	if len(glueBroadMatches) != 2 {
		t.Errorf("glue broad matches: got %d, want 2: %v", len(glueBroadMatches), glueBroadMatches)
	}
}

func TestMatchServiceURLs_NoFalsePositive_Bedrock(t *testing.T) {
	allURLs := parseIndexLLMSTxtURLs([]byte(testIndex))

	// bedrock/latest/userguide should NOT match bedrock-agentcore.
	prefix := servicePrefix("https://docs.aws.amazon.com/bedrock/latest/userguide")
	matches := matchServiceURLs(allURLs, prefix)

	if len(matches) != 1 {
		t.Fatalf("bedrock userguide matches: got %d, want 1: %v", len(matches), matches)
	}
	if matches[0] != "https://docs.aws.amazon.com/bedrock/latest/userguide/llms.txt" {
		t.Errorf("wrong match: %s", matches[0])
	}
}

func TestMatchServiceURLs_BroadBedrockPrefix(t *testing.T) {
	allURLs := parseIndexLLMSTxtURLs([]byte(testIndex))

	// bedrock/latest (without guide family) should match userguide + APIReference
	// but NOT bedrock-agentcore.
	prefix := servicePrefix("https://docs.aws.amazon.com/bedrock/latest")
	matches := matchServiceURLs(allURLs, prefix)

	if len(matches) != 2 {
		t.Fatalf("bedrock/latest matches: got %d, want 2: %v", len(matches), matches)
	}

	for _, m := range matches {
		if containsString(m, "bedrock-agentcore") {
			t.Errorf("false positive: matched bedrock-agentcore URL: %s", m)
		}
	}
}

func TestMatchServiceURLs_NoMatch(t *testing.T) {
	allURLs := parseIndexLLMSTxtURLs([]byte(testIndex))

	prefix := servicePrefix("https://docs.aws.amazon.com/lambda/latest")
	matches := matchServiceURLs(allURLs, prefix)

	if len(matches) != 0 {
		t.Errorf("lambda matches: got %d, want 0: %v", len(matches), matches)
	}
}

func TestSelectBestMatch(t *testing.T) {
	// When base_url is bedrock/latest/userguide, selectBestMatch should prefer
	// the userguide URL over the APIReference URL.
	urls := []string{
		"https://docs.aws.amazon.com/bedrock/latest/userguide/llms.txt",
		"https://docs.aws.amazon.com/bedrock/latest/APIReference/llms.txt",
	}
	prefix := servicePrefix("https://docs.aws.amazon.com/bedrock/latest/userguide")
	best := selectBestMatch(urls, prefix)

	want := "https://docs.aws.amazon.com/bedrock/latest/userguide/llms.txt"
	if best != want {
		t.Errorf("selectBestMatch() = %q, want %q", best, want)
	}
}

func TestNewHierarchicalFetcher_MissingLLMSTxtURL(t *testing.T) {
	cfg := ProviderConfig{
		Slug:          "test",
		BaseURL:       "https://example.com",
		FetchStrategy: StrategyHierarchical,
	}
	_, err := NewHierarchicalFetcher(cfg)
	if err == nil {
		t.Fatal("expected error for missing llms_txt_url")
	}
}

func TestNewHierarchicalFetcher_MissingBaseURL(t *testing.T) {
	cfg := ProviderConfig{
		Slug:          "test",
		LLMSTxtURL:    "https://example.com/llms.txt",
		FetchStrategy: StrategyHierarchical,
	}
	_, err := NewHierarchicalFetcher(cfg)
	if err == nil {
		t.Fatal("expected error for missing base_url")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
