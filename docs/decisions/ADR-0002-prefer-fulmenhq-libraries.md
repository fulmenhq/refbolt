# ADR-0002: Prefer Ecosystem Libraries Over Custom Implementations

**Status**: Accepted
**Date**: 2026-03-22
**Deciders**: @3leapsdave

## Context

The Fulmen and 3 Leaps ecosystems maintain shared libraries across multiple language ecosystems — gofulmen, tsfulmen, rsfulmen, pyfulmen (fulmenhq org), plus general-purpose libraries like ipcprims, sfetch, and seclusor (3leaps org). These libraries encapsulate common patterns (path matching, config loading, schema validation, hashing, etc.) that are tested and maintained centrally.

When projects implement functionality that already exists in these libraries, the result is duplicated code that diverges over time and misses upstream bug fixes.

## Decision

When implementing functionality in fularchive (or any fulmenhq/3leaps project), prefer an existing ecosystem library over a custom implementation when suitable functionality exists. For Go projects, check gofulmen first; for cross-language concerns, check the 3leaps org.

Deviation is acceptable when:

- The library does not cover the use case
- The dependency cost outweighs the benefit (e.g., pulling in a large module for a trivial helper)
- Performance or correctness requirements demand a specialized implementation

When deviating, leave a brief comment explaining why the library was not used.

## Consequences

- Reduces duplicated code across projects
- Bug fixes and improvements in shared libraries propagate automatically
- Adds a build-time dependency on the relevant module
- Contributors should be familiar with the library surface area for their language

## Alternatives Considered

- **Always roll custom**: Maximum independence per project, but duplicates effort and diverges over time. Rejected because the maintenance cost across the org is too high.
- **Mandate library use with no exceptions**: Too rigid — some cases genuinely warrant a local implementation. Rejected in favor of a prefer-with-justification policy.
