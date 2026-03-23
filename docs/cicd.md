# CI/CD Guide

## Test Modes

refbolt tests are organized into tiers that balance coverage with CI feasibility.

### Short mode (`go test -short`)

Runs only fast, hermetic unit tests. No network access, no git operations. This is the recommended mode for CI pipelines.

```bash
go test -short ./...
```

**What runs:**

- Config parsing and schema validation
- Commit message formatting
- Jina header stripping, HTML detection
- llms.txt/llms-full.txt splitting
- GitHub raw URL building and pattern matching
- Hierarchical URL matching and prefix logic

**What is skipped:**

- Live network fetches (Jina, OpenAI, xAI, Anthropic, AWS)
- Git integration tests (require writable temp repos with git config)

### Full mode (`go test ./...`)

Runs all tests including live network fetches and git integration. Requires:

- Network access to external APIs
- `git` on PATH with user.name/user.email configured
- Optional: `JINA_API_KEY` for authenticated Jina tests (anonymous works but may 429)

```bash
# Local development — full suite
make test

# With Jina auth for reliable rate limits
source ~/devsecops/vars/fulmenhq-refbolt-jina.sh
make test
```

### Tests skipped in short mode

| Package             | Test                                          | Reason         |
| ------------------- | --------------------------------------------- | -------------- |
| `internal/git`      | `TestIntegration_SyncGitCommit`               | Git operations |
| `internal/git`      | `TestIntegration_PreExistingDirtBlocksSync`   | Git operations |
| `internal/provider` | `TestHTTPFetcher_XAI_LLMSTxt`                 | Live network   |
| `internal/provider` | `TestHTTPFetcher_Pydantic_LLMSFullTxt`        | Live network   |
| `internal/provider` | `TestHTTPFetcher_Anthropic_LLMSFullTxt`       | Live network   |
| `internal/provider` | `TestHTTPFetcher_OpenAI_JinaWithOpenAPI`      | Live network   |
| `internal/provider` | `TestHTTPFetcher_Jina_OpenAI`                 | Live network   |
| `internal/provider` | `TestHTTPFetcher_Jina_Auto_Fallback`          | Live network   |
| `internal/provider` | `TestHierarchicalFetcher_AWSGlue`             | Live network   |
| `internal/provider` | `TestHierarchicalFetcher_AWSBedrockUserguide` | Live network   |

### CI configuration

For GitHub Actions or similar:

```yaml
- name: Run tests (short mode)
  run: go test -short ./...
```

For nightly or pre-merge full validation:

```yaml
- name: Run full test suite
  run: go test ./...
  env:
    JINA_API_KEY: ${{ secrets.JINA_API_KEY }}
```

## Build

```bash
make build          # → bin/refbolt
make clean          # purge bin/, dist/, Go caches
```

## Makefile Targets

| Target                     | Description                              | CI-safe |
| -------------------------- | ---------------------------------------- | ------- |
| `build`                    | Build binary to `bin/refbolt`            | Yes     |
| `test`                     | Full test suite (includes network tests) | Nightly |
| `test-short`               | Short tests only (no network, no git)    | Yes     |
| `test-cov`                 | Tests with coverage report               | Nightly |
| `fmt`                      | Format code and Markdown via goneat      | Yes     |
| `lint`                     | Go vet + goneat assess                   | Yes     |
| `check-all`                | fmt + lint + test                        | Nightly |
| `clean`                    | Purge bin/, dist/, Go caches             | Yes     |
| `release-build`            | Build multi-platform release artifacts   | Yes     |
| `release-checksums`        | Generate SHA256SUMS/SHA512SUMS           | Yes     |
| `release-sign`             | Sign checksum manifests (local only)     | No      |
| `release-download`         | Download CI-built release assets         | No      |
| `release-export-keys`      | Export public signing keys               | No      |
| `release-verify-keys`      | Verify exported keys are public-only     | No      |
| `release-verify-checksums` | Verify checksums against artifacts       | No      |
| `release-upload`           | Upload provenance to GitHub Release      | No      |

## CI Workflows

### `.github/workflows/ci.yml`

Runs on push to `main` and on pull requests:

1. **format-check** — yamlfmt + prettier via goneat-tools-runner container
2. **build-test** — fmt diff check, golangci-lint, `make test-short`, `make build`, smoke test

All builds use `CGO_ENABLED=0` — refbolt is pure Go.

### `.github/workflows/release.yml`

Runs on `v*` tag push:

1. Validates VERSION file matches tag
2. Runs lint + test-short
3. Builds multi-platform release artifacts via `make release-build`
4. Creates a **draft** GitHub Release with artifacts attached

Draft releases require manual review and publishing. Signing is done locally after downloading CI-built artifacts (see `RELEASE_CHECKLIST.md`).

## Release Signing

### Environment Variables

Source CI/CD credentials before signing:

```bash
source ~/devsecops/vars/fulmenhq-refbolt-cicd.sh
export REFBOLT_VERSION_TAG=v<version>
```

| Variable               | Purpose                          | Source                     |
| ---------------------- | -------------------------------- | -------------------------- |
| `REFBOLT_GPG_HOMEDIR`  | GnuPG home directory for signing | `fulmenhq-refbolt-cicd.sh` |
| `REFBOLT_PGP_KEY_ID`   | PGP signing key fingerprint      | `fulmenhq-refbolt-cicd.sh` |
| `REFBOLT_MINISIGN_KEY` | Path to minisign private key     | `fulmenhq-refbolt-cicd.sh` |
| `REFBOLT_MINISIGN_PUB` | Path to minisign public key      | `fulmenhq-refbolt-cicd.sh` |
| `REFBOLT_VERSION_TAG`  | Release tag (e.g., `v0.2.0`)     | Set separately per release |

The first four are stable across releases. `REFBOLT_VERSION_TAG` is set separately so the vars file does not change per release.
