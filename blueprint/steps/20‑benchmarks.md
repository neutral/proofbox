---
id: step.20.benchmarks
depends_on:
  - step.18.persist‑multi‑ver
tags: [benchmark, step]
---

## Objective

Add benchmarks for hasher, lookup, and commit; save under `bench/`.

## Implements

- **Non‑Functional Goals – Performance & Scalability**.
  _What happens_:

  - Encodes spec's benchmark aspirations ("bench throughput ≥ 20 k inserts/s") into repeatable Go benchmarks and spills raw numbers to files for trend tracking.

## Done When ✓

- [ ] `go test -bench ./...` writes three result files.
