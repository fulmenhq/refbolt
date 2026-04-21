// Package assets embeds the provider catalog, schema, and registry into the binary.
//
// IMPORTANT: The files in this directory are DERIVED COPIES — do NOT edit them directly.
//
//	Source of truth:
//	  catalog.yaml   ← configs/providers.yaml
//	  schema.yaml    ← schemas/providers/v0/providers.schema.yaml
//	  registry.jsonl ← registry/providers.jsonl
//
//	To update: run `make embed-assets` (also runs automatically as part of `make build`).
//	Edits to catalog.yaml / schema.yaml / registry.jsonl will be silently overwritten on the next build.
package assets

import _ "embed"

//go:embed catalog.yaml
var Catalog []byte

//go:embed schema.yaml
var Schema []byte

//go:embed registry.jsonl
var Registry []byte
