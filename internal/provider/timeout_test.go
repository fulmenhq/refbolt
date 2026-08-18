package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestJinaRetry_TimeoutThenSuccess(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			// First attempt: sleep longer than the initial timeout to force deadline exceeded.
			time.Sleep(200 * time.Millisecond)
			return
		}
		// Second attempt: respond immediately with Markdown.
		w.Header().Set("Content-Type", "text/markdown")
		if _, err := fmt.Fprint(w, "# Success\n\nRetried content."); err != nil {
			t.Errorf("writing test response: %v", err)
		}
	}))
	defer srv.Close()

	cfg := ProviderConfig{
		Slug:          "test-timeout",
		BaseURL:       srv.URL,
		FetchStrategy: StrategyJina,
		FetchTimeout:  100 * time.Millisecond,
	}
	f := &HTTPFetcher{
		cfg:    cfg,
		client: newHTTPClient(),
	}

	// fetchURL is the Jina proxy URL (test server); sourceURL is the original page URL.
	page, err := f.doJinaFetchWithRetry(context.Background(), srv.URL+"/page", "https://example.com/page", "test.md")
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if !strings.Contains(string(page.Content), "Retried content") {
		t.Errorf("expected retried content, got: %s", page.Content)
	}
	if page.SourceURL != "https://example.com/page" {
		t.Errorf("SourceURL = %q, want original URL not Jina proxy", page.SourceURL)
	}
	if attempts.Load() != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts.Load())
	}
}

func TestJinaRetry_NonTimeoutNotRetried(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := fmt.Fprint(w, "Internal Server Error"); err != nil {
			t.Errorf("writing test response: %v", err)
		}
	}))
	defer srv.Close()

	cfg := ProviderConfig{
		Slug:          "test-no-retry",
		BaseURL:       srv.URL,
		FetchStrategy: StrategyJina,
		FetchTimeout:  5 * time.Second,
	}
	f := &HTTPFetcher{
		cfg:    cfg,
		client: newHTTPClient(),
	}

	_, err := f.doJinaFetchWithRetry(context.Background(), srv.URL+"/page", "https://example.com/page", "test.md")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if attempts.Load() != 1 {
		t.Errorf("expected 1 attempt (no retry on 5xx), got %d", attempts.Load())
	}
}

func TestJinaRetry_BothTimeout(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		// Always timeout.
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()

	cfg := ProviderConfig{
		Slug:          "test-both-timeout",
		BaseURL:       srv.URL,
		FetchStrategy: StrategyJina,
		FetchTimeout:  50 * time.Millisecond,
	}
	f := &HTTPFetcher{
		cfg:    cfg,
		client: newHTTPClient(),
	}

	_, err := f.doJinaFetchWithRetry(context.Background(), srv.URL+"/page", "https://example.com/page", "test.md")
	if err == nil {
		t.Fatal("expected error after both attempts timeout")
	}
	if !strings.Contains(err.Error(), "Hint: increase fetch_timeout") {
		t.Errorf("expected actionable hint in error, got: %v", err)
	}
	// Error should reference the source URL, not the Jina proxy URL.
	if !strings.Contains(err.Error(), "example.com") {
		t.Errorf("expected source URL in error, got: %v", err)
	}
	if attempts.Load() != 2 {
		t.Errorf("expected 2 attempts (initial + retry), got %d", attempts.Load())
	}
}

func TestDefaultFetchTimeout(t *testing.T) {
	cfg := ProviderConfig{
		Slug: "test-default",
	}
	if got := cfg.EffectiveFetchTimeout(); got != DefaultFetchTimeout {
		t.Errorf("EffectiveFetchTimeout() = %v, want %v", got, DefaultFetchTimeout)
	}
}

func TestCustomFetchTimeout(t *testing.T) {
	cfg := ProviderConfig{
		Slug:         "test-custom",
		FetchTimeout: 90 * time.Second,
	}
	if got := cfg.EffectiveFetchTimeout(); got != 90*time.Second {
		t.Errorf("EffectiveFetchTimeout() = %v, want 90s", got)
	}
}

func TestIsTimeoutError(t *testing.T) {
	if isTimeoutError(fmt.Errorf("random error")) {
		t.Error("non-timeout error should return false")
	}
	if !isTimeoutError(context.DeadlineExceeded) {
		t.Error("context.DeadlineExceeded should be timeout")
	}
	if !isTimeoutError(fmt.Errorf("wrapped: %w", context.DeadlineExceeded)) {
		t.Error("wrapped DeadlineExceeded should be timeout")
	}
}
