---
title: "refbolt Architecture"
description: "Complete architecture for refbolt — CLI, providers, storage model, toolbox integration, and deployment topology"
author: "3leapsdave"
date: "2026-03-21"
status: "draft"
version: "0.1.0"
tags: ["refbolt", "architecture", "docs", "toolbox"]
---

# refbolt Architecture

## Vision Alignment

refbolt delivers clean, versioned Markdown archives of web documentation with zero hand-fetching. It is container-first, license-clean, and purpose-built for Lanyte native backends (especially Grok).

## Architecture Principles

1. **Native where possible** — use `.md` suffix, `/llms-full.txt`, and official OpenAPI endpoints first.
2. **Container-native** — the CLI runs anywhere; fulmen-toolbox provides the blessed images.
3. **Date-versioned + symlinks** — `provider/YYYY-MM-DD/` + `latest/` symlink for instant access.
4. **Git-aware** — optional auto-diff, commit, and PR creation on changes.
5. **Config-driven** — everything controlled by `providers.yaml` + env vars.
6. **Zero AGPL** — Jina Reader is sidecar-only (never baked in).

## Platform Topology

```
┌─────────────────────────────────────────────────────────────┐
│                    refbolt (Docker / Toolbox)             │
│                                                             │
│  ┌────────────────────┐  ┌───────────────────────────────┐  │
│  │   CLI (Go)         │  │   Providers                   │  │
│  │  - sync            │  │   - native (.md / OpenAPI)   │  │
│  │  - status          │  │   - markdown suffix          │  │
│  │  - diff            │  │   - jina-reader fallback     │  │
│  └─────────┬──────────┘  └───────────────┬───────────────┘  │
│            │                             │                   │
│  ┌─────────┴──────────┐  ┌───────────────┴───────────────┐  │
│  │   Storage Layer    │  │   Toolbox Integration         │  │
│  │  /data/archive/    │  │   - ghcr.io/fulmenhq/refbolt│  │
│  │  YYYY-MM-DD + latest│ │   - runner image + supercronic│  │
│  └────────────────────┘  └───────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Component Architecture

### 1. CLI (Go binary)

Cobra-based commands:

- `refbolt sync [--providers openai,anthropic,xai] [--git-commit]`
- `refbolt status`
- `refbolt diff`
- `refbolt serve` (future: tiny HTTP viewer)

### 2. Providers (pluggable)

| Provider  | Type     | Fetch Method                     | Outputs                     |
| --------- | -------- | -------------------------------- | --------------------------- |
| openai    | native   | Official OpenAPI + llms-full.txt | openapi.json, llms-full.txt |
| anthropic | markdown | `.md` suffix + llms-full.txt     | messages.md, tools.md, …    |
| xai       | markdown | Native Markdown + OpenAPI        | inference.md, openapi.json  |
| generic   | jina     | https://r.jina.ai/ fallback      | clean.md                    |

### 3. Storage Layer

Controlled by `REFBOLT_ARCHIVE_ROOT` (default: `/data/archive`)

```
archive/
├── openai/
│   ├── 2026-03-21/
│   │   ├── openapi.json
│   │   ├── llms-full.txt
│   │   └── changelog.md (auto-generated diff)
│   └── latest/ → symlink
├── anthropic/
├── xai/
├── custom/
└── index.json (metadata + hashes)
```

### 4. fulmen-toolbox Integration

Two images (planned, not yet published):

- `ghcr.io/fulmenhq/refbolt:latest` — slim CLI only
- `ghcr.io/fulmenhq/refbolt-runner:latest` — + supercronic, pandoc, git, jq, yq

Toolbox will handle builds, multi-arch, Cosign signing, and SBOMs.

## Deployment Models

### Current: Local binary

```bash
make build
./bin/refbolt sync --all --verbose
```

### Planned: Docker (container-first)

```bash
docker run --rm -v ./archive:/data ghcr.io/fulmenhq/refbolt refbolt sync --all
```

### Planned: Runner image + docker-compose

Daily cron via supercronic (Dockerfile and docker-compose.yml not yet checked in).

### Planned: CI via GitHub Actions

Daily workflow that runs sync and opens PR on changes.

## Open Design Questions

- Should we ship a tiny Tauri-based offline viewer as a companion binary?
- Add Slack/Teams webhook on breaking changes?
- Support custom providers via external YAML URL?
- Optional blockchain provenance for archive snapshots (future)?
