---
title: "refbolt Vision"
description: "Vision and strategic rationale for refbolt — the FulmenHQ general-purpose web documentation archiver"
author: "3leapsdave"
date: "2026-03-21"
status: "draft"
version: "0.1.0"
tags: ["refbolt", "docs", "archive", "llm", "fulmen-toolbox"]
---

# refbolt Vision

## The Problem We See

Every team building native LLM clients (especially for frontier models like Grok, Claude, and GPT) faces the same friction: API reference documentation changes constantly, and keeping local, clean, versioned copies is manual, error-prone, and brittle.

- Hand-fetching Markdown pages is tedious and breaks the moment a URL changes.
- Existing clients' built-in "browse" tools are flaky and not designed for offline-first or version-pinned archives.
- Full-site mirrors (httrack/wget) produce bloated, noisy HTML.
- No lightweight, container-native tool exists that produces clean Markdown + OpenAPI/JSON trees optimized for native backends and Tauri apps.

This pain is especially acute in Lanyte, where we are building 100% native X/Grok interfaces to give Grok agents lower friction and true sovereignty.

## What refbolt Is

**refbolt** is a lightweight, container-first CLI that automatically snapshots web documentation sites — especially frontier LLM APIs — into clean, date-versioned Markdown + JSON trees.

It lives in the Fulmen toolbox as the canonical way to keep documentation alive and offline-ready across all Fulmen projects.

**Tagline**: "Your immutable docs mirror in an ever-changing web."

## Core Beliefs

1. **Documentation is code** — it deserves versioning, provenance, and offline access.
2. **Native-first wins** — especially for Grok agents and Lanyte backends. Flattening everything to OpenAI compatibility is a tax we refuse to pay.
3. **Container-native by design** — run once in CI, run forever in toolbox images.
4. **Zero license risk** — built with the same MIT/Apache discipline as fulminar and gofulmen.
5. **Extensible from day one** — LLM providers today, any site tomorrow.

## What Success Looks Like

A Lanyte engineer (or any Fulmen developer) runs:

```bash
docker run --rm -v ./archive:/data ghcr.io/fulmenhq/refbolt refbolt sync --all
```

…and instantly gets a fresh, clean archive containing:

```
xai/2026-03-21/inference.md + openapi.json
anthropic/2026-03-21/messages.md + llms-full.txt
openai/2026-03-21/openapi.json + llms-full.txt
```

The Tauri app points at latest/ for instant offline reference. The Lanyte native backends load pinned Markdown at compile time. GitHub Actions auto-detect changes and open PRs. Hand-fetching is dead forever.

## Strategic Position

refbolt sits at the intersection of:

- Lanyte native LLM backends (especially Grok for lower-friction agent execution)
- Fulmen toolbox infrastructure
- Any team that wants sovereign, offline, versioned documentation

It is not a full web scraper. It is a focused, opinionated docs archiver optimized for the exact workflow we need in 2026.

## Audience

**Primary**: Lanyte core team, Fulmen toolbox consumers, any developer maintaining native LLM clients.

**Secondary**: Teams building Tauri apps or self-hosted tools that need offline reference docs.

## Technology Choices

- Go CLI (static binary, fast, cross-platform)
- Docker-first deployment via fulmen-toolbox
- Native Markdown fetching where providers support it (.md suffix, /llms-full.txt)
- Jina Reader fallback (Apache 2.0) for noisy sites
- Git-aware diff/commit for change detection

## Long-Term Trajectory

**Near-term (2026)**: Support OpenAI + Anthropic + xAI/Grok with daily syncs.

**Medium-term**: Add arbitrary sites via config, Tauri offline viewer mode, Slack/Teams notifications on breaking changes.

**Long-term**: Become the default "docs mirror" tool across the entire Fulmen ecosystem and beyond.
