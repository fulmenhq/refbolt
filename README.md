# refbolt

**Bolt down the reference docs and ship faster.**

API docs move. Refbolt pins them down. It fetches provider documentation — via
llms.txt, GitHub raw, or direct HTTP — and produces clean, date-versioned Markdown
trees ready for offline consumption. One command, reproducible snapshots, no drift.

## Features

- 24 providers across 8 topics (LLM APIs, cloud infra, data platforms, design platforms, and more)
- 5 fetch strategies: native, jina, auto, github-raw, llmstxt-hierarchical
- Incremental sync — skips unchanged providers via per-provider `.sync-meta.json` hints
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

### Run in Docker

Prefer containers? The repo ships a `docker-compose.yml` with a one-shot CLI
and a scheduled runner — see the [Docker section](#docker) below.

## Prerequisites

refbolt works out of the box for most providers. Two optional API keys unlock higher rate limits:

| Variable       | When needed                                          | How to get it                        |
| -------------- | ---------------------------------------------------- | ------------------------------------ |
| `JINA_API_KEY` | Providers using Jina Reader (OpenAI)                 | https://jina.ai/reader               |
| `GITHUB_TOKEN` | GitHub-backed providers (Trino, kubectl, Mattermost) | GitHub Settings → Developer → Tokens |

Set them before syncing:

```bash
export JINA_API_KEY=jina_...
export GITHUB_TOKEN=ghp_...
```

Without these keys, anonymous access works but may hit rate limits on repeated syncs. See [docs/development.md](docs/development.md) for details.

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

### Browse the catalog

See everything refbolt knows about before you sync:

```bash
refbolt catalog list                    # table of all providers
refbolt catalog list --topic llm-api    # filter by topic
refbolt catalog list --strategy jina    # filter by fetch strategy
refbolt catalog list --json             # machine-readable JSON
refbolt catalog show anthropic          # full detail for one provider
refbolt catalog topics                  # topic summary with counts
```

The `catalog` command reads from data embedded in the binary — it runs without
a `providers.yaml` and has no network dependency.

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

The repo ships a `docker-compose.yml` with three services — a one-shot CLI, a
scheduled runner, and a git-aware runner profile. All three share a host
bind-mounted `./archive/` directory so snapshots stay visible to your editor.

```bash
# 1. Generate a providers.yaml at the repo root (the compose services mount it)
refbolt init --all --output providers.yaml

# 2. Build the images
docker compose build

# 3. Run a one-shot sync
docker compose run --rm refbolt sync --all --verbose

# 4. Or start the scheduled runner (daily 06:00 UTC by default)
docker compose up -d runner
docker compose logs -f runner

# 5. Git-aware scheduled runner (mounts worktree + ~/.ssh for signed pushes)
docker compose --profile git up -d runner-git

# Stop everything — the ./archive/ directory persists
docker compose down
```

Archive storage is a host bind mount by default so the snapshot tree is
directly visible to your editor and diff tools. For orchestrated or NFS
deployments, add a `compose.override.yml` — see
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the override pattern and
forward plan for object-store backends.

For scripting or CI without compose, the raw `docker run` recipes still work:

```bash
# One-shot sync with explicit mounts
docker run --rm \
  -v ./providers.yaml:/work/providers.yaml:ro \
  -v ./archive:/data/archive \
  refbolt:local sync --all --verbose
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
