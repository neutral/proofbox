---
id: step.03.in‑mem‑skeleton
depends_on:
  - step.02.node‑structs
tags: [tree, step]
---

## Objective

Implement an in‑memory JMT that supports `Put`, `Get`, and `Commit` **without persistence**.

## Done When ✓

- [ ] `pkg/jmt/tree.go` with staging map and versioned root map.
- [ ] All methods compile; tests cover insert/read path.
- [ ] Bench throughput ≥ 20 k inserts/s in RAM.
