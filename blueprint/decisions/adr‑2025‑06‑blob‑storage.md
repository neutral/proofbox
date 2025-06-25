---
id: adr.2025‑06‑blob‑storage
depends_on:
  - nfr.storage.efficiency
---

## Context

Value blobs can be inlined inside the LeafNode or stored in a separate column‑family keyed by `Key`.

## Decision

_Inline up to 256 bytes, else external CF._

## Consequences

- Fast reads for small values.
  − Two Pebble gets for large values.
- Keeps LeafNode size bounded for hashing performance.

## Alternatives Considered

1. Always inline (simple but large nodes).
2. Always external CF (smaller nodes but extra I/O).
