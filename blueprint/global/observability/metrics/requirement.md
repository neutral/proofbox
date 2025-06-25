---
id: nfr.observability.metrics
tags: [observability]
---

## Description

Operators need real‑time insight into performance and resource use.

## Default Target

Metric: `/metrics` endpoint exports `jmt_commit_latency_seconds`, `jmt_proof_size_bytes`, `jmt_db_live_bytes`.

## Measurement

Integration test hits `/metrics` and parses exposition format.

## Rationale

Prometheus compatibility is de‑facto standard in cloud‑native deployments.
