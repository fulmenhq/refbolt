# fularchive

> Archive web docs (especially frontier LLM APIs) into clean, versioned Markdown trees.

## Overview

fularchive is a lightweight, container-first CLI that periodically snapshots documentation sites (OpenAI, Anthropic, xAI/Grok, etc.) into a structured, date-versioned archive. Outputs are clean Markdown + JSON/OpenAPI where native, perfect for offline reference in Tauri apps, Lanyte backends, or any Fulmen project.

**Not related to Apple's .doccarchive format** — this is a general web → Markdown archiver.

## Features

- Native Markdown fetching (`.md` suffix or `/llms-full.txt` where available)
- Jina Reader fallback for noisy sites
- Git-aware diff/commit/PR for change detection
- Envvar/config-driven tree (e.g. `/data/archive/openai/2026-03-21/`)
- Daily cron via supercronic in runner image

## Quick Start

```bash
# One-shot sync (all providers)
docker run --rm -v ./archive:/data ghcr.io/fulmenhq/fularchive fularchive sync --all

# Or use the runner image for scheduled jobs
docker-compose up -d
```

## Intended Use

fularchive is designed for local, offline developer reference — keeping version-pinned copies of documentation you're actively building against. It is not intended for republishing, redistributing, or publicly hosting archived content. Respect the terms of service of the documentation sites you archive.

## Documentation

- [Architecture](docs/ARCHITECTURE.md) — component design, storage model, deployment topology
- [Vision](docs/VISION.md) — strategic rationale and long-term trajectory
- [Decision Records](docs/decisions/) — ADRs, SDRs, DDRs

## License

Dual-licensed under [Apache License 2.0](LICENSE) and [MIT License](LICENSE-MIT). See the respective files for details.
