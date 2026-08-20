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
**Default branch**: `main`

### Fetch Quirks (as of 2026-03-22)

- **Strict scope control required**: The full repo is large; keep this provider limited to the kubectl reference subtree.
- **Auth recommended**: GitHub tree discovery should use `GITHUB_TOKEN` for routine runs.
- **Hugo `_index.md` convention**: Archive `_index.md` pages as `index.md` so directory landing pages remain stable.
- **Subtree size drifted upward**: The kubectl subtree currently contains 121 Markdown files, which is higher than the older ~50-page estimate in the original task brief.
- **robots.txt nuance**: Kubernetes disallows `/docs/reference/kubernetes-api/`, but does not specifically disallow the kubectl reference path mirrored by this provider.

## xAI / Grok

**Base URL**: `https://docs.x.ai`
**llms.txt**: `https://docs.x.ai/llms.txt` — ~164 sections, ~1.4MB, `===/<path>===` delimited
**Individual .md**: Available at `/developers/**/*.md`, `/grok-bot/**/*.md`, `/build/features/**/*.md`

### Content map (full sync via llms.txt)

A single `refbolt sync --provider xai` archives the **entire** docs.x.ai site — not
just the REST API. For Cursor and local Grok agents, the high-value surfaces are:

| Path prefix                           | ~sections | What agents get                                                                 |
| ------------------------------------- | --------- | ------------------------------------------------------------------------------- |
| `/developers/**`                      | ~80       | Grok REST API — inference, tools, voice, files, models                          |
| `/grok-bot/**`                        | 13        | Grok Bot product docs — Cursor SSO, dashboard admin, team rollout, privacy      |
| `/build/features/**`                  | ~20       | Grok CLI/TUI — reads `.cursor/mcp.json`, `.cursor/hooks.json`, `.cursor/rules/` |
| `/build/cli/**`, `/build/headless/**` | ~30       | Grok CLI reference, config, permissions, sandbox                                |

**Cursor-specific entry points** (also listed as supplemental paths in `providers.yaml`):

| Page                                      | Archive path (under `llm-api/xai/latest/`) |
| ----------------------------------------- | ------------------------------------------ |
| Docs MCP setup for Cursor                 | `developers/docs-mcp.md`                   |
| Grok Bot overview                         | `grok-bot/overview.md`                     |
| Team admin (Cursor dashboard, MCP policy) | `grok-bot/teams-and-enterprises.md`        |
| Cursor MCP config compat                  | `build/features/mcp-servers.md`            |
| Cursor hooks compat                       | `build/features/hooks.md`                  |

Live MCP endpoint (not archived — connect at runtime): `https://docs.x.ai/api/mcp`

**Agent usage:** sync once, then point Cursor `@Docs` or a local agent context root at
`<archive_root>/llm-api/xai/latest/`. Grok Bot and Cursor integration docs are **not**
under `spacex-data` — that topic is REST API reference only.

### Fetch Quirks (as of 2026-08-20)

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

## DigitalOcean

**Base URL**: `https://docs.digitalocean.com` (scoped per product via URL prefix)
**llms-full.txt**: `https://docs.digitalocean.com/llms-full.txt` (40MB, 4,083 pages)
**Strategy**: `native` + `llms_txt_url` with URL prefix filtering (FA-090)

### Architecture

DigitalOcean publishes a single 40MB `llms-full.txt` containing all documentation. Sections are delimited by `Source: <url>` lines (not `URL:` like Anthropic). Each provider entry scopes to a URL prefix via `base_url`, so only matching sections are archived.

### Opt-In by Product

Six product areas are configured, each fetching from the same shared `llms-full.txt` but scoping differently:

| Slug                    | Scope                  | Pages |
| ----------------------- | ---------------------- | ----- |
| `digitalocean-api`      | `/reference/api`       | ~226  |
| `digitalocean-doctl`    | `/reference/doctl`     | ~542  |
| `digitalocean-k8s`      | `/products/kubernetes` | ~355  |
| `digitalocean-dbs`      | `/products/databases`  | ~213  |
| `digitalocean-droplets` | `/products/droplets`   | ~60   |
| `digitalocean-spaces`   | `/products/spaces`     | ~40   |

### Known Limitations

- **Repeated fetches**: Each scoped provider re-downloads the full 40MB file. Shared fetch cache planned for v0.0.3.
- **No auth required**: Public endpoint, no rate limit issues observed.

### Status

Verified. All 6 scoped providers produce correct filtered archives.

## Cloudflare

**Base URL**: `https://developers.cloudflare.com` (per-product)
**Strategy**: `native` + `llms_txt_url` with YAML frontmatter splitting

### Architecture

Cloudflare has a three-tier llms.txt architecture:

1. **Top-level index** (`/llms.txt`) — links to per-product `llms.txt` files
2. **Per-product index** (`/workers/llms.txt`) — page link catalog
3. **Per-product full** (`/workers/llms-full.txt`) — complete rendered Markdown

Each product's `llms-full.txt` is self-contained (50-300KB). No shared bulk file like DigitalOcean.

### Section Delimiter Format

Cloudflare uses **YAML frontmatter** (`---/title:/---`) as section delimiters — different from Anthropic (`URL:`) and DigitalOcean (`Source:`). refbolt's `SplitFrontmatterFullTxt` handles this format and strips Cloudflare boilerplate (Skip to content, Was this helpful, Edit page, Copy page).

### Opt-In by Product

| Slug                 | Product | llms-full.txt            | Pages |
| -------------------- | ------- | ------------------------ | ----- |
| `cloudflare-workers` | Workers | `/workers/llms-full.txt` | ~407  |
| `cloudflare-pages`   | Pages   | `/pages/llms-full.txt`   | ~118  |
| `cloudflare-r2`      | R2      | `/r2/llms-full.txt`      | ~88   |
| `cloudflare-kv`      | KV      | `/kv/llms-full.txt`      | ~29   |

P2 products (D1, Durable Objects, Queues, Workers AI, etc.) documented as commented entries in `providers.yaml`.

### robots.txt

Explicitly AI-friendly: `Content-Signal: ai-train=yes, search=yes, ai-input=yes`. No restrictions on developer docs. No auth required.

### Status

Verified. Clean Markdown output — no JSX components, no MDX imports, no framework artifacts.

## Hetzner

Hetzner is a multi-surface family spanning two separate documentation sites
with no first-party source repository or `llms.txt`. refbolt ships three
P1 provider slugs under `cloud-infra`:

### Strategy table

| Slug                 | Strategy     | Source                                                                                                              | Scope                                          |
| -------------------- | ------------ | ------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| `hetzner-cloud-api`  | `github-raw` | [`MaximilianKoestler/hcloud-openapi`](https://github.com/MaximilianKoestler/hcloud-openapi) → `openapi/hcloud.json` | Cloud API OpenAPI spec (unofficial derivative) |
| `hetzner-cloud`      | `jina`       | `docs.hetzner.com/cloud/`                                                                                           | Cloud product docs (curated paths)             |
| `hetzner-networking` | `jina`       | `docs.hetzner.com/networking/`                                                                                      | Networking product docs (curated paths)        |

### Selection guidance

Which subset should you enable?

- **"I'm building against Hetzner Cloud from code."** `hetzner-cloud-api`
  alone is usually enough — the OpenAPI spec covers every endpoint,
  parameter, and schema. Add `hetzner-cloud` if you also want narrative
  context (getting-started walkthroughs, conceptual explanations).
- **"I'm operating Hetzner Cloud services (setup, troubleshooting)."**
  `hetzner-cloud` + `hetzner-networking`. The API spec is overkill for
  operational tasks.
- **"I need API integration _and_ operational context."** All three.

Each slug is independently opt-in via `refbolt sync --provider <slug>`,
mirroring the AWS and DigitalOcean multi-surface pattern. `refbolt
catalog show <slug>` renders per-surface detail with archive paths and
credential requirements.

### Known limitations

1. **Unofficial OpenAPI spec — conscious tradeoff.** Hetzner publishes
   official Cloud API specs at `docs.hetzner.cloud`, but they are served
   only through the interactive API-reference viewer — **not hosted as
   raw files on a public GitHub repo**, which is what our `github-raw`
   strategy requires. The community-maintained
   [`MaximilianKoestler/hcloud-openapi`](https://github.com/MaximilianKoestler/hcloud-openapi)
   repo tracks the official spec and improves it for codegen
   (consolidated components, unique operation IDs, ~1.2MB vs the official
   ~2.9MB). Actively maintained, MIT licensed, v0.28.0 as of Nov 2025.
   The asymmetry vs. our Figma pattern matters: Figma's official repo
   _is_ the authoritative source; this one is a derivative, so users are
   getting a github-raw-fetchable, codegen-friendly version of the
   Hetzner spec — not the canonical Hetzner-published JSON served via
   `docs.hetzner.cloud`.
2. **Exit criteria for the unofficial spec.** If the upstream repo goes
   90+ days without commits, or demonstrably diverges from the official
   spec, we file a follow-up to reassess — either switch to a Jina-based
   fetch of `docs.hetzner.cloud`, or disable the provider until Hetzner
   publishes a fetchable raw spec. This is an intentional monitoring
   task, not a scheduled migration.
3. **Jina providers have no fetch-level skip.** `hetzner-cloud` and
   `hetzner-networking` re-download every configured path on every sync
   (Jina Reader exposes no cache headers). Content-hash dedup at the
   writer level still prevents unnecessary disk writes and git churn
   (same limitation as OpenAI).
4. **Path curation is manual.** The YAML lists a curated starter set
   (~7 paths for Cloud, ~5 for Networking) — not every possible page on
   `docs.hetzner.com`. Extend per demand, not by exhaustive crawl.
   Sitemap-based discovery (FA-052) would eliminate this when
   prioritized.
5. **No fallback if Jina fails.** If a page returns empty or errors after
   FA-080's retry, it's skipped with a warning. We do not scrape the
   JS-rendered HTML ourselves.

### Deferred surfaces

Not oversights — deliberate P1 scope discipline:

| Slug (future)       | Strategy     | Why deferred                                                                                         |
| ------------------- | ------------ | ---------------------------------------------------------------------------------------------------- |
| `hetzner-storage`   | `jina`       | Storage Box / Storage Share — P2 if user-surveyed demand.                                            |
| `hetzner-robot`     | `jina`       | Dedicated server / Robot panel — narrower audience.                                                  |
| `hetzner-community` | `github-raw` | `hetzneronline/community-content` — community-authored, different editorial bar from reference docs. |

To request promotion to P1, open a ticket referencing this section with
the use case.

### Status

Verified. Three P1 slugs ship in this cycle. Live-fetch verification
covers all three surfaces; see the FA-091 PR body for the smoke-test
table.

## Mattermost

**Strategy**: `github-raw`

### Providers

| Slug                   | Source Repo                                     | Docs Path                 | Files    |
| ---------------------- | ----------------------------------------------- | ------------------------- | -------- |
| `mattermost-api`       | `mattermost/mattermost`                         | `api/v4/source/`          | ~56 YAML |
| `mattermost-integrate` | `mattermost/mattermost-developer-documentation` | `site/content/integrate/` | ~54 MD   |

### Notes

- API source is OpenAPI v4 YAML (split across 56 files), not Markdown.
- Integration guides are standard Markdown from the developer docs repo.
- `GITHUB_TOKEN` recommended for both.

### Status

Verified.

## Nextcloud

**Strategy**: `github-raw`
**Source**: `nextcloud/documentation`, path `admin_manual/`
**Format**: RST (Sphinx)

### Notes

- Single repo contains 3 manuals (admin, user, developer). We start with admin only.
- 23 subdirs: installation, configuration, maintenance, groupware, office, AI, GDPR, webhooks, ExApps.
- CC BY 3.0 license.
- `GITHUB_TOKEN` recommended.

### Status

Verified. ~173 RST files archived.

## Stalwart Mail Server

**Strategy**: `github-raw`
**Source**: `stalwartlabs/website`, path `docs/`
**Format**: Docusaurus Markdown

### Notes

- Covers JMAP, IMAP, SMTP, CalDAV, CardDAV, spam filtering, clustering.
- 17 sections: install, config, server, storage, auth, email, MTA, collaboration, encryption, spam, sieve, cluster, telemetry, management, API, development.
- `GITHUB_TOKEN` recommended.

### Status

Verified. ~270 Markdown files archived.

## Figma

**Strategy**: `github-raw`
**Source**: `figma/rest-api-spec`, path `openapi/`
**Format**: OpenAPI 3.1.0 YAML

### Scope: API spec only

refbolt archives the official Figma REST API OpenAPI specification from
GitHub. We deliberately do not archive Figma's narrative developer
documentation at `developers.figma.com` — their `robots.txt` explicitly
blocks AI crawlers (`anthropic-ai`, `GPTBot`, `CCBot`, etc.). Per
[SDR-0001](../decisions/SDR-0001-ethical-fetching-policy.md) (ethical
fetching policy), we respect this restriction.

The OpenAPI spec is MIT licensed, covers all REST API endpoints, and is
the authoritative machine-readable reference. Contributors adding Figma
surfaces should keep this provider scoped to the GitHub spec repo — if
narrative docs become useful, open a separate ticket rather than
extending this entry.

### Notes

- Single YAML file (~368KB, ~10K lines) — `estimated_pages: 1` in the registry
- MIT licensed
- `GITHUB_TOKEN` recommended for tree-API rate limits on repeated syncs
- `github_branch` is intentionally omitted; the fetcher resolves the repo default

### Status

Verified.

## r/SpaceX API (community open data)

**Base URL**: `https://docs.spacexdata.com` (Postman mirror; source is GitHub)
**GitHub source**: `r-spacex/SpaceX-API`
**Docs path**: `docs/`
**Strategy**: `github-raw`

### Multi-surface family

The r/SpaceX REST API is a **community** project (Apache 2.0), not official SpaceX.
refbolt archives Markdown from the GitHub `docs/` tree — the same content served at
`docs.spacexdata.com`. There is no `llms.txt`; `github-raw` is the agent-friendly path.

**Opt-in by surface** — like AWS and DigitalOcean, each resource/version is a separate
provider slug under topic `spacex-data`. There is no monolithic "sync all SpaceX" mode.

| Slug                   | Resource                                            | ~files |
| ---------------------- | --------------------------------------------------- | ------ |
| `spacex-api-guides`    | Cross-cutting guides (pagination, queries, clients) | 4      |
| `spacex-launches-v4`   | Launches v4                                         | 9      |
| `spacex-launches-v5`   | Launches v5                                         | 9      |
| `spacex-rockets-v4`    | Rockets                                             | 4      |
| `spacex-capsules-v4`   | Capsules                                            | 4      |
| `spacex-cores-v4`      | Cores                                               | 4      |
| `spacex-crew-v4`       | Crew                                                | 4      |
| `spacex-dragons-v4`    | Dragons                                             | 4      |
| `spacex-history-v4`    | History                                             | 4      |
| `spacex-company-v4`    | Company info                                        | 2      |
| `spacex-landpads-v4`   | Landing pads                                        | 4      |
| `spacex-launchpads-v4` | Launch pads                                         | 4      |
| `spacex-payloads-v4`   | Payloads                                            | 4      |
| `spacex-roadster-v4`   | Roadster ephemeris                                  | 3      |
| `spacex-ships-v4`      | Recovery ships                                      | 4      |
| `spacex-starlink-v4`   | Starlink _satellite catalog_ (r/SpaceX data API)    | 4      |

**Demo surfaces:**

- **New user / fun**: `spacex-rockets-v4` — simple read-only GET endpoints, small archive.
- **Involved interaction**: `starlink-api-v2-status` — aviation flight POST + telemetry
  streaming; requires enterprise OIDC setup to call live APIs (docs are still public).

`GITHUB_TOKEN` recommended for tree API rate limits on repeated syncs.

### Status

Verified (`spacex-rockets-v4` live fetch).

## Starlink Public API v2 (official enterprise)

**Base URL**: `https://starlink.readme.io`
**Strategy**: `native` (`.md` suffix endpoints)
**llms.txt**: `https://starlink.readme.io/llms.txt` (index with links — not a content dump)

### Multi-surface family

Official SpaceX Starlink enterprise API documentation. API calls require a business/
enterprise account with a V2 service account (OIDC); **documentation is public**.

Device and local-router guides (RADIUS, captive portal, local HTTPS router API) are
**deliberately omitted** — device integration, not the public REST API surface.

| Slug                        | Surface                                          | ~pages |
| --------------------------- | ------------------------------------------------ | ------ |
| `starlink-api-v2-guides`    | Getting started, auth, service lines, data pools | ~23    |
| `starlink-api-v2-reference` | Full v2 endpoint reference                       | ~67    |
| `starlink-api-v2-status`    | Aviation flight status + telemetry stream/query  | 6      |

Paths are curated from `llms.txt`; refbolt fetches each `.md` URL directly. Path
entries are **relative to `base_url`** (no leading slash) so URLs resolve to
`…/docs/getting-started.md`, not `…/getting-started.md`. No Jina required.

### Status

Verified (`starlink-api-v2-status` live fetch).

## Social Platform APIs (X & YouTube)

Topic `social-platform` archives **official X (Twitter) Platform API** and **YouTube Data API v3**
documentation. Opt-in by surface — there is no monolithic firehose sync.

**Important:** This topic is distinct from:

| Topic / provider | What it is                                     | What it is NOT                      |
| ---------------- | ---------------------------------------------- | ----------------------------------- |
| `llm-api/xai`    | xAI Grok API, Grok Bot, Cursor MCP             | X Platform REST API for posting/DMs |
| `spacex-data/*`  | r/SpaceX launch data + Starlink enterprise API | X/Twitter social platform           |

### X Developer Platform (`docs.x.com`)

**Base URL**: `https://docs.x.com`
**Strategy**: `native` — append `.md` to any docs URL
**Index**: nested `llms.txt` per surface (not the 5.4MB root `llms-full.txt` firehose)

X publishes agent-friendly docs: nested indexes, native Markdown, OpenAPI at
`openapi.json`, and MCP/skill.md tooling. refbolt curates paths from each nested
`llms.txt` (probed 2026-08-20) and fetches individual `.md` pages — incremental,
surface-scoped sync without re-downloading `llms-full.txt` per provider.

#### Selection guidance

| Use case                                        | Start with                                    |
| ----------------------------------------------- | --------------------------------------------- |
| "Build messaging/DM integrations on X"          | `xdev-api-messaging` + `xdev-platform-guides` |
| "Full X API v2 reference (posts, users, media)" | `xdev-api-v2`                                 |
| "Official Python/TypeScript SDKs"               | `xdev-xdk-python` / `xdev-xdk-typescript`     |
| "Onboarding, auth, rate limits, OpenAPI"        | `xdev-platform-guides`                        |

#### Opt-in surfaces

| Slug                   | Surface                                                         | ~pages |
| ---------------------- | --------------------------------------------------------------- | ------ |
| `xdev-platform-guides` | Overview, auth fundamentals, tutorials, AI/agent tools, OpenAPI | 16     |
| `xdev-api-messaging`   | DMs, X Chat, webhooks, Account Activity, X Activity             | 66     |
| `xdev-api-v2`          | Core v2 reference (posts, users, lists, spaces, media, streams) | 316    |
| `xdev-xdk-python`      | Python XDK                                                      | 68     |
| `xdev-xdk-typescript`  | TypeScript/JavaScript XDK                                       | 105    |

Paths are listed explicitly in `configs/providers.yaml` (generated from nested
`llms.txt` indexes). To refresh after X doc changes, re-probe
`docs.x.com/<surface>/llms.txt` and update the path lists.

#### Known quirks

- Slugs use `xdev-` prefix (not `x-`) — provider slug schema requires `[a-z][a-z0-9]+` after the first character; single-letter `x-` prefixes fail validation.
- Native `.md` pages may contain JSX component stubs (`export const Button`); content is still usable for agents.
- Use HTTP/1.1 (refbolt default) — same compatibility note as xAI docs.

#### Status

Verified (`xdev-platform-guides`, `xdev-api-messaging` live fetch 2026-08-20).

### YouTube Data API v3 (`developers.google.com/youtube/v3`)

**Base URL**: `https://developers.google.com`
**Strategy**: `jina` for HTML narrative/reference pages; Discovery REST JSON via `openapi_url`
**Auth**: `JINA_API_KEY` recommended (rate limits without key)

Google publishes HTML-only developer docs — no `llms.txt`, no native `.md` suffix.
robots.txt is permissive (only `/youtube/partner/` disallowed). refbolt uses Jina
Reader for guides and reference pages, plus the Google Discovery REST document
(~520KB JSON) as machine-readable schema.

#### Selection guidance

| Use case                             | Start with                                                                                     |
| ------------------------------------ | ---------------------------------------------------------------------------------------------- |
| "Search YouTube programmatically"    | `youtube-data-api-guides` (implementation/search) + `youtube-data-api-reference` (search.list) |
| "OAuth setup, quotas, quickstarts"   | `youtube-data-api-guides`                                                                      |
| "Endpoint shapes, parameters, types" | `youtube-data-api-reference` (includes Discovery JSON)                                         |

#### Opt-in surfaces

| Slug                         | Surface                                                               | ~pages |
| ---------------------------- | --------------------------------------------------------------------- | ------ |
| `youtube-data-api-guides`    | Getting started, OAuth, implementation guides, quickstarts, libraries | 43     |
| `youtube-data-api-reference` | REST resource/method reference + Discovery JSON                       | 77     |

Key search endpoint: `GET https://www.googleapis.com/youtube/v3/search` — documented at
`/youtube/v3/docs/search/list`.

#### Status

Verified (`youtube-data-api-guides` live fetch 2026-08-20).
