package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchDoesNotPublishPartialOrInvalidDocuments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok.md":
			_, _ = fmt.Fprint(w, "# Good document")
		case "/html.md":
			_, _ = fmt.Fprint(w, "<!DOCTYPE html><html>Site navigation</html>")
		case "/empty.md":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	for _, cfg := range []ProviderConfig{
		{Paths: []string{"ok.md", "missing.md"}},
		{Paths: []string{"ok.md"}, OpenAPIURL: srv.URL + "/missing-spec"},
		{Paths: []string{"html.md"}},
		{Paths: []string{"empty.md"}},
		{LLMSTxtURL: srv.URL + "/html.md"},
	} {
		cfg.BaseURL, cfg.FetchStrategy = srv.URL, StrategyNative
		f, _ := NewHTTPFetcher(cfg)
		pages, err := f.Fetch(context.Background())
		if err == nil || len(pages) != 0 {
			t.Fatalf("invalid or partial provider published %d pages: %v", len(pages), err)
		}
	}
}

func TestMarkdownHTMLExampleIsNotAnHTMLDocument(t *testing.T) {
	if looksLikeHTML([]byte("# HTML example\n\n```html\n<!DOCTYPE html>\n<html>\n```")) {
		t.Fatal("Markdown code example treated as an HTML response")
	}
}
