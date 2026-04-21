package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// setupRealCatalogAndRegistry wires the real configs/providers.yaml and
// registry/providers.jsonl into the config package's embedded asset globals
// for the duration of the test. Exits the test if either fixture is missing.
func setupRealCatalogAndRegistry(t *testing.T) {
	t.Helper()

	root := findConfigRoot(t)
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

	SetEmbeddedAssets(catalogBytes, schemaBytes)
	SetEmbeddedRegistry(registryBytes)
	t.Cleanup(func() {
		SetEmbeddedAssets(nil, nil)
		SetEmbeddedRegistry(nil)
	})
}

func findConfigRoot(t *testing.T) string {
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

func TestCatalogEntries_CountAndSort(t *testing.T) {
	setupRealCatalogAndRegistry(t)

	entries, err := CatalogEntries()
	if err != nil {
		t.Fatalf("CatalogEntries: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("got zero entries")
	}

	// Expected: 24 providers. Exact count guards against future drift; if
	// the catalog grows on purpose, update this number alongside the change.
	const expected = 24
	if len(entries) != expected {
		t.Errorf("CatalogEntries count = %d, want %d", len(entries), expected)
	}

	for i := 1; i < len(entries); i++ {
		if entries[i-1].Provider.Slug >= entries[i].Provider.Slug {
			t.Errorf("entries not sorted at %d: %q >= %q",
				i, entries[i-1].Provider.Slug, entries[i].Provider.Slug)
		}
	}
}

func TestCatalogEntryBySlug_RegistryOnlySlugIsUnknown(t *testing.T) {
	setupRealCatalogAndRegistry(t)

	// aws-cli is in the registry but not in the catalog; catalog is
	// authoritative, so it must surface as unknown (not list-able).
	_, err := CatalogEntryBySlug("aws-cli")
	if err == nil {
		t.Fatal("expected unknown-slug error for registry-only entry")
	}
	var unk ErrUnknownSlug
	if !errors.As(err, &unk) {
		t.Fatalf("want ErrUnknownSlug, got %T: %v", err, err)
	}
	if unk.Slug != "aws-cli" {
		t.Errorf("unk.Slug = %q, want aws-cli", unk.Slug)
	}
}

func TestCatalogEntryBySlug_TypoSuggestsCorrection(t *testing.T) {
	setupRealCatalogAndRegistry(t)

	_, err := CatalogEntryBySlug("anthroipc") // transposed letters
	var unk ErrUnknownSlug
	if !errors.As(err, &unk) {
		t.Fatalf("want ErrUnknownSlug, got %T: %v", err, err)
	}
	if unk.Suggested != "anthropic" {
		t.Errorf("Suggested = %q, want anthropic", unk.Suggested)
	}
}

func TestCatalogEntryBySlug_KnownSlugIncludesRegistry(t *testing.T) {
	setupRealCatalogAndRegistry(t)

	e, err := CatalogEntryBySlug("anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Provider.Slug != "anthropic" {
		t.Errorf("Provider.Slug = %q", e.Provider.Slug)
	}
	if e.Registry == nil {
		t.Fatal("Registry enrichment missing")
	}
	if e.Registry.EstimatedPages == 0 {
		t.Error("EstimatedPages should be non-zero for anthropic")
	}
	if e.Registry.Description == "" {
		t.Error("Description should not be empty for anthropic")
	}
}

func TestCatalogEntryBySlug_MissingRegistryDegradesGracefully(t *testing.T) {
	setupRealCatalogAndRegistry(t)

	// Simulate a catalog-only provider by clearing the registry.
	SetEmbeddedRegistry(nil)
	e, err := CatalogEntryBySlug("anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Registry != nil {
		t.Errorf("Registry should be nil when enrichment missing")
	}
}

func TestProvidersByTopic_FiltersAndErrorsOnUnknown(t *testing.T) {
	setupRealCatalogAndRegistry(t)

	llm, err := ProvidersByTopic("llm-api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(llm) == 0 {
		t.Fatal("expected at least one llm-api provider")
	}
	for _, e := range llm {
		if e.TopicSlug != "llm-api" {
			t.Errorf("wrong topic %q in llm-api filter", e.TopicSlug)
		}
	}

	_, err = ProvidersByTopic("not-a-topic")
	var unk ErrUnknownTopic
	if !errors.As(err, &unk) {
		t.Fatalf("want ErrUnknownTopic, got %T: %v", err, err)
	}
	if len(unk.Valid) == 0 {
		t.Error("ErrUnknownTopic.Valid should list available topics")
	}
}

func TestProvidersByStrategy_RejectsInvalidAndFiltersValid(t *testing.T) {
	setupRealCatalogAndRegistry(t)

	_, err := ProvidersByStrategy("bogus")
	var unk ErrUnknownStrategy
	if !errors.As(err, &unk) {
		t.Fatalf("want ErrUnknownStrategy, got %T: %v", err, err)
	}

	jina, err := ProvidersByStrategy("jina")
	if err != nil {
		t.Fatalf("jina filter: %v", err)
	}
	for _, e := range jina {
		if string(e.Provider.FetchStrategy) != "jina" {
			t.Errorf("non-jina entry %q in jina filter", e.Provider.Slug)
		}
	}
}

func TestTopicSummaries_CountsMatchEntries(t *testing.T) {
	setupRealCatalogAndRegistry(t)

	summaries, err := TopicSummaries()
	if err != nil {
		t.Fatalf("TopicSummaries: %v", err)
	}
	if len(summaries) != 8 {
		t.Errorf("want 8 topics, got %d", len(summaries))
	}

	// Verify counts sum back to the total provider count.
	total := 0
	for _, s := range summaries {
		total += s.ProviderCount
	}
	entries, err := CatalogEntries()
	if err != nil {
		t.Fatalf("CatalogEntries: %v", err)
	}
	if total != len(entries) {
		t.Errorf("topic counts sum to %d, entries = %d", total, len(entries))
	}
}

func TestSuggestSlug_PrefersUnambiguousPrefix(t *testing.T) {
	got := suggestSlug("anthr", []string{"anthropic", "openai", "xai"})
	if got != "anthropic" {
		t.Errorf("got %q, want anthropic", got)
	}
}

func TestSuggestSlug_FallsBackToLevenshtein(t *testing.T) {
	got := suggestSlug("openia", []string{"anthropic", "openai", "xai"})
	if got != "openai" {
		t.Errorf("got %q, want openai", got)
	}
}

func TestSuggestSlug_EmptyWhenFarOff(t *testing.T) {
	got := suggestSlug("zzzzzzz", []string{"anthropic", "openai", "xai"})
	if got != "" {
		t.Errorf("got %q, want empty (no candidates close enough)", got)
	}
}
