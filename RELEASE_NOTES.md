# Release Notes

This document tracks release notes for refbolt releases.

> **Convention**: Keep only the latest 3 releases here to prevent file bloat. Older releases are archived in `docs/releases/`.

## [0.1.0] - 2026-03-21

### Project Scaffolding

**Release Type**: Initial Setup
**Status**: ✅ Released

First release establishing the refbolt project structure, build system, and agentic development framework.

#### Overview

- Dual MIT/Apache-2.0 licensing
- Role catalog with 7 roles tailored for document archiving
- Makefile with goneat DX tooling integration
- Governance files (AGENTS.md, MAINTAINERS.md, REPOSITORY_SAFETY_PROTOCOLS.md)
- Productbook delivery board in `fulmenhq-productbook-internal`

#### What's Next

- Go module initialization with CLI skeleton (cobra)
- Provider framework and registry
- First providers: Anthropic, OpenAI
- Dockerfile for CLI and runner images
