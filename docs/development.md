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

| Variable               | Purpose                           | Default                  |
| ---------------------- | --------------------------------- | ------------------------ |
| `REFBOLT_CONFIG`       | Path to providers config file     | `configs/providers.yaml` |
| `REFBOLT_ARCHIVE_ROOT` | Base directory for archive output | `/data/archive`          |

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

| Strategy     | When Used                                            | Auth                         |
| ------------ | ---------------------------------------------------- | ---------------------------- |
| `native`     | Provider serves `.md` or `llms-full.txt` directly    | None needed                  |
| `jina`       | Provider serves HTML only                            | `JINA_API_KEY` (optional)    |
| `auto`       | Try native first, fall back to Jina if HTML detected | `JINA_API_KEY` (optional)    |
| `github-raw` | Docs hosted as Markdown in a GitHub repo             | `GITHUB_TOKEN` (recommended) |

## Running a Sync

```bash
# All providers
./bin/refbolt sync --all --verbose

# Output lands in the archive_root (default: /data/archive)
# Override with REFBOLT_ARCHIVE_ROOT=/tmp/archive
```

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

## Adding a New Provider

1. Add entry to `configs/providers.yaml` under the appropriate topic
2. Add registry entry to `registry/providers.jsonl`
3. Document fetch quirks in `docs/providers/README.md`
4. Review provider's TOS per [SDR-0001](decisions/SDR-0001-ethical-fetching-policy.md)
5. Run `make test` to verify
