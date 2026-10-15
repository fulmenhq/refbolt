package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrontmatterRepeatedTitlesKeepDistinctSources(t *testing.T) {
	content := ""
	for _, name := range []string{"alpha", "beta"} {
		content += fmt.Sprintf("---\ndescription: Content before the title\ntitle: Concepts\n---\n[View as Markdown](https://developers.cloudflare.com/workers/%s.md)\n# Concepts\n%s content\n", name, name)
	}
	content += "---\ntitle: Concepts\n---\n# Concepts\nFallback one\n---\ntitle: Concepts\n---\n# Concepts\nFallback two\n"
	pages, err := SplitFrontmatterFullTxt([]byte(content), "https://developers.cloudflare.com/workers/llms-full.txt")
	if err != nil || len(pages) != 4 {
		t.Fatalf("split: %d pages, %v", len(pages), err)
	}
	for i, want := range []string{"workers/alpha.md", "workers/beta.md", "concepts.md", "concepts-2.md"} {
		if pages[i].Path != want {
			t.Errorf("page %d path = %q, want %q", i, pages[i].Path, want)
		}
	}
}

func TestDumpMarkersInsideCodeArePreserved(t *testing.T) {
	tests := []struct {
		name, input, marker string
		split               func([]byte, string) ([]Page, error)
	}{
		{"xai", "===/real===\n# Real\n```text\n===/fake===\n```\nend", "===/fake===", SplitLLMSTxt},
		{"source", "Source: https://example.com/real\n# Real\n~~~text\nSource: https://example.com/fake\n~~~\nend", "Source: https://example.com/fake", SplitLLMSFullTxt},
		{"frontmatter", "---\ntitle: Real\n---\n# Real\n```yaml\n---\ntitle: Fake\n---\n```\nend", "title: Fake", SplitFrontmatterFullTxt},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pages, err := tt.split([]byte(tt.input), "https://example.com/llms-full.txt")
			if err != nil || len(pages) != 1 {
				t.Fatalf("split: %d pages, %v", len(pages), err)
			}
			if !strings.Contains(string(pages[0].Content), tt.marker) || !strings.Contains(string(pages[0].Content), "end") {
				t.Fatalf("lost code or trailing content: %s", pages[0].Content)
			}
		})
	}
}

func TestBulkHintsCannotHideSupplementalChanges(t *testing.T) {
	for _, cfg := range []ProviderConfig{
		{LLMSTxtURL: "https://example.com/llms-full.txt", Paths: []string{"extra.md"}},
		{LLMSFullTxtURL: "https://example.com/llms-full.txt", OpenAPIURL: "https://example.com/spec"},
	} {
		cfg.BaseURL, cfg.FetchStrategy = "https://example.com", StrategyNative
		f, _ := NewHTTPFetcher(cfg)
		if _, err := f.CheckHints(context.Background()); err == nil {
			t.Fatal("single HEAD incorrectly represents multiple sources")
		}
	}
}

func TestNativeSpecChangeParticipatesInIncrementalHints(t *testing.T) {
	specVersion := "v1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := "page-v1"
		if r.URL.Path == "/spec" {
			version = specVersion
		}
		w.Header().Set("ETag", version)
		if r.Method == http.MethodHead {
			return
		}
		if r.Header.Get("If-None-Match") == version {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_, _ = fmt.Fprint(w, version)
	}))
	defer srv.Close()
	f, _ := NewHTTPFetcher(ProviderConfig{BaseURL: srv.URL, FetchStrategy: StrategyNative, Paths: []string{"page.md"}, OpenAPIURL: srv.URL + "/spec"})
	dir := t.TempDir()
	f.SetIncrementalContext(dir, FetchHint{})
	pages, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pages {
		if err := os.WriteFile(filepath.Join(dir, p.Path), p.Content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	prior := f.LastFetchHint()
	if prior.PathHints["openapi.yaml"].ETag != "v1" {
		t.Fatal("spec fetch hint missing")
	}
	specVersion = "v2"
	current, err := f.CheckHints(context.Background())
	if err != nil || PathHintsMatch(prior.PathHints, current.PathHints) {
		t.Fatalf("changed spec was skipped: %v", err)
	}
	f.SetIncrementalContext(dir, prior)
	pages, err = f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pages {
		if p.Path == "openapi.yaml" && string(p.Content) == "v2" {
			return
		}
	}
	t.Fatal("updated spec missing from incremental fetch")
}

func TestScopeMatchesDirectoryBoundary(t *testing.T) {
	pages := []Page{{SourceURL: "https://example.com/api/page"}, {SourceURL: "https://example.com/apiculture/page"}}
	if got := FilterByBaseURL(pages, "https://example.com/api"); len(got) != 1 {
		t.Fatalf("scope matched sibling: %v", got)
	}
}
