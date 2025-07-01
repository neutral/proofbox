# ProofBox Architecture Overview

This document provides a comprehensive overview of ProofBox's architecture, explaining how components work together to provide a versioned, verifiable key-value store.

## System Architecture

ProofBox is organized in a layered architecture:

```
┌─────────────────────────────────────────┐
│    CLI (cmd/pb)  │  User Applications  │
├─────────────────────────────────────────┤
│           Tree Package                  │  ← Core API
├─────────────────────────────────────────┤
│    Proof    │   Types   │   Crypto     │  ← Core Support
├─────────────────────────────────────────┤
│    Codec    │   Metrics │   Fuzz       │  ← Additional
├─────────────────────────────────────────┤
│         Storage Abstraction             │
├─────────────────────────────────────────┤
│    PebbleDB    │    Memory Store       │  ← Backends
└─────────────────────────────────────────┘
```

## Core Components

### 1. Tree Package (`pkg/tree`)
The heart of ProofBox, implementing the Jellyfish Merkle Tree:

- **Tree**: Main structure managing versions and operations
- **Nodes**: Internal nodes (branching) and leaf nodes (data)
- **Version Manager**: Tracks multiple versions and their relationships
- **Reader/Writer**: Separates read and write operations

Key design decisions:
- Single writer, multiple readers (SWMR) concurrency model
- Copy-on-write for version isolation
- Lazy loading for memory efficiency

### 2. Storage Layer (`pkg/storage`)
Abstracts persistent storage with a comprehensive interface:

```go
type Storage interface {
    Get(key []byte) ([]byte, error)
    Put(key []byte, value []byte) error
    Delete(key []byte) error
    NewBatch() Batch
    NewSnapshot() Snapshot
    NewIterator(opts *IteratorOptions) Iterator
    Metrics() Metrics
    Close() error
}
```

Implementations:
- **PebbleDB**: Production storage backend (LSM-tree based)
- **Memory**: For testing and temporary usage

### 3. Proof System (`pkg/proof`)
Generates and verifies cryptographic proofs:

- **Generator**: Creates inclusion/exclusion proofs
- **Verifier**: Validates proofs independently
- **Optimization**: Minimizes proof size while maintaining security

### 4. Codec (`pkg/codec`)
Handles serialization/deserialization:

- Binary encoding for space efficiency
- Type-safe node encoding/decoding
- Batch operations support
- Forward compatibility considerations

### 5. Types (`pkg/types`)
Common types and utilities:

- `Key`: 256-bit keys with nibble operations
- `Hash`: SHA-256 based hashing
- `NodeKey`: Composite key for storage
- Error types and interfaces

### 6. Crypto (`pkg/crypto`)
Cryptographic operations:

- SHA-256 hashing implementation
- Empty tree hash constants
- Hasher interface for extensibility

## Data Flow

### Write Operation
```
1. User: put(key, value)
2. Tree: Create new version
3. Tree: Copy-on-write path from root to leaf
4. Codec: Serialize modified nodes
5. Storage: Batch write to PebbleDB
6. Tree: Update root hash for new version
7. Return: New version number
```

### Read Operation
```
1. User: get(key, version)
2. Tree: Look up root hash for version
3. Storage: Create snapshot
4. Tree: Traverse from root following key path
5. Storage: Load nodes as needed
6. Return: Value (if found)
```

### Proof Generation
```
1. User: prove(key, version)
2. Tree: Create reader for version
3. Proof: Traverse tree collecting sibling hashes
4. Proof: Package proof data
5. Return: Compact proof structure
```

## Concurrency Model

ProofBox uses a carefully designed concurrency model:

### Lock Hierarchy
To prevent deadlocks, locks must be acquired in this order:
1. Tree lock (for version management)
2. Storage locks (for node access)
3. Node locks (if needed)

### Read Concurrency
- Multiple readers can operate simultaneously
- Readers use storage snapshots for consistency
- No blocking between readers

### Write Serialization
- Only one writer at a time
- Writers don't block readers
- Batch operations are atomic

## Storage Design

### Key Structure
Storage keys encode version and path:
```
[version:8 bytes][path:variable]
```

This enables:
- Efficient version-based queries
- Natural ordering for range scans
- Easy garbage collection of old versions

### Value Storage
Two strategies supported:
1. **Inline**: Small values stored with nodes
2. **Separate**: Large values stored separately

### Optimization for SSDs
- Append-only writes reduce write amplification
- Batch operations minimize I/O operations
- LSM-tree structure aligns with SSD characteristics

## Memory Management

### Node Cache
- LRU cache for frequently accessed nodes
- Configurable size limits
- Automatic eviction under pressure

### Version Management
- Keep recent versions in memory
- Lazy loading of historical versions
- Garbage collection for old versions

## Security Architecture

### Cryptographic Guarantees
- SHA-256 for all hashing operations
- Merkle tree structure ensures tamper detection
- Root hash represents entire state

### Trust Model
- Trust the root hash source
- No trust needed for proof verification
- Storage corruption detected via hash validation

## Performance Characteristics

### Time Complexity
- Get/Put: O(log n) where n = number of keys
- Proof generation: O(log n)
- Proof verification: O(log n)
- Batch operations: Amortized O(log n) per operation

### Space Complexity
- Tree nodes: O(n)
- Proof size: O(log n)
- Version overhead: O(m × k) where m = modified nodes, k = versions

## Extension Points

### Custom Storage Backends
Implement the `Storage` interface for:
- Cloud storage (S3, GCS)
- Distributed storage
- Encrypted storage

### Alternative Hash Functions
While SHA-256 is standard, the design supports:
- Blake2b for performance
- SHA-3 for future-proofing
- Custom domain-specific hashes

### Metrics and Monitoring
Pluggable metrics system supports:
- Prometheus integration
- Custom metric collectors
- Performance profiling

## Design Principles

1. **Simplicity**: Prefer simple, correct solutions
2. **Performance**: Optimize for SSD storage patterns
3. **Correctness**: Comprehensive testing and verification
4. **Extensibility**: Clean interfaces for customization
5. **Security**: Cryptographic integrity by default

## Future Considerations

Potential enhancements maintaining current architecture:
- Sharding for horizontal scaling
- Compression for space efficiency
- Parallel proof generation
- Historical version pruning
- Remote proof verification service

---

For implementation details, see the [Design Decisions](design-decisions.md) document. For performance characteristics, see [Performance](performance.md).