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