---
id: step.09.perf‑bench
depends_on:
  - step.08.non‑inclusion‑proofs
  - nfr.performance.throughput
tags: [benchmark, step]
---

## Objective

Add micro & macro benchmarks, Prometheus metrics, and autosave `bench.txt`.

## Done When ✓

- [ ] `go test -bench` covers Put, Get, Prove, Verify.
- [ ] Results exported to `bench.txt`.
- [ ] `jmt_commit_latency_seconds` appears at `/metrics`.
