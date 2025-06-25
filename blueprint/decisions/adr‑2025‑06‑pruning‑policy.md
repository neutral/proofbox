---
id: adr.2025‑06‑pruning‑policy
depends_on:
  - nfr.storage.efficiency
---

## Context

Old versions accumulate stale nodes indefinitely.

## Decision

Retain last **N = 100 k** versions; nightly job sweeps nodes whose `Version < tip‑N`.

## Consequences

- Predictable storage bound.
  − Nodes needed for audit beyond N must be restored from archive.

## Alternatives Considered

Time‑based (30 days) vs count‑based — count chosen for deterministic behaviour.
