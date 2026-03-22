---
title: AGENTS.md
description: Agentic attribution and development standards for fularchive
author_of_record: Dave Thompson (@3leapsdave)
status: draft
---

# fularchive – AI Agent Guide

**Project**: fularchive
**Purpose**: Archive web docs (especially frontier LLM APIs) into clean, versioned Markdown trees
**Maintainers**: See `MAINTAINERS.md`

## Operating Model

| Aspect   | Setting                                  |
| -------- | ---------------------------------------- |
| Mode     | Supervised (human reviews before commit) |
| Role     | devlead (default)                        |
| Identity | Per session (no persistent memory)       |

See [3leaps-crucible agent-identity standard](https://crucible.3leaps.dev/repository/agent-identity) for operating modes and attribution patterns.

## Read First

1. **Confirm your role.** Roles are defined in [`config/agentic/roles/`](config/agentic/roles/). Default to `devlead` if unspecified.
2. **Check `AGENTS.local.md`** if it exists (gitignored) for machine-specific instructions, credential guidance, and tactical session overrides. This file is the final authority on local environment configuration.
3. **Read `MAINTAINERS.md`** for human maintainer contacts.
4. **Read files before editing them.**
5. **Read the ADRs** in `docs/decisions/` — they capture key decisions on provider architecture, archive format, and container strategy.

## Quick Reference

| Task           | Command          | Notes                       |
| -------------- | ---------------- | --------------------------- |
| Build          | `make build`     | Builds `bin/fularchive`     |
| Tests          | `make test`      | Must pass before committing |
| Format + Lint  | `make fmt`       | Format + vet                |
| Quality checks | `make check-all` | fmt + lint + test           |

## Architecture Overview

fularchive is a container-first CLI for periodic documentation archival.

```
┌──────────────────────────────────────┐
│         Provider Registry            │
│  (openai, anthropic, xai, etc.)      │
└──────┬───────────────────────────────┘
       │
  ┌────▼────────┐    ┌───────────────┐
  │  Fetcher    │    │  Git Differ   │
  │ (md/jina)   │    │ (commit/PR)   │
  └─────────────┘    └───────────────┘
       │                    │
  ┌────▼────────────────────▼──────────┐
  │      Archive Tree (date-versioned) │
  │  /data/archive/<provider>/<date>/  │
  └────────────────────────────────────┘
```

Key conventions:

- Provider configs define URL patterns, native Markdown availability, fallback strategies
- Date-versioned output tree under configurable root
- Git-aware change detection for incremental updates
- Container-first: CLI works standalone, runner image adds cron scheduling

## Role-Based Development

Agents operate in role contexts. Each role has defined scope, mindset, and escalation paths.

Full role definitions live in [`config/agentic/roles/`](config/agentic/roles/) as YAML files following the [crucible role-prompt schema](https://schemas.3leaps.dev/agentic/v0/role-prompt.schema.json).

### Catalog Roles

| Role       | Focus                                                                  |
| ---------- | ---------------------------------------------------------------------- |
| `cxotech`  | Strategic architecture, product decisions, delivery coordination       |
| `devlead`  | Core implementation, CLI, provider framework                           |
| `devrev`   | Code review, bug finding, four-eyes audit                              |
| `infoarch` | Archive output quality, Markdown standards, provider schema governance |
| `qa`       | Testing, validation, coverage                                          |
| `releng`   | Release engineering, CI/CD validation                                  |
| `secrev`   | Security analysis, credential handling                                 |

### Role Notes

- **cxotech covers delivery coordination** for this project. No separate deliverylead role is needed at this scale. If the project grows to 3+ concurrent streams with external contributors, consider adding deliverylead.
- **releng covers CI/CD** for this project. No separate cicd role — releng handles both release coordination and pipeline authoring/validation.
- **infoarch is a first-class production role** here, not just "docs work." fularchive's core value proposition is clean, well-structured Markdown output. infoarch owns the quality bar for archive output, provider config schemas, and metadata standards.
- **Documentation work** (ADRs, architecture docs, schemas) is done by whichever role owns the task — devlead for implementation docs, cxotech for architecture docs, infoarch for output standards and schemas.

### fularchive-Specific Roles (Inline)

#### archiver – Archive Provider Implementation

- **Scope**: Provider packages (Anthropic, OpenAI, xAI, etc.), fetch strategies, URL patterns, Jina Reader fallback
- **Responsibilities**: Provider config authoring, fetch logic, output formatting, archive tree compliance
- **Escalates to**: devlead for framework/API design decisions, infoarch for output quality concerns, secrev for credential handling

## Session Protocol

### Startup

1. Read `AGENTS.local.md` if it exists (gitignored; machine-specific instructions, credential guidance, local overrides)
2. Identify your role from context or request assignment; read the role YAML from `config/agentic/roles/`
3. Scan relevant code before making changes
4. Review relevant ADRs in `docs/decisions/` for the area you're working in

### Before Committing

1. Run quality gates: `make check-all`
2. Verify tests pass
3. Stage all modified files
4. Use proper attribution format (see below)

### Escalation

Escalate to maintainers (see `MAINTAINERS.md`) for:

- Releases and version tags
- Breaking changes
- Security concerns (especially credential handling)
- Architectural decisions

## Commit Attribution

**MANDATORY** — all AI-assisted commits must use the exact format below. No exceptions.

### Why `noreply@fulmenhq.dev`

AI model providers ship default Co-Authored-By emails like `noreply@anthropic.com`. A GitHub user has associated such an email with their account, causing them to appear as a contributor on any repository that uses that default attribution. This is email squatting — it creates false contributor provenance.

We use `noreply@fulmenhq.dev` (a domain we control) to eliminate this attack vector entirely. **Never use `noreply@anthropic.com`, `noreply@openai.com`, or any other model provider email in Co-Authored-By lines.**

### Format

```
<type>(<scope>): <subject>

<body>

Changes:
- Specific change 1
- Specific change 2

Generated by <Model> via <Interface> under supervision of @3leapsdave

Co-Authored-By: <Model> <noreply@fulmenhq.dev>
Role: <role>
Committer-of-Record: Dave Thompson <dave.thompson@3leaps.net> [@3leapsdave]
```

### Rules

1. **`Co-Authored-By` email MUST be `noreply@fulmenhq.dev`** — never a model provider domain
2. **`<Model>` is the specific model name** (e.g., `Claude Opus 4.6`, `Claude Sonnet 4.6`) — not just `Claude`
3. **`<Interface>` is the tool used** (e.g., `Claude Code`, `API`)
4. **`Committer-of-Record` identifies the human** who supervised and approved the commit
5. **`Role` matches the active role slug**
6. **Types**: `feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`
7. **Scopes**: `provider`, `fetcher`, `archive`, `git`, `config`, `docker`

## DO / DO NOT

### DO

- Run `make check-all` before commits
- Read files before editing them
- Keep changes minimal and focused
- Test manually when changing CLI behavior
- Prefer 3 Leaps ecosystem libraries over custom implementations when suitable functionality exists — see [ADR-0002](docs/decisions/ADR-0002-prefer-fulmenhq-libraries.md)

### DO NOT

- Push without maintainer approval
- Skip quality gates
- Commit secrets or credentials
- Commit `.plans/` contents (gitignored)
- Create unnecessary files
- Touch code outside your task scope
- Use `noreply@anthropic.com` or any model provider email in attribution

## References

- `AGENTS.local.md` - Machine-specific instructions (gitignored; read if present)
- `MAINTAINERS.md` - Human maintainers
- `README.md` - Project overview
- `docs/decisions/` - Architecture Decision Records
- `config/agentic/roles/` - Role catalog
- `.plans/` - Scratch workspace for session-level planning (gitignored, ephemeral — not the source of truth for delivery planning; see `AGENTS.local.md` for productbook location)

### Upstream

- [crucible/config/agentic/roles/](https://github.com/fulmenhq/crucible/tree/main/config/agentic/roles) - Baseline role definitions
- [crucible agent-identity standard](https://crucible.3leaps.dev/repository/agent-identity) - Operating modes and attribution
