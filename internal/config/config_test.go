package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fulmenhq/fularchive/internal/config"
)

func TestLoad_NoConfigFile(t *testing.T) {
	// Point to a nonexistent config so we exercise the "no file" path.
	t.Setenv("FULARCHIVE_CONFIG", "/tmp/nonexistent-fularchive-config.yaml")

	if err := config.Load(); err != nil {
		t.Fatalf("Load() with missing config should succeed: %v", err)
	}

	// Defaults should apply.
	if got := config.ArchiveRoot(); got != "/data/archive" {
		t.Errorf("ArchiveRoot() = %q, want /data/archive", got)
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	// Find project root (walk up to find go.mod).
	root := findProjectRoot(t)
	configPath := filepath.Join(root, "configs", "providers.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skipf("configs/providers.yaml not found at %s", configPath)
	}

	t.Setenv("FULARCHIVE_CONFIG", configPath)

	if err := config.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	providers := config.Providers()
	if len(providers) == 0 {
		t.Error("Expected at least one provider from configs/providers.yaml")
	}

	// Check that our seed providers are present.
	slugs := map[string]bool{}
	for _, s := range providers {
		slugs[s] = true
	}
	for _, want := range []string{"anthropic", "openai", "xai"} {
		if !slugs[want] {
			t.Errorf("Missing expected provider %q in %v", want, providers)
		}
	}

	topics := config.TopicSlugs()
	if len(topics) == 0 {
		t.Error("Expected at least one topic")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("FULARCHIVE_CONFIG", "/tmp/nonexistent-fularchive-config.yaml")
	t.Setenv("FULARCHIVE_ARCHIVE_ROOT", "/custom/path")

	if err := config.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if got := config.ArchiveRoot(); got != "/custom/path" {
		t.Errorf("ArchiveRoot() = %q, want /custom/path", got)
	}
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}
