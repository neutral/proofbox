---
id: step.13.proof‑empty
depends_on:
  - step.12.proof‑inclusion
tags: [proof, step]
---

## Objective

Generate and verify exclusion proofs (empty subtree case).

## Implements

- **§7.2 Exclusion Proof – Empty Subtree**.
  _What happens_:

  - Proof generator locates the first missing child and supplies default digest evidence; verifier checks empty‑subtree condition.

## Done When ✓

- [ ] Absent keys verify as non‑members.
