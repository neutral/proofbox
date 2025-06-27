# Cryptographic Integrity Specification

## Overview
The JMT ensures cryptographic integrity through its Merkle tree structure. See [../../spec/jmt-overview.md](../../spec/jmt-overview.md) for general JMT introduction.

## Cryptographic Properties

### Hash Function
- A secure cryptographic hash function (e.g. SHA-3 or Blake2) is used to compute node digests
- All references to "digest" imply a cryptographic hash output
- The specific hash algorithm must be consistent across all implementations

### Node Digests
Each node (internal or leaf) has a **cryptographic digest** that contributes to the Merkle root:

- **Leaf nodes**: Digest computed as `H(key || H(value))` to incorporate the key and avoid collision attacks
- **Internal nodes**: Digest computed by hashing concatenation of child digests for all 16 children in index order
  - Missing children use a default digest value
  - Deterministic and consistent computation required

### Integrity Guarantees
As an authenticated structure, JMT ensures:
- Any tampering with data can be detected by verifying against the known root hash
- The design assumes the cryptographic hash is secure (preimage-resistant, collision-resistant)
- Negligible chance of malicious collision or preimage
- Strong integrity guarantees for the entire state

### Default Hash Constants
The value of the default placeholder hash for empty subtrees must be decided and agreed upon (often the hash of an empty string or zero). This constant is part of proof verification.

## Security Assumptions
- Cryptographic hash function remains unbroken
- Keys are properly randomized (typically 256-bit hashed addresses)
- Storage layer maintains data integrity