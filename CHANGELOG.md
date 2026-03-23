# Changelog

All notable changes to this project will be documented in this file. Older entries are archived under `docs/releases/` once we ship tagged versions.

> **Maintenance**: Keep only the 10 most recent releases in reverse-chronological order. Purge older entries when adding new releases.

## [Unreleased]

## [0.0.1] - 2026-03-23

First functional release of refbolt — container-first CLI for archiving web documentation into clean, date-versioned Markdown trees.

### Providers

- **xAI / Grok**: llms.txt split strategy, 96 pages verified (PR#2)
- **Pydantic**: llms-full.txt single-file strategy, 1.7MB archive (direct push)
- **Anthropic**: llms-full.txt URL-based splitter, 488 pages from `platform.claude.com` (PR#4)
- **OpenAI**: Jina Reader HTML-to-Markdown conversion, 3 doc pages + OpenAPI spec from `manual_spec` branch (PR#7)
- **Trino**: GitHub raw fetch strategy, 641 Markdown files from `trinodb/trino` (PR#6)
- **Kubernetes kubectl**: GitHub raw fetch, 121 files from `kubernetes/website` (PR#8)
- **AWS Glue**: Hierarchical llms.txt strategy via AWS top-level index (PR#10)
- **AWS Bedrock**: User Guide + API Reference as separate provider entries, hierarchical strategy (PR#10)

### Fetch Strategies

- **native**: Direct `.md` or `llms-full.txt` fetch
- **jina**: Jina Reader HTML-to-Markdown conversion with `JINA_API_KEY` auth support
- **auto**: Try native first, fall back to Jina if HTML detected
- **github-raw**: GitHub tree API discovery + `raw.githubusercontent.com` content fetch with default branch resolution
- **llmstxt-hierarchical**: Top-level llms.txt index → per-service fetch with `base_url` prefix matching

### Core

- Go CLI with Cobra (`sync`, `version` commands)
- 3-layer config: defaults → `configs/providers.yaml` → `REFBOLT_*` env vars
- JSON Schema validation for provider configuration
- Date-versioned archive tree with `latest` symlink
- Provider registry (`registry/providers.jsonl`) with capability metadata

### Git Automation (PR#12)

- `--git-commit`: stage archive changes and commit with structured message
- `--git-push`: push after commit (requires `--git-commit`)
- `--git-branch`: push destination (default: current branch)
- `--git-trailer`: repeatable trailer lines for attribution compliance
- Pre-flight validation: git on PATH, archive inside worktree, canonicalized paths
- Safety: archive-only staging, no force push, no empty commits, pre-existing dirt detection

### Container Images

- **CLI image** (`Dockerfile`): `gcr.io/distroless/static-debian12`, `CGO_ENABLED=0`, ~8MB (PR#9)
- **Runner image** (`Dockerfile.runner`): `debian:trixie-slim` + supercronic + git + openssh-client, ~80MB (PR#11, PR#13)
- `make docker-build` and `make docker-build-runner` targets
- Mounted config, crontab, and credentials — nothing baked in
- `REFBOLT_GIT_SAFE_DIRECTORY` for mounted worktree ownership

### CI/CD (PR#14)

- **CI workflow** (`.github/workflows/ci.yml`): format-check → build-test on push to main and PRs
- **Release workflow** (`.github/workflows/release.yml`): `v*` tag trigger, VERSION validation, 5-platform cross-build, draft GitHub Release
- `test-short` mode for CI (no live network tests)
- Full signing/release target chain in Makefile
- `RELEASE_CHECKLIST.md` with env var table for operator signing handoff

### Documentation

- Ethical fetching policy (SDR-0001)
- Provider-specific fetch quirks and verification status
- Development guide with env vars, fetch strategies, git automation, containerized usage
- CI/CD guide with workflow triggers, job tables, signing env vars
- Architecture and vision documents
- Decision records: llms.txt primary fetch, ecosystem libraries, archive tree structure, ethical fetching

### Project

- Renamed from `fularchive` to `refbolt` (PR#5)
- Dual MIT / Apache-2.0 license
- Multi-agent development: Alfa (provider quality) and Bravo (GitHub raw fetch / containers)
- 7 agentic roles in `config/agentic/roles/`
