# Release Notes

This document tracks release notes for refbolt releases.

> **Convention**: Keep only the latest 3 releases here to prevent file bloat. Older releases are archived in `docs/releases/`.

## [0.0.4] - 2026-04-22

**Operational foundation, provider browsing, and four new providers.**

### Highlights

- **`docker-compose.yml`**: Three services (one-shot CLI, scheduled runner, git-aware runner via `--profile git`) around a host-bind `./archive` directory. Turnkey scheduled archiving in one command. (FA-012)
- **`refbolt catalog` command**: Browse the embedded catalog without cloning the repo. `list` / `show <slug>` / `topics` subcommands with registry enrichment, `--topic` / `--strategy` filters, `--json` envelope, and "did you mean?" suggestions. (FA-101)
- **27 providers total**: +Figma REST API (new `design-platform` topic), +Hetzner Cloud API / Cloud Docs / Networking Docs. 23 → 27 providers, 7 → 8 topics. (FA-104, FA-091)
- **Doc refresh**: `docs/ARCHITECTURE.md` fully rewritten for v0.0.3 reality; DDR-0001 reconciled against the writer; runner examples standardized on `TZ=UTC`. (FA-100, FA-105)
- **First-run usability polish** (FA-111): `refbolt --version` now works; `--verbose` actually enables debug output; every credential warning surfaces a get-a-key URL; `refbolt sync` with no selectors shows concrete hints; duplicate `Error:` lines fixed. New "Getting Started (5 minutes)" numbered walkthrough in the README.
- **Bundled fixes**: `refbolt init` now emits schema-valid YAML; compose correctly honors user config and archive root; catalog filtered totals describe the result set, not the full catalog.

### New Providers

| Provider                | Slug                 | Strategy             | Pages | Topic           |
| ----------------------- | -------------------- | -------------------- | ----- | --------------- |
| Figma REST API          | `figma-api`          | github-raw (OpenAPI) | 1     | design-platform |
| Hetzner Cloud API       | `hetzner-cloud-api`  | github-raw (OpenAPI) | 1     | cloud-infra     |
| Hetzner Cloud Docs      | `hetzner-cloud`      | jina                 | ~15   | cloud-infra     |
| Hetzner Networking Docs | `hetzner-networking` | jina                 | ~10   | cloud-infra     |

### Browse the catalog

Quickest way to feel the new command — pick a headline provider and look:

```bash
refbolt catalog show hetzner-cloud-api      # one of the new Hetzner surfaces
refbolt catalog show figma-api              # the new design-platform provider
```

Full command set:

```bash
refbolt catalog list                        # all providers, table
refbolt catalog list --topic cloud-infra    # filter by topic
refbolt catalog show <slug>                 # per-provider detail
refbolt catalog topics                      # topic summary + counts
refbolt catalog list --json                 # machine-readable
```

### Install

```bash
# macOS
brew install fulmenhq/tap/refbolt

# Windows
scoop bucket add fulmenhq https://github.com/fulmenhq/scoop-bucket
scoop install refbolt

# From source
make install    # → ~/.local/bin/refbolt

# Go install
go install github.com/fulmenhq/refbolt@v0.0.4
```

### Upgrade notes

- Regenerate any cached `providers.yaml` that was produced by pre-v0.0.4 `refbolt init` — the emitter had indentation and duration-format bugs that caused `refbolt validate` to reject the output. `refbolt init --all --output providers.yaml --force` regenerates cleanly.
- `refbolt sync --all` picks up Figma automatically via the new `design-platform` topic. Exclude with `--exclude-provider figma-api` if unwanted.
- Compose users who overrode `REFBOLT_CONFIG` on `runner-git` to point at `/workspace/configs/providers.yaml` should drop that override — the service now reads the repo-root `providers.yaml` by default.

---

## [0.0.3] - 2026-04-02

**Windows ARM64, Scoop support, and `make install`.**

### Highlights

- **6-platform builds**: Added `windows/arm64` — all cross-compiled from single runner (`CGO_ENABLED=0`)
- **`make install`**: Build + copy to `~/.local/bin` for local development
- **Scoop (Windows)**: `scoop bucket add fulmenhq https://github.com/fulmenhq/scoop-bucket && scoop install refbolt`
- **CI/CD docs**: Full cross-compilation rationale, build target table, package distribution channels

### Install

```bash
# macOS
brew install fulmenhq/tap/refbolt

# Windows
scoop bucket add fulmenhq https://github.com/fulmenhq/scoop-bucket
scoop install refbolt

# From source
make install    # → ~/.local/bin/refbolt
```

---

## [0.0.2] - 2026-04-02

**16 new providers, incremental sync, user-facing config, and public-readiness.**

### Highlights

- **24 providers total**: +DigitalOcean (6), Cloudflare (4), Mattermost (2), Nextcloud, Stalwart
- **Incremental sync**: Skip unchanged providers via tree SHA / HEAD hints. Write-level SHA-256 dedup. `--force` bypass.
- **User-facing config**: `refbolt init` generates config from embedded catalog, `refbolt validate` checks it. Works without cloning the repo.
- **Provider filtering**: `--provider`, `--topic`, `--exclude-provider` with union semantics
- **First-run guidance**: Credential hints for JINA_API_KEY and GITHUB_TOKEN in init, validate, and README

### New Providers

| Provider                | Strategy               | Pages | Topic             |
| ----------------------- | ---------------------- | ----- | ----------------- |
| DigitalOcean API        | llmstxt-split (scoped) | ~226  | cloud-infra       |
| DigitalOcean doctl      | llmstxt-split (scoped) | ~542  | cloud-infra       |
| DigitalOcean Kubernetes | llmstxt-split (scoped) | ~355  | cloud-infra       |
| DigitalOcean Databases  | llmstxt-split (scoped) | ~213  | cloud-infra       |
| DigitalOcean Droplets   | llmstxt-split (scoped) | ~60   | cloud-infra       |
| DigitalOcean Spaces     | llmstxt-split (scoped) | ~40   | cloud-infra       |
| Cloudflare Workers      | native + frontmatter   | ~407  | cloud-infra       |
| Cloudflare Pages        | native + frontmatter   | ~118  | cloud-infra       |
| Cloudflare R2           | native + frontmatter   | ~88   | cloud-infra       |
| Cloudflare KV           | native + frontmatter   | ~29   | cloud-infra       |
| Mattermost API v4       | github-raw             | ~56   | collaboration     |
| Mattermost Integration  | github-raw             | ~54   | collaboration     |
| Nextcloud Admin         | github-raw             | ~173  | self-hosted-suite |
| Stalwart Mail Server    | github-raw             | ~270  | self-hosted-suite |

### New Commands

- **`refbolt init`**: Generate `providers.yaml` from embedded catalog. `--topic`, `--all`, `--output`. YAML to stdout, status to stderr.
- **`refbolt validate`**: Check config against embedded schema. Strict exit codes + advisory credential warnings.

### Quick Start

```bash
# Install
brew install fulmenhq/tap/refbolt

# Generate config and sync
refbolt init --topic llm-api --output providers.yaml
export JINA_API_KEY=jina_...    # optional, for OpenAI
export GITHUB_TOKEN=ghp_...     # optional, for GitHub-backed providers
refbolt sync --all
```

### What's Next (v0.0.3)

- docker-compose for turnkey scheduled archiving (FA-012)
- Hetzner API docs — multi-strategy provider (FA-091)
- Shared fetch cache for DigitalOcean bulk file (tech debt from FA-090)
