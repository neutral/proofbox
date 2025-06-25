---
id: adr.2025‑06‑concurrency
depends_on:
  - nfr.reliability.crash‑safety
---

## Context

Spec assumes a single writer; some deployments may require concurrent block executors.

## Decision

Keep **single writer, many readers** model for v0.1. Implement write‑side channel serializer; revisit multi‑writer later.

## Consequences

- Simpler code and proofs.
  − Throughput limited by one writer goroutine.

## Alternatives Considered

Pebble transactions per block (complex, little gain at block cadence).
