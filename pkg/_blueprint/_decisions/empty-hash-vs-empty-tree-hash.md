---
id: adr.empty-hash-design
status: accepted
date: 2024-01-15
---

# ADR: EmptyHash vs EmptyTreeHash Design

## Status

Accepted

## Context

In the JMT, we need to distinguish between:
1. A missing/empty child slot in an internal node
2. An all-zero hash value

The Diem JMT implementation uses two different constants:
- `EMPTY_HASH`: 32 zero bytes
- `EMPTY_TREE_HASH`: A specific hash value for empty subtrees

## Decision

Follow the Diem implementation:
- Use `types.EmptyHash()` for zero hash values
- Use `crypto.EmptyTreeHash` as a sentinel for missing children
- Internal nodes use `EmptyTreeHash` when computing hash for missing children

## Consequences

### Positive
- **Compatibility**: Matches reference implementation
- **Distinction**: Can differentiate between zero and missing
- **Deterministic**: Empty slots always hash the same way
- **Type Safety**: Different types prevent confusion

### Negative
- **Two Concepts**: Must understand both EmptyHash and EmptyTreeHash
- **Documentation**: Requires clear explanation of the difference
- **Potential Confusion**: Developers might use the wrong one

### Neutral
- Similar pattern in other Merkle tree implementations
- Extra constant but clearer semantics

## Implementation Details

```go
// pkg/types/hash.go
func EmptyHash() Hash {
    return Hash{} // All zeros
}

// pkg/crypto/constants.go
var EmptyTreeHash = computeEmptyTreeHash()

func computeEmptyTreeHash() types.Hash {
    hasher := sha256.New()
    hasher.Write([]byte("EMPTY_TREE"))
    var hash types.Hash
    copy(hash[:], hasher.Sum(nil))
    return hash
}

// Usage in InternalNode
for nibble := 0; nibble < 16; nibble++ {
    if child, exists := children[nibble]; exists {
        // Use actual child hash
        hasher.Write(child.Hash[:])
    } else {
        // Use EmptyTreeHash for missing children
        hasher.Write(crypto.EmptyTreeHash[:])
    }
}
```

## Alternatives Considered

1. **Only Zero Hash**
   - Use Hash{} for both purposes
   - Rejected: Can't distinguish missing from actual zero

2. **Nil for Missing**
   - Use *Hash with nil for missing
   - Rejected: Requires nil checks everywhere

3. **Skip Missing Children**
   - Don't include missing children in hash
   - Rejected: Different child patterns would hash the same

4. **Special Nibble Value**
   - Use nibble 0xFF for "missing"
   - Rejected: Reduces valid nibble space

## Notes

This decision was discovered during implementation when reviewing the Diem codebase. The distinction is important for:
- Proof verification
- Hash computation correctness
- Compatibility with other implementations

## References

- Diem JMT source code
- Initial confusion resolved in step 05 implementation
- crypto/constants.go implementation