# refbolt

**Bolt down the reference docs and ship faster.**

API docs move. Refbolt pins them down. It fetches provider documentation — via
llms.txt, GitHub raw, or direct HTTP — and produces clean, date-versioned Markdown
trees ready for offline consumption. One command, reproducible snapshots, no drift.

## Features

- 64 providers across 11 topics (frontier labs, inference hosts, cloud infra, SpaceX/Starlink data APIs, data platforms, design platforms, and more)
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
refbolt init --topic frontier-labs --output providers.yaml

# Start with one native source from that config
refbolt sync --config providers.yaml --provider xai
```

### Build from source

```bash
make build
./bin/refbolt sync --provider xai --verbose
```

### Run in Docker

Prefer containers? The repo ships a `docker-compose.yml` with a one-shot CLI
and a scheduled runner — see the [Docker section](#docker) below.

## Getting Started (5 minutes)

Fresh install from brew or scoop — here's the end-to-end flow:

```bash
# 1. Browse what's available without syncing anything
refbolt catalog list

# 2. Try a focused native sync — xAI docs, no credentials needed
refbolt sync --provider xai

# 3. Customize — generate a providers.yaml you can edit
refbolt init --topic frontier-labs --output providers.yaml

# 4. (Edit providers.yaml — delete topics/providers you don't want)

# 5. Sync only the providers remaining in your config
refbolt sync --config providers.yaml --all
```

The xAI example archives its native documentation export and pages without API
credentials. The `frontier-labs` topic also includes Anthropic and OpenAI API
exports; generate that topic's config when you want all three.

Use `--provider`, `--topic`, or an explicit curated config for routine syncs.
With no user config, `sync --all` selects the full embedded catalog, including
large GitHub trees, social API references, and Jina-backed sources.

## Prerequisites

Credentials are source-specific:

| Variable       | When needed                                                                                                        | How to get it                      |
| -------------- | ------------------------------------------------------------------------------------------------------------------ | ---------------------------------- |
| `JINA_API_KEY` | Set before syncing Jina-backed Hetzner and YouTube documentation                                                   | https://jina.ai/reader             |
| `GITHUB_TOKEN` | Recommended for GitHub-backed sources, including Trino, kubectl, Mattermost, Nextcloud, Stalwart, Figma and SpaceX | https://github.com/settings/tokens |

Native sources such as xAI, Anthropic, OpenAI and DigitalOcean do not require
these credentials. `refbolt catalog show <slug>` shows the selected strategy and
credential requirements. `refbolt validate` checks the config and warns about
missing credential variables. See [docs/development.md](docs/development.md) for
source-specific setup.

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
refbolt catalog list --topic frontier-labs    # filter by topic
refbolt catalog list --strategy jina    # filter by fetch strategy
refbolt catalog list --json             # machine-readable JSON
refbolt catalog show anthropic          # full detail for one provider
refbolt catalog topics                  # topic summary with counts
```

The `catalog` command reads from data embedded in the binary — it runs without
a `providers.yaml` and has no network dependency.

### Selective sync

Not every project needs all 64 providers. Pick what you need:

```bash
# Frontier lab APIs (xAI / Grok Bot + Cursor, Anthropic, OpenAI)
refbolt sync --topic frontier-labs

# Third-party serverless inference hosts (DeepInfra, Fireworks, Together)
refbolt sync --topic inference-host

# xAI API, Grok Bot and agent documentation
refbolt sync --provider xai

# Only Anthropic and OpenAI
refbolt sync --provider anthropic --provider openai

# All configured providers except OpenAI
refbolt sync --config providers.yaml --all --exclude-provider openai

# Only one AWS service
refbolt sync --provider aws-bedrock-userguide

# SpaceX / Starlink data APIs (opt-in by surface — see below)
refbolt sync --provider spacex-rockets-v4
refbolt sync --topic spacex-data --exclude-provider starlink-api-v2-reference
```

### SpaceX & Starlink data APIs (`spacex-data`)

refbolt archives two related but distinct documentation families under topic
`spacex-data`:

| Family                 | Prefix              | Source                                    | Strategy     |
| ---------------------- | ------------------- | ----------------------------------------- | ------------ |
| r/SpaceX community API | `spacex-*`          | `r-spacex/SpaceX-API` on GitHub (`docs/`) | `github-raw` |
| Starlink Public API v2 | `starlink-api-v2-*` | `starlink.readme.io` (native `.md`)       | `native`     |

There is no monolithic "sync all SpaceX" mode. Each resource or API surface is a
separate provider slug — the same opt-in pattern as AWS (`aws-bedrock-userguide`)
and DigitalOcean (`digitalocean-api`).

**Quick demos:**

```bash
refbolt init --topic spacex-data --output providers.yaml

# Fun / new user — read-only rocket specs (~4 pages)
refbolt sync --provider spacex-rockets-v4

# Involved — aviation flight status + telemetry (docs public; live API needs enterprise OIDC)
refbolt sync --provider starlink-api-v2-status
```

Archived docs land at `<archive_root>/spacex-data/<slug>/latest/`. Point Cursor
`@Docs` or your agent context at that tree for offline reference.

### xAI / Grok Bot + Cursor agent docs (`xai`)

The `xai` provider archives the **full** docs.x.ai site — REST API, Grok Bot team
rollout (Cursor SSO/dashboard), and Grok CLI compat with `.cursor/mcp.json` and
`.cursor/hooks.json`. One sync covers it:

```bash
refbolt sync --provider xai
# → <archive_root>/frontier-labs/xai/latest/
```

Key pages for Cursor/local agents: `developers/docs-mcp.md`, `grok-bot/overview.md`,
`grok-bot/teams-and-enterprises.md`, `build/features/mcp-servers.md`,
`build/features/hooks.md`. Live docs MCP: `https://docs.x.ai/api/mcp`.

Details: [docs/providers/README.md](docs/providers/README.md#xai--grok).

### Inference providers (two flat topics)

“Inference providers” is a documentation umbrella, not a nested archive
directory. Topics stay flat per [DDR-0001](docs/decisions/DDR-0001-archive-tree-structure.md):

| Topic            | Who                                                                                                                                                                                | Archive path                                   |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| `frontier-labs`  | First-party lab APIs: `xai`, `anthropic`, `openai`                                                                                                                                 | `<archive_root>/frontier-labs/<slug>/latest/`  |
| `inference-host` | Hosted inference APIs: open-weights `deepinfra`, `fireworks`, `together`, `groq`, `cerebras`, `sambanova`, `hyperbolic`, `featherless`, `novita`; System One `typesafe` (not chat) | `<archive_root>/inference-host/<slug>/latest/` |

AWS Bedrock stays under `cloud-infra`. OpenRouter is not in the catalog.
`typesafe` is a structured-decision API (Jev / System One Choice/Score/Noul), not
an OpenAI-compat or open-weights chat host.

Existing `<archive_root>/llm-api/` trees (the former topic slug) may remain until
you re-sync into `frontier-labs/` or prune them.

```bash
refbolt sync --provider deepinfra
# → <archive_root>/inference-host/deepinfra/latest/
```

## Docker

The repo ships a `docker-compose.yml` with three services — a one-shot CLI, a
scheduled runner, and a git-aware runner profile. All three share a host
bind-mounted `./archive/` directory so snapshots stay visible to your editor.

```bash
# 1. Generate a providers.yaml at the repo root (the compose services mount it)
refbolt init --topic frontier-labs --output providers.yaml

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
  refbolt:local sync --config /work/providers.yaml --all --verbose
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
