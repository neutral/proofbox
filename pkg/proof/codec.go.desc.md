# codec.go

Provides efficient binary serialization and deserialization for Merkle proofs, optimized for network transmission and storage.

## Key Components

- **ProofCodec**: Main codec for proof serialization/deserialization
- **Encode/Decode**: Converts between Proof structs and byte arrays
- **Magic bytes**: Uses 0x70, 0x72 ('pr') for format identification

## Serialization Format

1. **Header**: Magic bytes + version (3 bytes)
2. **Common fields**: Type, key, version, root hash
3. **Type-specific data**:
   - Inclusion: Value length + value bytes
   - Neighbor: Neighbor key, value hash, diverge depth, path
   - Empty: No additional data
4. **Siblings**: Count + array of (depth, nibble, children map)

## Design Decisions

- Variable-length encoding for space efficiency
- Size validation (MaxProofSize = 64KB) prevents DoS attacks
- Binary format chosen over JSON for ~3x size reduction
- Forward-compatible versioning for future extensions