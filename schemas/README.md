# Schemas Directory

JSON Schema definitions for refbolt configuration validation.

## Directory Structure

Schemas follow the Fulmen topical catalog convention:

```
schemas/
└── {topic}/
    └── {version}/
        └── {name}.schema.yaml
```

YAML format is used (instead of JSON) to allow inline comments.

```
schemas/
└── providers/
    └── v0/
        └── providers.schema.yaml    # Provider configuration schema
```

## Schema ID Convention

Schema IDs follow `{topic}/{version}/{name}`:

```
providers/v0/providers
```

## Supported JSONSchema Drafts

All schemas use Draft 2020-12. The validation system (via gofulmen/schema) supports all major drafts, but new schemas should target 2020-12.

## Validation

Schemas are validated at runtime via gofulmen's schema catalog and can be meta-validated with goneat:

```go
import "github.com/fulmenhq/gofulmen/schema"

catalog := schema.NewCatalog("./schemas")
diags, err := catalog.ValidateDataByID("providers/v0/providers", configData)
```

## See Also

- [gofulmen/schema](https://github.com/fulmenhq/gofulmen) — Schema validation library
- [JSON Schema](https://json-schema.org/) — Official specification
