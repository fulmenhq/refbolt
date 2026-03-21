package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fulmenhq/fularchive/internal/config"
	"github.com/fulmenhq/fularchive/internal/provider"
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

func TestTopics_ParsesGitHubRawFields(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "providers.yaml")
	configBody := []byte(`archive_root: /tmp/archive
topics:
  - slug: data-platform
    providers:
      - slug: trino
        name: Trino
        base_url: https://trino.io/docs/current
        fetch_strategy: github-raw
        github_repo: trinodb/trino
        github_docs_path: docs/src/main/sphinx/
        github_branch: master
        auth_env_var: GITHUB_TOKEN
        enabled: false
        rate_limit:
          requests_per_second: 3
          burst: 2
        paths:
          - "**/*.md"
`)
	if err := os.WriteFile(configPath, configBody, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("FULARCHIVE_CONFIG", configPath)

	if err := config.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	topics := config.Topics()
	if len(topics) != 1 {
		t.Fatalf("len(Topics()) = %d, want 1", len(topics))
	}
	if len(topics[0].Providers) != 1 {
		t.Fatalf("len(Topics()[0].Providers) = %d, want 1", len(topics[0].Providers))
	}

	got := topics[0].Providers[0]
	if got.FetchStrategy != provider.StrategyGitHubRaw {
		t.Fatalf("FetchStrategy = %q, want %q", got.FetchStrategy, provider.StrategyGitHubRaw)
	}
	if got.GitHubRepo != "trinodb/trino" {
		t.Fatalf("GitHubRepo = %q, want trinodb/trino", got.GitHubRepo)
	}
	if got.GitHubDocsPath != "docs/src/main/sphinx/" {
		t.Fatalf("GitHubDocsPath = %q, want docs/src/main/sphinx/", got.GitHubDocsPath)
	}
	if got.GitHubBranch != "master" {
		t.Fatalf("GitHubBranch = %q, want master", got.GitHubBranch)
	}
	if got.AuthEnvVar != "GITHUB_TOKEN" {
		t.Fatalf("AuthEnvVar = %q, want GITHUB_TOKEN", got.AuthEnvVar)
	}
	if got.Enabled == nil || *got.Enabled {
		t.Fatalf("Enabled = %v, want false", got.Enabled)
	}
	if got.RateLimit == nil {
		t.Fatal("RateLimit = nil, want parsed config")
	}
	if got.RateLimit.RequestsPerSecond != 3 {
		t.Fatalf("RateLimit.RequestsPerSecond = %v, want 3", got.RateLimit.RequestsPerSecond)
	}
	if got.RateLimit.Burst != 2 {
		t.Fatalf("RateLimit.Burst = %d, want 2", got.RateLimit.Burst)
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
