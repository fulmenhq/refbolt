# refbolt

**Bolt down the reference docs and ship faster.**

API docs move. Refbolt pins them down. It fetches provider documentation — via
llms.txt, GitHub raw, or direct HTTP — and produces clean, date-versioned Markdown
trees ready for offline consumption. One command, reproducible snapshots, no drift.

## Features

- Native Markdown fetching (`.md` suffix or `/llms-full.txt` where available)
- Jina Reader fallback for noisy sites
- Git-aware diff/commit/PR for change detection
- Envvar/config-driven tree (e.g. `/data/archive/openai/2026-03-21/`)
- Container-ready design (Dockerfile planned)

## Quick Start

```bash
# Build from source
make build

# One-shot sync (all providers)
./bin/refbolt sync --all --verbose
```

## Intended Use

refbolt is designed for local, offline developer reference — keeping version-pinned copies of documentation you're actively building against. It is not intended for republishing, redistributing, or publicly hosting archived content. Respect the terms of service of the documentation sites you archive.

## Documentation

- [Architecture](docs/ARCHITECTURE.md) — component design, storage model, deployment topology
- [Vision](docs/VISION.md) — strategic rationale and long-term trajectory
- [Decision Records](docs/decisions/) — ADRs, SDRs, DDRs

## License

Dual-licensed under [Apache License 2.0](LICENSE) and [MIT License](LICENSE-MIT). See the respective files for details.
