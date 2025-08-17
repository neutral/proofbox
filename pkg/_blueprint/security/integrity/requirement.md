---
id: nfr.security.integrity
tags: [security]
---

## Description

All untrusted data (proofs, keys, values) must be authenticated before use.

## Default Target

Metric: 100 % of externally supplied hashes re‑computed and constant‑time compared.

## Measurement

Static analysis (`gosec`) plus unit tests injecting malformed proofs must fail with `ErrIntegrity`.

## Rationale

Protects against state‑root forgery and timing side‑channel attacks.
