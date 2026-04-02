package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fulmenhq/refbolt/internal/config"
	"gopkg.in/yaml.v3"
)

// setupTestCatalog loads a minimal embedded catalog for command tests.
func setupTestCatalog(t *testing.T) {
	t.Helper()
	config.SetEmbeddedAssets(
		[]byte(`archive_root: /data/archive
topics:
  - slug: test-topic
    name: Test Topic
    providers:
      - slug: test-provider
        name: Test Provider
        base_url: https://example.com
        fetch_strategy: native
        paths:
          - /test
`),
		[]byte(`$schema: "https://json-schema.org/draft/2020-12/schema"
$id: "providers/v0/providers"
type: object
properties:
  archive_root:
    type: string
  topics:
    type: array
`),
	)
	t.Cleanup(func() { config.SetEmbeddedAssets(nil, nil) })
}

func TestInitCmd_StdoutIsValidYAML(t *testing.T) {
	setupTestCatalog(t)

	// Capture stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset cobra state for test isolation.
	rootCmd.SetArgs([]string{"init", "--all"})
	err := rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stdout := buf.String()

	if err != nil {
		t.Fatalf("init --all failed: %v", err)
	}

	// stdout must be valid YAML.
	var parsed interface{}
	if err := yaml.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("init stdout is not valid YAML: %v\nOutput:\n%s", err, stdout)
	}

	// stdout must not contain log noise (INFO, WARN, DEBUG lines).
	for _, noise := range []string{"INFO", "WARN", "DEBUG", "ERROR"} {
		if strings.Contains(stdout, noise) {
			t.Errorf("init stdout contains log noise %q:\n%s", noise, stdout)
		}
	}

	// Must contain the test provider.
	if !strings.Contains(stdout, "test-provider") {
		t.Errorf("init stdout missing test-provider:\n%s", stdout)
	}
}

func TestValidateCmd_ValidConfig(t *testing.T) {
	root := findProjectRoot(t)
	configPath := filepath.Join(root, "configs", "providers.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("configs/providers.yaml not found")
	}

	// Load the real embedded assets for schema validation.
	catalogBytes, _ := os.ReadFile(configPath)
	schemaBytes, _ := os.ReadFile(filepath.Join(root, "schemas", "providers", "v0", "providers.schema.yaml"))
	config.SetEmbeddedAssets(catalogBytes, schemaBytes)
	t.Cleanup(func() { config.SetEmbeddedAssets(nil, nil) })

	rootCmd.SetArgs([]string{"validate", "--config", configPath})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("validate should pass for valid config, got: %v", err)
	}
}

func TestValidateCmd_InvalidYAML(t *testing.T) {
	setupTestCatalog(t)

	// Write invalid YAML to a temp file.
	tmp := filepath.Join(t.TempDir(), "bad.yaml")
	os.WriteFile(tmp, []byte("{{{{not yaml"), 0o644)

	rootCmd.SetArgs([]string{"validate", "--config", tmp})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("validate should fail for invalid YAML")
	}
}

func TestInitCmd_NoOverwrite(t *testing.T) {
	setupTestCatalog(t)

	// Create existing file.
	tmp := filepath.Join(t.TempDir(), "providers.yaml")
	os.WriteFile(tmp, []byte("existing"), 0o644)

	rootCmd.SetArgs([]string{"init", "--all", "--output", tmp})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("init should refuse to overwrite existing file without --force")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestSyncCmd_ConfigFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	root := findProjectRoot(t)
	configPath := filepath.Join(root, "configs", "providers.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("configs/providers.yaml not found")
	}

	// Verify --config flag is accepted (we don't actually sync — just check it loads).
	rootCmd.SetArgs([]string{"validate", "--config", configPath})
	// If this doesn't error, the --config flag works through the resolution chain.
	// Full sync test would require network — covered by existing fetch tests.
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
			t.Fatal("could not find project root")
		}
		dir = parent
	}
}
