# DDR-0002: Hierarchical Provider Entries Must Be Guide-Specific

**Status**: Accepted
**Date**: 2026-03-22
**Deciders**: @3leapsdave

## Context

Cloud platforms (AWS, Azure, GCP) organize documentation into multiple guide families per service. For example, AWS Glue has both a Developer Guide (`/glue/latest/dg/`) and a Web API Reference (`/glue/latest/webapi/`), each with its own `llms.txt`. AWS Bedrock has User Guide and API Reference. Azure and GCP follow similar patterns.

The `llmstxt-hierarchical` strategy matches `base_url` as a prefix against a top-level index. A broad `base_url` like `/glue/latest` matches multiple guide families, and the fetcher picks one by index order — silently non-deterministic if the upstream index is reordered.

This was caught during FA-050/FA-030 review when `aws-glue` was configured with `/glue/latest` and matched both `/glue/latest/dg/llms.txt` and `/glue/latest/webapi/llms.txt`.

## Decision

**Every hierarchical provider entry MUST use a guide-specific `base_url`** — one that resolves to exactly one `llms.txt` in the upstream index.

Rules:

1. **One provider entry = one guide family.** If a service has User Guide, API Reference, and Developer Guide, each is a separate provider entry (e.g., `aws-bedrock-userguide`, `aws-bedrock-apiref`).

2. **`base_url` must be specific enough to match exactly one index entry.** Use `/glue/latest/dg`, not `/glue/latest`. Use `/s3/latest/userguide`, not `/s3/latest`.

3. **Slug must encode the guide family** when a service has multiple guides. Convention: `{cloud}-{service}-{guide}` (e.g., `aws-glue-dg`, `aws-bedrock-apiref`, `azure-storage-rest`).

4. **Config review must verify match cardinality.** When adding a hierarchical provider, run the match logic against the live index and confirm exactly one URL matches. The unit test suite includes a broad-prefix test case as a reminder.

5. **This rule applies to all cloud platforms**, not just AWS. Azure (`learn.microsoft.com`) and GCP (`cloud.google.com`) publish similar multi-guide structures and will hit the same ambiguity if configured with broad prefixes.

## Consequences

- **More provider entries** for multi-guide services — Bedrock alone needs two entries instead of one. This is intentional: explicit is better than implicit.
- **No silent index-order dependency** — each entry is deterministic regardless of upstream index changes.
- **Slug proliferation is bounded** — most services have 1–3 guide families, and only opted-in services get entries.
- **Config reviewers have a clear rule to check** — "does `base_url` match exactly one `llms.txt`?" is a binary test.

## Alternatives Considered

- **Multi-match aggregation** (one provider entry archives all matched guides): considered, but creates archive path collisions (`llms.txt` filename overlap) and mixes content semantics. Would require subpath-aware archiving.
- **Selection field** (e.g., `guide_family: userguide`): adds schema complexity for a problem solved by prefix specificity. Rejected as over-engineering.
- **Runtime ambiguity warning** (log a warning on multi-match, pick first): hides a config error behind a log line. Config should be correct at rest, not just at runtime.
