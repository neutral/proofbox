---
id: step.07.versioning‑commit
depends_on:
  - step.06.pebble‑persistence
tags: [versioning, step]
---

## Objective

Finalize multi‑version support, immutable reads, and `Versions()` enumeration.

## Done When ✓

- [ ] Interleaved commits produce correct roots.
- [ ] `Get(oldVersion,key)` still works after 10 newer commits.
- [ ] Enumeration speed meets NFR.
