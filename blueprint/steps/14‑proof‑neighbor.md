---
id: step.14.proof‑neighbor
depends_on:
  - step.13.proof‑empty
tags: [proof, step]
---

## Objective

Generate and verify exclusion proofs (neighbor leaf case).

## Implements

- **§7.2 Exclusion Proof – Neighbor Leaf** case.
  _What happens_:

  - When traversal collides with a different leaf, generator includes that leaf + siblings.
  - Upper bound on proof size ("≤ path + 1") tests the _Concise Proof Format_ claim (§Advantages list, "Optimised proof format").

## Done When ✓

- [ ] Proof size ≤ path length + 1 in tests.
