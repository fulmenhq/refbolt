package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestBulkFailureCannotPublishSupplementalSources(t *testing.T) {
	for _, failure := range []string{"missing", "empty", "html"} {
		for _, source := range []string{"index", "full"} {
			for _, supplemental := range []string{"paths", "schema", "both"} {
				t.Run(failure+"/"+source+"/"+supplemental, func(t *testing.T) {
					srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						switch r.URL.Path {
						case "/ok.md", "/schema":
							_, _ = fmt.Fprint(w, "# Valid supplemental artifact")
						default:
							switch failure {
							case "missing":
								http.NotFound(w, r)
							case "html":
								_, _ = fmt.Fprint(w, "<!DOCTYPE html><html>Error page</html>")
							case "empty":
								w.WriteHeader(http.StatusOK)
							}
						}
					}))
					defer srv.Close()
					cfg := ProviderConfig{Slug: "bulk", BaseURL: srv.URL, FetchStrategy: StrategyNative}
					if source == "index" {
						cfg.LLMSTxtURL = srv.URL + "/llms.txt"
					} else {
						cfg.LLMSFullTxtURL = srv.URL + "/llms-full.txt"
					}
					if supplemental != "schema" {
						cfg.Paths = []string{"/ok.md"}
					}
					if supplemental != "paths" {
						cfg.OpenAPIURL = srv.URL + "/schema"
					}
					f, err := NewHTTPFetcher(cfg)
					if err != nil {
						t.Fatal(err)
					}
					pages, err := f.Fetch(context.Background())
					if err == nil || !strings.Contains(err.Error(), "incomplete provider bulk") || len(pages) != 0 {
						t.Fatalf("failed bulk source published %d supplemental artifacts: %v", len(pages), err)
					}
				})
			}
		}
	}
}

func TestMarkdownHTMLExampleIsNotAnHTMLDocument(t *testing.T) {
	if looksLikeHTML([]byte("# HTML example\n\n```html\n<!DOCTYPE html>\n<html>\n```")) {
		t.Fatal("Markdown code example treated as an HTML response")
	}
}
