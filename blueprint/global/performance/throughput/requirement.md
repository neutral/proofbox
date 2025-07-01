---
id: nfr.performance.throughput
tags: [performance]
---

## Description

The write path must sustain high aggregate throughput during block execution bursts.

## Default Target

Metric: ≥ 50 000 `Put` operations per second on an 8‑core 2024 laptop (p95).

## Measurement

`go test -bench=.^BenchmarkPut$ ./...` produces `bench.txt`; CI checks p95 ≥ 50 k/s.

Note: Benchmarks require proper metrics initialization. If encountering "duplicate metrics collector" errors, ensure only one metrics instance is created during test execution.

## Rationale

Throughput at this scale keeps block‑time constant when processing 10 k mutated accounts.