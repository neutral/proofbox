---
id: nfr.storage.efficiency
tags: [storage]
---

## Description

On‑disk bytes must largely represent live data; amplification and waste must stay low.

## Default Target

Metric: ≥ 90 % of disk bytes are live versions; Pebble compaction < 5 % wall‑time.

## Measurement

`pebble` metrics exported via `/metrics`; integration test asserts ratio after 20 k blocks.

## Rationale

Lower storage cost and faster snapshot shipping for archival nodes.
