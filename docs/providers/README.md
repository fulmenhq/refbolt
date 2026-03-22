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

## OpenAI

**Base URL**: `https://platform.openai.com`
**llms-full.txt**: Not tested.
**OpenAPI spec**: Was at `https://raw.githubusercontent.com/openai/openai-openapi/master/openapi.yaml` — now 404 (likely moved to `main` branch).

### Fetch Quirks (as of 2026-03-21)

- **No native .md**: Pages are HTML; need Jina Reader fallback.
- **OpenAPI spec**: Try `main` branch instead of `master`.

### Status

Partially working. 2 pages fetched via HTML. OpenAPI URL needs updating.
