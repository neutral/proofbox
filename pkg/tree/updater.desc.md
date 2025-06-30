# TreeUpdater

The TreeUpdater struct handles the logic for modifying the Jellyfish Merkle Tree during insert and update operations.

## Purpose

TreeUpdater encapsulates the complex logic of tree modifications, managing:
- Version transitions
- Node creation and updates
- Batch operations for atomic commits
- Value storage with content-addressed hashing

## Key Design Decisions

1. **Separation of Concerns**: TreeUpdater is separate from the main Tree struct to isolate mutation logic and make it easier to test.

2. **Write Buffering**: All node modifications are buffered in memory (`nodeWrites` map) before being written to the batch, allowing for validation and rollback if needed.

3. **Content-Addressed Value Storage**: Values are stored in PebbleDB using their SHA-256 hash as part of the key. This is NOT an external CAS like IPFS, but rather a key naming strategy within PebbleDB itself:
   ```go
   // Compute hash for verification
   valueHash := sha256.Sum256(value)
   
   // Use hash as part of PebbleDB key
   key := 'v' || version || valueHash
   pebbleDB.Set(key, value)
   ```
   
   This approach is essential for Merkle trees because:
   - **Verification**: The leaf node stores the value hash, allowing cryptographic verification during retrieval
   - **Integrity**: Can't retrieve a tampered value (hash wouldn't match)
   - **Proofs**: Can prove a value exists without revealing it
   
   All storage remains in the same PebbleDB instance - we're just using content-addressing as the key naming strategy for the verifiable lookup pathway required by Merkle trees.

4. **Version Immutability**: Each operation creates new nodes at the new version, preserving historical state.

## Single-Child Chains (Important Design Decision)

When inserting keys with long common prefixes, this implementation creates chains of single-child internal nodes. This is **intentional behavior** following the JMT specification, not a bug or inefficiency.

Example: Two keys differing only in their last nibble will create 63 internal nodes, each with one child, before the final divergence node with two children.

This design choice prioritizes:
- Implementation simplicity
- Specification compliance  
- Correctness over optimization


### Why This is Acceptable

1. **JMT Specification**: Explicitly rejects extension nodes for simplicity
2. **Random Keys**: In blockchain use cases, keys are hashes with uniform distribution
3. **Rare in Practice**: Long common prefixes are statistically improbable with hash-based keys
4. **Trade-off**: Simplicity and correctness outweigh the inefficiency in edge cases

## Known Limitations

1. **Deep Trees**: Keys with very long common prefixes can create trees approaching MaxTreeDepth (64 levels)
2. **Memory Usage**: Single-child chains use more memory than strictly necessary
3. **No Path Compression**: Unlike Patricia tries, we don't compress single-child paths

These are acceptable trade-offs per the JMT specification's philosophy of "simplicity over optimization".

## Thread Safety

TreeUpdater instances are not thread-safe and should only be used within the context of a single Put operation, which is already serialized by the Tree's writeMu mutex.