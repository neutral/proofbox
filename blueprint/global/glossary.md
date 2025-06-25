---
id: global.glossary
tags: [reference]
---

# Glossary

| Term              | Definition                                                                            |
| ----------------- | ------------------------------------------------------------------------------------- |
| JMT               | **Jellyfish Merkle Tree** — 16‑way sparse Merkle trie used in Move‑based blockchains. |
| Version           | Monotonically increasing `uint64` identifying each committed state.                   |
| Root              | 32‑byte SHA‑256 digest committing to the entire tree at a given version.              |
| Nibble            | 4‑bit value (0‑15); two packed per byte in path encodings.                            |
| Pebble            | Go‑native LSM storage engine compatible with RocksDB on‑disk format.                  |
| Proof (inclusion) | A set of sibling hashes that reconstructs the path root for a present key.            |
| Proof (non‑incl.) | Same, but accompanied by the nearest existing leaf showing absence.                   |
| DefaultHash       | `SHA256([]byte{})` — constant hash of the empty string.                               |
