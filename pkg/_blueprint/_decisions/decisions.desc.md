# Architecture Decision Records

## Purpose

Documents significant design decisions made during the implementation of the Jellyfish Merkle Tree. Each decision record captures the context, alternatives considered, and rationale for the chosen approach.

## Current Decisions

1. **001-two-node-types-only.md** - Why we use only Leaf and Internal nodes (no extension nodes)
2. **002-separate-codec-package.md** - Codec logic in separate package from tree logic
3. **003-version-not-serialized.md** - Version provided externally, not in serialized data
4. **004-deterministic-child-ordering.md** - Children encoded in sorted nibble order
5. **005-thread-safe-nodes.md** - Built-in thread safety with RWMutex
6. **006-visitor-pattern.md** - Visitor pattern for extensible node operations
7. **007-no-panics-in-library.md** - All errors returned as values, never panic
8. **008-empty-hash-vs-empty-tree-hash.md** - Distinction between zero hash and missing children
9. **009-value-storage-strategy.md** - Lazy loading of values with hash validation
10. **010-single-child-chains.md** - Accepting single-child chains as per JMT specification

## Decision Process

- Decisions are numbered sequentially
- Status: draft → accepted/rejected → deprecated
- Include context, alternatives, and consequences
- Reference implementation code where applicable

## Template

```markdown
---
id: adr.XXX.short-name
status: draft|accepted|rejected|deprecated
date: YYYY-MM-DD
---

# ADR-XXX: Title

## Status
[Current status]

## Context
[Background and problem statement]

## Decision
[What we decided]

## Consequences
### Positive
### Negative
### Neutral

## Alternatives Considered
[Other options evaluated]

## References
[Links to code, papers, discussions]
```
