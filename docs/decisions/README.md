# Decision Records

This directory is the **single authoritative location** for all decision records in fularchive.

## Record Types

| Prefix | Type                         | Use When                                                      |
| ------ | ---------------------------- | ------------------------------------------------------------- |
| ADR    | Architecture Decision Record | Technical architecture choices (provider design, storage, CI) |
| SDR    | Security Decision Record     | Security posture, credential handling, trust boundaries       |
| DDR    | Data Decision Record         | Archive format, schema governance, output conventions         |

## Index

### Architecture Decisions

| ADR                                           | Decision                           | Date       |
| --------------------------------------------- | ---------------------------------- | ---------- |
| [ADR-0001](ADR-0001-llmstxt-primary-fetch.md) | llms.txt as primary fetch strategy | 2026-03-21 |

### Security Decisions

| SDR                                             | Decision                | Date       |
| ----------------------------------------------- | ----------------------- | ---------- |
| [SDR-0001](SDR-0001-ethical-fetching-policy.md) | Ethical fetching policy | 2026-03-21 |

### Data Decisions

| DDR                                            | Decision               | Date       |
| ---------------------------------------------- | ---------------------- | ---------- |
| [DDR-0001](DDR-0001-archive-tree-structure.md) | Archive tree structure | 2026-03-21 |

## Creating a New Decision Record

1. Copy `ADR-template.md` (same template works for SDR/DDR — just change the prefix)
2. Assign the next number for the appropriate type (zero-padded, e.g., ADR-0001, SDR-0001)
3. Keep it short (1-2 pages)
4. Link to relevant code and docs
5. Update this index
