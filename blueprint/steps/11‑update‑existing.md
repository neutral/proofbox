---
id: step.11.update‑existing
depends_on:
  - step.10.insert‑basic
tags: [update, step]
---

## Objective

Handle insertions that update an existing key's value.

## Implements

- §6.2 Insertion – "Existing leaf, same key → update" path.
  _What happens_:

  - Tests that digest changes but topology is stable; this ensures clients can detect updates via root‑hash change.

## Done When ✓

- [ ] Root digest changes while tree shape stays identical.
