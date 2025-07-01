# Storage Efficiency Specification

## Overview
JMT achieves excellent storage efficiency through multiple design choices that minimize both space usage and I/O operations.

## Space Optimizations

### Sparse Structure
- Only stores nodes that exist (no empty subtrees)
- Single default hash represents all empty regions
- Dramatic reduction vs full tree: stores O(N) nodes for N keys vs O(2^256)

### Compact Node Keys
- NodeKey typically ~12 bytes (8-byte version + few nibbles)
- Compare to 32-byte hash keys in traditional designs
- Reduces database index size significantly

### Persistence with Delta Storage
When updating m out of n leaves (see [SN-010]):
- Creates O(m · log n) new nodes
- Unchanged subtrees shared between versions
- Storage growth proportional to changes, not state size

## I/O Optimizations

### Minimized Read Amplification
- 16-ary structure reduces tree height
- Fewer nodes to traverse (log₁₆ N vs log₂ N)
- Point lookups efficient in LSM stores

### Eliminated Write Amplification
- Append-only writes avoid compaction
- Sequential key pattern optimal for LSM
- 90%+ reduction in I/O vs random writes

## Practical Impact

### Example: 1 Billion Keys
- Average path length: ~8 nibbles
- Average NodeKey size: ~12 bytes
- Tree height: ~8-10 levels (vs 30+ for binary)
- Proof size: ~8-10 hashes (vs 30+ for binary)

### Storage Growth Pattern
For distributed systems with:
- 1000 transactions per batch
- 10 entities modified per transaction
- Result: ~10,000 new nodes per batch
- At 1KB per node: ~10MB per batch
- Sustainable on commodity SSDs

## Design Trade-offs
- Larger branching factor (16) means larger internal nodes
- But fewer internal nodes overall
- Net positive due to reduced tree height
- Optimal for read-heavy distributed workloads