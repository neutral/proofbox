---
id: nfr.performance.latency
tags: [performance]
---

## Description

End‑to‑end proof verification by a stateless client **MUST** complete quickly.

## Default Target

Metric: proof verify p95 ≤ 300 µs on a 2024‑era laptop CPU.

## Measurement

Benchmarks in `go test -bench=.^Verify$ ./...` exported to `bench.txt`; CI verifies target.

## Rationale

Fast verification is essential for light‑client UX and validator timers.
