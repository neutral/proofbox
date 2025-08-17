---
id: req.cli.basic
depends_on:
  - req.storage.put‑commit
  - req.storage.get
  - req.proof.generate
  - req.proof.verify
tags: [cli]
---

## 1 Purpose & Value

As a **dev‑ops engineer** I need `jmtcli` commands to smoke test and automate maintenance scripts.

## 2 Scope

Sub‑commands: put, get, root, prove, verify.

## 3 Actors & Preconditions

Actor: shell script.
Preconditions: `jmtcli` installed via `go install ./cmd/jmtcli`.

## 4 Functional Requirements

FR‑CLI‑1 — Each command returns exit code 0 on success, non‑0 on error.
FR‑CLI‑2 — `--help` lists flags via Cobra auto‑gen.
FR‑CLI‑3 — JSON output option for scripting.

## 5 Flows

Standard Cobra CLI flow.

## 6 NFR Overrides

None.

## 7 External Interfaces & Data

Binary **jmtcli**.

## 8 Acceptance Criteria

CI executes smoke test script covering all sub‑commands.

## 9 Open Issues & Risks

Cross‑platform packaging (brew, scoop) left for operators.
