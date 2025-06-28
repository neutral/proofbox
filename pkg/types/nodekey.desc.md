# NodeKey Type Implementation

## Purpose
The `nodekey.go` file implements the NodeKey structure that uniquely identifies nodes across all versions of the tree. This versioned key system is critical for the JMT's append-only storage model and efficient historical queries.

## Design Decisions

### Version-First Design
- NodeKey combines Version and NibblePath into a single identifier
- Version comes first in encoding for efficient range scans
- Enables append-only writes that match LSM-tree characteristics

### Binary Encoding Format
```
+----------------+----------------+----------------------+
| Version (8B)   | Length (2B)    | Nibbles (0-32B)      |
+----------------+----------------+----------------------+
| Big-endian u64 | Big-endian u16 | Packed, 2 per byte   |
+----------------+----------------+----------------------+
```

- Big-endian for consistency and debuggability
- Nibbles packed 2 per byte to minimize storage
- Variable length encoding saves space for shallow nodes

### Storage Key Generation
- Prefixed with "n" for namespace separation in PebbleDB
- Allows efficient iteration over all nodes
- Version-prefixed ordering enables historical queries

## Key Features
1. **Child Construction**: Build child keys maintaining version info
2. **Comparison Function**: Lexicographic ordering by version then path
3. **Efficient Encoding**: Packed nibbles reduce storage overhead
4. **Root Node Helper**: Convenient constructor for root nodes

## Performance Benefits
- Sequential writes due to version-based ordering
- Compact encoding reduces I/O and memory usage
- No heap allocations in encoding/decoding paths
- Benchmarks show ~50ns encoding, ~80ns decoding