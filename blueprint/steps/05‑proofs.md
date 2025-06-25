---
id: step.05.proofs
depends_on:
  - step.04.insert‑update
  - req.proof.generate
tags: [proof, step]
---

## Objective

Implement inclusion & preliminary non‑inclusion proof generation plus in‑memory verification.

## Done When ✓

- [ ] `Proof` struct and encode/decode helpers.
- [ ] 1 000‑sample test round‑trips verify.
- [ ] Proof length distribution logged to `bench.txt`.
