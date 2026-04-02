# Release Notes

This document tracks release notes for refbolt releases.

> **Convention**: Keep only the latest 3 releases here to prevent file bloat. Older releases are archived in `docs/releases/`.

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

## [0.0.1] - 2026-03-23

**Bolt down the reference docs and ship faster.**

First functional release of refbolt — a container-first CLI for archiving web documentation into clean, date-versioned Markdown trees.

### Highlights

- **8 providers verified**: xAI, Anthropic, Pydantic, OpenAI, Trino, kubectl, AWS Glue, AWS Bedrock
- **5 fetch strategies**: native, jina, auto, github-raw, llmstxt-hierarchical
- **Git automation**: `--git-commit`, `--git-push`, `--git-trailer` for scheduled archive workflows
- **Two container images**: slim CLI (distroless, ~8MB) and scheduled runner (Debian + supercronic + git, ~80MB)
- **CI/CD**: GitHub Actions for CI on push/PR, draft release on `v*` tags with manifest-only signing handoff

### Providers

| Provider               | Strategy             | Pages       | Status   |
| ---------------------- | -------------------- | ----------- | -------- |
| xAI / Grok             | llmstxt-split        | 96          | Verified |
| Anthropic              | llmstxt-split        | 488         | Verified |
| Pydantic               | llmstxt-single       | ~100        | Verified |
| OpenAI                 | jina                 | 3 + OpenAPI | Verified |
| Trino                  | github-raw           | 641         | Verified |
| kubectl                | github-raw           | 121         | Verified |
| AWS Glue               | llmstxt-hierarchical | ~300        | Verified |
| AWS Bedrock (UG + API) | llmstxt-hierarchical | ~400        | Verified |

### Quick Start

```bash
# Build from source
make build
./bin/refbolt sync --all --verbose

# Docker
make docker-build
docker run --rm \
  -v ./archive:/data/archive \
  refbolt:local sync --all --verbose
```

### What's Next

- docker-compose for turnkey scheduled archiving (FA-012)
- Sitemap-based discovery for providers without llms.txt (FA-052)
- Tag/search CLI for provider registry (FA-053)
