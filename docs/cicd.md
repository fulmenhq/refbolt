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
- `JINA_API_KEY` for the live Jina transport checks (see [Live Jina gate](#live-jina-gate)). Anonymous Jina access works for ad hoc runs but may return 429.

```bash
# Set JINA_API_KEY from your local secret store, then run the full suite
make test

# Run only the live Jina release gate
make test-live-jina
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

Full validation runs are not scheduled in CI. The live Jina checks run through the gate below, not through a full-suite job.

## Live Jina gate

Two tests make real Jina Reader requests and are a **required** release check:

- `TestHTTPFetcher_Jina_OpenAI`
- `TestHTTPFetcher_Jina_Auto_Fallback`

They are transport integration checks against a stable docs URL. They are not a test of the current OpenAI provider config, which uses native fetch.

Behavior:

- A missing or blank `JINA_API_KEY` **fails** the check with `JINA_API_KEY is required for live Jina checks`. It does not skip.
- Short (under 1000 bytes) or HTML-looking responses **fail**. They do not skip.
- Short mode (`go test -short`, `make test-short`) skips the live tests, so PR CI stays hermetic.
- The auto-fallback test checks the route on the wire: a direct request to the provider host must come before a request to the Jina proxy. Content checks run after that as a second guard.

Where it runs:

| Trigger                                     | Runs                                                                            | Secret                     |
| ------------------------------------------- | ------------------------------------------------------------------------------- | -------------------------- |
| Pull request                                | No                                                                              | Not provided               |
| Push to `main`                              | `ci.yml` calls `live-jina.yml`                                                  | `JINA_API_KEY` (CI secret) |
| Manual (`workflow_dispatch`) on `main` only | `ci.yml` calls `live-jina.yml`                                                  | `JINA_API_KEY` (CI secret) |
| Tag `v*`                                    | `release.yml` calls `live-jina.yml` on the tagged commit; `release` waits on it | `JINA_API_KEY` (CI secret) |

A tag cannot publish unless the tagged commit passes the live check itself. A passing `main` run does not substitute for it.

Credentials:

- `JINA_API_KEY` is a repository secret. It belongs to a dedicated CI key, separate from any local development key.
- Rotate the CI key every 90 days. Replace the repository secret when you rotate.
- No key material appears in workflow files, logs, or commit messages.

Local reminder: run `make test`, or the two tests above, with your local key before you open a PR that touches providers or the Jina path. A failed live check at tag time is the failure this step prevents.

## Build

```bash
make build          # → bin/refbolt (current platform)
make install        # → build + copy to ~/.local/bin
make build-all      # → bin/ with all 6 platform binaries
make clean          # purge bin/, dist/, Go caches
```

## Cross-Compilation and CGO

**refbolt is `CGO_ENABLED=0`.** This is a deliberate architectural choice that simplifies the entire build and release pipeline:

- **Single-runner cross-compilation**: All 6 platform binaries are built from one `ubuntu-latest` runner using `GOOS`/`GOARCH` environment variables. No platform-specific runners, no build matrices, no C toolchains.
- **Static binaries**: No libc dependency, no dynamic linking. Binaries run on any machine with the matching OS/arch — no runtime dependencies.
- **No native ARM64 runners needed**: `windows/arm64` and `linux/arm64` are cross-compiled from x64. Go's compiler handles this natively when CGO is disabled.

**If CGO is ever needed** (e.g., for SQLite bindings, system TLS, or native crypto), the build pipeline would need:

- Separate build jobs per platform (or a cross-compiler like `zig cc`)
- Native ARM64 runners for ARM64 binaries (the org has `windows-latest-arm64-s` and `ubuntu-latest-arm64-s` available)
- Platform-specific Dockerfiles for containerized builds

For now, staying CGO-free keeps the pipeline simple. Track this decision if dependencies change.

### Build Targets

| Platform      | GOOS    | GOARCH | Binary Name                 |
| ------------- | ------- | ------ | --------------------------- |
| Linux x64     | linux   | amd64  | `refbolt-linux-amd64`       |
| Linux ARM64   | linux   | arm64  | `refbolt-linux-arm64`       |
| macOS x64     | darwin  | amd64  | `refbolt-darwin-amd64`      |
| macOS ARM64   | darwin  | arm64  | `refbolt-darwin-arm64`      |
| Windows x64   | windows | amd64  | `refbolt-windows-amd64.exe` |
| Windows ARM64 | windows | arm64  | `refbolt-windows-arm64.exe` |

## Makefile Targets

| Target                     | Description                              | CI-safe |
| -------------------------- | ---------------------------------------- | ------- |
| `build`                    | Build binary to `bin/refbolt`            | Yes     |
| `install`                  | Build + copy to `~/.local/bin`           | No      |
| `build-all`                | Build all 6 platform binaries            | Yes     |
| `test`                     | Full test suite (includes network tests) | Nightly |
| `test-short`               | Short tests only (no network, no git)    | Yes     |
| `test-cov`                 | Tests with coverage report               | Nightly |
| `fmt`                      | Format code and Markdown via goneat      | Yes     |
| `lint`                     | Go vet + goneat assess                   | Yes     |
| `check-all`                | fmt + lint + test                        | Nightly |
| `clean`                    | Purge bin/, dist/, Go caches             | Yes     |
| `embed-assets`             | Sync embedded catalog/schema copies      | Yes     |
| `release-build`            | Build 6-platform release artifacts       | Yes     |
| `release-checksums`        | Generate SHA256SUMS/SHA512SUMS           | Yes     |
| `release-sign`             | Sign checksum manifests (local only)     | No      |
| `release-download`         | Download CI-built release assets         | No      |
| `release-export-keys`      | Export public signing keys               | No      |
| `release-verify-keys`      | Verify exported keys are public-only     | No      |
| `release-verify-checksums` | Verify checksums against artifacts       | No      |
| `release-upload`           | Upload provenance to GitHub Release      | No      |

## Package Distribution

| Channel  | Repository                  | Update mechanism                                |
| -------- | --------------------------- | ----------------------------------------------- |
| Homebrew | `fulmenhq/homebrew-tap`     | Update `Formula/refbolt.rb` with new checksums  |
| Scoop    | `fulmenhq/scoop-bucket`     | Update `bucket/refbolt.json` with new checksums |
| Docker   | `ghcr.io/fulmenhq/refbolt`  | Rebuild images (when registry is set up)        |
| Go       | `go install github.com/...` | Automatic via Go module proxy                   |

## CI Workflows

refbolt is pure Go with `CGO_ENABLED=0`. This means all cross-compilation happens from a single Linux runner — no platform-specific build matrices, no C toolchains, no manual Go bindings steps. As long as we stay CGO-free, the CI pipeline remains a single-job build.

### `.github/workflows/ci.yml`

**Trigger:** push to `main` or any pull request.

| Job            | Depends on     | What it does                                                                                                                 |
| -------------- | -------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `format-check` | —              | yamlfmt + prettier via `goneat-tools-runner` container                                                                       |
| `build-test`   | `format-check` | `make fmt` + diff check, golangci-lint v2, `make test-short`, `make build`, smoke test (`refbolt version`, `refbolt --help`) |

Both jobs run in the `ghcr.io/fulmenhq/goneat-tools-runner:v0.3.3` container. Go is installed via `actions/setup-go` from the `go.mod` minimum (1.26.9) with `GOTOOLCHAIN=local`. Tests run in short mode — no live network calls, no git integration tests.

### `.github/workflows/release.yml`

**Trigger:** push of a `v*` tag (e.g., `git push origin v0.2.0`).

| Step                    | What it does                                                            |
| ----------------------- | ----------------------------------------------------------------------- |
| Validate VERSION        | Fails if `VERSION` file content does not match the pushed tag           |
| Lint + test             | `make lint` + `make test-short`                                         |
| Build release artifacts | `make release-build` — 6 binaries (linux/darwin/windows × amd64/arm64)  |
| Publish draft release   | `softprops/action-gh-release` with `draft: true` + all `dist/release/*` |

The release is created as a **draft**. After CI completes:

1. Download the CI-built artifacts locally (`make release-download`)
2. Sign checksum manifests with minisign/PGP (`make release-sign`)
3. Upload provenance assets (`make release-upload`)
4. Review and publish the draft release on GitHub

See `RELEASE_CHECKLIST.md` for the full step-by-step procedure.

### What does NOT trigger CI

- Pushes to feature branches without a PR — open a PR to get CI
- Nightly full test suite — not configured; run `make test` locally for live network coverage

Manual workflow dispatch is configured. On `main`, it runs the live Jina gate
alongside the other CI jobs. On other refs, the live Jina job is skipped.

## Release Signing

### Environment Variables

The following variables must be set before signing. Store them in a credentials file outside the repo to keep signing keys out of version control.

| Variable               | Purpose                          | Example                                                  |
| ---------------------- | -------------------------------- | -------------------------------------------------------- |
| `REFBOLT_GPG_HOMEDIR`  | GnuPG home directory for signing | `~/vault/fulmenhq-gpg`                                   |
| `REFBOLT_PGP_KEY_ID`   | PGP signing key fingerprint      | `448A539320A397AF!`                                      |
| `REFBOLT_MINISIGN_KEY` | Path to minisign private key     | `~/vault/fulmenhq-minisign/fulmenhq-release-signing.key` |
| `REFBOLT_MINISIGN_PUB` | Path to minisign public key      | `~/vault/fulmenhq-minisign/fulmenhq-release-signing.pub` |
| `REFBOLT_VERSION_TAG`  | Release tag for this release     | `v0.2.0`                                                 |

The first four are stable across releases. `REFBOLT_VERSION_TAG` is set per release so the credentials file does not need to change.

```bash
# Source your credentials file, then set the release tag
source <your-credentials-file>
export REFBOLT_VERSION_TAG=v<version>
```
