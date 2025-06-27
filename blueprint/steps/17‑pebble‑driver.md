---
id: step.17.pebble‑driver
depends_on:
  - step.16.update‑batch
tags: [storage, step]
---

## Objective

Wrap Pebble DB with `Get`, `Set`, and `WriteBatch`.

## Implements

- **§5 Persistent Storage Requirements S3** (batch commit) and **Architecture Overview** storage layer.
  _What happens_:

  - Abstracts Pebble operations behind a small interface so tree logic remains DB‑agnostic.

## Done When ✓

- [ ] Opening DB twice retains data.
