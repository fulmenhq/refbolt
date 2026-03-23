# Release Notes

This document tracks release notes for refbolt releases.

> **Convention**: Keep only the latest 3 releases here to prevent file bloat. Older releases are archived in `docs/releases/`.

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
