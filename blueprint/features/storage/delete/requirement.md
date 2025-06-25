---
id: req.storage.delete
depends_on:
  - goal.proofbox
  - adr.2025‑06‑deletion‑semantics
tags: [api, storage]
---

## 1 Purpose & Value

As a **node operator** I want to delete a key and recompute the root so that I can remove erroneous or obsolete items.

## 2 Scope

Marking leaves as absent and proof updates.

## 3 Actors & Preconditions

Actor: admin tool.
Preconditions: key exists (or not) in current version.

## 4 Functional Requirements

FR‑DEL‑1 — `Delete(key)` stages a removal.
FR‑DEL‑2 — `Commit` updates root and removes leaf in subsequent `Get`.
FR‑DEL‑3 — Deletion proof passes `Verify`.

## 5 Flows

Same as Insert but leaf removed and path possibly collapsed (per ADR).

## 6 NFR Overrides

None (efficiency NFR impacted; see ADR).

## 7 External Interfaces & Data

API — `Tree.Delete`.

## 8 Acceptance Criteria

Integration test deletes a key and verifies absence afterwards.

## 9 Open Issues & Risks

Performance of mass deletes; bloom filters for stale nodes.
