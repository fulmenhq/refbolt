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
**Default branch**: `master`

### Fetch Quirks (as of 2026-03-22)

- **No llms.txt**: Discovery comes from the GitHub tree API, not the published site.
- **Default branch is not `main`**: The source repo still uses `master`. Leave `github_branch` unset or set it explicitly when testing branch-specific fetches.
- **Auth recommended**: Tree discovery can hit GitHub anonymous limits quickly during repeated syncs. Use `GITHUB_TOKEN`.
- **Sphinx tree**: Connector, function, and SQL reference pages live under the same docs subtree and can be archived with `**/*.md`.
- **robots.txt allows current docs**: `/docs/current/` is allowed for user agents, which aligns with the published site path this provider mirrors.

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
**llms.txt**: `https://platform.claude.com/llms.txt` — 62KB index, 606 page links
**llms-full.txt**: `https://platform.claude.com/llms-full.txt` — 24MB full dump, 488 sections
**Individual .md**: Available at `/docs/en/<path>.md` (307 redirects for renamed pages)

**Important**: Both `.txt` endpoints are at the **domain root** (`platform.claude.com/`), NOT under `/docs/`. The old `/docs/llms.txt` and `/docs/llms-full.txt` paths return 404.

### Delimiter Format

Anthropic does **not** use xAI-style `===/path===` delimiters. Sections are separated by a combination of horizontal rule, heading, and URL line:

```
<end of previous page content>

---

# Page Title

URL: https://platform.claude.com/docs/en/some/path

# Page Title  (duplicate heading — stripped by splitter)

<page content starts here>
```

The `URL: ` line is the reliable split point. `SplitLLMSFullTxt()` handles this format. The duplicate `# Title` after the URL line is stripped from archived content.

### Fetch Quirks (as of 2026-03-21)

- **Domain migration**: `docs.anthropic.com` → `platform.claude.com/docs` (301 redirect). Old domain still redirects correctly.
- **Next.js SPA**: Despite being a Next.js app, static `.txt` endpoints at the root are served correctly with `Content-Type: text/plain`.
- **`.md` suffix**: Individual pages at `/docs/en/<path>.md` return clean Markdown. Some paths redirect via 307 (e.g., tool-use moved from `build-with-claude/` to `agents-and-tools/tool-use/overview`).
- **File size**: `llms-full.txt` is ~24MB — scanner buffer needs to be increased beyond Go's default 64KB max line size.
- **Content**: Includes Anthropic-specific JSX components (`<Tabs>`, `<Steps>`, `<Tip>`, `<CardGroup>`) in the Markdown. These render in Anthropic's docs but are passthrough text in raw Markdown.

### Recommended Strategy

Primary: Fetch `llms-full.txt` and split on `URL:` delimiters (one HTTP request, 488 pages).
Supplement: Fetch individual `.md` pages for targeted updates between full syncs.

### Status

Verified. Full pipeline working — native `llms-full.txt` fetch, URL-based splitting, archive tree output.

## AWS (Hierarchical llms.txt)

AWS publishes a top-level `llms.txt` index (~290KB) at `https://docs.aws.amazon.com/llms.txt` that links to per-service `llms.txt` files. refbolt uses the `llmstxt-hierarchical` strategy to fetch individual services from this index.

### Opt-In by Service

**AWS is not a crawl target.** Each AWS service/guide family is a separate provider entry in `configs/providers.yaml`. Users activate only the services relevant to their project. There is no "sync all AWS" mode.

Multi-guide services (e.g., Bedrock) use separate entries per guide family to avoid archive path collisions:

```yaml
- slug: aws-bedrock-userguide
  base_url: https://docs.aws.amazon.com/bedrock/latest/userguide
  fetch_strategy: llmstxt-hierarchical
  llms_txt_url: https://docs.aws.amazon.com/llms.txt

- slug: aws-bedrock-apiref
  base_url: https://docs.aws.amazon.com/bedrock/latest/APIReference
  fetch_strategy: llmstxt-hierarchical
  llms_txt_url: https://docs.aws.amazon.com/llms.txt
```

### Service Matching

Matching uses the `base_url`-derived path prefix — `/bedrock/latest/userguide/` will NOT match `/bedrock-agentcore/latest/...`. This is intentional to prevent false positives in the large AWS catalog.

**Important**: Each provider entry must use a guide-specific `base_url` that matches exactly one `llms.txt` in the upstream index. Do not use broad prefixes like `/glue/latest` when a service has multiple guide families — use `/glue/latest/dg` instead. See [DDR-0002](../decisions/DDR-0002-hierarchical-guide-specificity.md) for the full rationale and rules.

### Content Format

AWS per-service `llms.txt` files are structured table-of-contents indexes (Markdown links to HTML pages), not content dumps. The raw `llms.txt` is archived as-is — it serves as a reference index for the service's documentation structure.

### Verified Services (as of 2026-03-22)

| Service               | llms.txt                                | Size  |
| --------------------- | --------------------------------------- | ----- |
| Glue User Guide       | `/glue/latest/dg/llms.txt`              | 228KB |
| Bedrock User Guide    | `/bedrock/latest/userguide/llms.txt`    | 181KB |
| Bedrock API Reference | `/bedrock/latest/APIReference/llms.txt` | 228KB |

### Probed Services (observation only)

S3 User Guide (157KB), CloudFormation User Guide (51KB), Lambda Developer Guide (81KB) — all return 200. The hierarchical pattern works broadly across AWS.

## OpenAI

**Base URL**: `https://platform.openai.com`
**llms.txt / llms-full.txt**: 404 — not available.
**OpenAPI spec**: `https://github.com/openai/openai-openapi` (branch `manual_spec`).

### Fetch Quirks (as of 2026-03-22)

- **No native .md**: Pages are HTML only. No llms.txt or llms-full.txt endpoints.
- **Requires Jina Reader**: Use `fetch_strategy: jina` or `auto` (auto detects HTML and falls back to Jina).
- **OpenAPI spec**: Available at `openai/openai-openapi` on GitHub, branch `manual_spec` (not `master` or `main`).

### Recommended Strategy

Use `fetch_strategy: jina` with key reference pages: `/docs/api-reference/chat`, `/docs/api-reference/responses`, `/docs/api-reference/assistants`. OpenAPI spec fetched directly from GitHub (`manual_spec` branch).

### Status

Verified. Chat, responses, and assistants pages archived via Jina Reader. OpenAPI spec fetched from GitHub.

## Jina Reader

**Service URL**: `https://r.jina.ai/<target-url>`
**License**: Apache 2.0 (service)
**Auth**: Optional `JINA_API_KEY` env var for higher rate limits.

### How It Works

Jina Reader converts any HTML page to clean Markdown by prepending `https://r.jina.ai/` to the target URL. refbolt sends `Accept: text/markdown` and strips the metadata header (Title, URL Source, Markdown Content lines) from the response.

### Configuration

Set `fetch_strategy: jina` on any provider that serves HTML instead of Markdown. The `auto` strategy will detect HTML responses and fall back to Jina automatically.

For authenticated access (higher rate limits), set the `JINA_API_KEY` environment variable. Provider-specific credentials (e.g., `OPENAI_API_KEY`) are never sent to Jina.

### Known Limitations

- Third-party service — subject to availability and rate limits (HTTP 429).
- Very large or complex pages may be truncated or fail (HTTP 422).
- Output quality varies by site complexity (JavaScript-heavy SPAs may produce sparse content).
- Free tier has lower rate limits; set `JINA_API_KEY` for production use.
