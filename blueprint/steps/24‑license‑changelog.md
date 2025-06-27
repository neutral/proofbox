---
id: step.24.license‑changelog
depends_on:
  - step.23.docs
tags: [legal, step]
---

## Objective

Insert MIT LICENSE headers and generate `CHANGELOG.md`.

## Implements

- **Open question – Legal/Oss housekeeping** (not explicit in spec but always required).
  _What happens_:

  - Ensures SPDX correctness and a Conventional‑Commit‑based changelog enumerating every step ID for traceability—supporting audits and diff‑driven verification mentioned under _Persistent Versioning_ ("can reconstruct any version").

## Done When ✓

- [ ] `addlicense -check ./...` passes and changelog lists all steps.
