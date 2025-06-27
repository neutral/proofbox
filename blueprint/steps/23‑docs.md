---
id: step.23.docs
depends_on:
  - step.22.ci
tags: [docs, step]
---

## Objective

Add README quick‑start and sync `docs/spec.md` to code.

## Implements

- **Scope & Assumptions** (spec evolves with code) and **Appendix A** references.
  _What happens_:

  - README quick‑start demonstrates an inclusion proof round‑trip, directly mapping to **§6.1 Lookup** and **§7.1 Proof Verification** so newcomers can reproduce behaviour.

## Done When ✓

- [ ] `make docs` runs with no diff.
