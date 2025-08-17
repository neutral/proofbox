---
id: req.metadata.versions
depends_on:
  - goal.proofbox
tags: [api, metadata]
---

## 1 Purpose & Value

As an **auditor** I want to list committed versions and their roots so that I can verify consistency over time.

## 2 Scope

`Versions()` enumeration, root retrieval.

## 3 Actors & Preconditions

Actor: audit tool.
Preconditions: DB opened read‑only or read‑write.

## 4 Functional Requirements

FR‑VERS‑1 — Versions returned in strictly increasing order.
FR‑VERS‑2 — Root for each version matches value returned by Commit.
FR‑VERS‑3 — Enumeration runs in < 100 ms for 100 k versions.

## 5 Flows

Iterate Pebble prefix for version index.

## 6 NFR Overrides

None.

## 7 External Interfaces & Data

API — `Tree.Versions`.
Data — compact version index.

## 8 Acceptance Criteria

Integration test populates 10 k versions, enumeration completes and hashes match.

## 9 Open Issues & Risks

Index maintenance overhead.
