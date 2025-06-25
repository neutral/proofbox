---
id: goal.proofbox
tags: [business, product]
---

# Goals

### HLG-1 – Provably Correct KV Store

Provide a provably‑correct key‑value store with a 256‑bit root hash.

**Definition of Done**

- ≥ 10 k random inclusion / non‑inclusion proofs verify against the published root.

### HLG-2 – Historic Version Support

Support historic versions at transaction cadence using append-only storage semantics.

**Definition of Done**

- Any root ≤ 10 k versions old reconstructs
- Diff scan < 1 µs/key.

### HLG-3 – Persistent State

Persistent state

**Definition of Done**

- Shutdown and boot up KV store from storage.

### HLG-4 – Proofs

Create concise, easy-to-verify Merkle proofs.

**Definition of Done**

- ...

### HLG-5 – I/O Minimisation on Commodity SSDs

Minimise I/O on commodity SSDs during bursts of 1 k – 10 k writes.

**Definition of Done**

- < 10 MB/s sustained write
- Pebble compaction < 5 % of wall time.
