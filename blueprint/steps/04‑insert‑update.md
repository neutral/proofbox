---
id: step.04.insert‑update
depends_on:
  - step.03.in‑mem‑skeleton
tags: [insert, step]
---

## Objective

Complete full insert & update logic with path cloning and digest recomputation.

## Done When ✓

- [ ] Path clone algorithm passes 100 random differential‑update tests.
- [ ] Leaf replacement reuses existing hashes where possible.
- [ ] No memory leaks (checked with `go test -run Leak`).
