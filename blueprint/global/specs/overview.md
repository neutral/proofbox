# Jellyfish Merkle Tree – Technical Specification

## Executive Summary

The Jellyfish Merkle Tree (JMT) is an **authenticated, sparse Merkle tree** optimized for storing versioned state in key-value databases. It was originally developed for Facebook's Diem (Libra) distributed system to balance **space efficiency, computational overhead, and I/O performance**.

The JMT draws inspiration from Ethereum's Patricia Merkle Tree (PMT) but introduces _key innovations_ in **node design, key schema, and proof structure** to better suit a **log-structured merge (LSM) storage engine** (e.g. RocksDB).

In essence, JMT combines a **16-ary Radix Merkle Tree** structure with **sparse Merkle optimizations** and a **versioned node key scheme**. This design enables:

- High throughput updates (minimal compaction overhead on disk)
- Relatively short proof sizes
- The ability to persist multiple historical versions of the state efficiently (persisted snapshots)

By using only two node types (internal and leaf) and eliminating certain complexity (like Ethereum's extension nodes), JMT achieves a simpler implementation without sacrificing performance (see [SN-001], [SN-006]).

## Key Features

1. **High branching factor (16)**: Reduces tree depth (~64 nibble levels for 256-bit keys) compared to binary (~256 levels). This improves read performance (shorter proof paths) at the cost of slightly larger node update overhead.

2. **Two node types only**: JMT uses only _Internal Nodes_ and _Leaf Nodes_. No extension nodes, simplifying implementation and reducing bugs.

3. **Versioned node keys**: Each node is stored/retrieved by a compound key = `(version, nibble_path)`. This enables append-only writes and efficient historic queries.

4. **Persistent structure**: Unchanged parts of the tree are carried over into new versions, while only affected parts are created anew.

5. **Optimized proof format**: JMT Merkle proofs are smaller on average due to sparse optimization and higher branching factor.

## Core Concepts

### Authenticated Key-Value Store (AKVS)

The JMT acts as an authenticated data structure that binds a set of key-value pairs to a single root hash (state commitment). A Merkle tree allows any client to verify that a given key maps to a given value (or is absent) in the committed state by supplying a _Merkle proof_. This fits the distributed system need for cryptographic state verification.

### Addressable Merkle Tree (AMT)

An AMT is a deterministic authenticated structure (usually a Merkle tree) that can store and map arbitrary binary keys to values. Each leaf in an AMT corresponds to a key-value pair, and the path from root to a leaf is determined by the bits of the key. This contrasts with UTXO-style Merkle trees (which index outputs by position) – in an AMT the key itself acts as the address within the tree. (see [SN-002])

### Radix Merkle Tree (AR<sub>r</sub>MT)

A Radix Merkle Tree generalizes the binary Merkle tree to an _r_-ary tree (branching factor _r_ > 2). Instead of each node having 2 children (for each bit), a node can have up to _r_ children, effectively compressing multiple bits of the key into a single tree level.

The JMT specifically uses _r = 16_ (a hexary tree), so each tree level covers 4 bits (a "nibble") of the key. The choice of _r = 16_ in JMT is a compromise balancing read and write costs (see [SN-003]).

### Sparse Merkle Tree (SMT)

A sparse Merkle tree conceptually represents the full binary trie of height _h_ (for _h_-bit keys) but avoids explicitly storing empty subtrees. A sparse Merkle tree optimizes by:

- Treating any _absent_ subtree as a constant default value (with a well-known hash)
- Collapsing any branch that leads to a single leaf

The JMT leverages sparseness: empty subtrees are not stored, and if a branch would have only one leaf, that leaf is linked directly to its highest branching ancestor. (see [SN-004])

### Jellyfish Merkle Tree (JMT)

The JMT can be viewed as a **Sparse Addressable Radix-16 Merkle Tree**. It inherits:

- AMT property (key-determined paths)
- 16-way branching factor (each node covers 4 bits of the key)
- Sparse optimizations (default placeholders and leaf compression)
- **Versioned Node Keys** for append-only storage optimization

This combination achieves effectively _zero compaction_ overhead for new state writes, saving over 90% of I/O operations compared to a hash-keyed approach (see [SN-005], [SN-015]).

## Feature Specifications

The detailed specifications have been organized alongside their corresponding features:

### Data Model Specifications

- [features/storage/get/specs/node-types.md](../../features/storage/get/specs/node-types.md) - Internal and Leaf node specifications
- [features/metadata/versions/specs/versioned-keys.md](../../features/metadata/versions/specs/versioned-keys.md) - NodeKey structure and versioning system

### Operation Specifications

- [features/storage/get/specs/lookup-operation.md](../../features/storage/get/specs/lookup-operation.md) - Query/lookup algorithm
- [features/put-commit/specs/insert-update-operation.md](../../features/put-commit/specs/insert-update-operation.md) - Insert and update operations
- [features/storage/delete/specs/delete-operation.md](../../features/storage/delete/specs/delete-operation.md) - Delete operation (incomplete)

### Proof Specifications

- [features/proof/generate/specs/proof-generation.md](../../features/proof/generate/specs/proof-generation.md) - Proof generation algorithm
- [features/proof/verify/specs/proof-verification.md](../../features/proof/verify/specs/proof-verification.md) - Proof verification process

### Performance & Storage Specifications

- [performance/throughput/specs/lsm-optimization.md](../performance/throughput/specs/lsm-optimization.md) - LSM storage optimization
- [performance/latency/specs/proof-performance.md](../performance/latency/specs/proof-performance.md) - Proof performance optimization
- [storage/efficiency/specs/storage-efficiency.md](../storage/efficiency/specs/storage-efficiency.md) - Storage efficiency design

### System Design Specifications

- [scalability/capacity/specs/scalability-design.md](../scalability/capacity/specs/scalability-design.md) - Scalability architecture
- [reliability/crash-safety/specs/persistence-reliability.md](../reliability/crash-safety/specs/persistence-reliability.md) - Persistence and reliability
- [security/integrity/specs/cryptographic-integrity.md](../security/integrity/specs/cryptographic-integrity.md) - Cryptographic integrity
- [features/metadata/versions/specs/versioning-system.md](../../features/metadata/versions/specs/versioning-system.md) - Versioning system details

## Scope & Assumptions

The complete specification focuses on the core data structure and algorithms of the Jellyfish Merkle Tree as used for an authenticated key-value store in distributed state management. Key assumptions include:

- **Key-Value Domain**: Fixed-length binary keys (typically 256-bit hashes)
- **Cryptographic Hashing**: Secure hash function (e.g. SHA-3 or Blake2)
- **Persistent Versioning**: Append-only, multi-version support
- **Storage Model**: LSM-tree based key-value database
- **Out of Scope**: Network protocols, caching strategies, specialized proofs

## Open Questions & TODOs

- Hash function specification
- Default hash constants
- Complete deletion algorithm
- Storage pruning strategy
- Concurrency and caching
- Proof serialization format

## Reference Mapping

The following table maps key concepts in the specifications to the supporting source snippets from original materials:

| Concept or Feature                     | Relevant Snippet IDs |
| -------------------------------------- | -------------------- |
| Addressable Merkle Tree (AMT)          | [SN-002]             |
| Radix Merkle Tree (branching factor)   | [SN-003]             |
| Sparse Merkle Tree (default/empty)     | [SN-004]             |
| JMT = Sparse AR16 Merkle (definition)  | [SN-005]             |
| Version-based node keys & benefits     | [SN-005], [SN-015]   |
| Two node types (no extension node)     | [SN-006], [SN-014]   |
| Concise proof (fewer siblings)         | [SN-007]             |
| Persistent versioning (delta updates)  | [SN-010]             |
| NodeKey structure (version + path)     | [SN-008], [SN-009]   |
| Internal node structure (children)     | [SN-011]             |
| Leaf node structure (key, blob hash)   | [SN-011]             |
| Insertion logic (new leaf/internal)    | [SN-012]             |
| Update logic (replace leaf value)      | [SN-013]             |
| Removal of extension nodes (rationale) | [SN-014]             |
| LSM-tree optimization (compaction)     | [SN-015]             |

## Source Snippets

**[SN-001]**: _JMT overview from Diem whitepaper (2021) – highlights efficiency and optimization for LSM storage._
"This paper presents Jellyfish Merkle Tree (JMT), a space-and-computation-efficient sparse Merkle tree optimized for Log-Structured Merge-tree (LSM-tree) based key-value storage, which is designed specially for the Diem Blockchain..."

**[SN-002]**: _Addressable Merkle Tree definition by Olshansky (2022) – explains what an AMT is._
"1. An Addressable Merkle Tree (AMT) is a cryptographically authenticated deterministic data structure backed by a key-value store database used for account-based (non-UTXO-based) systems to map keys (i.e. addresses) to arbitrary binary data in each leaf node."

**[SN-003]**: _Addressable Radix Merkle Tree (ARMT) and branching factor trade-offs (Olshansky, 2022)._
"2. An Addressable Radix Merkle Tree (AR₁₆MT) ... is a generalization of an AMT, a binary tree, where r > 2. With a key size of h bits, and each node having at most r children, the height of the tree is logᵣ(2ʰ)..."

**[SN-004]**: _Sparse Merkle Tree optimizations (Olshansky, 2022) – empty subtrees and single-leaf compression._
"3. A Sparse Merkle Tree (SMT) ... The two main optimizations are: Empty Subtrees are replaced with a constant default placeholder node value. Single-Leaf Subtrees are replaced with one node reflecting the one leaf value."

**[SN-005]**: _JMT as Sparse AR16MT with versioned node keys (Olshansky, 2022) – key features summary._
"4. A Jellyfish Merkle Tree (JMT) is a Sparse Addressable Radix Merkle Tree with r=16 (Sparse AR₁₆MT) that balances the tradeoff of storage, compute, read and write operations using two node types: Internal Node and Leaf (Data) Node..."

**[SN-006]**: _"Less Complexity – only two node types" (Diem whitepaper, 2021)._
"• Less Complexity: JMT has only two physical node types, Internal Node and Leaf Node."

**[SN-007]**: _"Concise Proof Format" with fewer siblings on average (Diem whitepaper, 2021)._
"• Concise Proof Format: The number of sibling digests in a JMT proof is less on average (Θ(log(number of existent leaves))) than that of the same ARMT without optimizations..."

**[SN-008]**: _Version-based NodeKey schema (Diem whitepaper, 2021)._
"JMT adopts a version-based node key schema, by splicing version and nibble path, as: version ‖ nibble path..."

**[SN-009]**: _Uniqueness of version+path keys across history (Diem whitepaper, 2021)._
"At any version, a nibble path itself can uniquely identify a node. Therefore, combining version and nibble path could similarly pinpoint any node across versions from the whole blockchain history."

**[SN-010]**: _Persistent data structure & node reuse between versions (Diem whitepaper, 2021)._
"The new tree reuses unchanged portions generated at previous versions, forming a persistent data structure. If an update modifies m out of n leaves, on average O(m ⋅ log n) new nodes are created..."

**[SN-011]**: _Libra Core implementation structures (Rust) for Node, InternalNode, LeafNode, Child._

```rust
pub struct NodeKey {
    version: Version,
    nibble_path: NibblePath,
}
pub enum Node {
    Null,
    Internal(InternalNode),
    Leaf(LeafNode),
}
pub struct LeafNode {
    account_key: HashValue,    // The hashed account address
    blob_hash: HashValue,      // The hash of the account state blob
    blob: AccountStateBlob,    // The account blob data
}
pub struct InternalNode {
    children: HashMap<Nibble, Child>, // Up to 16 children
}
pub struct Child {
    pub hash: HashValue,       // The hash of this child node
    pub version: Version,      // Version when child was created
    pub is_leaf: bool,         // Whether the child is a leaf
}
```

**[SN-012]**: _Insertion logic – internal node case and new leaf creation (Diem whitepaper, 2021)._
"• Internal Node: Create a new leaf node from k and d with its node key as v ‖ current nibble path ‖ n..."

**[SN-013]**: _Update logic – leaf keys match case (Diem whitepaper, 2021)._
"– If the keys match, the insertion becomes an update. We just have to replace the leaf node with its new value and update its node key with new version v."

**[SN-014]**: _Rationale for removing extension nodes (Diem whitepaper, 2021)._
"It is noted that we expressly abandon an extension node, introduced by PMT [Ethereum], for the reasons below..."

**[SN-015]**: _LSM storage optimization – sequential writes and 90% reduction in compaction (Diem whitepaper, 2021)._
"Our experiment shows this schema saves IOPS and disk bandwidth by more than 90% in contrast to hash-based node keys."
