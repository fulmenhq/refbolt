# DDR-0001: Archive Tree Structure

**Status**: Accepted
**Date**: 2026-03-21
**Updated**: 2026-09-15
**Deciders**: @3leapsdave

## Context

refbolt needs a filesystem layout for archived documentation that is stable
enough for downstream consumers (editors, agents, diff tools, and anything
reading from a shared archive volume) to depend on, while supporting
date-keyed snapshots and multiple providers grouped by topic.

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
        ├── latest → 2026-03-22
        └── .sync-meta.json
```

- **Topic** (e.g. `frontier-labs`, `inference-host`): groups providers by domain.
  Slug-formatted. Topics are **flat** — nested topic directories are not
  supported. The former `llm-api` topic was split in 2026-09 (see Amendment
  below).
- **Provider** (e.g. `xai`): one directory per documentation source.
  Slug-formatted.
- **Date** (`YYYY-MM-DD`): each sync writes into a directory keyed by the
  host process's local calendar date (`time.Now().Format("2006-01-02")` in
  `internal/archive/writer.go:39` — no explicit timezone conversion). The
  first sync on a given day creates the directory; subsequent same-day syncs
  reuse it. **Operational note**: scheduled runners should set `TZ=UTC` (or
  another consistent zone) so daylight-saving boundaries do not produce
  unexpected directory names. `docker-compose.yml`'s `runner` and
  `runner-git` services default `TZ=UTC` to make this easy.
- **Content dedup, then overwrite** (`internal/archive/writer.go:50-69`):
  before each page write, the writer compares new content against any file
  already present at the same path using SHA-256. Matching content is left
  untouched (`WriteStat.Skipped++`); differing content is overwritten in
  place (`WriteStat.Written++`); pages not yet present are created. This
  bounds same-day re-sync churn without introducing per-sync snapshots.
- **`latest/` symlink**: points to the most recent date directory. Updated
  atomically after any write. Non-fatal if the target filesystem does not
  support symlinks (certain Windows + FAT mounts) — a warning is logged and
  the sync continues.
- **`.sync-meta.json`**: per-provider incremental-sync state; lives at the
  provider level (outside the date directories) so it survives across day
  boundaries. See [FA-095 / `internal/sync/metadata.go`].
- **Page paths**: mirror the source URL path structure with `.md` extension
  added where missing.

## Consequences

- Downstream consumers can point at `<root>/<topic>/<provider>/latest/` for
  always-current docs.
- Historical snapshots are preserved across calendar days and diffable
  (`diff -r 2026-03-21/ 2026-03-22/`); same-day re-syncs overwrite changed
  files in place rather than proliferating directories.
- Disk growth is bounded by **unique content**, not by sync frequency —
  same-day re-runs dedup cleanly; cross-day runs accrete only when content
  actually changes upstream.
- Git workflows stay quiet: `--git-commit` stages archive files, and dedup
  means unchanged content produces no diff. A per-sync-snapshot scheme would
  churn git history on every retry, which would be actively harmful here.
- Topic/provider slugs are enforced by schema validation, preventing
  filesystem-unsafe names.

### Amendment 2026-09-15: `llm-api` → `frontier-labs` + `inference-host`

Topics remain first-order and **flat**. Nested trees such as
`inference-providers/frontier-labs/...` need a separate DDR, schema change,
and archive migration — deferred.

The former `llm-api` topic is **renamed** to `frontier-labs` (`xai`,
`anthropic`, `openai`). Third-party serverless open-weights hosts are a new
topic `inference-host` (`deepinfra`, `fireworks`, `together`). AWS Bedrock
stays under `cloud-infra`.

“Inference providers” is a **documentation umbrella only** — not a nested
directory.

**Archive migration:** existing `<archive_root>/llm-api/` trees may remain on
disk until operators re-sync into `frontier-labs/` or prune the old path.
`latest/` under the new topic is independent; there is no automatic rename of
on-disk archives.

### Design-choice matrix

Why calendar-day keying instead of per-sync snapshots:

| Consideration            | Calendar-day keying (current)  | Per-sync snapshots (rejected)       |
| ------------------------ | ------------------------------ | ----------------------------------- |
| Primary use (daily cron) | One dir per day; natural fit   | Equivalent                          |
| Same-day retries         | Overwrites cleanly; no clutter | Directory proliferation             |
| Disk growth              | Bounded by unique content      | Grows per sync, regardless of diffs |
| Git commit churn         | Low — only changed files stage | High — new tree every sync          |
| Object-store fit         | OK                             | Natural (timestamped keys)          |

### Forward plan: object-storage backend

Per-sync immutable snapshots remain interesting for a planned object-storage
archive backend (S3 / R2 / GCS), where timestamped object keys and bucket
lifecycle rules make full per-sync retention cheap and natural. When that
backend lands, this decision may be revisited for the object-store path;
calendar-day keying is expected to remain the model for filesystem backends.
See [docs/ARCHITECTURE.md → Future: Object-Store Backend](../ARCHITECTURE.md#future-object-store-backend).

## Alternatives Considered

- **Per-sync immutable snapshots** (e.g., timestamped subdirectories like
  `2026-03-22T14:30:00Z/`): rejected for filesystem backends because same-day
  retries would multiply directories with no material content difference,
  and git workflows would churn on every retry. Attractive for object-store
  backends; see Forward plan above.
- **Content-addressable storage** (e.g., by file SHA): over-engineered for
  the current use case and loses the browsable-per-date affordance.
- **Flat provider directories** (no topic grouping): simpler but doesn't
  scale to non-LLM doc sites.
- **Git-only versioning** (no date directories): considered, but the
  date-directory approach works without git and is easier to browse.
