---
id: step.10.insert‑basic
depends_on:
  - step.09.tree‑skeleton
  - step.03.key‑util
tags: [insert, step]
---

## Objective

Support inserting first key/value into an empty tree.

## Implements

- **§6.2 Insertion & Update** case "Empty Slot → create new leaf" as the very first insertion.
  _What happens_:

  - Adds ability to store first `(key,value)` and compute a non‑default root digest, preparing for later branching logic.

## Done When ✓

- [ ] After insert, `Get` returns stored value.
