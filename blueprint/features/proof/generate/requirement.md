---
id: req.proof.generate
depends_on:
  - goal.proofbox
  - req.storage.get
  - nfr.performance.latency
tags: [api, proof]
---

## 1 Purpose & Value

As a **light client** I want `Prove(version,key)` so that a server can send me a compact inclusion or non‑inclusion proof.

## 2 Scope

Paths for present and absent keys; sibling hash collection.

## 3 Actors & Preconditions

Actor: RPC handler.
Preconditions: version exists.

## 4 Functional Requirements

FR‑PROVE‑1 — Proof contains ≤ 64 sibling entries (each with up to 16 child hashes).
FR‑PROVE‑2 — Generation time ≤ 200 µs p95.
FR‑PROVE‑3 — Proof round‑trips with Verify.

## 5 Flows

Standard Merkle proof construction while walking the path.

## 6 NFR Overrides

None.

## 7 External Interfaces & Data

API — `proof.Generate(tree.Reader(version), key)` via `proof.Generator`.
Data — sibling data containing hash and children map.

## 8 Acceptance Criteria

Proof verifies for deterministic test keys. Random key testing recommended for future enhancement.

## 9 Open Issues & Risks

Binary proof format implemented in codec package.