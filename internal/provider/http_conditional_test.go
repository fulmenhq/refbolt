package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPathHintsMatch(t *testing.T) {
	stored := map[string]FetchHint{
		"a.md": {ETag: `"v1"`},
		"b.md": {ETag: `"v2"`},
	}
	current := map[string]FetchHint{
		"a.md": {ETag: `"v1"`},
		"b.md": {ETag: `"v2"`},
	}
	if !PathHintsMatch(stored, current) {
		t.Fatal("expected matching path hints")
	}
	current["b.md"] = FetchHint{ETag: `"v3"`}
	if PathHintsMatch(stored, current) {
		t.Fatal("expected mismatch when etag differs")
	}
}

func TestHTTPFetcher_FetchNativeConditional_NotModified(t *testing.T) {
	const body = "# Cached page\n"
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Header.Get("If-None-Match") == `"abc"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"abc"`)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	dir := t.TempDir()
	latest := filepath.Join(dir, "latest")
	if err := os.MkdirAll(latest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(latest, "page.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	f := &HTTPFetcher{
		cfg: ProviderConfig{
			Slug:          "test",
			BaseURL:       srv.URL,
			FetchStrategy: StrategyNative,
		},
		client:           srv.Client(),
		archiveLatestDir: latest,
	}

	page, hint, err := f.fetchNativeConditional(context.Background(), srv.URL+"/page.md", "page.md", FetchHint{ETag: `"abc"`})
	if err != nil {
		t.Fatalf("fetchNativeConditional: %v", err)
	}
	if hits != 1 {
		t.Fatalf("expected 1 HTTP round trip, got %d", hits)
	}
	if string(page.Content) != body {
		t.Fatalf("content = %q, want cached body", string(page.Content))
	}
	if hint.ETag != `"abc"` {
		t.Fatalf("hint etag = %q", hint.ETag)
	}
}

func TestHTTPFetcher_CheckPathHints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		switch r.URL.Path {
		case "/docs/a.md":
			w.Header().Set("ETag", `"a"`)
		case "/docs/b.md":
			w.Header().Set("ETag", `"b"`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	f, err := NewHTTPFetcher(ProviderConfig{
		Slug:          "starlink-test",
		BaseURL:       srv.URL + "/docs",
		FetchStrategy: StrategyNative,
		Paths:         []string{"a.md", "b.md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	f.client = srv.Client()

	hint, err := f.CheckHints(context.Background())
	if err != nil {
		t.Fatalf("CheckHints: %v", err)
	}
	if len(hint.PathHints) != 2 {
		t.Fatalf("len(PathHints) = %d, want 2", len(hint.PathHints))
	}
	if hint.PathHints["a.md"].ETag != `"a"` {
		t.Fatalf("a.md etag = %q", hint.PathHints["a.md"].ETag)
	}
}

func TestHTTPFetcher_UsesPerPathHints_NativeOnly(t *testing.T) {
	native, _ := NewHTTPFetcher(ProviderConfig{
		Slug:          "x",
		BaseURL:       "https://example.com",
		FetchStrategy: StrategyNative,
		Paths:         []string{"a.md", "b.md"},
	})
	if !native.usesPerPathHints() {
		t.Fatal("native multi-path should use per-path hints")
	}

	withBulk, _ := NewHTTPFetcher(ProviderConfig{
		Slug:          "anthropic-like",
		BaseURL:       "https://example.com/docs",
		FetchStrategy: StrategyNative,
		LLMSTxtURL:    "https://example.com/llms-full.txt",
		Paths:         []string{"a.md", "b.md"},
	})
	if withBulk.usesPerPathHints() {
		t.Fatal("native provider with llms_txt_url must not use per-path provider skip")
	}

	jina, _ := NewHTTPFetcher(ProviderConfig{
		Slug:          "y",
		BaseURL:       "https://example.com",
		FetchStrategy: StrategyJina,
		Paths:         []string{"/a", "/b"},
	})
	if jina.usesPerPathHints() {
		t.Fatal("jina should not use per-path hints")
	}
}
