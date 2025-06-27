---
id: step.02.hasher
depends_on:
  - step.01.project‑scaffold
tags: [crypto, step]
---

## Objective

Add `Hasher` interface plus `DefaultSHA256` and constant `DefaultDigest`.

## Implements

- **Glossary – Digest**, **§3 Conventions & Parameters** (`HASH_LEN`, _Hash Function_), and **Security Requirement R1** (128‑bit+ collision resistance).
  _What happens_:

  - Define the `Hasher` abstraction so that later steps can swap SHA‑256 for another algorithm without touching the tree code ("Hash Function is assumed but not fixed", original text).
  - Provide a constant `DefaultDigest` = `H("")`, aligning with **§3 Default Digest** and the "Sparse Merkle default hash" discussion (§Sparse Merkle Tree concept).

## Done When ✓

- [ ] Unit test asserts `DefaultDigest == sha256.Sum256(nil)`.
