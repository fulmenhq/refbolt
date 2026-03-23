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

## Makefile targets

| Target      | Description                              | CI-safe |
| ----------- | ---------------------------------------- | ------- |
| `build`     | Build binary to `bin/refbolt`            | Yes     |
| `test`      | Full test suite (includes network tests) | Nightly |
| `test-cov`  | Tests with coverage report               | Nightly |
| `fmt`       | Format code and Markdown via goneat      | Yes     |
| `check-all` | fmt + lint + test                        | Nightly |
| `clean`     | Purge bin/, dist/, Go caches             | Yes     |
