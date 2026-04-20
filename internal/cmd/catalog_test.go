package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fulmenhq/refbolt/internal/config"
)

// setupCatalogFixture loads the real configs/providers.yaml + real
// registry/providers.jsonl into the embedded asset globals for the test.
// If a test needs a registry with specific slugs omitted, call
// overrideRegistry afterwards.
func setupCatalogFixture(t *testing.T) {
	t.Helper()

	root := findProjectRoot(t)
	catalogBytes, err := os.ReadFile(filepath.Join(root, "configs", "providers.yaml"))
	if err != nil {
		t.Skipf("configs/providers.yaml missing: %v", err)
	}
	schemaBytes, err := os.ReadFile(filepath.Join(root, "schemas", "providers", "v0", "providers.schema.yaml"))
	if err != nil {
		t.Skipf("schema missing: %v", err)
	}
	registryBytes, err := os.ReadFile(filepath.Join(root, "registry", "providers.jsonl"))
	if err != nil {
		t.Skipf("registry/providers.jsonl missing: %v", err)
	}

	config.SetEmbeddedAssets(catalogBytes, schemaBytes)
	config.SetEmbeddedRegistry(registryBytes)
	t.Cleanup(func() {
		config.SetEmbeddedAssets(nil, nil)
		config.SetEmbeddedRegistry(nil)
	})
}

// clearCatalogFlags zeroes the package-level flag globals so the previous
// test's --topic/--strategy/--json don't leak into the next.
func clearCatalogFlags(t *testing.T) {
	t.Helper()
	listTopic = ""
	listStrategy = ""
	listJSON = false
	t.Cleanup(func() {
		listTopic = ""
		listStrategy = ""
		listJSON = false
	})
}

// overrideRegistry replaces the embedded registry with the supplied lines.
func overrideRegistry(t *testing.T, jsonl string) {
	t.Helper()
	config.SetEmbeddedRegistry([]byte(jsonl))
}

// runCatalog executes rootCmd with the given args, capturing stdout and
// stderr separately so the data-vs-status contract can be asserted. The
// returned err is the command's RunE error if any.
func runCatalog(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	err = rootCmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestCatalogList_DefaultTable(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	stdout, stderr, err := runCatalog(t, "catalog", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Header + row for every catalog provider, no log noise in stdout.
	if !strings.HasPrefix(stdout, "SLUG") {
		t.Errorf("stdout should start with SLUG header, got: %q", truncate(stdout, 80))
	}
	// Spot-check a few known slugs are present.
	for _, slug := range []string{"anthropic", "openai", "trino"} {
		if !strings.Contains(stdout, slug) {
			t.Errorf("stdout missing expected slug %q", slug)
		}
	}
	for _, noise := range []string{"INFO", "WARN", "ERROR", "level=info"} {
		if strings.Contains(stdout, noise) {
			t.Errorf("stdout contains log noise %q", noise)
		}
	}

	if !strings.Contains(stderr, "providers across") {
		t.Errorf("stderr should carry the hint line, got: %q", stderr)
	}
}

func TestCatalogList_JSONEnvelope(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	stdout, _, err := runCatalog(t, "catalog", "list", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var envelope struct {
		Version        string           `json:"version"`
		TopicsTotal    int              `json:"topics_total"`
		ProvidersTotal int              `json:"providers_total"`
		Providers      []map[string]any `json:"providers"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if envelope.TopicsTotal != 7 {
		t.Errorf("topics_total = %d, want 7", envelope.TopicsTotal)
	}
	if envelope.ProvidersTotal != 23 {
		t.Errorf("providers_total = %d, want 23", envelope.ProvidersTotal)
	}
	if envelope.Version == "" {
		t.Error("version should not be empty")
	}
	if len(envelope.Providers) != envelope.ProvidersTotal {
		t.Errorf("providers array length %d != providers_total %d",
			len(envelope.Providers), envelope.ProvidersTotal)
	}
}

func TestCatalogList_JSONMissingRegistryDegrades(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	// Registry with only xai enriched; every other catalog slug should
	// serialize with null estimated_pages/description.
	overrideRegistry(t, `{"slug":"xai","description":"xAI test","estimated_pages":99}`+"\n")

	stdout, _, err := runCatalog(t, "catalog", "list", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var envelope struct {
		Providers []map[string]any `json:"providers"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("json parse: %v", err)
	}

	var sawNull, sawEnriched bool
	for _, p := range envelope.Providers {
		if p["slug"] == "xai" {
			sawEnriched = true
			if p["estimated_pages"] != float64(99) {
				t.Errorf("xai.estimated_pages = %v, want 99", p["estimated_pages"])
			}
		} else {
			// Every non-xai provider should have null enrichment fields.
			if p["estimated_pages"] != nil {
				t.Errorf("%v.estimated_pages should be null, got %v", p["slug"], p["estimated_pages"])
			}
			if p["description"] != nil {
				t.Errorf("%v.description should be null, got %v", p["slug"], p["description"])
			}
			sawNull = true
		}
	}
	if !sawEnriched || !sawNull {
		t.Errorf("expected both enriched and degraded rows; enriched=%v degraded=%v", sawEnriched, sawNull)
	}
}

func TestCatalogList_FilterByTopic(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	stdout, _, err := runCatalog(t, "catalog", "list", "--topic", "llm-api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Every data row should mention llm-api as topic column (or the slug is
	// a known llm-api provider). Quick check: expect anthropic present,
	// trino (data-platform) absent.
	if !strings.Contains(stdout, "anthropic") {
		t.Error("llm-api filter missing anthropic")
	}
	if strings.Contains(stdout, "trino") {
		t.Error("llm-api filter should not contain trino")
	}
}

func TestCatalogList_UnknownTopicErrors(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	_, _, err := runCatalog(t, "catalog", "list", "--topic", "not-a-topic")
	if err == nil {
		t.Fatal("expected error for unknown topic")
	}
	if !strings.Contains(err.Error(), "unknown topic") {
		t.Errorf("error should mention unknown topic, got: %v", err)
	}
}

func TestCatalogList_FilterByStrategy(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	stdout, _, err := runCatalog(t, "catalog", "list", "--strategy", "jina")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// openai is the lone jina provider in the catalog.
	if !strings.Contains(stdout, "openai") {
		t.Error("jina filter missing openai")
	}
	if strings.Contains(stdout, "anthropic") {
		t.Error("jina filter should not contain anthropic (native)")
	}
}

func TestCatalogList_UnknownStrategyErrors(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	_, _, err := runCatalog(t, "catalog", "list", "--strategy", "bogus")
	if err == nil {
		t.Fatal("expected error for unknown strategy")
	}
	if !strings.Contains(err.Error(), "unknown strategy") {
		t.Errorf("error should mention unknown strategy, got: %v", err)
	}
}

// TestCatalogList_ConfigFlagSilentlyIgnored locks in devrev's review guard:
// the --config flag is accepted (since it's a persistent flag on root) but
// must never trigger config loading on catalog subcommands. A completely
// bogus path must not cause an error.
func TestCatalogList_ConfigFlagSilentlyIgnored(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	bogusPath := filepath.Join(t.TempDir(), "absolutely-does-not-exist.yaml")
	stdout, _, err := runCatalog(t, "catalog", "list", "--config", bogusPath)
	if err != nil {
		t.Fatalf("--config should be silently ignored on catalog list, got error: %v", err)
	}
	if !strings.HasPrefix(stdout, "SLUG") {
		t.Error("catalog list output did not render — config load may have run")
	}
	// Reset the root configFlag so other tests don't see a bogus path.
	t.Cleanup(func() { configFlag = "" })
}

func TestCatalogShow_KnownSlug(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	stdout, _, err := runCatalog(t, "catalog", "show", "anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"anthropic",
		"Topic:",
		"Strategy:",
		"Archive output:",
		"<archive_root>/llm-api/anthropic",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q", want)
		}
	}
}

func TestCatalogShow_RegistryOnlySlugIsUnknown(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	_, _, err := runCatalog(t, "catalog", "show", "aws-cli")
	if err == nil {
		t.Fatal("aws-cli (registry-only) should not be listable")
	}
	if !errIsUnknownSlug(err) {
		t.Errorf("expected ErrUnknownSlug, got %T: %v", err, err)
	}
}

func TestCatalogShow_TypoGetsSuggestion(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	_, _, err := runCatalog(t, "catalog", "show", "anthroipc")
	if err == nil {
		t.Fatal("expected error for typo'd slug")
	}
	if !strings.Contains(err.Error(), "anthropic") {
		t.Errorf("error should suggest anthropic, got: %v", err)
	}
}

func TestCatalogTopics_RendersTableAndCounts(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	stdout, stderr, err := runCatalog(t, "catalog", "topics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, slug := range []string{
		"llm-api", "cloud-infra", "python-libs",
	} {
		if !strings.Contains(stdout, slug) {
			t.Errorf("topics output missing %q", slug)
		}
	}
	if !strings.Contains(stderr, "7 topics") {
		t.Errorf("stderr should mention '7 topics', got: %q", stderr)
	}
}

func TestCatalogList_StdoutJSONParsesCleanly(t *testing.T) {
	setupCatalogFixture(t)
	clearCatalogFlags(t)

	// Mirrors the contract test devrev called out: redirecting stdout to a
	// file must produce standalone valid JSON — no status lines, no hints.
	stdout, stderr, err := runCatalog(t, "catalog", "list", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var anything any
	if err := json.Unmarshal([]byte(stdout), &anything); err != nil {
		t.Fatalf("stdout should be clean JSON: %v\n---\n%s", err, stdout)
	}
	// JSON mode should emit nothing to stderr (no "N providers across…" hint).
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("stderr should be empty in JSON mode, got: %q", stderr)
	}
}

// truncate is a tiny display helper for nicer test failure output.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// readAll is used below to satisfy unused-variable warnings if future
// tests need a scratch buffer. Currently a no-op helper.
var _ = io.ReadAll
