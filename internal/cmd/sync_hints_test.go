package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fulmenhq/refbolt/internal/provider"
)

func TestCaptureFetchHint_PreservesBulkETagWithPathHints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			if r.URL.Path == "/llms-full.txt" {
				w.Header().Set("ETag", `"bulk"`)
				return
			}
		case http.MethodGet:
			switch r.URL.Path {
			case "/llms-full.txt":
				w.Header().Set("ETag", `"bulk"`)
				_, _ = w.Write([]byte("bulk content"))
				return
			case "/docs/extra.md":
				w.Header().Set("ETag", `"extra"`)
				_, _ = w.Write([]byte("# extra"))
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	f, err := provider.NewHTTPFetcher(provider.ProviderConfig{
		Slug:          "mixed",
		BaseURL:       srv.URL + "/docs",
		FetchStrategy: provider.StrategyNative,
		LLMSTxtURL:    srv.URL + "/llms-full.txt",
		Paths:         []string{"extra.md"},
	})
	if err != nil {
		t.Fatal(err)
	}

	latest := filepath.Join(t.TempDir(), "latest")
	if err := os.MkdirAll(latest, 0o755); err != nil {
		t.Fatal(err)
	}
	f.SetIncrementalContext(latest, provider.FetchHint{})

	ctx := context.Background()
	if _, err := f.Fetch(ctx); err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	hint := captureFetchHint(ctx, f)
	if hint.ETag != `"bulk"` {
		t.Fatalf("bulk etag = %q, want %q", hint.ETag, `"bulk"`)
	}
	if len(hint.PathHints) != 1 {
		t.Fatalf("len(PathHints) = %d, want 1", len(hint.PathHints))
	}
	if hint.PathHints["extra.md"].ETag != `"extra"` {
		t.Fatalf("extra.md etag = %q", hint.PathHints["extra.md"].ETag)
	}
}
