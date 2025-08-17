---
id: req.proof.verify
depends_on:
  - goal.proofbox
  - nfr.performance.latency
  - nfr.security.integrity
tags: [api, proof]
---

## 1 Purpose & Value

As a **light client** I want `Verify(root,key,value,proof)` so that I can trust server responses offline.

## 2 Scope

Hash re‑computation, constant time compare.

## 3 Actors & Preconditions

Actor: wallet application.
Preconditions: root was obtained from a trusted source.

## 4 Functional Requirements

FR‑VERIFY‑1 — Returns `true` only for valid proofs.
FR‑VERIFY‑2 — Execution time ≤ 300 µs p95.
FR‑VERIFY‑3 — Constant‑time comparison to mitigate timing attacks.

## 5 Flows

Re‑hash leaf, fold sibling list, compare to root.

## 6 NFR Overrides

None.

## 7 External Interfaces & Data

API — `Tree.Verify`.
Data — proof binary blob.

## 8 Acceptance Criteria

Unit tests with forged proofs must return `false`.

## 9 Open Issues & Risks

None.
