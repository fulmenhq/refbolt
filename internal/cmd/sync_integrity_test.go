package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fulmenhq/refbolt/internal/config"
)

func TestSyncReturnsFailureWithoutRecordingEmptyProvider(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/good.md":
			_, _ = fmt.Fprint(w, "# Good content")
		case "/empty.md":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	root := t.TempDir()
	t.Setenv("REFBOLT_ARCHIVE_ROOT", root)
	setupTestCatalog(t)
	body := fmt.Sprintf(`archive_root: %q
topics:
  - slug: test-topic
    providers:
      - slug: missing
        base_url: %q
        fetch_strategy: native
        paths: [/missing.md]
      - slug: empty
        base_url: %q
        fetch_strategy: native
        paths: [/empty.md]
      - slug: good
        base_url: %q
        fetch_strategy: native
        paths: [/good.md]
`, root, srv.URL, srv.URL, srv.URL)
	cfg := filepath.Join(t.TempDir(), "providers.yaml")
	if err := os.WriteFile(cfg, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.Load(config.LoadOptions{ConfigPath: cfg}); err != nil {
		t.Fatal(err)
	}
	syncAll, syncForce = true, true
	providerSlugs, topicSlugs, excludeProviders = nil, nil, nil
	gitCommit, gitPush = false, false
	t.Cleanup(func() { syncAll, syncForce = false, false })
	syncCmd.SetContext(context.Background())
	err := syncCmd.RunE(syncCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "missing") || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("incomplete sync should report both failed providers: %v", err)
	}
	for _, slug := range []string{"missing", "empty"} {
		if _, err := os.Stat(filepath.Join(root, "test-topic", slug, ".sync-meta.json")); !os.IsNotExist(err) {
			t.Fatalf("failed provider %s recorded sync metadata", slug)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "test-topic", "good", "latest", "good.md")); err != nil {
		t.Fatalf("successful provider was not archived: %v", err)
	}
}
