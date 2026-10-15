package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHierarchicalFetchCuratedPages(t *testing.T) {
	var base string
	var fetched int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/llms.txt":
			_, _ = fmt.Fprintf(w, "[llms.txt](%s/service/guide/llms.txt)", base)
		case "/service/guide/llms.txt":
			_, _ = fmt.Fprintf(w, "[Guide](%s/service/guide/page.md)\n[Sibling](%s/service/other/page.md)", base, base)
		case "/service/guide/page.md":
			fetched++
			_, _ = fmt.Fprint(w, "# Native guide page")
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL
	for _, paths := range [][]string{{"page.md", "page.md"}, {"not-indexed.md"}, {"../other/page.md"}, {"**/*.md"}} {
		f, err := NewHierarchicalFetcher(ProviderConfig{BaseURL: base + "/service/guide", LLMSTxtURL: base + "/llms.txt", Paths: paths})
		if err != nil {
			t.Fatal(err)
		}
		pages, err := f.Fetch(context.Background())
		if paths[0] != "page.md" {
			if err == nil {
				t.Fatalf("accepted unbounded or out-of-scope paths %v", paths)
			}
			continue
		}
		if err != nil || len(pages) != 2 || pages[1].Path != "page.md" || string(pages[1].Content) != "# Native guide page" {
			t.Fatalf("curated fetch = %v, %v", pages, err)
		}
	}
	if fetched != 1 {
		t.Fatalf("curated page fetched %d times, want 1", fetched)
	}
}

func TestHierarchicalFetchRejectsAmbiguousGuides(t *testing.T) {
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/llms.txt" {
			t.Errorf("ambiguous selection fetched a guide: %s", r.URL.Path)
		}
		_, _ = fmt.Fprintf(w, "[llms.txt](%s/service/one/llms.txt)\n[llms.txt](%s/service/two/llms.txt)", base, base)
	}))
	defer srv.Close()
	base = srv.URL
	f, _ := NewHierarchicalFetcher(ProviderConfig{BaseURL: base + "/service", LLMSTxtURL: base + "/llms.txt"})
	if _, err := f.Fetch(context.Background()); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous guide selection = %v", err)
	}
}
