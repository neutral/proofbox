---
id: req.storage.put‑commit
depends_on:
  - goal.proofbox
  - nfr.reliability.crash‑safety
  - nfr.performance.throughput
tags: [api, storage]
---

## 1 Purpose & Value

As a **state manager** I want to call `Put(key,value)` repeatedly and then `Commit(version)` so that all modified entities persist atomically and I get the new root hash.

## 2 Scope

**In scope** — staging key‑value pairs, atomic batch write, root computation.
**Out of scope** — proof generation (handled in another feature).

## 3 Actors & Preconditions

Actor: execution engine.
Precondition: Tree opened successfully, `key` = 32 bytes.

## 4 Functional Requirements

FR‑PUT‑1 — System **SHALL** accept `Put` for any 32‑byte key.
FR‑PUT‑2 — System **SHALL** detect duplicate keys within the same stage and keep the last.
FR‑COMMIT‑1 — System **SHALL** persist all staged changes with one Pebble batch.
FR‑COMMIT‑2 — System **SHALL** return the root hash that re‑hashes to the same value in unit tests.

## 5 Flows

**Main**

1. Executor calls `Put` N times → leaf nodes staged.
2. Executor calls `Commit(version)` → batch write, root returned.

## 6 NFR Overrides

None (global NFRs apply).

## 7 External Interfaces & Data

_API_ — `Tree.Put`, `Tree.Commit` in `pkg/jmt`.
_Data_ — staged node set (in‑memory), Pebble batch on commit.

## 8 Acceptance Criteria

GIVEN a fresh Tree
WHEN I `Put` 3 keys and `Commit(42)`
THEN `Commit` returns a non‑zero root and Pebble contains exactly the new nodes at version 42.

## 9 Open Issues & Risks

None.
