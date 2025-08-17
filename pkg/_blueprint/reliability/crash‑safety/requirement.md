---
id: nfr.reliability.crash‑safety
tags: [reliability]
---

## Description

A power loss or process crash must never corrupt committed state.

## Default Target

Metric: After a forced kill during Commit(), DB re‑opens cleanly and verifies with zero hash mismatches.

## Measurement

CI step: start Commit, `kill -9`, restart, run `jmtcli root`; expect previous root.

## Rationale

Distributed systems cannot afford state forks caused by storage corruption.
