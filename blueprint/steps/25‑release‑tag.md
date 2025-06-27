---
id: step.25.release‑tag
depends_on:
  - step.24.license‑changelog
tags: [release, step]
---

## Objective

Create annotated git tag `v0.1.0` and push.

## Implements

- **Architecture Overview** final sentence: "root hash of a given version serves as the authenticated state identifier;" the git tag v0.1.0 acts as the implementation identifier.
  _What happens_:

  - Freezes the code so external auditors can reference both spec version and implementation tag, completing the journey from high‑level design to working library.

## Done When ✓

- [ ] `git describe --tags` prints `v0.1.0`.
