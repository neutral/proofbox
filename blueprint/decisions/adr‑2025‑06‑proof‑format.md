---
id: adr.2025‑06‑proof‑format
depends_on:
  - nfr.performance.latency
---

## Context

Proofs need a stable, language‑agnostic wire representation.

## Decision

Use **length‑prefixed binary**:
`uvarint(numHashes) || hashes[...] || leafFlag || (leafKey || leafHash || leafValueLen || leafValue?)`

## Consequences

- Zero external dep, matches Go’s `encoding/binary`.
  − Slightly larger than protobuf varint‑optimised, but negligible (~40 B).

## Alternatives Considered

1. protobuf (heavy dep).
2. flatbuffers (overkill).
3. JSON (too big).
