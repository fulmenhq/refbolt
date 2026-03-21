# Provider Notes

Operational notes for each documentation provider — fetch quirks, domain changes, auth requirements, and known issues. Updated as providers change behavior.

## GitHub-Hosted Markdown Providers

Providers using the `github-raw` strategy rely on two GitHub surfaces:

- `api.github.com` for recursive tree discovery
- `raw.githubusercontent.com` for file content download

### Operating Guidance (as of 2026-03-21)

- **Authenticated API access is the expected mode**: Set `GITHUB_TOKEN` or the provider's configured `auth_env_var` for normal syncs. Anonymous GitHub API access is supported only as a best-effort fallback for local testing.
- **Auth applies only to the API**: Send the token to `api.github.com` tree requests. Do not send it to `raw.githubusercontent.com`.
- **403 and 429 are operational signals**: They usually mean the anonymous budget is exhausted or secondary throttling has kicked in. Retry later or use authenticated access.
- **Keep scope tight**: Large repos need narrow `github_docs_path` and path filters to avoid truncated trees and excessive request volume.

## Trino

**Base URL**: `https://trino.io/docs/current`
**GitHub source**: `trinodb/trino`
**Docs path**: `docs/src/main/sphinx/`

### Fetch Quirks (as of 2026-03-21)

- **No llms.txt**: Discovery comes from the GitHub tree API, not the published site.
- **Auth recommended**: Tree discovery can hit GitHub anonymous limits quickly during repeated syncs. Use `GITHUB_TOKEN`.
- **Sphinx tree**: Connector, function, and SQL reference pages live under the same docs subtree and can be archived with `**/*.md`.

## Kubernetes kubectl

**Base URL**: `https://kubernetes.io/docs/reference/kubectl`
**GitHub source**: `kubernetes/website`
**Docs path**: `content/en/docs/reference/kubectl/`

### Fetch Quirks (as of 2026-03-21)

- **Strict scope control required**: The full repo is large; keep this provider limited to the kubectl reference subtree.
- **Auth recommended**: GitHub tree discovery should use `GITHUB_TOKEN` for routine runs.
- **Hugo `_index.md` convention**: Archive `_index.md` pages as `index.md` so directory landing pages remain stable.

## xAI / Grok

**Base URL**: `https://docs.x.ai`
**llms.txt**: `https://docs.x.ai/llms.txt` — 96 sections, ~875KB, `===/<path>===` delimited
**Individual .md**: Available at `/developers/**/*.md` (append `.md` to HTML paths)

### Fetch Quirks (as of 2026-03-21)

- **Accept header**: Returns 404 for `Accept: text/markdown`. Use `Accept: */*`.
- **TLS ALPN**: Go's default HTTP/2 ALPN negotiation causes 404. Use HTTP/1.1 only (`NextProtos: ["http/1.1"]`).
- **No sitemap.xml**: 404. Use llms.txt as the page index instead.
- **No OpenAPI spec**: Not published. API is OpenAI-compatible but with extensions (tools, model names).

### Recommended Strategy

Primary: Fetch `llms.txt` and split on delimiters (one HTTP request, full site).
Supplement: Fetch individual `.md` pages for targeted updates between full syncs.

## Anthropic

**Base URL**: `https://platform.claude.com/docs` (migrated from `docs.anthropic.com` circa March 2026)
**llms-full.txt**: Not found at new domain (404). Needs investigation.
**Individual .md**: Untested at new domain.

### Fetch Quirks (as of 2026-03-21)

- **Domain migration**: `docs.anthropic.com` → `platform.claude.com/docs` (301 redirect).
- **SPA rendering**: New domain serves Next.js SPA — static `.txt` endpoints may not exist.
- Pages serve rich Markdown content via HTML (clean conversion via Jina Reader or similar).

### Status

Partially working. 3 pages fetched via HTML. Need to investigate `llms-full.txt` equivalent and `.md` suffix support at new domain.

## OpenAI

**Base URL**: `https://platform.openai.com`
**llms-full.txt**: Not tested.
**OpenAPI spec**: Was at `https://raw.githubusercontent.com/openai/openai-openapi/master/openapi.yaml` — now 404 (likely moved to `main` branch).

### Fetch Quirks (as of 2026-03-21)

- **No native .md**: Pages are HTML; need Jina Reader fallback.
- **OpenAPI spec**: Try `main` branch instead of `master`.

### Status

Partially working. 2 pages fetched via HTML. OpenAPI URL needs updating.
