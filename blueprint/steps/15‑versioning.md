---
id: step.15.versioning
depends_on:
  - step.11.update‑existing
tags: [versioning, step]
---

## Objective

Implement `Begin(version)` and path‑cloning for new versions.

## Implements

- **§4.3 NodeKey**, **§6.2 Persistent Structure** ("path cloning"), and **§5 S2 sequential writes**.
  _What happens_:

  - `Begin(version)` forks the root; ancestor duplication realises the _persistent functional tree_ described under _Persistent Versioning_.

## Done When ✓

- [ ] Old root still verifiable after a new commit.
