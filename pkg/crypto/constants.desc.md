# Crypto Constants Description

## Overview
This file defines cryptographic constants used throughout the Jellyfish Merkle Tree implementation. Currently, it contains the critical `EmptyTreeHash` constant.

## EmptyTreeHash

```go
var EmptyTreeHash = types.Hash{
    0x5b, 0xa9, 0x3c, 0x9d, 0xb0, 0xcf, 0xf9, 0x3f,
    0x52, 0xb5, 0x21, 0xd7, 0x81, 0x4c, 0x7f, 0xa0,
    0x8a, 0xbe, 0x86, 0x13, 0x5f, 0x74, 0x6b, 0x46,
    0x68, 0xb0, 0x13, 0xbc, 0xd1, 0xc4, 0x8e, 0x9b,
}
```

### Purpose
`EmptyTreeHash` is a **sentinel value** representing empty/non-existent nodes in the sparse Merkle tree structure.

### Why Not Use SHA-256("")?

This is a critical design decision. We intentionally use a different value than `DefaultDigest` (SHA-256 of empty string) because:

1. **Semantic Distinction**
   - `DefaultDigest`: "This leaf contains empty data"
   - `EmptyTreeHash`: "No node exists at this position"

2. **Security**
   - Prevents confusion between storing empty values and non-existent keys
   - Enables proper existence proofs vs. empty value proofs
   - Avoids potential attack vectors from ambiguity

3. **Sparse Tree Optimization**
   - Allows efficient representation of large empty subtrees
   - Can skip computation for known-empty branches

### The Specific Value

The value `5ba93c9d...` appears to be:
- A predetermined constant from the Jellyfish Merkle Tree specification
- Chosen to be unlikely to collide with any real hash output
- Consistent across all implementations for compatibility

### Usage in Tree Operations

```go
// When computing parent hash with missing child:
if leftChild == nil {
    leftHash = EmptyTreeHash
} else {
    leftHash = leftChild.Hash()
}

// In proofs for non-existent keys:
proof.AppendEmptyNode(EmptyTreeHash)
```

## Design Principles

### Why a Package Variable?
- Makes the value discoverable and documented
- Ensures consistent usage across the codebase
- Can be verified in tests against the specification

### Why Hardcoded?
- This value MUST be deterministic across all nodes
- Cannot be computed at runtime (would defeat its purpose)
- Part of the protocol specification

## Future Constants

This file is designed to hold other cryptographic constants as needed:
```go
// Potential future additions:
var (
    // Domain separation tags
    InternalNodeTag = []byte{0x00}
    LeafNodeTag     = []byte{0x01}
    
    // Version-specific salts
    V1Salt = []byte("JMT-V1")
)
```

## Critical Invariant

**EmptyTreeHash must NEVER equal the hash of any actual data**. This is why it's not derived from any hash function but is instead a carefully chosen constant that all implementations must use exactly.