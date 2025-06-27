---
id: step.03.key‑util
depends_on:
  - step.02.hasher
tags: [types, step]
---

## Objective

Introduce `type Key [32]byte` and helper `Nibble(key Key, depth int)`.

## Implements

- **Glossary – Key, Nibble**
- **§4.2 Digest Computation** (leaf digest includes the full key)
  _What happens_:

  - Makes keys an explicit fixed‑size array (`[32]byte`), matching the 256‑bit assumption in §3.
  - Supplies `Nibble(k, depth)` so traversal algorithms (insert, lookup) can follow key bits as 4‑bit chunks, reflecting the **radix‑16 design** (§4.1 Node Types / Internal Node).

## Done When ✓

- [ ] Fuzz test of `Nibble` across 64 depths passes.
