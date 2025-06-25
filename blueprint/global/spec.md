# Jellyfish Merkle Tree – Technical Specification

## Executive Summary

The Jellyfish Merkle Tree (JMT) is an **authenticated, sparse Merkle tree** optimized for storing blockchain state
in key-value databases. It was originally developed for Facebook's Diem (Libra) blockchain to balance **space efficiency,
computational overhead, and I/O performance**. The JMT draws inspiration from Ethereum’s Patricia Merkle Tree
(PMT) but introduces _key innovations_ in **node design, key schema, and proof structure** to better suit a
**log-structured merge (LSM) storage engine** (e.g. RocksDB). In essence, JMT combines a
**16-ary Radix Merkle Tree** structure with **sparse Merkle optimizations** and a **versioned node key scheme**. This design
enables high throughput updates (minimal compaction overhead on disk), relatively short proof sizes, and the ability to
persist multiple historical versions of the state efficiently (persisted snapshots). By using only two node types (internal
and leaf) and eliminating certain complexity (like Ethereum’s extension nodes), JMT achieves a simpler implementation
without sacrificing performance (see \[SN-001], \[SN-006]). Overall, this specification describes the high-level
data model, operations, and design rationale of JMT, which can guide an automated code generation phase for implementations
in Rust or other languages.

## Scope & Assumptions

This document focuses on the **core data structure and algorithms** of the Jellyfish Merkle Tree as used for an
authenticated key-value store in blockchain state management. It defines the JMT’s data model (nodes, keys, values) and
operations (insertion, update, lookup, proof generation) at a high level, suitable for code generation. We assume the
following:

- **Key-Value Domain**: The JMT is used to store a mapping from _keys_ to _values_, e.g. account addresses to account state.
  Keys are treated as fixed-length binary identifiers (typically cryptographic hashes of actual addresses or IDs, e.g. 256-bit
  values) to ensure a uniform size. Each key’s bits determine a unique path in the Merkle tree. Values are arbitrary binary
  blobs (e.g. serialized account data) stored in leaf nodes.
- **Cryptographic Hashing**: A secure cryptographic hash function (e.g. SHA-3 or Blake2) is used to compute node digests
  from child hashes or leaf data. The specific hash algorithm is assumed but not fixed in this spec. All references to
  “digest” imply a cryptographic hash output. <!-- TODO: Specify the hash function and parameters (hash length, etc.) in the implementation. -->
- **Persistent Versioning**: The tree supports **multiple versions** of the state. Each update to the key-value set produces
  a new version (typically incremented per block or transaction) which results in a new Merkle root hash. The JMT is
  _append-only_: it never overwrites nodes for a new state but instead creates new nodes for changes, while reusing untouched
  parts of the previous version (persistent functional tree). We assume an ever-increasing integer `version` is provided with
  each batch of updates to identify the state version (see \[SN-010]). Historical versions may be retained to allow proofs
  against old states.
- **Storage Model**: JMT nodes are stored in an **underlying key-value database** (such as RocksDB or a similar LSM-tree
  store). We assume the storage layer supports ordered keys and point lookups by key. The JMT will generate a set of key-value
  entries to be inserted for each new version (representing newly created nodes), and possibly a set of “stale” nodes that can
  be pruned. Actual database transactions or concurrency control are beyond the scope of this spec. We treat the storage as an
  abstract persistent dictionary mapping _NodeKey → NodeData_.
- **Out of Scope**: This spec does not cover client synchronization protocols, network communication of proofs, or specifics
  of batch commit strategies. It also does not detail potential optimizations like caching or parallel processing of nodes.
  Such concerns are left to the implementation. Additionally, specialized proof types (e.g. range proofs) and advanced
  features like garbage-collecting old versions are mentioned only briefly if at all. <!-- TODO: Design and document a pruning/garbage-collection strategy for stale nodes in long-running deployments. -->

All numeric examples (e.g. tree heights, byte sizes) are based on the Diem implementation and are for illustration. The spec
focuses on the logical structure and can be adapted to different environments as long as the core properties hold.

## Core Concepts

**Authenticated Key-Value Store (AKVS)** – The JMT acts as an authenticated data structure that binds a set of key-value
pairs to a single root hash (state commitment). A Merkle tree allows any client to verify that a given key maps to a given
value (or is absent) in the committed state by supplying a _Merkle proof_. This fits the blockchain need for cryptographic
state verification. The JMT is an instance of an _addressable_ Merkle tree where keys determine paths.

**Addressable Merkle Tree (AMT)** – An AMT is a deterministic authenticated structure (usually a Merkle tree) that can store
and map arbitrary binary keys to values. Each leaf in an AMT corresponds to a key-value pair, and the path from root to a
leaf is determined by the bits of the key. This contrasts with UTXO-style Merkle trees (which index outputs by position) –
in an AMT the key itself acts as the address within the tree. In blockchain contexts (account-based models), AMTs allow
efficient proofs of inclusion for accounts or other state objects. (see \[SN-002])

**Radix Merkle Tree (AR<sub>r</sub>MT)** – A Radix Merkle Tree generalizes the binary Merkle tree to an _r_-ary tree (branching
factor _r_ > 2). Instead of each node having 2 children (for each bit), a node can have up to _r_ children, effectively
compressing multiple bits of the key into a single tree level. For a key of fixed length _h_ bits and a radix _r_, the tree
height is approximately **log<sub>r</sub>(2^h)**. The JMT specifically uses _r = 16_ (a hexary tree), so each tree level
covers 4 bits (a “nibble”) of the key. Higher branching factors reduce tree height (improving read/query paths) at the cost
of more work when updating (since an update may need to rewrite larger node structures with many children). In general,
**larger _r_** yields shorter paths but higher write amplification, whereas **smaller _r_** yields longer paths but potentially
cheaper updates. The choice of _r = 16_ in JMT is a compromise balancing read and write costs (see \[SN-003]).

**Sparse Merkle Tree (SMT)** – A sparse Merkle tree conceptually represents the full binary trie of height _h_ (for _h_-bit
keys) but avoids explicitly storing empty subtrees. In a naive “perfect” Merkle tree of 2^h leaves, most leaves would be
empty (for unused keys), which is highly inefficient. A sparse Merkle tree optimizes this by treating any _absent_ subtree as
a constant default value (with a well-known hash) instead of a chain of empty nodes, and by collapsing any branch that leads
to a single leaf. This means we do not store paths of “only child” nodes – they are compressed into the leaf itself. The net
effect is drastically fewer nodes stored than the theoretical maximum. All empty leaves share a single **default hash**
(e.g. H(“”) or some agreed constant) which is used in proofs. The JMT leverages sparseness: empty subtrees are not stored,
and if a branch would have only one leaf, that leaf is linked directly to its highest branching ancestor. This
minimizes storage and reduces the length of proofs (fewer intermediate hashes). (see \[SN-004])

**Jellyfish Merkle Tree (JMT)** – The JMT can be viewed as a **Sparse Addressable Radix-16 Merkle Tree**. It inherits the
AMT property (key-determined paths), uses a 16-way branching factor (each node covers 4 bits of the key), and applies the
sparse optimizations (default placeholders and leaf compression). In addition, the JMT introduces a critical improvement for
performance on disk-based storage: **Versioned Node Keys**. Every node in the tree is identified not just by its position (path)
but also by the _version_ at which it was created. By prefixing each node’s key in the storage engine with a version number,
JMT ensures that newer versions of the tree append their nodes in lexicographic order, rather than overwriting or randomly
updating existing ones. This exploits the LSM-tree storage’s append-friendly nature, drastically
reducing write amplification due to compaction (JMT achieves effectively _zero compaction_ overhead for new state writes, as
demonstrated in Diem’s experiments – saving over 90% of I/O operations compared to a hash-keyed approach, see \[SN-015]). In
summary, key distinguishing concepts of JMT include:

- **High branching factor (16)**: Reduces tree depth (\~64 nibble levels for 256-bit keys) compared to binary (\~256 levels).
  This improves read performance (shorter proof paths) at the cost of slightly larger node update overhead (writing up to 16
  hashes per internal node update).
- **Two node types only**: JMT uses only _Internal Nodes_ and _Leaf Nodes_. It explicitly does **not use extension nodes**
  (unlike Ethereum’s trie) because any sequence of single-child nodes is rare in a large sparse tree and removing the
  extension concept simplifies implementation and reduces bugs. Every non-leaf in JMT is a 16-way branching
  internal node, unless it’s effectively a leaf reached at a shorter path.
- **Versioned node keys**: Each node is stored/retrieved by a compound key = `(version, nibble_path)`. The version (e.g. block
  height or txn index) indicates when the node was created, and the nibble path encodes its position in the tree. This scheme
  means that the “same” logical node (same path) created in two different versions will have distinct storage keys, so past
  versions of the tree remain accessible without conflict. It also guarantees that all nodes of an older version have smaller
  keys (lexicographically) than any node of a newer version, enabling append-only writes (see \[SN-005] and \[SN-015]).
- **Persistent structure**: Because of versioning, the JMT naturally supports persistence. Unchanged parts of the tree are
  carried over (referenced by identical nibble paths) into new versions, while only the parts of the tree affected by an update
  are created anew. An update affecting _m_ leaves out of _n_ total will create _O(m · log n)_ new nodes (recomputing only the
  path for each changed leaf). This significantly saves space when changes are small relative to the state size,
  and allows efficient storage of historical states by storing deltas (see \[SN-010]).
- **Optimized proof format**: JMT Merkle proofs follow the standard inclusion/exclusion proof concepts but are optimized to be
  smaller on average. Because the tree is sparse and 16-ary, the number of siblings in a proof is closer to _log<sub>16</sub>(N_active)_
  rather than _log<sub>2</sub>(N_possible)_. In other words, JMT proofs include far fewer default-hash siblings than a full binary
  tree would. The whitepaper notes the average number of proof elements is Θ(log _number of existing leaves_) vs log(2^h) for a
  dense tree. Additionally, JMT proofs categorize into three cases: _Inclusion_ proofs (target leaf exists), and two
  types of _Exclusion_ proofs – one where a different leaf with a conflicting prefix is encountered (proving the queried key is absent),
  and one where an empty placeholder is encountered (proving no leaf exists at that path). These proof types ensure that a
  verifier can conclusively check non-membership without needing all default siblings (see \[SN-007]).

## High-Level Data Model & Operations

### Data Model

**Node Types** – The JMT defines exactly two concrete node types (excluding a trivial `Null` placeholder for an empty tree):

- **Internal Node**: An internal node represents a branching point in the trie and can have up to 16 children (one for each possible nibble 0x0 through 0xF). It is effectively a radix-16 node that compresses 4 levels of a binary trie into one level. An internal node in storage holds an array or map of child pointers, where each _occupied child slot_ contains:

  - the child’s **hash digest**,
  - the **version** at which that child node was created,
  - and a flag indicating if the child is a leaf or an internal node.
    Any _unoccupied child slot_ is considered an empty subtree (and implicitly represents the default hash). For example, if an internal node at path `0x996F` (in hex nibbles) has children at only 4 of the 16 positions (say indices 0, 7, B, F), it will store those indices and their corresponding child versions/hashes, and omit the empty slots or mark them as empty. (The `JellyfishMerkleTree` implementation uses a `HashMap<Nibble, Child>` to store only present children, see \[SN-011].)

- **Leaf Node**: A leaf node contains the actual key-value mapping. It holds the full key (typically a 256-bit hashed address) of the account or item it represents, and either the value or a hash of the value (depending on whether values are stored inline or separately). In Diem’s implementation, the leaf stores the hash of the account blob and a pointer to the blob itself. For simplicity, we can consider the leaf node to contain:

  - the **key** (full length, to differentiate among siblings in case of collisions),
  - the **value hash** (H(value)),
  - and optionally the actual **value data** (or a reference to it) if the tree is storing values inline.
    The leaf node’s position in the tree is determined by the key’s bits. By definition, a leaf node has no children; it is the terminal node for that key’s path. All leaf nodes are at _some position at or above the maximum tree depth_ (they could be higher in case the tree compressed a single-leaf subtree).

Each node (internal or leaf) has a **cryptographic digest** that contributes to the Merkle root. For leaf nodes, the digest might be computed as `H(key || H(value))` (one common scheme, to incorporate the key to avoid certain collision attacks). For internal nodes, the digest is computed by hashing the concatenation of child digests (for all 16 children in index order, using a default digest for missing children). The exact hash computation is not critical for this spec, but it is important that it’s deterministic and consistent.

**Node Key (Storage Key)** – In the persistent key-value store, each node is stored with a key called `NodeKey`. A `NodeKey` is a tuple `(version, nibble_path)`:

- `version` – the version number (typically a 64-bit integer) when this node was _created_. This is not stored within the node itself but used as part of its lookup key in the DB.
- `nibble_path` – the sequence of nibbles from the root to this node’s position. This path can be represented as a byte string (each byte holding two nibbles) or any suitable compact form.

For example, the root node of version 42 will have `NodeKey(version=42, nibble_path="")` (empty path). A child of the root at nibble `A` (10 in decimal) that was first created in version 42 might have `NodeKey(42, path="A")`. If at version 43, a new leaf appears under that path `A` with further path `B7...`, intermediate new internal nodes will be created with `NodeKey(43, path="A")`, `NodeKey(43, path="AB")`, etc., possibly reusing existing nodes from version 42 for any unchanged branches. The key point is that **NodeKey globally identifies a node across the entire history** of the tree: no two different nodes share the same NodeKey, and given a NodeKey one can determine exactly which version and path the node belongs to. This scheme contrasts with using the node’s hash as a key: by using version+path, JMT keys are much shorter on average (only as long as the path length, typically much less than 32 bytes) and preserve a sort order aligned with version (newer version = larger key). (see \[SN-008], \[SN-009])

**Example**: Consider a small JMT storing two keys: `K1` and `K2`. Let their 4-bit nibble representations share a common prefix nibble `0x9`, but differ at the next nibble. Suppose `K1 = 0x9ABC...` and `K2 = 0x9F10...`. When the first key `K1` is inserted at version 1, it will create a leaf at path `9ABC...` and all necessary internal nodes along `9`, `9A`, `9AB`, etc. If `K2` is inserted at version 2, the insertion process will find the existing internal nodes for prefix `9`, and `9A` but will diverge at `9A` vs `9F`. At the nibble `9`, child `A` exists (leading to `K1`), but child `F` is empty. So a new leaf for `K2` is added at child `F` of node `9` at version 2. Additionally, the internal node at `9` itself was originally created at version 1 (with only child `A`). At version 2, that node will be updated (creating a _new_ node for path `9` version 2) now having two children (`A` and `F`). The new `NodeKey` for the internal node at path `9` would be `(version=2, nibble_path="9")`, while the old version 1 node at path `9` remains in storage as it was. The root’s child pointer for nibble `9` will now point to the version 2 node. In effect, the tree at version 2 reuses the path `9A...` for `K1` unchanged (except ancestors), and adds new branches for `K2`. This demonstrates persistence: version 1’s nodes are not modified in place, so one could still reconstruct the version 1 state root if needed.

### Operations

#### Lookup (Query)

**Goal**: Retrieve the value associated with a key `k` in a given version `v`, or determine that no such key exists, and provide a proof (if needed).

**Procedure**: To lookup key `k` at version `v`, the algorithm navigates from the root:

1. **Start at Root**: Determine the `NodeKey` for the root of version `v`. By convention in JMT, the root’s NodeKey may be just the version number (since the root has an empty nibble path). In practice, the root NodeKey for version `v` can be formed as `(v, path="")`. (In Diem’s implementation, the version itself was used as the root key reference out-of-band).
2. **Traverse Internal Nodes**: Fetch the node with the current `NodeKey` from storage (or cache). If it’s an **Internal Node**, do the following:

   - Take the next nibble of the target key `k` (according to the current depth in the tree) and interpret it as an index 0–15.
   - Check if the internal node has a child at that index. If **no child is present** at that nibble, then the search terminates: the key `k` was not found (the path leads to an empty subtree). We would produce an _exclusion proof_ indicating an empty node at that position (see proof section). If a child _is_ present, retrieve the child’s stored `version` and digest from the internal node structure.
   - Compute the next `NodeKey`: this is `(child_version, nibble_path = current_path + <nibble>)`. Essentially, append the nibble to the path and use the child’s recorded version. This derived NodeKey pinpoints the child node in storage.
   - Set this as the current node and repeat the fetch in storage.

3. If the fetched node is a **Leaf Node** (meaning we’ve either reached a terminal or stepped into a leaf), then read the leaf’s stored key. Compare it to the target key `k`:

   - If the keys match exactly, we found the target. Return the leaf’s value (or a pointer to it) as the result. A proof of inclusion can be generated by collecting all the sibling hashes on the path we traversed.
   - If the keys do _not_ match, this means we ended up at a different key’s leaf. This situation arises when the target key doesn’t exist, and the search path led into the location of another existing leaf (they shared a prefix up to a point, then diverged, but because the divergent branch for `k` was never created, `k`’s path fell into the other leaf). In this case, we conclude `k` is not in the tree. We produce an _exclusion proof_ citing this other leaf as evidence (often called a “neighbor proof” – the existence of a different leaf with the same prefix prevents `k` from existing). The lookup result is null/not found.

This lookup algorithm reads at most one node per tree level (per nibble of the key) until it finds a leaf or empty slot. Because the tree is sparse and branching factor 16, the _worst-case_ lookup is 64 steps for a 256-bit key (if the tree is maximally deep), but on average it is shorter if keys don’t extend all the way or the tree has few levels populated. Each node retrieval is a point DB lookup, which is efficient in an LSM store especially when keys are clustered by version.

#### Insertion & Update

**Goal**: Insert or update a key-value pair `(k, value)` at the next version `v`. If `k` already exists, its value is updated; if not, a new leaf is created. The result is a new root hash for version `v` and a batch of new nodes to persist.

**Procedure**: Insertion in JMT follows a similar traversal as lookup, with modifications when a missing branch or leaf is encountered:

1. **Traverse as in lookup**: Start from the previous version’s root (`version = v-1`) or an empty tree if this is the genesis version. (In practice, JMT often batches many insert/update operations as part of producing a new version; here we describe a single key insertion for clarity.) Walk down the tree following `k`’s nibbles, using the same child selection logic as lookup:

   - If an internal node does not have a child for a needed nibble, that is the point of insertion (an empty slot where the new leaf should go).
   - If a leaf node is reached, that is a point of insertion or update (depending on whether it’s the same key).

2. **Modify at insertion point**: Two scenarios:

   - **Empty Slot** (internal node child missing): We need to create a new leaf node for `k`. This happens when no leaf existed for `k`’s prefix. We create a **Leaf Node** with key `k` and the given `value`. This leaf’s `NodeKey` will be `(v, current_path + <nibble>)` where `current_path` is the path of the internal node and `<nibble>` is the index where the leaf is inserted. We insert this new leaf’s hash and version into the parent internal node at that index. The parent internal node itself will be updated (we create a _new_ Internal Node object for version `v` reflecting the added child). This new internal node will have the same children as the old one plus the new leaf. (All ancestors up to root will also be cloned/updated as described below.)
   - **Existing Leaf encountered**: We have reached a leaf node during traversal. There are two sub-cases:

     - If the existing leaf’s key equals `k`, then this operation is an **update** of an existing key. We will create a new Leaf Node for `k` at version `v` containing the new value, and mark the old leaf node as stale. Essentially, we replace the leaf node with a new version. The parent internal node will be updated to point to the new leaf (and its version), and the leaf’s hash will change to reflect the new value. No new branching occurs in this sub-case.
     - If the existing leaf’s key is different from `k`, we have a **key collision on a prefix**. The single leaf in the parent internal node actually represents a subtree that would diverge if both keys were present. To resolve this, we must introduce one or more new internal nodes to represent the diverging path. Concretely: let `L` be the existing leaf’s key and `K` = `k` the new key. They share a prefix (the path we followed so far) and differ at some next nibble. At the current internal node (the parent of that leaf), instead of a leaf, we will create a new Internal Node (call it `X`) at the point of divergence. `X` will have two children: one for `L` and one for `K` (at their next nibble). We remove the existing leaf from the parent and instead insert `X` as the child for the common prefix nibble. Then under `X`, we add the old leaf and a new leaf for `k` as children at their respective divergent nibble indices. If their keys diverge by more than one nibble (i.e., they have a common prefix of length > current*depth), we may need to create a \_chain* of internal nodes from the current position down to the level where the keys differ – each internal node in this chain will have one child leading to the next, effectively filling in the gap in the trie where no nodes existed before. This chain ends with the new internal node `X` that has two children (the two leaves). All these new internal nodes are created at version `v`. Finally, we place `X` into the parent (which itself is recreated for version `v`). In summary, the old leaf is "pushed down" one or more levels and a new leaf for `k` is inserted alongside it. This handles the collision gracefully and maintains the trie structure.

3. **Update ancestors**: After inserting or updating at the found position, we have potentially created a new leaf and some new internal nodes. Now, **all ancestor nodes along the path back to the root** must be updated for version `v` because the subtree hashes along that path have changed. Starting from the parent of the inserted/updated node up to the root:

   - For each ancestor node encountered on the original search path, create a new Internal Node for version `v` that is a copy of the previous version’s node, but with the child pointer (hash, version) updated to the new child node (or new child node inserted if it was an empty slot).
   - Recompute that internal node’s hash based on its children’s hashes.
   - This new internal node’s `NodeKey` will be `(v, path_of_node)`.
   - Continue upward, so that the root itself is replaced by a new root node at version `v` (with a new hash). This new root is the state commitment for the updated state.

All new nodes (leaf or internal) created during the insertion are marked with the current version `v` and will be inserted into storage with their NodeKey. Meanwhile, the old nodes from the previous version remain untouched (but may be considered stale from the perspective of version `v` since they are no longer part of the latest state). The set of new nodes and the set of obsolete nodes can be collected as a "batch". For example, the Libra implementation returns a `TreeUpdateBatch` containing `node_batch` (new nodes to insert) and `stale_node_index_batch` (identifiers of replaced nodes that could be deleted). The actual writing to the database can then be done in a single transaction for atomicity.

**Complexity**: An insertion or update touches O(log<sub>16</sub> N) nodes on the path (worst-case \~64). If a new leaf is added, one or more internal nodes might be created for branching (in the worst case of completely different key, one new internal per remaining nibble, but on average far fewer because keys rarely differ in every remaining bit). Every touched node up to root is duplicated for the new version. Thus insert complexity is O(log N) for both time and space. By leveraging the versioned store, the writes are sequential in the key space (sorted by version and path), which is optimal for LSM writes (see Non-Functional Goals).

#### Deletion (Removal)

_Deletion_ of a key is not explicitly described in the original JMT whitepaper, but it can be handled in a similar persistent manner: to delete a key, one would create a new version, remove the leaf node for that key, and update all ancestors. If the removal causes an internal node to have only one child remaining, one could theoretically remove that internal node and connect the child upwards (the inverse of the insertion collision process). However, such node merging (handling the sparse optimization in deletion) must be done carefully. The Libra code marks nodes as stale but doesn’t immediately merge nodes; the tree remains sparse (it might leave a chain of single-child nodes, which is acceptable). From a spec perspective, deletion will create a new version where the leaf is not present and any internal nodes on the removed path are updated (some may become single-child nodes). We mention deletion for completeness but do not define it fully here. <!-- TODO: Define the deletion algorithm and how single-child internal nodes might be handled or left as is, in line with persistent storage strategy. -->

#### Proof Generation

One of the main purposes of the JMT is to generate **Merkle proofs** to verify state elements. Given the above lookup procedure:

- An **Inclusion Proof** for key `k` at version `v` consists of the hash of the leaf node for `k` (including perhaps the key and value hash) and the set of all **sibling node hashes** on the path from the leaf to the root at version `v`. In JMT, because of the sparse 16-ary structure, many siblings may be default placeholders. The proof can omit long runs of default siblings by instead including an indication of an empty subtree. (The Diem implementation collapses consecutive empty levels: for example, if from a certain depth onwards no nodes exist on one side, the proof can just include one default hash for that entire empty branch.) The verification algorithm will hash the leaf with its key and value, then hash it with siblings (in the correct positions, inserting default hashes where indicated) upward until reaching the root hash, which should match the known state commitment.
- An **Exclusion Proof** for key `k` at version `v` can take two forms, as noted earlier:

  1. **Neighbor proof**: show a leaf of a different key `L` that shares a prefix with `k` (and typically that the divergence point in the path had no other branch for `k`). For instance, if `k` would have branched left at some internal node where the only child is a leaf `L` going right, the proof will include leaf `L` and siblings up to root. The verifier sees that `L`’s key is different but has the common prefix, implying `k` could not exist or it would have appeared as a sibling at that branch.
  2. **Empty subtree proof**: show that at some internal node, the child pointer for `k`’s nibble is empty (and the hash there is the known default hash). In practice, the proof will include the default hash (or a compressed indicator of an empty path) and siblings up to root. The verifier can confirm that the path for `k` terminates early in an empty subtree.

The JMT proof format as described in the whitepaper includes a struct with a `Leaf` (containing address and value_hash) and a vector of `HashValue` siblings. For inclusion, the provided leaf will match the query key; for exclusion, either a different leaf is provided or the proof’s leaf can be a default placeholder. The verifier needs to distinguish these cases (the proof type) and then perform the appropriate hash recomputations. This specification does not delve into the low-level serialization of proofs, but the concept is that any proof is simply the minimal set of hashes (and possibly one key) needed to recompute the root. The branching factor being 16 means proof paths are shorter and potentially fewer hashes than an equivalent binary Merkle proof for the same number of leaves (especially when large portions of the tree are empty).

## Architecture Overview

At a high level, the Jellyfish Merkle Tree system can be thought of as two layers:

1. **In-memory logic layer**: which handles the trie navigation, node creation, hashing, and assembly of updates or proofs.
2. **Storage layer**: an underlying key-value store where node data is persisted using the composite keys.

When a new version of the state is committed, the JMT logic produces a set of new nodes (each with a `NodeKey(v, nibble_path)`) and typically marks old nodes as stale. These are then written to the database. The _root hash_ of the JMT at a given version serves as the authenticated state identifier (e.g., to be stored in a block header).

Below is a diagram stub illustrating the relationships between JMT nodes, their keys, and the storage:

```plantuml
@startuml JMT_Architecture
' TODO: Insert architecture diagram illustrating node relationships, versioned keys, and storage.
' For example, show a root referencing internal nodes (with version labels) and leaves, and how NodeKeys are composed.
@enduml
```

_(Diagram description)_: In a typical diagram, each node would be labeled with its `NodeKey`. Arrows from an Internal node to its children would be labeled with the nibble (0–F) of the path. The storage key for a child might be shown as "(version ‖ path)" on the arrow. One could also illustrate two versions of the tree side by side: e.g., version 5 and version 6, where most nodes are shared except one branch that got updated at version 6 (thus new nodes at that branch with `version=6` in their keys). This highlights how new version nodes extend the structure without altering old ones.

The architecture is intentionally simple: node retrieval is done by primary key lookup in the database, and updates are bulk-inserted. There is no complex pointer-chasing or memory management beyond what the database does. The design benefits from database features:

- **Ordered storage**: By constructing NodeKeys such that all nodes of earlier versions sort before nodes of later versions, sequential writing is achieved (important for LSM-trees). The ordering is essentially by version first, then by path bytes.
- **Sharding by version**: The version prefix could be used to separate data into different physical shards or files if needed (e.g., pruning old versions by dropping entire prefixes).
- **Concurrency**: Read operations (proof generation) can run without locks as they operate on an immutable version of the tree. Writes for a new version can be done in one batch, making coordination simpler. (Only one writer per version is assumed in blockchain context, typically the block executor.)

The **JMT code architecture** (e.g., in Rust) typically provides an API like `put_value(key, value, version)` which returns the new root hash and a batch of writes. The internal structures include a representation of nodes (as Rust enums or classes) and functions for hashing and proof construction. This spec aims to remain language-agnostic, but these concepts map directly to implementations (see Appendix A for references to actual code definitions \[SN-011]).

## Non-Functional Goals

The Jellyfish Merkle Tree’s design was guided by several non-functional goals, primarily performance and maintainability:

- **High Throughput for Updates**: The use of **versioned node keys** and sorted insertion order means that writing a new version’s state is as efficient as appending new records to the database. Compaction overhead in an LSM store is essentially eliminated because new keys come in ascending order (no random writes). In tests, this resulted in over _90% reduction in IOPS and disk bandwidth_ usage compared to a Merkle tree that uses random hash keys for nodes (which cause writes all over the key space) (see \[SN-015]). This makes JMT highly scalable for frequent state updates, as expected in a blockchain (e.g., thousands of transactions per block producing new states).
- **Low Storage Overhead**: JMT stores fewer nodes than a naive trie due to its sparse optimization. It also avoids storing large hash keys as node identifiers. On average, a NodeKey in JMT is on the order of a few bytes (version + a handful of nibbles) rather than a 32-byte hash. For example, with 1 billion leaves, average path length might be \~8 nibbles, meaning \~12-byte average NodeKey. This yields space savings in the database indices. Furthermore, unchanged nodes are not duplicated for new versions, so the storage growth is proportional to changes. Historical versions do consume extra space, but only for the parts of the state that changed – this is optimal for applications where each block or transaction only affects a small portion of the state.
- **Fast Proof Verification**: Proof sizes in JMT are smaller on average than in a full binary Merkle tree for the same key space, because the branching factor is higher and default siblings are collapsed. This means less data needs to be transmitted and hashed when sending proofs to clients. In practical terms, a proof in JMT contains at most 64 sibling hashes (for a 256-bit key, if every level had a sibling), but typically much fewer because many subtrees are empty and can be represented by a single default hash. Shorter proofs improve network efficiency and verification time for light clients.
- **Simplicity and Maintainability**: The JMT avoids the complexity of multiple node types (no separate extension node logic) and uses a uniform approach for all branches. This simplifies implementation: developers only need to handle internal nodes and leaves, reducing potential bug surface. The absence of extension nodes (which in Ethereum’s PMT handle streaks of single-child nodes) was a deliberate choice to keep the codebase cleaner, given that such streaks are unlikely in a large, sparse address space (especially when keys are hashed) \[SN-014]. Additionally, by aligning with a persistent versioned model, the JMT fits well with functional programming techniques (treating the tree as immutable between versions), which can make reasoning and testing easier.
- **Consistency and Integrity**: As an authenticated structure, JMT ensures that any tampering with the data can be detected by verifying against the known root hash. This is a fundamental property of Merkle trees. Non-functional in the sense of security, the design assumes the cryptographic hash is secure (preimage-resistant, collision-resistant) so that the chance of a malicious collision or preimage is negligible. If that holds, the JMT provides strong integrity guarantees for the state.
- **Scalability**: JMT is designed to handle very large state sizes (e.g., millions to billions of keys). The combination of on-disk storage, sparse structure, and versioning means the tree can scale without needing to reside entirely in memory. Only the part of the tree being accessed or updated needs to be loaded. The sorted-order writes and point reads are operations that scale well in modern storage engines. Moreover, reading older versions is possible (for audit or syncing purposes) as long as those nodes are retained. This contributes to the scalability and flexibility of the system.
- **Flexibility**: The JMT’s structure is not tied to a specific use-case; it could be implemented in different languages and storage backends. The spec is high-level enough that code generation could target, say, a Rust in-memory variant or a SQL-based store if needed. The design also allows additional features like batched updates (inserting multiple leaves in one version) and even range proofs (the whitepaper mentions the possibility of range proofs for consecutive leaves) with proper design. <!-- TODO: Consider extending the specification with support for range proofs or multi-key proofs if required by the application. -->

In summary, JMT achieves a mix of **performance optimizations** (for I/O and storage) and **implementation simplicity**. It is well-suited for blockchain state management where frequent updates occur and historical states may need to be preserved or verified. These non-functional goals guided the design choices validated by the Diem team’s research and can inform anyone re-implementing the JMT.

## Open Questions & TODOs

While the Jellyfish Merkle Tree is a well-defined structure, certain aspects are left for implementers or future refinements:

- **Hash Function and Key Size**: The spec assumes a generic cryptographic hash function. A concrete implementation must choose one (e.g. SHA-256 as in Libra/Diem, or another function) and ensure all participants use the same. The output size of the hash (e.g. 256 bits) determines the tree height (number of nibbles). <!-- TODO: Specify the hash algorithm (and consider if configurable) and ensure all tree computations use it consistently. -->
- **Default Hash Constants**: The value of the default placeholder hash for empty subtrees must be decided (often the hash of an empty string or zero). This constant needs to be agreed upon as it’s part of proof verification. Our spec does not fix it. <!-- TODO: Define the default empty subtree hash constant. -->
- **Deletion Algorithm**: As noted, deletion handling is not fully specified here. In practice, removing a key would involve creating a new version without that key and updating ancestor hashes. The rules for when to remove an internal node (if its children count goes to 1) or whether to leave tombstones need to be defined to maintain an optimal sparse structure. <!-- TODO: Develop and document the deletion process, including whether to merge single-child paths or leave them. -->
- **Storage Pruning**: Over time, old versions’ nodes might consume a lot of storage. A strategy to prune or compact the store by removing stale nodes is needed. The Libra implementation tracks stale nodes (by version and NodeKey) so they can be deleted after a certain point. Implementers need to decide when it’s safe to drop older versions (e.g. after finality or checkpoints). This is application-specific and not covered in the core spec. <!-- TODO: Implement pruning of old versions (maybe keep last N versions or prune at checkpoints) and ensure no needed proofs are lost. -->
- **Concurrency and Caching**: The spec assumes a single-threaded update to generate each new version. In real systems, concurrency control (e.g., if multiple transactions update different parts of the tree in parallel) and node caching strategies (to avoid re-reading frequently accessed internal nodes from DB) can significantly impact performance. These are beyond the scope of the spec, but worth noting for implementation. <!-- TODO: Investigate caching hot internal nodes and thread-safe batch updates if multiple writers are considered. -->
- **Alternate Branching Factors**: We fixed the radix at 16 due to its proven balance in Diem. If an implementer wanted to use a different branching factor (say 8 or 32), the concept remains similar but would affect performance trade-offs and proof sizes. Our spec is written for r=16. Adjustments for other r would include changing the nibble size (e.g. 3 bits for 8, 5 bits for 32) and how NodeKey paths are encoded. This is an open parameter to explore if needed.
- **Proof Serialization**: The exact encoding of proofs (e.g., how to indicate an empty subtree compression, how to encode sibling positions) is left unspecified. Any code generation should define a struct or format compatible with clients. <!-- TODO: Define the binary format for Merkle proofs (perhaps using an existing standard or a simple length-prefixed array of hashes). -->

These open questions do not undermine the core functionality of JMT but are considerations for a complete production implementation. They are marked for future resolution as needed in the context where the JMT is employed.

## Reference Mapping

The following table maps key concepts in this specification to the supporting source snippets (from original materials):

| Concept or Feature                     | Relevant Snippet IDs |
| -------------------------------------- | -------------------- |
| Addressable Merkle Tree (AMT)          | \[SN-002]            |
| Radix Merkle Tree (branching factor)   | \[SN-003]            |
| Sparse Merkle Tree (default/empty)     | \[SN-004]            |
| JMT = Sparse AR16 Merkle (definition)  | \[SN-005]            |
| Version-based node keys & benefits     | \[SN-005], \[SN-015] |
| Two node types (no extension node)     | \[SN-006], \[SN-014] |
| Concise proof (fewer siblings)         | \[SN-007]            |
| Persistent versioning (delta updates)  | \[SN-010]            |
| NodeKey structure (version + path)     | \[SN-008], \[SN-009] |
| Internal node structure (children)     | \[SN-011]            |
| Leaf node structure (key, blob hash)   | \[SN-011]            |
| Insertion logic (new leaf/internal)    | \[SN-012]            |
| Update logic (replace leaf value)      | \[SN-013]            |
| Removal of extension nodes (rationale) | \[SN-014]            |
| LSM-tree optimization (compaction)     | \[SN-015]            |

Each snippet provides direct quotations or code from the Diem JMT whitepaper (2021), Daniel Olshansky’s 5-point summary (2022), or the Libra Core implementation in Rust, as indicated. These sources reinforce the design choices and properties discussed.

## Appendix A – Source Snippets

**SN-001**: _JMT overview from Diem whitepaper (2021) – highlights efficiency and optimization for LSM storage._
“_This paper presents Jellyfish Merkle Tree (JMT), a space-and-computation-efficient sparse Merkle tree optimized for Log-Structured Merge-tree (LSM-tree) based key-value storage, which is designed specially for the Diem Blockchain. JMT was inspired by Patricia Merkle Tree (PMT), a sparse Merkle tree structure that powers the widely known Ethereum network. JMT further makes quite a few optimizations in node key, node types and proof format to find the ideal balance…_” (This describes the JMT as a sparse Merkle tree with various optimizations for performance, setting the stage for its design goals.)

**SN-002**: _Addressable Merkle Tree definition by Olshansky (2022) – explains what an AMT is._
“_1. An Addressable Merkle Tree (AMT) is a cryptographically authenticated deterministic data structure backed by a key-value store database used for account-based (non-UTXO-based) systems to map keys (i.e. addresses) to arbitrary binary data in each leaf node._” (Relevance: Defines the concept of an AMT, which the JMT is an instance of, highlighting that keys map to values in leaves in an authenticated structure.)

**SN-003**: _Addressable Radix Merkle Tree (ARMT) and branching factor trade-offs (Olshansky, 2022)._
“_2. An Addressable Radix Merkle Tree (AR₁₆MT) ... is a generalization of an AMT, a binary tree, where r > 2. With a key size of h bits, and each node having at most r children, the height of the tree is logᵣ(2ʰ). The tradeoffs of r are: (a) A large r is good for read-heavy applications but results in greater write amplification and higher storage costs when updating internal path nodes. (b) A small r is good for write-heavy applications, but results in higher I/O when querying or updating the tree._” (Relevance: Describes the nature of a radix-Merkle tree and the impact of branching factor on reads/writes. JMT chooses r=16 as a balance between these trade-offs.)

**SN-004**: _Sparse Merkle Tree optimizations (Olshansky, 2022) – empty subtrees and single-leaf compression._
“_3. A Sparse Merkle Tree (SMT) ... accounts for the fact that Perfect Merkle Trees are never needed for account-based models since most keys will not have data, thereby removing the need to store 2ʰ keys. The two main optimizations are: - Empty Subtrees are replaced with a constant default placeholder node value. - Single-Leaf Subtrees are replaced with one node reflecting the one leaf value._” (Relevance: Explains how sparse Merkle trees avoid storing large numbers of empty or single-child nodes. JMT uses exactly these optimizations to reduce tree size and proof length.)

**SN-005**: _JMT as Sparse AR16MT with versioned node keys (Olshansky, 2022) – key features summary._
“_4. A Jellyfish Merkle Tree (JMT) is a Sparse Addressable Radix Merkle Tree with r=16 (Sparse AR₁₆MT) that balances the tradeoff of storage, compute, read and write operations using two node types: Internal Node and Leaf (Data) Node. It optimizes for Log-Structured Merge-tree (LSM-tree) based key-value storage databases (e.g. RocksDB) by using Version-Based Node Keys whereby a monotonically increasing version number (i.e. # of transactions applied to the state) is prefixed to the node path in order to: (– Enable version-based sharding; – Automatically sort node updates written (i.e. appended) to disk in lexicographic order, reducing the disk bandwidth and IOPS of LSM-tree compaction.)_” (Relevance: Captures the essence of JMT’s structure and its special sauce: version-prefixed keys for efficient storage. It lists the two node types and the benefits of the versioned key scheme on LSM-tree performance.)

**SN-006**: _“Less Complexity – only two node types” (Diem whitepaper, 2021)._
“_• Less Complexity: JMT has only two physical node types, Internal Node and Leaf Node._” (Relevance: Notes that the JMT simplifies the Merkle tree design by using just internal and leaf nodes, avoiding additional node types such as extension nodes. This is a conscious design choice for simplicity.)

**SN-007**: _“Concise Proof Format” with fewer siblings on average (Diem whitepaper, 2021)._
“_• Concise Proof Format: The number of sibling digests in a JMT proof is less on average (Θ(log(number of existent leaves))) than that of the same ARMT without optimizations (log(number of maximum leaves), i.e., the height of the equivalent AMT), requiring less computation and space._” (Relevance: Quantifies the proof size improvement. JMT proofs grow with the log of actual populated leaves, rather than the maximum tree height, thanks to skipping default siblings. This supports JMT’s goal of shorter proofs.)

**SN-008**: _Version-based NodeKey schema (Diem whitepaper, 2021)._
“_JMT adopts a version-based node key schema, by splicing version and nibble path, as: version ‖ nibble path, where the node of this key is created at version and nibble path is the sequence of nibbles on the path from the root node to this node following the given key._” (Relevance: Defines how node keys are formed by concatenating the version with the path. This is fundamental to how JMT addresses nodes in storage and ensures uniqueness across versions.)

**SN-009**: _Uniqueness of version+path keys across history (Diem whitepaper, 2021)._
“_At any version, a nibble path itself can uniquely identify a node. Therefore, combining version and nibble path could similarly pinpoint any node across versions from the whole blockchain history._” (Relevance: Emphasizes that adding the version prefix allows one to uniquely identify nodes even when multiple versions exist. It’s why JMT can maintain multiple versions without key collisions in the database.)

**SN-010**: _Persistent data structure & node reuse between versions (Diem whitepaper, 2021)._
“_The new tree reuses unchanged portions generated at previous versions, forming a persistent data structure. If an update modifies m out of n leaves, on average O(m ⋅ log n) new nodes are created in the tree that differ from the previous version. This approach allows any upper-layer application to store multiple versions of the state efficiently by only recording the “delta”._” (Relevance: Describes the versioning/persistence aspect of JMT. Only a logarithmic number of new nodes are created per updated leaf, and unchanged parts are reused, which is key to JMT’s efficiency in storing incremental states.)

**SN-011**: _Libra Core implementation structures (Rust) for Node, InternalNode, LeafNode, Child._

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

_(Relevance: These code definitions from Libra’s implementation illustrate the actual data model: a NodeKey composed of version and path, and Node as an enum of either Internal or Leaf. The InternalNode holds a map of children keyed by nibble, with each child storing its hash, creation version, and type. The LeafNode holds the full key (account_key) and the hashes of its data. This confirms and complements the abstract description of JMT’s node structure in this spec.)_

**SN-012**: _Insertion logic – internal node case and new leaf creation (Diem whitepaper, 2021)._
“_• Internal Node: Create a new leaf node from k and d with its node key as v ‖ current nibble path ‖ n. Fill version as v and digest as hash(k ‖ hash(d)) into the current internal node at index n._”
“_• Leaf Node: … – If the keys mismatch, the new leaf node will be created likewise. Moreover, a series of cascading internal nodes will be created one by one to represent the shared nibble path within all unvisited nibbles of both keys. Also, both leaf nodes will be grafted onto the bottom one as child nodes. Afterwards, the uppermost new internal node will be positioned in place of the old leaf node in its parent._” (Relevance: Describes how insertion works when encountering an empty slot (new leaf directly) and when a leaf with a different key is encountered (create new internal nodes and a new leaf, split the path). It corresponds to our insertion algorithm discussion, detailing the creation of new leaf and internal nodes at the point of divergence.)

**SN-013**: _Update logic – leaf keys match case (Diem whitepaper, 2021)._
“_– If the keys match, the insertion becomes an update. We just have to replace the leaf node with its new value and update its node key with new version v._” (Relevance: Describes the simple case of updating an existing key: a new leaf node with the updated value is created (at the new version) replacing the old one. This supports our explanation that updates don’t add new branches, they just replace the leaf and update ancestor hashes.)

**SN-014**: _Rationale for removing extension nodes (Diem whitepaper, 2021)._
“_It is noted that we expressly abandon an extension node, introduced by PMT \[Ethereum], for the reasons below: • The efficacy brought about by extension nodes will diminish quickly as the tree size grows and in turn becomes less and less sparse. The chance that two leaf nodes share a long common prefix of keys is so rare that the case where an extension node can substitute a long chain of internal nodes is very uncommon. • Removing the extension node effectively contributes to less complicated code in favor of less potential bug surfaces._” (Relevance: Explains why JMT does not include extension nodes. Essentially, they provide little benefit in a large sparse tree (keys rarely share very long prefixes), and eliminating them makes the implementation simpler and less error-prone. This design decision is a key difference from Ethereum’s trie and is reflected in JMT’s two-node-type approach.)

**SN-015**: _LSM storage optimization – sequential writes and 90% reduction in compaction (Diem whitepaper, 2021)._
“_Given the JMT node key schema, we could insert the new nodes generated by each version sequentially to append to the current key set in storage according to the lexicographic order because our key schema ensures keys of a high version are always lexicographically greater than those at a lower version. In this case, compaction is no longer necessary as the keys inserted are already ordered. Our experiment shows this schema saves IOPS and disk bandwidth by more than 90% in contrast to hash-based node keys._” (Relevance: This empirical result underlines one of JMT’s biggest advantages: by writing new version nodes in sorted order, RocksDB/LevelDB doesn’t need to constantly reorganize data (compaction). The quote explicitly states compaction becomes unnecessary and quantifies the I/O savings (>90%) compared to a baseline where node keys are random hashes. This validates the choice of versioned, sorted keys in the JMT design.)
