# refbolt – Repository Safety Protocols

This document outlines the safety protocols for refbolt repository operations.

## Quick Reference

- **Human Oversight Required**: All merges, tags, and publishes need @3leapsdave approval
- **Use Make Targets**: Prefer `make` commands for consistency and safety
- **Plan Changes**: Document work plans in `.plans/` before structural changes
- **Incident Response**: Follow escalation process to @3leapsdave for critical issues

## High-Risk Operations

### Version Bumps

- **Process**: Update VERSION constant in relevant files
- **Verification**: Run `make test && make build` after version bump
- **Approval**: Major version bumps require @3leapsdave approval

### Release Operations

- **Tagging**: Only @3leapsdave can create release tags
- **Publishing**: Verify GitHub release works post-publish

### Structural Changes

- **Package Reorganization**: Document in feature briefs, get approval
- **CLI Changes**: Breaking CLI changes require deprecation period
- **Config Changes**: New config keys must have defaults

## Incident Response

### Build Failures

1. Check `make test` and `make lint` output
2. Fix failing tests or linting issues
3. Verify fix with fresh build
4. Document root cause in commit message

### Critical Security Issue

1. **DO NOT commit fixes to public main branch immediately**
2. Contact @3leapsdave via direct channel
3. Create private security branch if needed
4. Coordinate hotfix release process

## Safety Checklist for Common Operations

### Before Every Commit

- [ ] Tests pass: `make test`
- [ ] Code formatted: `make fmt`
- [ ] Lint clean: `make lint`
- [ ] Attribution trailers included

### Before Every PR

- [ ] README updated (if user-facing changes)
- [ ] Tests cover new functionality
- [ ] No hardcoded secrets

## Guardrails

### Automated Protections

- `.plans/` is gitignored (planning files never committed)
- `.env` is gitignored (secrets never committed)
- Make targets enforce quality gates

### Manual Protections

- All releases require @3leapsdave approval
- Breaking changes require deprecation period
- Security issues handled privately first

## Escalation Paths

1. Check README and docs/ first
2. Ask @3leapsdave
3. Better to pause than proceed incorrectly

## Incident Recording

Operational incidents are documented in `docs/ops/incidents/` using the format `INC-NNN-<slug>.md`.

## References

- `AGENTS.md` - Agent guidelines and startup protocol
- `MAINTAINERS.md` - Agent identities and responsibilities
