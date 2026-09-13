# ADR-0001: llms.txt as Primary Fetch Strategy

**Status**: Accepted
**Date**: 2026-03-21
**Deciders**: @3leapsdave

## Context

Documentation providers offer content through multiple channels: individual HTML pages, `.md` suffix variants, OpenAPI specs, and single-file dumps (`llms.txt`, `llms-full.txt`). We need a fetch strategy that is reliable, efficient, and respectful of provider infrastructure.

## Decision

When a provider offers a `llms.txt` or `llms-full.txt` endpoint, refbolt uses it as the **primary fetch strategy**:

1. Fetch the single file in one HTTP request.
2. Split it into individual pages using section delimiters (e.g. `===/<path>===` for xAI).
3. Write each section as a separate file in the archive tree.
4. Supplement with individual `.md` page fetches for targeted updates.

The `llms_txt_url` field in provider config controls this. When absent, refbolt falls back to fetching individual paths. Optional `llms_full_txt_url` names the complete dump when `llms.txt` is only an index (see the 2026-09 note below).

## Consequences

- **Efficiency**: One HTTP request archives an entire documentation site (xAI: ~181 sections from `llms-full.txt`, ~1.5MB).
- **Reliability**: Single endpoint is less likely to hit rate limits or trigger bot detection than crawling dozens of pages.
- **Provider alignment**: `llms.txt` / `llms-full.txt` files are explicitly published for programmatic consumption — fetching them is the intended use case.
- **Format dependency**: We depend on the delimiter format (`===/<path>===` for xAI; `URL:` / `Source:` / YAML frontmatter elsewhere), which could change. The splitter is isolated in `internal/provider/llmstxt.go` for easy adaptation.
- **Not universal**: Not all providers offer `llms.txt`. The fallback to individual page fetching must remain robust.
- **Index vs dump (2026-09)**: Some sites now publish `llms.txt` as a markdown link catalog and put the concatenated dump at `llms-full.txt`. Native fetch treats zero split sections as a signal to try `llms_full_txt_url` or a sibling `llms-full.txt`. Auto-detected siblings are accepted only when they use xAI-style `===/<path>===` markers, so unrelated firehoses (e.g. docs.x.com) are not ingested.

## Alternatives Considered

- **Crawl HTML and convert**: Works broadly but is slower, noisier, and more likely to trigger rate limits or bot detection.
- **Individual `.md` pages only**: More targeted but requires maintaining a complete URL list per provider.
- **Sitemap-based discovery**: Most doc sites don't publish sitemaps. Where available, this could supplement llms.txt in future.
