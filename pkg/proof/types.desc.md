# types.go

Defines the core proof data structures and types for the Jellyfish Merkle Tree proof system.

## Key Types

- **ProofType**: Enumeration of proof variants (Inclusion, ExclusionEmpty, ExclusionNeighbor)
- **Proof**: Main proof structure containing type, key, value, siblings, and verification data
- **SiblingData**: Stores sibling information at each tree level with full children map for JMT hash computation
- **NeighborLeafData**: Contains neighbor leaf information for exclusion proofs

## Design Decisions

### Why Children Map Instead of Hash Array
The SiblingData structure includes a full `Children` map for all 16 nibbles rather than just non-empty siblings because:
1. JMT hash format requires knowing which nibbles have children (to include nibble prefix)
2. Empty positions need explicit EmptyTreeHash without nibble prefix
3. Simplifies verifier logic by avoiding nibble ordering complexity

### Depth-Based Sibling Storage
Siblings are stored with their depth rather than in a simple array because:
1. Enables sparse proof representation (skip levels with no siblings)
2. Supports future optimization for batch proofs with shared paths
3. Makes debugging easier by explicitly showing tree structure

### Separate Proof Types
Three distinct proof types rather than a single polymorphic type:
1. Type safety ensures correct data is present for each proof variant
2. Simplifies verification logic with type-specific code paths
3. Enables future optimizations specific to each proof type

### Version and RootHash Inclusion
Every proof includes version and expected root hash:
1. Prevents proof replay attacks across different tree versions
2. Enables stateless verification without tree access
3. Supports concurrent proof generation/verification for different versions