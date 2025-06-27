---
id: step.12.proof‑inclusion
depends_on:
  - step.11.update‑existing
tags: [proof, step]
---

## Objective

Generate and verify inclusion proofs.

## Implements

- **§7.1 Inclusion Proof** plus **Security R3** (verifier recomputes root).
  _What happens_:

  - Collects sibling hashes along the path and a leaf blob, then runs verification logic mirroring the spec's pseudo‑code.

## Done When ✓

- [ ] 1 000 random inclusion proofs verify.
