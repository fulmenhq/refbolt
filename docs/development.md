# Development Guide

## Prerequisites

- Go 1.25+ (`go version`)
- goneat (`make bootstrap` to install)
- git

## Build & Test

```bash
make build          # → bin/refbolt
make test           # run all tests (includes live network tests)
make test-cov       # tests with coverage report
make fmt            # format code and Markdown
make check-all      # fmt + lint + test
make clean          # purge bin/, dist/, Go caches
```

## Environment Variables

refbolt uses environment variables for credentials and configuration. Secrets are never stored in config files — only env var _names_ appear in `configs/providers.yaml` and provider schemas.

### Credential Variables

| Variable       | Purpose                          | Required | Notes                                                                                                                                  |
| -------------- | -------------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `JINA_API_KEY` | Jina Reader authenticated access | Optional | Higher rate limits for HTML-to-Markdown conversion. Without it, anonymous access works but may hit 429 rate limits on repeated syncs.  |
| `GITHUB_TOKEN` | GitHub API authenticated access  | Optional | Required for `github-raw` providers (Trino, kubectl) to avoid anonymous rate limits. GitHub tree API is very restrictive without auth. |

### Configuration Variables

| Variable                     | Purpose                                       | Default                                                                      |
| ---------------------------- | --------------------------------------------- | ---------------------------------------------------------------------------- |
| `REFBOLT_CONFIG`             | Path to providers config file                 | _(unset)_ — resolution chain below                                           |
| `REFBOLT_ARCHIVE_ROOT`       | Base directory for archive output             | from `archive_root:` in config; `./archive` when running on embedded catalog |
| `REFBOLT_GIT_SAFE_DIRECTORY` | Git safe.directory path for mounted worktrees | `/workspace` (runner image only; no default for the CLI)                     |

`REFBOLT_CONFIG` follows a resolution chain rather than a single default (see
[docs/ARCHITECTURE.md → Configuration Model](ARCHITECTURE.md#resolution-chain)):

1. `--config <path>` (explicit flag)
2. `REFBOLT_CONFIG` env var
3. `./providers.yaml`
4. `~/.config/refbolt/providers.yaml`
5. Embedded catalog (zero-config fallback)

All config keys can be overridden via env vars with the `REFBOLT_` prefix (e.g., `REFBOLT_ARCHIVE_ROOT=/tmp/archive`).

### Loading Credentials

Credential files live outside the repo in `~/devsecops/vars/` and are sourced before running:

```bash
source ~/devsecops/vars/fulmenhq-refbolt-jina.sh
```

Each file exports a single variable (e.g., `export JINA_API_KEY=<key>`).

### Security: Credential Isolation

Provider credentials (e.g., `OPENAI_API_KEY`, `GITHUB_TOKEN`) are used only for requests to their respective services. They are never forwarded to third-party services like Jina Reader. The Jina fetcher exclusively uses `JINA_API_KEY`.

## Fetch Strategies

| Strategy               | When Used                                                                    | Auth                         |
| ---------------------- | ---------------------------------------------------------------------------- | ---------------------------- |
| `native`               | Provider serves `.md`, `llms.txt`, `llms-full.txt`, or an OpenAPI endpoint   | None needed                  |
| `jina`                 | Provider serves HTML only                                                    | `JINA_API_KEY` (optional)    |
| `auto`                 | Try native first, fall back to Jina if HTML detected                         | `JINA_API_KEY` (optional)    |
| `github-raw`           | Docs hosted as Markdown in a GitHub repo                                     | `GITHUB_TOKEN` (recommended) |
| `llmstxt-hierarchical` | Cloud-provider aggregate indexes (AWS, Azure, GCP) with per-service llms.txt | None needed                  |

See [docs/ARCHITECTURE.md → Fetch Strategies](ARCHITECTURE.md#4-fetch-strategies) for splitter variants and implementation pointers.

## Running a Sync

At least one selector is required: `--all`, `--provider`, or `--topic`.

```bash
# All enabled providers
./bin/refbolt sync --all --verbose

# Specific providers by slug
./bin/refbolt sync --provider openai --provider anthropic

# All providers in a topic
./bin/refbolt sync --topic llm-api

# Union: topic + explicit provider
./bin/refbolt sync --topic llm-api --provider trino

# Exclude from --all
./bin/refbolt sync --all --exclude-provider trino

# Output lands in archive_root from your providers.yaml (or ./archive on embedded catalog).
# Override per-run with REFBOLT_ARCHIVE_ROOT=/tmp/archive
```

### Selection Semantics

| Flag                 | Behavior                                                    |
| -------------------- | ----------------------------------------------------------- |
| `--all`              | All enabled providers                                       |
| `--provider <slug>`  | Specific provider(s), ignores `enabled: false` (repeatable) |
| `--topic <slug>`     | All enabled providers in topic(s) (repeatable)              |
| `--exclude-provider` | Remove from resolved set after union + enabled filtering    |

- `--provider` overrides `enabled: false` (explicit request wins)
- `--all` and `--topic` skip providers with `enabled: false`
- Unknown slugs produce an error
- Same slug in `--provider` and `--exclude-provider` is an error
- Empty resolved set after filtering is an error

## Git Automation

After a sync writes files, refbolt can optionally stage, commit, and push the archive changes. All git operations are opt-in and scoped to the archive root — config files, code, and credentials are never staged.

### Flags

| Flag            | Default | Description                                              |
| --------------- | ------- | -------------------------------------------------------- |
| `--git-commit`  | false   | Stage archive changes and commit with structured message |
| `--git-push`    | false   | Push after commit (requires `--git-commit`)              |
| `--git-branch`  | (none)  | Remote branch to push to (default: current branch)       |
| `--git-trailer` | (none)  | Trailer line to append to commit message (repeatable)    |

### Examples

```bash
# Commit archive changes after sync (no push)
./bin/refbolt sync --all --git-commit

# Commit and push to current branch
./bin/refbolt sync --all --git-commit --git-push

# Push to a specific remote branch
./bin/refbolt sync --all --git-commit --git-push --git-branch archive/daily

# Add attribution trailers (for repos with AGENTS.md compliance)
./bin/refbolt sync --all --git-commit \
  --git-trailer "Co-Authored-By: Claude Opus 4.6 <noreply@fulmenhq.dev>"
```

### Commit Message Format

```
refbolt sync: 2026-03-22

Providers updated:
- xai: 96 files (llm-api)
- anthropic: 488 files (llm-api)

Archive root: /data/archive
```

### Safety

- Only archive files are staged — `git add` is scoped to `archive_root`
- No commit if nothing changed (no empty commits)
- No `--force` push, ever
- `git` must be on PATH — clear error if missing
- Archive root must be inside a git worktree — clear error if not
- Non-archive working tree changes are left untouched

### Containerized Git Automation

For containerized `--git-commit` and `--git-push` runs, mount a git worktree instead of only a bare archive directory. The archive root must live inside that worktree.

```bash
docker run --rm \
  -e REFBOLT_CONFIG=/workspace/providers.yaml \
  -e REFBOLT_ARCHIVE_ROOT=/workspace/archive \
  -e REFBOLT_GIT_SAFE_DIRECTORY=/workspace \
  -e TZ=America/New_York \
  -v "$PWD:/workspace" \
  -v ./examples/crontab-git:/etc/refbolt/crontab:ro \
  -v "$HOME/.ssh:/root/.ssh:ro" \
  refbolt-runner:local
```

Notes:

- `REFBOLT_GIT_SAFE_DIRECTORY=/workspace` avoids Git's ownership check on mounted worktrees
- The SSH mount path above assumes the container runs as `root`; if that changes, mount keys under the active user's `HOME`
- HTTPS auth also works if you mount a credential helper or provide `GIT_ASKPASS`
- Existing non-git schedules can keep using `examples/crontab` with a plain archive volume

## Adding a New Provider

1. Add entry to `configs/providers.yaml` under the appropriate topic
2. Add registry entry to `registry/providers.jsonl`
3. Document fetch quirks in `docs/providers/README.md`
4. Review provider's TOS per [SDR-0001](decisions/SDR-0001-ethical-fetching-policy.md)
5. Run `make test` to verify
