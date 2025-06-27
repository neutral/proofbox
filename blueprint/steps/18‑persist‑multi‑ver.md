---
id: step.18.persist‑multi‑ver
depends_on:
  - step.17.pebble‑driver
  - step.15.versioning
tags: [storage, step]
---

## Objective

Flush `UpdateBatch` to Pebble and enable historical `Get`.

## Implements

- **§5 S1/S2** fully (actual disk persistence) and **§6.2** ("flush batch in one transaction").
  _What happens_:

  - Enables historical lookup by reconstructing NodeKeys with embedded version—key to auditors verifying old blocks.

## Done When ✓

- [ ] Querying version N‑2 after three commits succeeds.
