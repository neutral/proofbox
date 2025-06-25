---
id: step.06.pebble‑persistence
depends_on:
  - step.05.proofs
  - nfr.reliability.crash‑safety
tags: [pebble, step]
---

## Objective

Introduce Pebble storage layer, encode/decode, and swap in for the in‑memory map.

## Done When ✓

- [ ] `pkg/storage/pebble.go` with key & value codecs.
- [ ] Commit uses `pebble.Batch`.
- [ ] Crash‑safety test passes (kill‑restart scenario).
