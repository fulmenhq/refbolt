package archive

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fulmenhq/refbolt/internal/provider"
)

func TestWriteRejectsCollisionsBeforeAnyWrite(t *testing.T) {
	root := t.TempDir()
	pages := []provider.Page{
		{Path: "safe.md", SourceURL: "https://example.com/safe", Content: []byte("safe")},
		{Path: "concepts.md", SourceURL: "https://example.com/alpha/concepts", Content: []byte("alpha")},
		{Path: "./concepts.md", SourceURL: "https://example.com/beta/concepts", Content: []byte("beta")},
	}
	if _, err := NewWriter(root).Write("topic", "provider", pages); err == nil {
		t.Fatal("distinct source collision accepted")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed batch left archive files: %v, %v", entries, err)
	}
}

func TestWriteSameSourceSupplementCountsOnce(t *testing.T) {
	root := t.TempDir()
	pages := []provider.Page{
		{Path: "guide.md", SourceURL: "https://example.com/guide", Content: []byte("old dump")},
		{Path: "guide.md", SourceURL: "https://example.com/guide.md", Content: []byte("fresh supplemental")},
	}
	stat, err := NewWriter(root).Write("topic", "provider", pages)
	if err != nil || stat.Total != 1 || stat.Written != 1 {
		t.Fatalf("write = %+v, %v", stat, err)
	}
	dateDir := findDateDir(t, filepath.Join(root, "topic", "provider"))
	got, err := os.ReadFile(filepath.Join(dateDir, "guide.md"))
	if err != nil || string(got) != "fresh supplemental" {
		t.Fatalf("content = %q, %v", got, err)
	}
}

func TestWriteRejectsEscapingPaths(t *testing.T) {
	for _, name := range []string{"../outside.md", "/absolute.md"} {
		_, err := NewWriter(t.TempDir()).Write("topic", "provider", []provider.Page{{Path: name, Content: []byte("no")}})
		if err == nil {
			t.Errorf("accepted unsafe path %q", name)
		}
	}
}
