// Package sync provides incremental sync metadata for skip-unchanged logic.
// Each provider gets a .sync-meta.json alongside its archive output.
package sync

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const metaFilename = ".sync-meta.json"

// SyncMeta holds per-provider sync state for incremental skip logic.
type SyncMeta struct {
	Provider    string    `json:"provider"`
	Strategy    string    `json:"strategy"`
	ConfigHash  string    `json:"config_hash"`
	LastSync    time.Time `json:"last_sync"`
	ContentHash string    `json:"content_hash"`
	FileCount   int       `json:"file_count"`
	Hint        FetchHint `json:"fetch_hint"`
}

// FetchHint holds strategy-specific metadata for short-circuiting fetches.
type FetchHint struct {
	ETag          string               `json:"etag,omitempty"`
	LastModified  string               `json:"last_modified,omitempty"`
	ContentLength int64                `json:"content_length,omitempty"`
	TreeSHA       string               `json:"tree_sha,omitempty"`
	PathHints     map[string]FetchHint `json:"path_hints,omitempty"`
}

// MetaPath returns the path to .sync-meta.json for a given provider.
func MetaPath(archiveRoot, topicSlug, providerSlug string) string {
	return filepath.Join(archiveRoot, topicSlug, providerSlug, metaFilename)
}

// Read loads metadata from disk. Returns nil, nil if the file doesn't exist.
func Read(path string) (*SyncMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading sync metadata: %w", err)
	}

	var meta SyncMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing sync metadata: %w", err)
	}
	return &meta, nil
}

// Write atomically writes metadata to disk (temp file + rename).
// Only call after a successful fetch/write cycle.
func Write(path string, meta *SyncMeta) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating metadata directory: %w", err)
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling sync metadata: %w", err)
	}
	data = append(data, '\n')

	// Atomic write: temp file in same directory, then rename.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing temp metadata: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		if cleanupErr := os.Remove(tmp); cleanupErr != nil && !os.IsNotExist(cleanupErr) {
			return fmt.Errorf("renaming temp metadata: %w (removing temp metadata: %v)", err, cleanupErr)
		}
		return fmt.Errorf("renaming temp metadata: %w", err)
	}
	return nil
}

// ConfigHash computes a SHA-256 fingerprint of provider config fields that
// affect output. If any of these change, stored metadata is invalidated.
func ConfigHash(fields map[string]string) string {
	// Sort keys for deterministic hashing.
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := sha256.New()
	for _, k := range keys {
		_, _ = fmt.Fprintf(h, "%s=%s\n", k, fields[k])
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}

// ContentHash computes SHA-256 of raw content bytes.
func ContentHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", h[:])
}

// ShouldSkip checks whether a provider can be skipped based on stored metadata.
// Returns true if config hasn't changed and hints suggest no upstream change.
// The caller must still verify strategy-specific hints (tree SHA, ETag, etc.).
func ShouldSkip(meta *SyncMeta, currentConfigHash string) bool {
	if meta == nil {
		return false // cold start — no metadata
	}
	if meta.ConfigHash != currentConfigHash {
		return false // config changed — must re-fetch
	}
	return true // config unchanged — caller checks strategy hints
}

// ProviderConfigFields extracts the config fields that affect output for hashing.
func ProviderConfigFields(slug, baseURL, strategy, llmsTxtURL, githubRepo, githubDocsPath, githubBranch string, paths []string) map[string]string {
	return map[string]string{
		"slug":             slug,
		"base_url":         baseURL,
		"fetch_strategy":   strategy,
		"llms_txt_url":     llmsTxtURL,
		"github_repo":      githubRepo,
		"github_docs_path": githubDocsPath,
		"github_branch":    githubBranch,
		"paths":            strings.Join(paths, ","),
	}
}
