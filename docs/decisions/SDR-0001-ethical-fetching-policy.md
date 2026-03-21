# SDR-0001: Ethical Fetching Policy

**Status**: Accepted
**Date**: 2026-03-21
**Deciders**: @3leapsdave

## Context

fularchive fetches documentation from third-party sites (xAI, Anthropic, OpenAI, etc.) and archives it locally. This creates obligations around how we interact with these sites — we must not violate terms of service, abuse rate limits, or use undocumented/private APIs.

During initial development we encountered HTTP compatibility issues (Accept headers, TLS ALPN negotiation) that required workarounds. We need a clear policy to distinguish between legitimate compatibility fixes and behavior that could be seen as circumventing site protections.

## Decision

fularchive will only fetch content through **publicly documented, intended-for-consumption interfaces**:

1. **Use official LLM-friendly endpoints first.** Providers that offer `llms.txt`, `llms-full.txt`, or explicit "View as Markdown" links are signaling intent for programmatic access. These are our primary fetch targets.

2. **No undocumented or private APIs.** We will not reverse-engineer internal API endpoints, use authentication tokens from browser sessions, or access content behind paywalls or login walls.

3. **Respect robots.txt and terms of service.** Before adding a new provider, review their robots.txt and TOS. If programmatic access is prohibited, do not add the provider. Document the review in `docs/providers/`.

4. **HTTP compatibility fixes are acceptable.** Adjusting User-Agent strings, Accept headers, or TLS settings to work with CDN infrastructure is standard HTTP client engineering — not circumvention. These fixes must be documented in `docs/providers/` with the factual behavior (not framed as bypassing security).

5. **Rate limit compliance.** Respect provider rate limits. Use the `rate_limit` config field. When in doubt, default to conservative limits (1 req/s).

6. **Review on each provider addition.** Adding a new provider requires reviewing its TOS and documenting the review decision in `docs/providers/`. This is a gate — not optional.

7. **No credential harvesting or session hijacking.** The `auth_env_var` field references environment variable _names_ for legitimate API keys (e.g., Jina Reader tokens). We never store, extract, or reuse credentials from browser sessions or other tools.

## Consequences

- Adding a new provider is slightly slower (requires TOS review).
- Some sites may not be archivable if their TOS prohibits programmatic access.
- We maintain a clean posture that protects the project and its users.
- Provider notes in `docs/providers/` serve as an audit trail for fetch decisions.

## Alternatives Considered

- **No formal policy**: Rejected. Without a policy, individual contributors make ad-hoc decisions about what's acceptable, leading to inconsistency and potential TOS violations.
- **Strict browser-only fetching (via headless Chrome)**: Rejected as over-engineering. The providers we target explicitly offer Markdown endpoints for programmatic consumption.
