# Jellyfish Merkle Tree Explained

The Jellyfish Merkle Tree (JMT) is the core data structure that powers ProofBox. This document explains what it is, how it works, and why it's ideal for building versioned, verifiable key-value stores.

## What is a Jellyfish Merkle Tree?

The Jellyfish Merkle Tree is a **sparse, versioned Merkle tree** optimized for key-value storage with cryptographic proofs. Originally developed for the Diem (formerly Libra) blockchain, it combines the benefits of:

- **Merkle Trees**: Cryptographic data structures that provide integrity proofs
- **Sparse Trees**: Efficient storage for large key spaces with few entries
- **Persistent Data Structures**: Version history with structural sharing
- **LSM-tree Optimization**: Append-only writes for modern storage engines

The name "Jellyfish" comes from its branching structure - like a jellyfish with many tentacles reaching down from the root.

## Core Concepts

### 256-bit Key Space

JMT uses 256-bit (32-byte) keys, providing an enormous address space:
- 2²⁵⁶ possible keys (more than atoms in the observable universe)
- Keys are typically hashes of actual data, ensuring uniform distribution
- Each key maps to a unique path through the tree

### Nibble-based Navigation

Keys are processed as **nibbles** (4-bit values) rather than bits:
- Each byte becomes 2 nibbles (0x0 to 0xF)
- A 256-bit key = 64 nibbles = maximum tree depth of 64
- This reduces tree height compared to binary trees (64 vs 256 levels)

### 16-way Branching (Hexary Tree)

Internal nodes can have up to 16 children:
```
         Root
      /   |   \
    0x0  0x5  0xF     ← First nibble determines branch
     |    |    |
   0x3   0x7  0xA     ← Second nibble
    ...  ...  ...
```

This branching factor is optimal because:
- Reduces proof size (fewer levels to traverse)
- Matches nibble size (4 bits = 16 values)
- Balances between tree height and node size

## Tree Structure

### Two Node Types

ProofBox's JMT uses only two node types:

#### 1. Internal Nodes
```
InternalNode {
    children: {
        0x0: (hash₀, version₀),
        0x5: (hash₅, version₅),
        0xF: (hash₁₅, version₁₅)
    }
}
```
- Store up to 16 child references
- Only non-empty children are stored (sparse)
- Each child reference includes its hash and version

#### 2. Leaf Nodes
```
LeafNode {
    key: [32]byte,
    valueHash: [32]byte,
    value: []byte (optional)
}
```
- Terminal nodes containing actual data
- Store the complete key (for verification)
- May store value inline or just its hash

### Hash Computation

Each node's hash uniquely identifies its content:

**Internal Node Hash**:
```
hash = SHA256(NodeTypeInternal || child₀ || child₁ || ... || child₁₅)
```
Where empty children use a predefined empty hash.

**Leaf Node Hash**:
```
hash = SHA256(NodeTypeLeaf || key || valueHash)
```

This ensures any change bubbles up to the root, providing tamper detection.

## Versioning and Persistence

### Append-Only Storage

JMT is designed for append-only storage:
- Each update creates a new version
- Nodes are never modified, only copied (copy-on-write)
- Storage key format: `(version, nibblePath)`

### Structural Sharing

Unchanged subtrees are shared between versions:
```
Version 1:        Version 2 (after updating key 0x5...):
    Root₁             Root₂
   /  |  \           /  |  \
  A   B   C        A   B'  C
      |                |
      D                D'
```
Only modified nodes (B' and D') are newly created.

## Proof Generation and Verification

### Inclusion Proofs

To prove key K exists with value V:
1. Traverse from root to leaf following K's nibbles
2. Collect sibling hashes at each level
3. Package: leaf data + sibling hashes + root hash

### Exclusion Proofs

To prove key K doesn't exist:
1. **Empty Path**: Show the path leads to an empty child
2. **Different Leaf**: Show the path leads to a leaf with different key

### Proof Size

For a tree with n entries:
- Average proof size: O(log₁₆ n) hash values
- Maximum proof size: 64 siblings (one per level)
- Typical proof: 1-2 KB for trees with millions of entries

## Why Jellyfish Merkle Tree?

### Advantages over Binary Merkle Trees

1. **Smaller Proofs**: 4x fewer levels due to 16-way branching
2. **Better Cache Locality**: Nodes pack more information
3. **Nibble Alignment**: Natural for hexadecimal operations

### Advantages over Ethereum's MPT

1. **Simpler Design**: Only 2 node types vs 3+
2. **No RLP Encoding**: Cleaner serialization
3. **Version Native**: Built-in version support

### Advantages over Regular Hash Maps

1. **Cryptographic Proofs**: Verify data without full database
2. **Historical Queries**: Access any past version
3. **Tamper Detection**: Any modification changes root hash

## Performance Characteristics

### Time Complexity
- **Get/Put**: O(log₁₆ n) ≈ O(log n)
- **Proof Generation**: O(log₁₆ n)
- **Proof Verification**: O(log₁₆ n)

### Space Complexity
- **Storage**: O(n × m) where m = number of versions
- **Memory**: O(log₁₆ n) for operations
- **Proof Size**: O(log₁₆ n) hashes

### Real-World Performance
- Trees with 1M entries: ~5 levels deep
- Trees with 1B entries: ~7-8 levels deep
- Proof size remains practical even for massive trees

## Design Trade-offs

### Why 16-way Branching?

The choice of radix-16 balances several factors:
- **Proof Size**: More branches = shallower tree = smaller proofs
- **Node Size**: Fewer branches = smaller nodes = less memory
- **CPU Cache**: 16 children fit well in modern CPU caches
- **Nibble Natural**: 4-bit nibbles map directly to 16 branches

### Why No Extension Nodes?

Unlike Ethereum's Patricia Merkle Tree, JMT omits extension nodes:
- **Simplicity**: Fewer node types = fewer bugs
- **Uniformity**: All operations follow same pattern
- **Trade-off**: Slightly deeper trees for sequential keys

### Why 256-bit Keys?

The 256-bit key size:
- Matches common hash outputs (SHA-256, Keccak-256)
- Provides collision resistance for hash-based keys
- Enables uniform key distribution
- Supports cryptographic commitments

## Implementation in ProofBox

ProofBox implements JMT with additional optimizations:

1. **Thread-Safe Operations**: Concurrent reads with single writer
2. **Lazy Value Loading**: Values loaded only when needed
3. **Cached Hashes**: Nodes cache computed hashes
4. **Batch Operations**: Amortize tree traversal costs
5. **Storage Abstraction**: Works with any key-value backend

## Summary

The Jellyfish Merkle Tree provides an elegant solution for versioned, verifiable storage:

- **Sparse**: Efficiently handles large key spaces
- **Versioned**: Natural support for historical queries
- **Verifiable**: Compact proofs of inclusion/exclusion
- **Optimized**: Designed for modern storage engines
- **Simple**: Only two node types to understand

These properties make JMT ideal for blockchain state storage, audit logs, configuration management, and any system requiring cryptographic integrity with version history.