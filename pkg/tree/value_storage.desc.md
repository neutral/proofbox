# Value Storage Design

This document clarifies the value storage approach used in the Jellyfish Merkle Tree implementation.

## Overview

Values in the JMT are stored using **content-addressed storage (CAS) within PebbleDB**. This is a critical distinction - we use CAS as a key naming strategy, not as an external service.

## What This Means

### What We're Doing ✅
```go
// 1. Compute hash of value (required for Merkle tree)
valueHash := sha256.Sum256(value)

// 2. Use hash as part of the key in PebbleDB
storageKey := []byte{'v'} + version + valueHash[:]

// 3. Store in the same PebbleDB instance
pebbleDB.Set(storageKey, value)
```

### What We're NOT Doing ❌
- NOT using IPFS
- NOT using external CAS services
- NOT using separate databases
- NOT making network calls
- NOT adding external dependencies

## Why Content-Addressed Storage?

In a Merkle tree, every lookup must follow a verifiable path:

```
Root Hash → Internal Nodes → Leaf Node → Value Hash → Value
```

The leaf node contains a `valueHash`. To maintain cryptographic verification:
1. We must fetch the value using this hash
2. We must verify the fetched value matches the hash
3. This proves the value hasn't been tampered with

### Why Not Direct Key-Based Storage?

```go
// This would break verification:
storageKey := []byte{'v'} + originalKey
pebbleDB.Set(storageKey, value)

// Why? The leaf contains valueHash, not originalKey
// No way to verify the retrieved value is correct!
```

## Storage Layout in PebbleDB

All data is in a single PebbleDB instance with different key prefixes:

```
PebbleDB Database:
├── 'n' + ...  → Tree nodes (internal & leaf nodes)
├── 'r' + ...  → Root hashes by version  
├── 'v' + ...  → Values by content hash
└── other prefixes for future use
```

## Implementation Details

### Value Storage Key Format
```
'v' || hash (32 bytes)
```

### Why NOT Include Version?
- Enables deduplication across versions (same value stored only once)
- Reduces storage for repeated values
- Simplifies garbage collection (ref counting or generation-based)
- Hash collisions are cryptographically improbable with SHA-256

### Retrieval Path
```go
// 1. Tree traversal finds leaf node
leaf := findLeaf(key)

// 2. Leaf contains value hash
valueHash := leaf.ValueHash()

// 3. Retrieve value using hash
storageKey := makeValueKey(version, valueHash)
value := pebbleDB.Get(storageKey)

// 4. Verify integrity (optional but recommended)
if sha256.Sum256(value) != valueHash {
    return ErrCorruptedValue
}
```

## Benefits

1. **Cryptographic Verification**: Required for Merkle tree security
2. **Tamper Detection**: Can't return wrong value without detection
3. **Proof Support**: Can prove value exists without revealing it
4. **Local Storage**: Everything in PebbleDB, no external dependencies
5. **Performance**: Sequential key layout good for LSM trees

## Common Misconceptions

### "CAS means IPFS/External Service"
No! CAS is just a technique where content is addressed by its hash. Git uses CAS locally, we use CAS in PebbleDB.

### "Two-tier means two databases"
No! Two-tier refers to logical separation:
- Tier 1: Tree structure (nodes)
- Tier 2: Value data
Both tiers are in the same PebbleDB with different key prefixes.

### "This is overkill for local storage"
No! This is the minimum required for a secure Merkle tree. Without hash-based retrieval, you can't verify values.

## Future Considerations

- Could use PebbleDB column families for better separation
- Could move values to different backend (S3, etc) without changing interface
- Could add compression (store compressed, hash uncompressed)
- Could remove version from key for cross-version deduplication

But for now, this simple approach in PebbleDB is correct, secure, and efficient.