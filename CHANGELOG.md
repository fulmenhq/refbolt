# Changelog

All notable changes to this project will be documented in this file. Older entries are archived under `docs/releases/` once we ship tagged versions.

> **Maintenance**: Keep only the 10 most recent releases in reverse-chronological order. Purge older entries when adding new releases.

## [Unreleased]

## [0.0.4] - 2026-04-22

Operational foundation, provider browsing, and four new providers.

### Added

- **`docker-compose.yml`**: CLI service, scheduled runner, and `--profile git` variant with SSH mount. Three services around a host-bind `./archive` directory. `REFBOLT_ARCHIVE_ROOT=/data/archive` pinned on CLI + runner so writes land on the bind mount regardless of user-config `archive_root`. (PR#30)
- **`refbolt catalog` command**: Read-only browse of the embedded catalog with registry enrichment. Subcommands `list`, `show <slug>`, `topics`. `--topic` / `--strategy` filters, `--json` envelope, "did you mean?" suggestions, stdout/stderr separation. Bypasses `config.Load` (silent-ignore of `--config` locked in by test). (PR#34)
- **Figma REST API provider** (`figma-api`): OpenAPI 3.1.0 spec via `github-raw` from `figma/rest-api-spec`. Spec-only by design — SDR-0001 respects `developers.figma.com` robots.txt. New `design-platform` topic. (PR#36)
- **Hetzner multi-surface family**: `hetzner-cloud-api` (OpenAPI via github-raw from `MaximilianKoestler/hcloud-openapi`), `hetzner-cloud` (Jina, narrative cloud docs), `hetzner-networking` (Jina, networking docs). User-configurable per-surface, like AWS/DO. (PR#37)
- **Embedded registry**: `registry/providers.jsonl` (28 entries) now embedded via `go:embed` and joined by slug into `refbolt catalog` output. (PR#34)
- **`Topic.Description` accessor**: `Topic` struct carries `Name` / `Description` from the catalog so `catalog topics` can render human-readable descriptions. (PR#34)
- **`refbolt --version` flag**: works alongside the existing `version` subcommand; same output format. (FA-111, PR#38)
- **`--verbose` / `-v` actually wired**: was parsed-and-ignored dead code. Now enables gofulmen debug-level logging in `config.Load()`, surfacing existing `log.Debug(...)` calls. Deeper verbose-logging design (per-page fetch, HTTP traces) remains v0.0.5+ work. (FA-111, PR#38)
- **Get-a-key URL hints**: `refbolt init` stderr credential block, `refbolt validate` warnings, and `refbolt catalog show <slug>` credentials line now surface the canonical token URL (`https://jina.ai/reader`, `https://github.com/settings/tokens`). `config.CredentialURL(envVar)` helper; unknown env vars omit the URL cleanly. (FA-111, PR#38)
- **`refbolt init` seed-flow tip**: when running without `--output`, stderr emits `Tip: to write this to a file, rerun with --output providers.yaml`. Teaches the stdout-by-default pattern without breaking pipe workflows. (FA-111, PR#38)
- **`refbolt validate` zero-config customization tip**: on the embedded-catalog fallback path, emits a tip pointing at `refbolt init --all --output providers.yaml`. Existing `Config source: (embedded catalog)` output unchanged. (FA-111, PR#38)
- **README "Getting Started (5 minutes)"**: numbered walkthrough for brew/scoop installers — browse → zero-config sync → init → edit → sync → credentials. (FA-111, PR#38)

### Changed

- **README.md**: Docker section leads with `docker compose` flow (one-shot, scheduled, git-aware profile). Raw `docker run` recipes kept as scripting alternative. New "Browse the catalog" pointer. Provider count reflects 27 across 8 topics. (PR#32, PR#34, PR#36, PR#37)
- **docs/ARCHITECTURE.md**: Full rewrite for v0.0.3 reality — real CLI surface, embedded catalog/registry, five fetch strategies with splitter variants, incremental sync (FA-095) with code pointers, date-versioned archive writer (DDR-0001), local-binary-first distribution. Dropped ghcr.io framing and commands that don't exist. (PR#33)
- **docs/decisions/DDR-0001**: Decision and Consequences rewritten to match `internal/archive/writer.go` exactly — calendar-day keying, dedup-then-overwrite via SHA-256, `latest/` symlink semantics preserved, forward-plan note for object-store backend. (PR#35)
- **docs/providers/README.md**: New Figma and Hetzner sections with scoping rationale and selection guidance. (PR#36, PR#37)
- **docs/development.md**: `REFBOLT_CONFIG` row now shows the real resolution chain (was claiming a stale default); runner-git example uses repo-root `providers.yaml`; fetch-strategy table gains missing `llmstxt-hierarchical` row; archive_root default comment corrected. TZ=UTC consistency across runner examples. (PR#33, PR#35)
- **examples/crontab, examples/crontab-git**: `TZ=UTC` guidance added with override notes for host-local schedules. (PR#35)
- **`refbolt sync` no-selector error**: replaced the curt `no providers selected; use --all, --provider, or --topic` one-liner with a multi-line hint block naming `--all`, `--topic`, `--provider`, and `refbolt catalog list`. (FA-111, PR#38)
- **Pluralization** across `init`, `validate`, and `catalog` surfaces: `1 provider` / `1 topic` instead of `1 providers` / `1 topics`. Shared `pluralize` helper in `internal/cmd/plural.go`. (FA-111, PR#38)

### Fixed

- **`refbolt init` emitted schema-invalid YAML**: Credential-hint comments were injected with a hardcoded 6-space indent while yaml.v3 marshals provider entries at 8 spaces — pulling every Jina/GITHUB_TOKEN provider out of its topic. `fetch_timeout` emitted compound Go duration strings (`1m30s`) that the schema's single-unit pattern rejects. Switched to `yaml.Node` `HeadComment` and single-unit-seconds format; added `TestInitCmd_RealCatalog_RoundTripsValid` regression test. (PR#30)
- **Compose `runner-git` hardcoded wrong config path**: `REFBOLT_CONFIG=/workspace/configs/providers.yaml` archived the bundled catalog instead of the user's repo-root `providers.yaml`. Fixed to `/workspace/providers.yaml`. (PR#30)
- **Compose missed `REFBOLT_ARCHIVE_ROOT`**: Writes landed in the container's ephemeral `/app/archive` instead of the host bind mount. Pinned on CLI + runner services. (PR#30)
- **Catalog list filtered totals misreported**: `--topic` / `--strategy` filters returned the filtered rows but emitted full-catalog `topics_total` in JSON and stderr hint. Now describe the rendered result set. (PR#34)
- **Singular/plural in catalog hint line**: `1 provider across 1 topic` (not `1 providers across 1 topics`). (PR#34)
- **Duplicate `Error:` lines on failed commands**: `rootCmd.SilenceErrors = true` so cobra stops auto-printing; `cmd/refbolt/main.go` is now the single stderr error source. Every failed command previously printed the error twice. (FA-111, PR#38)

## [0.0.3] - 2026-04-02

Build and distribution release.

### Build

- **Windows ARM64**: Add `windows/arm64` cross-compiled binary (6 platforms total, all `CGO_ENABLED=0` from single runner) (PR#28)
- **`make install`**: Build + copy to `~/.local/bin` (INSTALL_DIR overridable) (PR#28)
- **Scoop**: Manifest in `fulmenhq/scoop-bucket` with x64 + ARM64 Windows support (PR#28)

### Documentation

- **docs/cicd.md**: CGO_ENABLED=0 rationale, cross-compilation explanation, 6-platform build table, package distribution channels, migration notes for future CGO needs (PR#28)
- **RELEASE_CHECKLIST.md**: Add Scoop update step after Homebrew (PR#28)

## [0.0.2] - 2026-04-02

16 new providers, incremental sync, user-facing config, and public-readiness.

### Added

- **16 new providers**: DigitalOcean (6), Cloudflare (4), Mattermost (2), Nextcloud, Stalwart (PR#20, #21, #22, #24)
- **Incremental sync**: per-provider `.sync-meta.json` with config hash, content hash, and strategy-specific hints (tree SHA, ETag/HEAD). `--force` bypass. (PR#23)
- **`refbolt init`**: generate `providers.yaml` from embedded catalog with topic/provider selection (PR#25)
- **`refbolt validate`**: standalone config validation against embedded schema with strict exit codes (PR#25)
- **Embedded catalog and schema**: binary ships with full provider catalog and JSON Schema via `go:embed` — no filesystem dependency (PR#25)
- **`--config` global flag**: explicit config path available on all commands (PR#25)
- **Config resolution chain**: `--config` → `REFBOLT_CONFIG` → `./providers.yaml` → `~/.config/refbolt/providers.yaml` → embedded catalog (PR#25)
- **Provider/topic filtering**: `--provider`, `--topic`, `--exclude-provider` flags with union semantics (PR#19)
- **YAML frontmatter splitter**: `SplitFrontmatterFullTxt` for Cloudflare-style `llms-full.txt` with boilerplate stripping (PR#24)
- **URL prefix filtering**: `FilterByBaseURL` scopes split pages by `base_url` for shared bulk files like DigitalOcean (PR#20)
- **First-run credential guidance**: `init` stderr hints, `validate` env var warnings, inline config comments, README prerequisites section (PR#26)

### Changed

- **Fetch timeout**: per-provider `fetch_timeout` field (default 30s), Jina retry with 2x timeout on deadline exceeded (PR#19)
- **Write-level dedup**: SHA-256 content hash comparison before writing — `WriteStat` with written/skipped counts (PR#23)

### Full release notes

See [docs/releases/v0.0.2.md](docs/releases/v0.0.2.md) for provider table, strategy details, and PR list.

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
