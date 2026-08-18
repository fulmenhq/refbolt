package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadWrite_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "topic", "provider", ".sync-meta.json")

	meta := &SyncMeta{
		Provider:    "trino",
		Strategy:    "github-raw",
		ConfigHash:  "sha256:abc123",
		LastSync:    time.Date(2026, 3, 27, 9, 15, 0, 0, time.UTC),
		ContentHash: "sha256:def456",
		FileCount:   642,
		Hint: FetchHint{
			TreeSHA: "abc123def456",
			PathHints: map[string]FetchHint{
				"page.md": {ETag: `"v1"`},
			},
		},
	}

	if err := Write(path, meta); err != nil {
		t.Fatal(err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}

	if got.Provider != meta.Provider {
		t.Errorf("Provider = %q, want %q", got.Provider, meta.Provider)
	}
	if got.ConfigHash != meta.ConfigHash {
		t.Errorf("ConfigHash = %q, want %q", got.ConfigHash, meta.ConfigHash)
	}
	if got.ContentHash != meta.ContentHash {
		t.Errorf("ContentHash = %q, want %q", got.ContentHash, meta.ContentHash)
	}
	if got.FileCount != meta.FileCount {
		t.Errorf("FileCount = %d, want %d", got.FileCount, meta.FileCount)
	}
	if got.Hint.TreeSHA != meta.Hint.TreeSHA {
		t.Errorf("Hint.TreeSHA = %q, want %q", got.Hint.TreeSHA, meta.Hint.TreeSHA)
	}
	if got.Hint.PathHints["page.md"].ETag != `"v1"` {
		t.Errorf("PathHints etag = %q, want %q", got.Hint.PathHints["page.md"].ETag, `"v1"`)
	}
}

func TestRead_NoFile(t *testing.T) {
	got, err := Read("/nonexistent/path/.sync-meta.json")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent file")
	}
}

func TestWrite_Atomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".sync-meta.json")

	// Write original.
	original := &SyncMeta{Provider: "original", ConfigHash: "sha256:aaa"}
	if err := Write(path, original); err != nil {
		t.Fatal(err)
	}

	// Write updated — should atomically replace.
	updated := &SyncMeta{Provider: "updated", ConfigHash: "sha256:bbb"}
	if err := Write(path, updated); err != nil {
		t.Fatal(err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider != "updated" {
		t.Errorf("expected updated provider, got %q", got.Provider)
	}

	// No temp file should remain.
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file should not exist after successful write")
	}
}

func TestConfigHash_Deterministic(t *testing.T) {
	fields := map[string]string{
		"slug":     "trino",
		"base_url": "https://trino.io",
		"paths":    "**/*.md",
	}
	h1 := ConfigHash(fields)
	h2 := ConfigHash(fields)
	if h1 != h2 {
		t.Errorf("ConfigHash not deterministic: %q != %q", h1, h2)
	}
}

func TestConfigHash_ChangedField(t *testing.T) {
	fields1 := map[string]string{
		"slug":     "trino",
		"base_url": "https://trino.io/docs/current",
	}
	fields2 := map[string]string{
		"slug":     "trino",
		"base_url": "https://trino.io/docs/latest",
	}
	h1 := ConfigHash(fields1)
	h2 := ConfigHash(fields2)
	if h1 == h2 {
		t.Error("ConfigHash should differ when fields change")
	}
}

func TestContentHash(t *testing.T) {
	h1 := ContentHash([]byte("hello world"))
	h2 := ContentHash([]byte("hello world"))
	h3 := ContentHash([]byte("different"))

	if h1 != h2 {
		t.Error("same content should produce same hash")
	}
	if h1 == h3 {
		t.Error("different content should produce different hash")
	}
	if h1[:7] != "sha256:" {
		t.Errorf("hash should start with sha256:, got %q", h1[:7])
	}
}

func TestShouldSkip_NilMeta(t *testing.T) {
	if ShouldSkip(nil, "sha256:abc") {
		t.Error("should not skip with nil metadata (cold start)")
	}
}

func TestShouldSkip_ConfigChanged(t *testing.T) {
	meta := &SyncMeta{ConfigHash: "sha256:old"}
	if ShouldSkip(meta, "sha256:new") {
		t.Error("should not skip when config hash changed")
	}
}

func TestShouldSkip_ConfigUnchanged(t *testing.T) {
	meta := &SyncMeta{ConfigHash: "sha256:same"}
	if !ShouldSkip(meta, "sha256:same") {
		t.Error("should skip when config hash unchanged")
	}
}

func TestMetaPath(t *testing.T) {
	got := MetaPath("/data/archive", "llm-api", "trino")
	want := "/data/archive/llm-api/trino/.sync-meta.json"
	if got != want {
		t.Errorf("MetaPath = %q, want %q", got, want)
	}
}
