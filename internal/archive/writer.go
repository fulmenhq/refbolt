// Package archive handles writing fetched pages to the date-versioned archive tree.
package archive

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fulmenhq/refbolt/internal/provider"
)

// WriteStat reports what happened during a write operation.
type WriteStat struct {
	Written int // files actually written (new or changed)
	Skipped int // files skipped (content hash unchanged)
	Total   int // Written + Skipped
}

// Writer writes fetched pages to the archive tree.
type Writer struct {
	root string
}

// NewWriter creates a writer rooted at the given archive directory.
func NewWriter(root string) *Writer {
	return &Writer{root: root}
}

// Write writes all pages for a given topic and provider into a date-versioned directory.
// Tree structure: <root>/<topic>/<provider>/<date>/<page-path>
// Also creates/updates a "latest" symlink pointing to the current date directory.
//
// Content dedup: before writing, compares SHA-256 of new content against
// existing file. Skips write if unchanged, avoiding disk churn and git noise.
func (w *Writer) Write(topicSlug, providerSlug string, pages []provider.Page) (WriteStat, error) {
	pages, err := uniquePages(pages)
	if err != nil {
		return WriteStat{}, err
	}
	date := time.Now().Format("2006-01-02")
	dateDir := filepath.Join(w.root, topicSlug, providerSlug, date)

	var stat WriteStat
	for _, page := range pages {
		if len(page.Content) == 0 {
			continue
		}

		dest := filepath.Join(dateDir, page.Path)

		// Hash-before-write: skip if existing file has identical content.
		if existingContent, err := os.ReadFile(dest); err == nil {
			if contentEqual(existingContent, page.Content) {
				stat.Skipped++
				stat.Total++
				continue
			}
		}

		// Ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return stat, fmt.Errorf("creating directory for %s: %w", page.Path, err)
		}

		if err := os.WriteFile(dest, page.Content, 0o644); err != nil {
			return stat, fmt.Errorf("writing %s: %w", page.Path, err)
		}
		stat.Written++
		stat.Total++
	}

	// Update "latest" symlink.
	if stat.Written > 0 {
		latestLink := filepath.Join(w.root, topicSlug, providerSlug, "latest")
		_ = os.Remove(latestLink)
		if err := os.Symlink(date, latestLink); err != nil {
			// Non-fatal — symlinks may not work on all filesystems.
			fmt.Printf("  ⚠ could not create latest symlink: %v\n", err)
		}
	}

	return stat, nil
}

// Validate the whole batch before writing. A supplemental fetch of the same
// source replaces its dump copy; different sources may never overwrite it.
func uniquePages(pages []provider.Page) ([]provider.Page, error) {
	result := make([]provider.Page, 0, len(pages))
	seen := make(map[string]int)
	for _, page := range pages {
		if len(page.Content) == 0 {
			continue
		}
		if !filepath.IsLocal(page.Path) {
			return nil, fmt.Errorf("unsafe archive path %q", page.Path)
		}
		page.Path = filepath.Clean(page.Path)
		if i, ok := seen[page.Path]; ok {
			prior := result[i]
			same := sameSource(prior.SourceURL, page.SourceURL) ||
				(prior.SourceURL == "" && page.SourceURL == "" && bytes.Equal(prior.Content, page.Content))
			if !same {
				return nil, fmt.Errorf("archive path %q belongs to distinct sources %q and %q", page.Path, prior.SourceURL, page.SourceURL)
			}
			result[i] = page
			continue
		}
		seen[page.Path] = len(result)
		result = append(result, page)
	}
	return result, nil
}

func sameSource(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	normalize := func(raw string) string {
		u, err := url.Parse(raw)
		if err != nil {
			return raw
		}
		u.Path = strings.TrimSuffix(u.Path, ".md")
		return u.String()
	}
	return normalize(a) == normalize(b)
}

// contentEqual compares two byte slices by SHA-256 hash.
// Uses hash comparison instead of bytes.Equal to avoid holding both
// full contents in memory for very large files.
func contentEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	// Fast path: if lengths match and are small, use direct comparison.
	if len(a) < 64*1024 {
		return bytes.Equal(a, b)
	}
	ha := sha256.Sum256(a)
	hb := sha256.Sum256(b)
	return ha == hb
}
