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

## Current Limitations (Step 09)

- Only supports empty tree insertion and single-key updates
- Leaf splitting not implemented
- Internal node updates not implemented
- These will be added in subsequent implementation steps

## Thread Safety

TreeUpdater instances are not thread-safe and should only be used within the context of a single Put operation, which is already serialized by the Tree's writeMu mutex.