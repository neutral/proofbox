---
id: step.05.leaf
depends_on:
  - step.02.hasher
  - step.03.key‑util
tags: [structs, step]
---

## Objective

Create `LeafNode` holding `Key`, `ValueHash`, and `Digest()`.

## Implements

- **§4.1 Leaf Node definition** and **§4.2 Leaf digest** (`0x00 || key || value_hash`).
  _What happens_:

  - Enforces digest formula exactly as per spec; tests ensure attackers cannot tamper with key or value without changing the digest (Security R3).

## Done When ✓

- [ ] Digest formula unit test passes.
