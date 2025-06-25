---
id: nfr.scalability.capacity
tags: [scalability]
---

## Description

Proofbox must handle very large trees and long histories without rewrites.

## Default Target

Metric: ≥ 1 billion leaves and ≥ 100 k versions with ≤ 512 MB RSS during a 10 k‑leaf block.

## Measurement

Load generator in `cmd/genbig` populates the DB; CI job asserts memory via `/usr/bin/time -v`.

## Rationale

Main‑net deployments grow for years; re‑sharding the DB would be operationally risky.
