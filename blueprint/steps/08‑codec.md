---
id: step.08.codec
depends_on:
  - step.07.node‑iface
tags: [codec, step]
---

## Objective

Provide binary encode/decode for `LeafNode` and `InternalNode`.

## Implements

- **§5 Persistent Storage Requirements S1** (node is persisted under its NodeKey exactly once).
  _What happens_:

  - Binary coders ensure deterministic serialisation so identical nodes in different runs hash to the same byte stream—critical for network proofs and storage hashing.

## Done When ✓

- [ ] Decode(Encode(x)) deep‑equals `x` in tests.
