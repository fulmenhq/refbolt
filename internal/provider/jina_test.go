package provider

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
)

// routeRecorder wraps the fetcher's transport and records each request host, so
// the auto test can prove the Jina proxy was reached on the wire.
type routeRecorder struct {
	base  http.RoundTripper
	hosts []string
}

func (r *routeRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.hosts = append(r.hosts, req.URL.Host)
	return r.base.RoundTrip(req)
}

// requireLiveJinaKey fails the live Jina check when the key is absent. These
// tests are a release gate: a missing secret must not look like a green skip.
func requireLiveJinaKey(t *testing.T) {
	t.Helper()
	if jinaAPIKey(ProviderConfig{}) == "" {
		t.Fatal("JINA_API_KEY is required for live Jina checks")
	}
}

// requireJinaContent fails on short or HTML-looking bodies. Jina transient
// short responses are treated as failures for the live gate, not skips.
func requireJinaContent(t *testing.T, content []byte) {
	t.Helper()
	if len(bytes.TrimSpace(content)) < 1000 {
		t.Fatalf("Jina returned insufficient content (%d bytes); expected a documentation page", len(content))
	}
	if looksLikeHTML(content) {
		t.Fatal("Jina returned HTML instead of Markdown")
	}
}

// TestHTTPFetcher_Jina_OpenAI is a transport integration check for the Jina
// Reader path against a stable docs URL. It is not a test of the current
// OpenAI provider config, which is native.
func TestHTTPFetcher_Jina_OpenAI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	requireLiveJinaKey(t)
	cfg := ProviderConfig{
		Slug:          "openai",
		Name:          "OpenAI",
		BaseURL:       "https://platform.openai.com",
		FetchStrategy: StrategyJina,
		Paths:         []string{"/docs/api-reference/chat"},
	}
	f, err := NewHTTPFetcher(cfg)
	if err != nil {
		t.Fatal(err)
	}

	pages, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) == 0 {
		t.Fatal("Expected at least 1 page from Jina")
	}

	page := pages[0]
	t.Logf("Page path: %s, size: %d bytes", page.Path, len(page.Content))

	requireJinaContent(t, page.Content)

	content := string(page.Content)

	// Jina header should be stripped — no "Markdown Content:" in output.
	if strings.Contains(content, "Markdown Content:") {
		t.Error("Jina metadata header was not stripped")
	}

	// Spot-check: OpenAI chat reference should mention "chat" or "completions".
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "chat") {
		t.Error("Expected content to mention 'chat'")
	}
}

// TestHTTPFetcher_Jina_Auto_Fallback exercises the auto strategy's Jina fallback
// as a transport integration check. The stable URL is not today's native
// OpenAI provider config. The route is proven on the wire: the direct request
// to the provider host must happen, and a request must reach r.jina.ai.
func TestHTTPFetcher_Jina_Auto_Fallback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	requireLiveJinaKey(t)
	cfg := ProviderConfig{
		Slug:          "openai",
		Name:          "OpenAI",
		BaseURL:       "https://platform.openai.com",
		FetchStrategy: StrategyAuto,
		Paths:         []string{"/docs/api-reference/chat"},
	}
	f, err := NewHTTPFetcher(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rec := &routeRecorder{base: f.client.Transport}
	f.client.Transport = rec

	pages, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) == 0 {
		t.Fatal("Expected at least 1 page from auto strategy with Jina fallback")
	}

	// Auto must try the direct provider host first, then fall back to the Jina
	// proxy. Check the order of first appearances, not just presence.
	directAt, jinaAt := -1, -1
	for i, h := range rec.hosts {
		if h == "platform.openai.com" && directAt < 0 {
			directAt = i
		}
		if h == "r.jina.ai" && jinaAt < 0 {
			jinaAt = i
		}
	}
	if directAt < 0 || jinaAt < 0 || jinaAt < directAt {
		t.Fatalf("auto strategy did not take direct-then-Jina route: hosts=%v", rec.hosts)
	}

	// Content is a second guard: clean, substantial Markdown with no HTML doctype.
	requireJinaContent(t, pages[0].Content)
}

func TestStripJinaHeader(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard header",
			input: "Title: My Page\n\nURL Source: https://example.com\n\nMarkdown Content:\n# Hello\n\nWorld",
			want:  "# Hello\n\nWorld",
		},
		{
			name:  "with published time",
			input: "Title: X\n\nURL Source: https://x.com\n\nPublished Time: Mon, 01 Jan 2026\n\nMarkdown Content:\nContent here",
			want:  "Content here",
		},
		{
			name:  "no header",
			input: "# Just markdown\n\nNo Jina header here.",
			want:  "# Just markdown\n\nNo Jina header here.",
		},
		{
			name:  "empty content after header",
			input: "Title: Empty\n\nURL Source: https://x.com\n\nMarkdown Content:\n",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(stripJinaHeader([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("stripJinaHeader() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLooksLikeHTML(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"html doctype", "<!DOCTYPE html><html><body>hi</body></html>", true},
		{"html tag", "<html lang=\"en\"><head></head></html>", true},
		{"markdown", "# Hello\n\nThis is **markdown**.", false},
		{"empty", "", false},
		{"html in markdown code block context", "Here is some text without html tags", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeHTML([]byte(tt.in)); got != tt.want {
				t.Errorf("looksLikeHTML(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestJinaAPIKey_NeverUsesProviderAuthEnvVar(t *testing.T) {
	// Simulate a provider with auth_env_var set to a provider-specific secret.
	// jinaAPIKey must NOT return this value — it would exfiltrate the provider's
	// credential to the third-party Jina Reader service.
	t.Setenv("OPENAI_API_KEY", "sk-provider-secret")
	t.Setenv("JINA_API_KEY", "")

	cfg := ProviderConfig{
		Slug:       "openai",
		AuthEnvVar: "OPENAI_API_KEY",
	}

	key := jinaAPIKey(cfg)
	if key == "sk-provider-secret" {
		t.Fatal("jinaAPIKey returned provider credential; this would leak secrets to Jina Reader")
	}
	if key != "" {
		t.Fatalf("jinaAPIKey returned unexpected value: %q", key)
	}
}

func TestJinaAPIKey_UsesJinaEnvVar(t *testing.T) {
	t.Setenv("JINA_API_KEY", "jina_test_key_123")

	cfg := ProviderConfig{
		Slug:       "openai",
		AuthEnvVar: "OPENAI_API_KEY",
	}

	key := jinaAPIKey(cfg)
	if key != "jina_test_key_123" {
		t.Fatalf("jinaAPIKey() = %q, want %q", key, "jina_test_key_123")
	}
}

func TestCheckJinaResponse_OK(t *testing.T) {
	// Nil error for 200.
	resp := &mockHTTPResponse{statusCode: 200}
	if err := checkJinaResponse(resp.toHTTPResponse(), "https://example.com"); err != nil {
		t.Errorf("expected nil error for 200, got: %v", err)
	}
}

func TestCheckJinaResponse_RateLimit(t *testing.T) {
	resp := &mockHTTPResponse{statusCode: 429, headers: map[string]string{"Retry-After": "30"}}
	err := checkJinaResponse(resp.toHTTPResponse(), "https://example.com")
	if err == nil {
		t.Fatal("expected error for 429")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("expected rate limit message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "30s") {
		t.Errorf("expected retry-after hint, got: %v", err)
	}
}

func TestCheckJinaResponse_Payment(t *testing.T) {
	resp := &mockHTTPResponse{statusCode: 402}
	err := checkJinaResponse(resp.toHTTPResponse(), "https://example.com")
	if err == nil {
		t.Fatal("expected error for 402")
	}
	if !strings.Contains(err.Error(), "payment") {
		t.Errorf("expected payment message, got: %v", err)
	}
}

// mockHTTPResponse is a minimal helper for testing checkJinaResponse.
type mockHTTPResponse struct {
	statusCode int
	headers    map[string]string
}

func (m *mockHTTPResponse) toHTTPResponse() *http.Response {
	resp := &http.Response{
		StatusCode: m.statusCode,
		Header:     http.Header{},
	}
	for k, v := range m.headers {
		resp.Header.Set(k, v)
	}
	return resp
}
