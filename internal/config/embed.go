package config

// SetEmbeddedAssets is called from main to inject build-time embedded
// catalog and schema. The signature is preserved for backward compatibility;
// registry bytes are set separately via SetEmbeddedRegistry so adding new
// assets does not cascade into every test's setup call.
func SetEmbeddedAssets(catalog, schema []byte) {
	embeddedCatalog = catalog
	embeddedSchema = schema
}

// SetEmbeddedRegistry is called from main to inject the build-time embedded
// provider registry (JSONL). Independent of SetEmbeddedAssets so callers can
// opt in without migrating older test fixtures.
func SetEmbeddedRegistry(registry []byte) {
	embeddedRegistry = registry
}

var (
	embeddedCatalog  []byte
	embeddedSchema   []byte
	embeddedRegistry []byte
)

// EmbeddedCatalog returns the built-in provider catalog YAML.
func EmbeddedCatalog() []byte {
	return embeddedCatalog
}

// EmbeddedSchema returns the built-in provider schema YAML.
func EmbeddedSchema() []byte {
	return embeddedSchema
}

// EmbeddedRegistry returns the built-in provider registry JSONL.
func EmbeddedRegistry() []byte {
	return embeddedRegistry
}
