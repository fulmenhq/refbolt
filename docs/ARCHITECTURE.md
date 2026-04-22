---
title: "refbolt Architecture"
description: "Component, storage, and deployment architecture for refbolt as of v0.0.3"
author: "3leapsdave"
date: "2026-04-20"
status: "active"
version: "1.0.0"
tags: ["refbolt", "architecture", "docs"]
---

# refbolt Architecture

## Purpose and Scope

refbolt fetches third-party documentation — LLM APIs, cloud platforms, data and
container tooling, collaboration suites, self-hosted services, and Python
libraries — and produces clean, date-versioned Markdown trees on local disk.
The target audience is anyone who wants a pinned, offline-reviewable copy of
the docs they build against: individual developers, agent pipelines that need
stable reference material, and teams sharing a common archive via bind-mounted
or network storage.

The CLI is a single self-contained Go binary. It ships with an embedded
provider catalog and schema, so running `refbolt sync --all` against a fresh
checkout produces a usable archive with no further configuration. Users who
want to pick a subset generate their own `providers.yaml` via `refbolt init`.

refbolt is not a crawler. It prefers official Markdown endpoints
(`llms.txt` / `llms-full.txt` / `.md` suffix), falls back to Jina Reader for
HTML-only sites, and documents its fetch stance per-provider under
`docs/providers/`. See [SDR-0001](decisions/SDR-0001-ethical-fetching-policy.md)
for the ethical-fetching policy.

## Architecture Principles

1. **Native where possible.** Prefer `llms.txt`, `.md`-suffixed pages, and
   OpenAPI specs over HTML scraping. Details: [ADR-0001](decisions/ADR-0001-llmstxt-primary-fetch.md).
2. **Self-contained binary.** The provider catalog and JSON Schema are
   embedded via `go:embed`. Users can override, but they never have to.
3. **Config-driven.** A single `providers.yaml` is the source of truth for
   what to fetch; env vars override paths and credentials.
4. **Incremental by default.** Per-provider `.sync-meta.json` lets sync skip
   unchanged providers; `--force` bypasses.
5. **Container-optional.** The CLI is a first-class local-binary workflow.
   Docker images and a `docker-compose.yml` exist for scheduled or
   cross-machine use, but nothing is Docker-only.
6. **Ethical by policy.** robots.txt and per-site ToS reviews are
   prerequisites for every new provider; no undocumented endpoints,
   session-credential harvesting, or paywall circumvention.
   See [SDR-0001](decisions/SDR-0001-ethical-fetching-policy.md).
7. **Prefer ecosystem libraries.** Shared utilities live in gofulmen /
   3leaps-org libraries rather than being re-rolled per project.
   See [ADR-0002](decisions/ADR-0002-prefer-fulmenhq-libraries.md).

## Platform Topology

```
                     ┌──────────────────────────────┐
                     │  User config (optional)      │
                     │   providers.yaml             │
                     └───────────────┬──────────────┘
                                     │
┌────────────────────────────────────▼────────────────────────────────────┐
│                              refbolt CLI                                 │
│  init · validate · sync · version   (Cobra; embedded catalog & schema)   │
└────────────────────┬─────────────────────────────────┬──────────────────┘
                     │                                 │
        ┌────────────▼──────────────┐     ┌────────────▼────────────┐
        │  Fetchers                 │     │  Incremental sync       │
        │  native · jina · auto     │     │  .sync-meta.json        │
        │  github-raw · hierarchical│     │  config/content hashes  │
        └────────────┬──────────────┘     └────────────┬────────────┘
                     │                                 │
        ┌────────────▼─────────────────────────────────▼────────────┐
        │  Archive writer                                           │
        │  <root>/<topic>/<provider>/<YYYY-MM-DD>/<path>            │
        │  + latest → <YYYY-MM-DD>   (SHA-256 dedup per file)       │
        └────────────────────────────┬──────────────────────────────┘
                                     │
                ┌────────────────────▼────────────────────┐
                │  Optional git automation                │
                │  --git-commit · --git-push · --git-*    │
                └─────────────────────────────────────────┘
```

Deployment targets — local binary, Docker CLI image, `docker compose` with CLI
and scheduled runner services — sit around this core. None are required.

## Component Architecture

### 1. CLI (Go + Cobra)

Four top-level commands. Everything else is discoverable via `--help`.

| Command            | Purpose                                                                                |
| ------------------ | -------------------------------------------------------------------------------------- |
| `refbolt init`     | Generate a `providers.yaml` from the embedded catalog (`--topic`, `--all`, `--output`) |
| `refbolt validate` | Validate a config against the embedded JSON Schema; warn about missing credentials     |
| `refbolt sync`     | Fetch, dedup, and write the selected providers; optionally commit and push             |
| `refbolt version`  | Print version, commit, and build date                                                  |

Global flags: `--config <path>`, `--verbose`.

`refbolt sync` flags:

- Selection: `--all`, `--provider <slug>` (repeatable), `--topic <slug>` (repeatable), `--exclude-provider <slug>` (repeatable)
- Incremental: `--force` (bypass `.sync-meta.json`)
- Git: `--git-commit`, `--git-push`, `--git-branch <name>`, `--git-trailer <line>` (repeatable)

Sources: `internal/cmd/{init,validate,sync,version}.go`, `internal/cmd/root.go`.

### 2. Embedded Catalog and Schema

The binary ships with a curated catalog of 27 providers across 8 topics plus
the JSON Schema that validates them:

- Source of truth: `configs/providers.yaml`, `schemas/providers/v0/providers.schema.yaml`
- Embedded copies: `assets/catalog.yaml`, `assets/schema.yaml` — refreshed by `make embed-assets`
- Zero-config fallback: when no user config is found, `sync` runs against the embedded catalog

Topics: `llm-api`, `python-libs`, `cloud-infra`, `data-platform`,
`container-platform`, `collaboration`, `self-hosted-suite`, `design-platform`.

### 3. Provider Registry

`registry/providers.jsonl` records capability metadata (llms.txt availability,
`md_suffix` pattern, GitHub source, OpenAPI, ToS-review status, verification
date, site quirks) for every provider known to the project. It currently
contains 28 entries, one more than the shipped catalog — `aws-cli` is
described in the registry but not yet wired into `configs/providers.yaml`.
See the Open Questions section.

The registry is documentation tooling (tracked, reviewable, human-readable),
not a runtime artifact. The binary never reads it.

### 4. Fetch Strategies

Five strategies. Each one handles a family of sites and produces a uniform
`[]provider.Page` for the archive writer.

| Strategy               | When to use                                                                           | Implementation                      |
| ---------------------- | ------------------------------------------------------------------------------------- | ----------------------------------- |
| `native`               | Site offers `llms.txt` / `llms-full.txt` / `.md` suffix / OpenAPI endpoint            | `internal/provider/http.go`         |
| `jina`                 | HTML-only site; use Jina Reader (`r.jina.ai`) for HTML→Markdown conversion            | `internal/provider/http.go`         |
| `auto`                 | Try `native` first; fall back to `jina` if the native endpoint returns nothing usable | `internal/provider/http.go`         |
| `github-raw`           | Docs hosted as Markdown in a GitHub repo (tree API discovery + raw content fetch)     | `internal/provider/github.go`       |
| `llmstxt-hierarchical` | Cloud-provider aggregate indexes (AWS, Azure, GCP) with per-service `llms.txt` files  | `internal/provider/hierarchical.go` |

The `native` strategy supports three splitter variants for `llms.txt` / `llms-full.txt`:

- **URL-delimited**: xAI-style `===/<path>===` section markers
- **Source-delimited**: `URL: <url>` markers between `# Title` headings
- **YAML frontmatter**: sections separated by `---` frontmatter blocks

Splitter dispatch lives in `internal/provider/llmstxt.go`.

Hierarchical resolution (DDR-0002) requires every hierarchical provider entry
to use a guide-specific `base_url` that matches exactly one entry in the upstream
index, so selection is deterministic.

### 5. Incremental Sync

Each provider writes a `.sync-meta.json` alongside its archive output at
`<root>/<topic>/<provider>/.sync-meta.json` (outside the date directories —
it's per-provider state, not per-snapshot). The file is a `SyncMeta` struct
with these fields (`internal/sync/metadata.go`):

| Field          | Purpose                                                                                    |
| -------------- | ------------------------------------------------------------------------------------------ |
| `config_hash`  | SHA-256 of the provider's resolved config fields; invalidates meta if config changes       |
| `last_sync`    | Timestamp of the last successful sync for this provider                                    |
| `content_hash` | SHA-256 of the aggregate content from the previous sync                                    |
| `file_count`   | Number of files written in the previous sync                                               |
| `fetch_hint`   | Strategy-specific upstream signals (`etag`, `last_modified`, `content_length`, `tree_sha`) |

Two levels of skip during `refbolt sync`:

1. **Fetch-level skip** — `internal/cmd/sync.go` computes the current
   `config_hash`, reads stored meta, and calls `ShouldSkip(meta, hash)`. If
   the config matches and the fetcher implements the `HintChecker` interface,
   the sync command calls `CheckHints(ctx)` to pull the current upstream
   hints (HEAD request for ETag/Last-Modified, GitHub tree SHA, etc.). If the
   returned hint matches the stored `fetch_hint`, the provider is skipped —
   no content download, no writer call.
2. **Write-level dedup** — when a fetch does happen, the archive writer
   hashes each page's content (SHA-256) against the file already on disk at
   the same path. Matching content bumps `WriteStat.Skipped`; changed or new
   content bumps `WriteStat.Written`.

`--force` on `refbolt sync` bypasses the fetch-level check entirely.
Write-level dedup is always applied.

Sources: `internal/sync/metadata.go`, `internal/provider/provider.go` (FetchHint + HintChecker interface), `internal/cmd/sync.go:83-110`, `internal/archive/writer.go:50-69`.

### 6. Archive Writer

Writes fetched pages to a date-versioned tree; implementation at
`internal/archive/writer.go`.

Tree structure:

```
<archive_root>/
└── <topic>/
    └── <provider>/
        ├── 2026-04-20/
        │   └── <page-path>.md
        ├── 2026-04-21/
        │   └── <page-path>.md
        ├── latest → 2026-04-21
        └── .sync-meta.json
```

- **Date-keyed directories** (`YYYY-MM-DD/`) — the writer formats
  `time.Now()` to a calendar date and writes pages under that subdirectory.
  **Same-day re-syncs reuse the directory**: changed files are overwritten
  in place, unchanged files are skipped via SHA-256 dedup. A new calendar
  day produces a new directory on the next sync.
- **`latest/` symlink** — updated atomically after any write to point at
  the current date directory. Downstream consumers (editors, agents, diff
  tools) read from `latest/` for always-current content.
- **SHA-256 content dedup** — before writing each page, the writer
  compares new content against the existing file at the same path. Matching
  content is skipped (`WriteStat.Skipped++`); changed or new content is
  written (`WriteStat.Written++`). This avoids disk churn and keeps git
  diffs clean when nothing meaningful changed.
- **Symlink failures are non-fatal** — on filesystems without symlink
  support (certain Windows + FAT mounts), a warning is logged and the
  run continues.

Design rationale: [DDR-0001](decisions/DDR-0001-archive-tree-structure.md).

### 7. Git Automation

Optional post-sync step. Flags live on `refbolt sync`:

| Flag            | Effect                                                                    |
| --------------- | ------------------------------------------------------------------------- |
| `--git-commit`  | Stage archive changes and commit with a structured message (format below) |
| `--git-push`    | Push to the remote after commit (requires `--git-commit`)                 |
| `--git-branch`  | Override the remote branch (default: current branch)                      |
| `--git-trailer` | Append trailer line(s) to the commit message (repeatable)                 |

Commit message format (`internal/git/message.go`):

```
refbolt sync: 2026-04-20

Providers updated:
- xai: 96 files (llm-api)
- anthropic: 488 files (llm-api)

Archive root: /data/archive

<trailers, if any>
```

Pre-flight checks: `git` must be on PATH, the archive root must live inside a
git worktree, and non-archive files in the worktree are left untouched (only
files under `<archive_root>` are staged). No `--force` push; no empty commits
(skipped if nothing changed).

## Deployment Models

### Local binary (primary)

Distribution channels:

- `brew install fulmenhq/tap/refbolt` (macOS / Linux Homebrew)
- `scoop install refbolt` (Windows, once the bucket is public)
- GitHub Releases (all platforms, including Windows ARM64)
- `go install github.com/fulmenhq/refbolt/cmd/refbolt@latest`
- `make install` from source (respects `DESTDIR` / `PREFIX`)

All distributions produce identical binaries — the embedded catalog and
schema mean no post-install configuration is required.

### Docker (CLI image)

`Dockerfile` produces a distroless image (~8 MB) intended for scripting and
CI use. One-shot usage:

```bash
make docker-build
docker run --rm refbolt:local version

docker run --rm \
  -e REFBOLT_CONFIG=/work/providers.yaml \
  -v ./providers.yaml:/work/providers.yaml:ro \
  -v ./archive:/data/archive \
  refbolt:local sync --all --verbose
```

### Docker runner (scheduled)

`Dockerfile.runner` adds supercronic, git, ssh, and a validated entrypoint
(`docker/runner-entrypoint.sh`) on top of the CLI. Useful when you want a
persistent scheduler without compose.

```bash
make docker-build-runner

# Non-git scheduled sync
# TZ=UTC keeps cron firing and archive date directories on a stable zone
# across DST boundaries (see DDR-0001); override for local-time scheduling.
docker run --rm \
  -e REFBOLT_CONFIG=/work/providers.yaml \
  -e TZ=UTC \
  -v ./providers.yaml:/work/providers.yaml:ro \
  -v ./archive:/data/archive \
  -v ./examples/crontab:/etc/refbolt/crontab:ro \
  refbolt-runner:local

# Git-aware scheduled sync
docker run --rm \
  -e REFBOLT_CONFIG=/workspace/providers.yaml \
  -e REFBOLT_ARCHIVE_ROOT=/workspace/archive \
  -e REFBOLT_GIT_SAFE_DIRECTORY=/workspace \
  -e TZ=UTC \
  -v "$PWD:/workspace" \
  -v ./examples/crontab-git:/etc/refbolt/crontab:ro \
  -v "$HOME/.ssh:/root/.ssh:ro" \
  refbolt-runner:local
```

### Docker Compose (turnkey scheduled archiving)

A top-level `docker-compose.yml` wires the CLI and runner images together around a
host-bind `./archive` directory. Generate a `providers.yaml` at the repo root first
(`make build && ./bin/refbolt init --all --output providers.yaml`, or `refbolt init`
from an installed binary); the file is gitignored and mounted read-only into the
containers. Three services ship by default:

| Service      | Purpose                       | Activation                                      |
| ------------ | ----------------------------- | ----------------------------------------------- |
| `refbolt`    | One-shot CLI                  | `docker compose run --rm refbolt sync --all -v` |
| `runner`     | Scheduled runner (plain cron) | `docker compose up -d runner`                   |
| `runner-git` | Scheduled runner with git/SSH | `docker compose --profile git up -d runner-git` |

Common commands:

```bash
docker compose build                                    # build both images
docker compose run --rm refbolt version                 # sanity-check CLI
docker compose run --rm refbolt sync --all --verbose    # one-shot sync
docker compose up -d runner                             # start scheduler
docker compose logs -f runner                           # tail runner logs
docker compose --profile git up -d runner-git           # git-aware scheduler
docker compose down                                     # stop services (archive dir persists)
```

Archive storage expectations:

- **Host bind mount is the default** — the archive writer is POSIX
  filesystem only, so `./archive:/data/archive` keeps the snapshot tree
  directly visible to editors, diff tools, and anything else running on the
  host. This matches the `docker run` recipes above.
- **Named volume or NFS mount** for orchestrated or sidecar setups: add a
  `compose.override.yml` replacing `./archive:/data/archive` with a volume
  reference (e.g. `archive-vol:/data/archive` plus a `volumes:` block). Compose
  merges the override automatically.
- **Forward plan**: direct writes to object storage (S3 / R2 / GCS) are
  planned as a new archive backend. Once that lands, ephemeral runners (cloud
  functions, lightweight sidecars) can skip the volume entirely and archive
  straight to a bucket. Until then, a host-accessible filesystem is required.

Other notes:

- `providers.yaml` is bind-mounted read-only from the project root so edits on the
  host apply without a rebuild. `REFBOLT_CONFIG` points at the mount path.
- The `runner-git` profile mounts `~/.ssh` read-only for signed commits and pushes.
  Users with passphrase-protected keys or non-default agent configurations will
  need to adjust the mount or run `ssh-agent` on the host.
- The raw `docker run` recipes above remain valid for users who prefer them or
  need to invoke the runner outside Compose.

### Future: Object-Store Backend

A planned archive backend that writes directly to S3 / R2 / GCS via the cloud
SDK, bypassing the POSIX filesystem. Makes fully ephemeral runners possible
(cloud functions, stateless sidecars) and lifts the "host-accessible
filesystem required" constraint. Not in v0.0.3; tracking ticket to follow.

## Configuration Model

### Resolution chain

Implemented at `internal/config/config.go` (`ResolveConfigPath`):

1. `--config <path>` (explicit flag)
2. `REFBOLT_CONFIG` env var
3. `./providers.yaml` (current directory)
4. `~/.config/refbolt/providers.yaml` (XDG user config)
5. Embedded catalog (zero-config fallback)

No default path beyond step 5 — the binary always has something to work with.

### Schema

JSON Schema 2020-12, stored at `schemas/providers/v0/providers.schema.yaml`
and embedded alongside the catalog. Two validation surfaces:

- **Strict** — `refbolt validate` fails on any violation and prints the
  specific instance locations.
- **Permissive startup** — `refbolt sync` accepts configs with optional
  fields missing and logs warnings rather than refusing to run.

### Credentials

Only env var names appear in provider config (`auth_env_var: GITHUB_TOKEN`);
credential values are always read from the environment at runtime. Never
stored in config files, never echoed in verbose output.

| Variable       | Used by                                               | Guidance                                                                         |
| -------------- | ----------------------------------------------------- | -------------------------------------------------------------------------------- |
| `JINA_API_KEY` | `jina` and `auto` strategies                          | Optional but recommended; anonymous Jina Reader is aggressively rate-limited     |
| `GITHUB_TOKEN` | `github-raw` strategy (Trino, kubectl, Mattermost, …) | Optional for small trees; required in practice once tree API rate limits kick in |

First-run guidance lands via `refbolt init` credential hints and
`refbolt validate` warnings. See
[SDR-0001](decisions/SDR-0001-ethical-fetching-policy.md) for the broader
credential-hygiene stance.

## Ethical Fetching

refbolt fetches only through publicly documented, intended-for-consumption
interfaces. No reverse-engineered endpoints, no session-token reuse, no
scraping past robots.txt or ToS restrictions. Every new provider requires a
robots.txt and ToS review, documented in `docs/providers/`.

The full policy and rationale: [SDR-0001](decisions/SDR-0001-ethical-fetching-policy.md).
Per-provider operational notes: [`docs/providers/README.md`](providers/README.md).

## Open Design Questions

- **Promote or retire `aws-cli`?** The registry describes it but the shipped
  catalog does not. Decide whether to wire it into `configs/providers.yaml`
  or remove the registry entry.
- **Object-store archive backend.** Priority and scope for S3 / R2 / GCS
  direct writes (see Forward above).
- **Breaking-change notifications.** Slack / Teams / webhook notifications on
  detected upstream drift — useful signal, adds a delivery surface.
- **Custom providers via external YAML URL.** Letting users point refbolt at
  a remote `providers.yaml` (signed, pinned) for shared team configs.
- **Decision-support UI.** A tiny offline viewer for browsing the archive
  without a full editor — still open, no design yet.
