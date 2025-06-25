---
id: step.02.node‑structs
depends_on:
  - step.01.hasher‑abstraction
tags: [structs, step]
---

## Objective

Define all in‑memory structs (`NodeKey`, `LeafNode`, `ChildPointer`, `InternalNode`) and basic constructors with validation.

## Done When ✓

- [ ] Types live in `pkg/nodes/`.
- [ ] Unit tests enforce key length, nil child semantics, default hash constant.
- [ ] Benchmarks show InternalNode zero‑alloc creation < 100 ns.
- [ ] CI green.
