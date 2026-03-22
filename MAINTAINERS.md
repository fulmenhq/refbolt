# refbolt – Maintainers

**Project**: refbolt
**Purpose**: Archive web docs (especially frontier LLM APIs) into clean, versioned Markdown trees
**Governance Model**: FulmenHQ / 3 Leaps Initiative

## Human Maintainers

### @3leapsdave (Dave Thompson)

- **Role**: Project Lead & Primary Maintainer
- **Responsibilities**: Architecture, integration oversight, production readiness
- **Contact**: dave.thompson@3leaps.net | GitHub [@3leapsdave](https://github.com/3leapsdave) | X [@3leapsdave](https://x.com/3leapsdave)
- **Supervision**: All AI agent contributions

## Agentic Roles

This repository uses role-based agentic development. Agents operate under roles defined in the [Role Catalog](config/agentic/roles/README.md).

### Available Roles

| Role       | Catalog                                             | Use When                                            |
| ---------- | --------------------------------------------------- | --------------------------------------------------- |
| `cxotech`  | [cxotech.yaml](config/agentic/roles/cxotech.yaml)   | Strategy, architecture, delivery coordination       |
| `devlead`  | [devlead.yaml](config/agentic/roles/devlead.yaml)   | Implementation, CLI, provider framework             |
| `devrev`   | [devrev.yaml](config/agentic/roles/devrev.yaml)     | Code review, bug finding, four-eyes audit           |
| `infoarch` | [infoarch.yaml](config/agentic/roles/infoarch.yaml) | Archive output quality, Markdown standards, schemas |
| `qa`       | [qa.yaml](config/agentic/roles/qa.yaml)             | Testing, validation, coverage                       |
| `releng`   | [releng.yaml](config/agentic/roles/releng.yaml)     | Release engineering, CI/CD validation               |
| `secrev`   | [secrev.yaml](config/agentic/roles/secrev.yaml)     | Security analysis, credential handling              |

### Operating Modes

**Supervised Mode** (current):

- All agent work requires human review before commit
- Human maintainer (@3leapsdave) is Committer-of-Record

**Autonomous Mode** (future):

- Agents operate within defined boundaries
- Escalation contact for issues: @3leapsdave
- Requires `Autonomous-Agent:` and `Escalation-Contact:` trailers

## Attribution Guidelines

### Required Trailers

```
Co-Authored-By: <Model> <noreply@fulmenhq.dev>
Role: <role>
Committer-of-Record: Dave Thompson <dave.thompson@3leaps.net> [@3leapsdave]
```

### Key Requirements

- Use `noreply@fulmenhq.dev` (NOT vendor defaults like `noreply@anthropic.com`)
- Include `Role:` trailer matching the operating role
- Include `Committer-of-Record:` for human accountability

## Governance Structure

- Human maintainers approve architecture, releases, and supervise AI agents
- AI agents execute tasks under defined roles with human oversight
- See `REPOSITORY_SAFETY_PROTOCOLS.md` for guardrails and escalation paths

## Communication Channels

- **Primary**: GitHub Issues and Pull Requests
- **Escalation**: Direct contact with @3leapsdave for critical issues

## Contribution Guidelines

All contributors (human and AI) must:

- Follow commit attribution standard
- Maintain test coverage
- Run `make check-all` before commits
- Coordinate breaking changes with @3leapsdave
