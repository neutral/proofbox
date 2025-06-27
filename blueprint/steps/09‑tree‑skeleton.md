---
id: step.09.tree‑skeleton
depends_on:
  - step.07.node‑iface
tags: [tree, step]
---

## Objective

Create `Tree` with `root Node`, `version uint64`, and `Get` on empty tree.

## Implements

- **§6 Algorithms – Lookup** initial conditions (root may be empty).
  _What happens_:

  - Creates minimal `Tree` struct with a nil‑root sentinel that returns `nil` for any GET, satisfying lookup step 1 ("Start at Root… if tree empty -> not found").

## Done When ✓

- [ ] `Get(anyVersion, anyKey)` returns nil on empty tree.
