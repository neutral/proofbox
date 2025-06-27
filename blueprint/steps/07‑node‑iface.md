---
id: step.07.node‑iface
depends_on:
  - step.05.leaf
  - step.06.internal
tags: [structs, step]
---

## Objective

Define `Node` interface satisfied by both node types.

## Implements

- Architecture's "in‑memory logic layer" abstraction → polymorphic `Node` interface.
  _What happens_:

  - Allows generic traversal code to call `Digest()` without type‑switching, keeping the implementation simple (Non‑functional goal: **Simplicity and Maintainability**).

## Done When ✓

- [ ] `var _ Node = (*LeafNode)(nil)` compiles for both nodes.
