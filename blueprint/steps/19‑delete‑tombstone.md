---
id: step.19.delete‑tombstone
depends_on:
  - step.18.persist‑multi‑ver
  - step.14.proof‑neighbor
tags: [deletion, step]
---

## Objective

Support delete via tombstone and its exclusion proof.

## Implements

- **Open Question – Deletion Algorithm** (spec calls for tombstones).
  _What happens_:

  - Writes a tombstoned leaf and produces proper exclusion proof, exercising both persistent‑versioning and proof logic under deletion.

## Done When ✓

- [ ] Deleted key absent in new version but provable in old.
