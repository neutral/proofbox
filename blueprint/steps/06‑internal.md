---
id: step.06.internal
depends_on:
  - step.02.hasher
tags: [structs, step]
---

## Objective

Add `InternalNode` with `map[byte]ChildMeta` and `Digest()`.

## Implements

- **§4.1 Internal Node** (map of ≤ 16 children) and **§4.2 Internal digest** (`0x01 || c0 … c15`).
  _What happens_:

  - Uses `map[byte]ChildMeta` to store only occupied slots, matching the _sparse optimisation_ note: "unoccupied child slot is considered a default digest."
  - Empty‑node digest test proves default‑hash concatenation logic is correct.

## Done When ✓

- [ ] Digest of empty node equals concat(DefaultDigest × 16).
