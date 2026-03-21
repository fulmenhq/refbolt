// Package archive handles writing fetched pages to the date-versioned archive tree.
package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fulmenhq/fularchive/internal/provider"
)

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
func (w *Writer) Write(topicSlug, providerSlug string, pages []provider.Page) (int, error) {
	date := time.Now().Format("2006-01-02")
	dateDir := filepath.Join(w.root, topicSlug, providerSlug, date)

	written := 0
	for _, page := range pages {
		if len(page.Content) == 0 {
			continue
		}

		dest := filepath.Join(dateDir, page.Path)

		// Ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return written, fmt.Errorf("creating directory for %s: %w", page.Path, err)
		}

		if err := os.WriteFile(dest, page.Content, 0o644); err != nil {
			return written, fmt.Errorf("writing %s: %w", page.Path, err)
		}
		written++
	}

	// Update "latest" symlink.
	if written > 0 {
		latestLink := filepath.Join(w.root, topicSlug, providerSlug, "latest")
		_ = os.Remove(latestLink)
		if err := os.Symlink(date, latestLink); err != nil {
			// Non-fatal — symlinks may not work on all filesystems.
			fmt.Printf("  ⚠ could not create latest symlink: %v\n", err)
		}
	}

	return written, nil
}
