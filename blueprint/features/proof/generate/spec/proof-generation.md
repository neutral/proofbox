# Proof Generation Specification

## Overview
One of the main purposes of the JMT is to generate **Merkle proofs** to verify state elements. Proofs allow clients to verify key-value pairs without accessing the entire tree.

## Proof Types

### Inclusion Proof
For key `k` that exists at version `v`:
- Contains hash of the leaf node for `k` (including key and value hash)
- Set of all **sibling node hashes** on path from leaf to root
- Due to sparse 16-ary structure, many siblings may be default placeholders
- Can omit long runs of default siblings (compression optimization)

### Exclusion Proof
For key `k` that doesn't exist at version `v`. Two forms:

1. **Neighbor Proof**
   - Shows leaf of different key `L` that shares prefix with `k`
   - Proves `k` couldn't exist without appearing as sibling at branch
   - Includes leaf `L` and siblings up to root

2. **Empty Subtree Proof**
   - Shows that at some internal node, child pointer for `k`'s nibble is empty
   - Includes default hash (or compressed indicator) and siblings to root
   - Verifier confirms path for `k` terminates in empty subtree

## Proof Format Optimization
JMT proofs are concise (see [SN-007]):
- Number of siblings: Θ(log(number of existent leaves))
- Versus log(2^h) for dense tree
- Shorter paths due to:
  - 16-ary branching (fewer levels)
  - Sparse optimization (collapsed empty regions)
  - Default hash compression

## Generation Algorithm
During lookup traversal:
1. Track path taken from root to leaf/empty
2. At each internal node, record:
   - Sibling hashes (or default indicators)
   - Node positions/nibbles
3. Package proof data:
   - Leaf data (if inclusion or neighbor proof)
   - Sibling hashes in correct order
   - Proof type indicator

## Proof Structure Example
```rust
struct SparseMerkleProof {
    leaf: Option<LeafNode>,     // Present for inclusion/neighbor
    siblings: Vec<HashValue>,   // Sibling hashes along path
    // Additional metadata for reconstruction
}
```

## Size Benefits
- Maximum proof size: 64 sibling hashes (256-bit key)
- Typical size: Much smaller due to sparsity
- Compression techniques further reduce transmission size
- Improved network efficiency for light clients