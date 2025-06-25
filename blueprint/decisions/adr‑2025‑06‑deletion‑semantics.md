---
id: adr.2025‑06‑deletion‑semantics
depends_on:
  - nfr.storage.efficiency
---

## Context

Requirement FR‑8 introduces `Delete(key)`. We must decide whether to

- (A) leave _tombstone_ leaves that keep the path fully populated, or
- (B) collapse single‑child chains to save space.

## Decision

**Pending** — to be determined before Step 08. Non‑decision recorded so
future contributors know this is open.

## Consequences

- A) simplifies proofs (same path length).
  − A) more disk usage.
- B) less storage.
  − B) proof format must encode “collapsed” steps; code more complex.

## Alternatives Considered

1. Store a tombstone flag in the leaf → hybrid.
2. External garbage‑collection process.
