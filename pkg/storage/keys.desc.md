# Key Encoder Description

## Purpose

The KeyEncoder interface abstracts the encoding of tree data structures into storage keys. This separation allows:

1. **Backend Optimization**: Different storage engines may prefer different key formats
2. **Migration Support**: Change key formats without modifying tree logic
3. **Testing**: Use simplified encodings for unit tests
4. **Compatibility**: Maintain backward compatibility with existing data

## Key Format Design

### Node Keys: `"n" + version(8) + nibble_count(2) + nibbles`

Node keys encode the full path from root to node:
- **Prefix "n"**: Identifies node keys for iteration
- **Version (8 bytes)**: Enables multi-version storage
- **Nibble count (2 bytes)**: Supports paths up to 512 nibbles (256-bit keys)
- **Packed nibbles**: Two nibbles per byte for space efficiency

This format ensures that nodes from the same version are stored together, improving locality.

### Root Keys: `"r" + version(8)`

Root keys are simple and sequential:
- **Prefix "r"**: Separates roots from other data
- **Version (8 bytes)**: Big-endian for natural ordering

Sequential ordering allows efficient version iteration and pruning.

### Value Keys: `"v" + hash(32)`

Values use content-addressing:
- **Prefix "v"**: Identifies value keys
- **Hash (32 bytes)**: SHA-256 hash of the value

Content-addressing enables automatic deduplication across versions.

### Alternative Value Keys: `"k" + key(32)`

Reserved for future key-based lookups:
- **Prefix "k"**: Different namespace from content-addressed values
- **Key (32 bytes)**: Original tree key

Currently unused but maintains format compatibility.

## Design Decisions

### Why Pack Nibbles?

Nibbles (4-bit values) are packed two per byte to:
1. Reduce storage overhead (50% space saving)
2. Minimize key size for better cache efficiency
3. Maintain readable hex representation in debug output

### Why Big-Endian?

Big-endian encoding ensures:
1. Natural lexicographic ordering matches numeric ordering
2. Efficient range scans for version queries
3. Compatibility with most database iteration APIs

### Why Content-Addressed Values?

Content-addressing provides:
1. Automatic deduplication across versions
2. Integrity verification without additional hashes
3. Simplified garbage collection

## Migration Considerations

When implementing a new KeyEncoder:

1. **Preserve Semantics**: Keys must maintain ordering properties
2. **Version Isolation**: Nodes from different versions must not overlap
3. **Prefix Design**: Choose prefixes that enable efficient iteration
4. **Length Limits**: Respect maximum key sizes of the storage backend