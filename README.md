# refbolt

**Bolt down the reference docs and ship faster.**

API docs move. Refbolt pins them down. It fetches provider documentation — via
llms.txt, GitHub raw, or direct HTTP — and produces clean, date-versioned Markdown
trees ready for offline consumption. One command, reproducible snapshots, no drift.

## Features

- 24 providers across 7 topics (LLM APIs, cloud infra, data platforms, and more)
- 5 fetch strategies: native, jina, auto, github-raw, llmstxt-hierarchical
- Incremental sync — skips unchanged providers using tree SHA / HEAD hints
- Provider/topic filtering: `--provider`, `--topic`, `--exclude-provider`
- Git automation: `--git-commit`, `--git-push`, `--git-trailer`
- Built-in provider catalog — works without cloning the repo
- Container-ready with slim CLI and scheduled runner images

## Quick Start

### Install and run (no source tree needed)

```bash
# Install via Homebrew
brew install fulmenhq/tap/refbolt

# Generate a config with the topics you need
refbolt init --topic llm-api --output providers.yaml

# Or start with everything and trim later
refbolt init --all --output providers.yaml

# Run your first sync
refbolt sync --all
```

### Build from source

```bash
make build
./bin/refbolt sync --all --verbose
```

## Configuration

refbolt ships with an embedded catalog of all supported providers. `refbolt init` generates a `providers.yaml` from this catalog with the topics you select.

### Config resolution

refbolt looks for config in this order:

1. `--config <path>` (explicit flag)
2. `REFBOLT_CONFIG` env var
3. `./providers.yaml` (current directory)
4. `~/.config/refbolt/providers.yaml` (user config)
5. Built-in catalog (zero-config fallback)

### Validate your config

```bash
refbolt validate                        # check ./providers.yaml
refbolt validate --config my-config.yaml  # check a specific file
```

### Selective sync

Not every project needs all 24 providers. Pick what you need:

```bash
# Just the LLM API docs
refbolt sync --topic llm-api

# Only Anthropic and OpenAI
refbolt sync --provider anthropic --provider openai

# Everything except the large GitHub-backed repos
refbolt sync --all --exclude-provider trino --exclude-provider kubernetes-kubectl

# Only one AWS service
refbolt sync --provider aws-bedrock-userguide
```

## Docker

```bash
# Generate config locally, then mount it
refbolt init --all --output providers.yaml

# One-shot sync
docker run --rm \
  -v ./providers.yaml:/work/providers.yaml:ro \
  -v ./archive:/data/archive \
  refbolt:local sync --all --verbose

# Scheduled runner (cron-based)
docker run -d \
  -v ./providers.yaml:/work/providers.yaml:ro \
  -v ./archive:/data/archive \
  -v ./crontab:/etc/refbolt/crontab:ro \
  refbolt-runner:local
```

## Intended Use

refbolt is designed for local, offline developer reference — keeping version-pinned copies of documentation you're actively building against. It is not intended for republishing, redistributing, or publicly hosting archived content. Respect the terms of service of the documentation sites you archive.

## Documentation

- [Architecture](docs/ARCHITECTURE.md) — component design, storage model, deployment topology
- [Development](docs/development.md) — build, test, env vars, fetch strategies, git automation
- [Providers](docs/providers/README.md) — per-provider fetch notes and verification status
- [CI/CD](docs/cicd.md) — workflow triggers, test tiers, signing
- [Decision Records](docs/decisions/) — ADRs, SDRs, DDRs

## License

Dual-licensed under [Apache License 2.0](LICENSE) and [MIT License](LICENSE-MIT). See the respective files for details.
