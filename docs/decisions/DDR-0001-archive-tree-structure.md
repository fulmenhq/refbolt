# DDR-0001: Archive Tree Structure

**Status**: Accepted
**Date**: 2026-03-21
**Deciders**: @3leapsdave

## Context

refbolt needs a filesystem layout for archived documentation that is stable enough for downstream consumers (Lanyte backends, Tauri apps) to depend on, while supporting date-versioned snapshots and multiple providers grouped by topic.

## Decision

The archive tree follows this structure:

```
<archive_root>/
└── <topic>/
    └── <provider>/
        ├── 2026-03-21/
        │   ├── developers/tools/overview.md
        │   ├── developers/rest-api-reference/inference/chat.md
        │   └── llms.txt
        ├── 2026-03-22/
        │   └── ...
        └── latest → 2026-03-22
```

- **Topic** (e.g. `llm-api`): groups providers by domain. Slug-formatted.
- **Provider** (e.g. `xai`): one directory per documentation source. Slug-formatted.
- **Date** (`YYYY-MM-DD`): each sync creates a new date directory. Files are never modified after creation.
- **`latest` symlink**: points to the most recent date directory. Updated atomically on each sync.
- **Page paths**: mirror the source URL path structure with `.md` extension added where missing.

## Consequences

- Downstream consumers can point at `<root>/llm-api/xai/latest/` for always-current docs.
- Historical snapshots are preserved and diffable (`diff -r 2026-03-21/ 2026-03-22/`).
- Disk usage grows linearly with sync frequency — future work may add deduplication or pruning.
- Topic/provider slugs are enforced by schema validation, preventing filesystem-unsafe names.

## Alternatives Considered

- **Flat provider directories** (no topic grouping): simpler but doesn't scale to non-LLM doc sites.
- **Git-only versioning** (no date directories): considered, but the date-directory approach works without git and is easier to browse.
- **Content-addressable storage**: over-engineered for the current use case.
