---
id: req.storage.get
depends_on:
  - goal.proofbox
  - nfr.performance.latency
tags: [api, storage]
---

## 1 Purpose & Value

As a **full node** I want to call `Get(version,key)` so that I can retrieve state at any historical block quickly.

## 2 Scope

**In scope** — point lookups, error handling for unknown versions.
**Out of scope** — range scans (future work).

## 3 Actors & Preconditions

Actor: state synchroniser.
Precondition: version exists; key = 32 bytes.

## 4 Functional Requirements

FR‑GET‑1 — System **SHALL** return `(value,ok=true)` if key present in version.
FR‑GET‑2 — System **SHALL** return `ok=false` when key absent.
FR‑GET‑3 — System **SHALL** return `ErrNoVersion` if version unknown.
FR‑GET‑4 — p99 latency ≤ 1 ms on cache‑warm path.

## 5 Flows

1. Caller invokes `Get`.
2. Tree descends at most 64 levels issuing Pebble Get per level.
3. Return result.

## 6 NFR Overrides

None.

## 7 External Interfaces & Data

API — `Tree.Get`.
Data — pebble.DB, internal node cache.

## 8 Acceptance Criteria

GIVEN a committed version
WHEN I `Get` an existing key
THEN the value matches the one used during Commit and completes in < 1 ms.

## 9 Open Issues & Risks

Potential optimisation: hot internal nodes cache.
