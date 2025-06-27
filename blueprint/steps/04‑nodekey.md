---
id: step.04.nodekey
depends_on:
  - step.03.key‑util
tags: [structs, step]
---

## Objective

Implement `NodeKey` `(Version uint64, Path []byte)` with `Encode/Decode`.

## Implements

- **§4.3 NodeKey Encoding** (version first, path bytes second)
- **§5 Persistent Storage Requirements S1/S2**
  _What happens_:

  - Encoders guarantee ordering so that "nodes of higher version sort lexicographically after lower versions," enabling append‑only LSM writes (Non‑functional goals → "LSM‑friendly writes").

## Done When ✓

- [ ] Round‑trip encode/decode test succeeds.
