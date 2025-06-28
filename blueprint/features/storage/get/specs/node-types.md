# Node Types Specification

## Overview
The JMT defines exactly two concrete node types (excluding a trivial `Null` placeholder for an empty tree). This simplicity is a key design feature (see [SN-006]).

## Internal Node

An internal node represents a branching point in the trie and can have up to 16 children (one for each possible nibble 0x0 through 0xF). It is effectively a radix-16 node that compresses 4 levels of a binary trie into one level.

### Storage Structure
An internal node in storage holds an array or map of child pointers, where each _occupied child slot_ contains:
- The child's **hash digest**
- The **version** at which that child node was created
- A flag indicating if the child is a **leaf or internal node**

Any _unoccupied child slot_ is considered an empty subtree (implicitly represents the default hash).

### Implementation Example
From the Libra Core implementation (see [SN-011]):
```rust
pub struct InternalNode {
    children: HashMap<Nibble, Child>, // Up to 16 children
}

pub struct Child {
    pub hash: HashValue,       // The hash of this child node
    pub version: Version,      // Version when child was created
    pub is_leaf: bool,         // Whether the child is a leaf
}
```

## Leaf Node

A leaf node contains the actual key-value mapping. It is the terminal node for a key's path.

### Storage Structure
A leaf node holds:
- The **key** (full length, typically 256-bit hashed address)
- The **value hash** `H(value)`
- Optionally the actual **value data** or a reference to it

### Implementation Example
From the Libra Core implementation:
```rust
pub struct LeafNode {
    account_key: HashValue,    // The hashed account address
    blob_hash: HashValue,      // The hash of the account state blob
    blob: AccountStateBlob,    // The account blob data
}
```

### Position Determination
The leaf node's position in the tree is determined by the key's bits. By definition, a leaf node has no children.

## Node Enum
The complete node type is typically represented as:
```rust
pub enum Node {
    Null,
    Internal(InternalNode),
    Leaf(LeafNode),
}
```

## Design Rationale
The absence of extension nodes (unlike Ethereum's PMT) was deliberate (see [SN-014]):
- In large sparse trees, the chance of two leaves sharing long common prefixes is rare
- Removing extension nodes contributes to less complicated code and fewer potential bugs
- The simplicity outweighs any minor space savings from extension nodes

## Implementation Requirements

### Thread Safety
All node types must be thread-safe for concurrent access:
- Use `sync.RWMutex` for protecting mutable state (cached hashes, children map)
- Hash computation must be deterministic regardless of concurrent access
- Child modifications must atomically invalidate parent hash caches
- Read operations should not block each other (use RLock)

### Memory Optimization

#### Sparse Representation
- Internal nodes use `map[Nibble]Child` instead of fixed arrays
- Empty children are not stored, saving memory in sparse trees
- Hash computation uses EmptyTreeHash for missing children

#### Hash Caching
- Computed hashes are cached to avoid recomputation
- Cache invalidation on any mutation ensures consistency
- Thread-safe access to cached values

#### Lazy Value Loading
- Leaf nodes support nil values with separate value storage
- Values loaded on-demand via SetValue() with hash verification
- See [Lazy Loading Specification](./lazy-loading.md) for details

### Deterministic Behavior

#### Hash Computation
- Internal nodes process children in nibble order (0x0 to 0xF)
- Hash format: `NodeType || child0 || child1 || ... || child15`
- Empty children contribute EmptyTreeHash to maintain fixed size

#### Cloning for Versioning
- Clone() creates new node instance with updated version
- Structural sharing: cloned nodes share immutable child references
- Enables efficient copy-on-write semantics

### Size Limits
- Maximum value size: 1MB (MaxValueSize = 1 << 20)
- Enforced at leaf node creation
- Prevents DoS attacks via large value storage