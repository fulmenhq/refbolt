# Role Catalog

Baseline role prompts for AI agent sessions in the FulmenHQ ecosystem.

**Schema**: [`role-prompt.schema.json`](https://schemas.3leaps.dev/agentic/v0/role-prompt.schema.json) (from [crucible](https://github.com/fulmenhq/crucible))

## Available Roles

| Role                                   | Slug       | Category   | Purpose                                                |
| -------------------------------------- | ---------- | ---------- | ------------------------------------------------------ |
| [CXO Technology](cxotech.yaml)         | `cxotech`  | governance | Strategy, architecture, delivery coordination          |
| [Development Lead](devlead.yaml)       | `devlead`  | agentic    | Implementation, CLI, provider framework                |
| [Development Reviewer](devrev.yaml)    | `devrev`   | review     | Four-eyes code review                                  |
| [Information Architect](infoarch.yaml) | `infoarch` | agentic    | Archive quality, schema governance, Markdown standards |
| [QA](qa.yaml)                          | `qa`       | review     | Testing, validation, coverage                          |
| [Release Engineering](releng.yaml)     | `releng`   | automation | Release engineering, CI/CD validation                  |
| [Security Review](secrev.yaml)         | `secrev`   | review     | Security analysis, credential handling                 |

## FulmenHQ Extensions

These roles extend the [crucible baseline](https://github.com/fulmenhq/crucible/tree/main/config/agentic/roles):

| Role       | Extension Purpose                                            |
| ---------- | ------------------------------------------------------------ |
| `cxotech`  | Strategic fulcrum — product + technical decisions            |
| `devlead`  | Adds refbolt provider framework and container patterns       |
| `infoarch` | Archive output quality, Markdown standards, provider schemas |

## Usage

Reference roles by slug in `AGENTS.md`:

```markdown
## Roles

| Role      | Prompt                       | Notes           |
| --------- | ---------------------------- | --------------- |
| `devlead` | [devlead.yaml](devlead.yaml) | Implementation  |
| `secrev`  | [secrev.yaml](secrev.yaml)   | Security review |
```

## Schema Validation

All role files conform to the [role-prompt schema](https://schemas.3leaps.dev/agentic/v0/role-prompt.schema.json). Each YAML file declares its schema via `yaml-language-server` directive.

> **Note:** The schema is currently referenced by remote URL. A future sync from `fulmenhq/crucible` may vendor it locally.

## Extending Roles

To extend a baseline role:

```yaml
slug: devlead
extends: https://schemas.3leaps.dev/roles/devlead.yaml
# Add or override fields
scope:
  - ...additional scope items...
```

## References

- [Crucible baseline roles](https://github.com/fulmenhq/crucible/tree/main/config/agentic/roles) — upstream baseline
- [AGENTS.md](../../../AGENTS.md) — role catalog and session protocol for this repo
