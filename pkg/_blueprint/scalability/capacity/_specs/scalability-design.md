# Scalability Design Specification

## Overview
JMT is designed to handle very large state sizes (millions to billions of keys) through careful architectural choices.

## Scalability Dimensions

### State Size Scalability
- **Sparse structure**: Handles 2^256 key space efficiently
- **On-disk storage**: Not limited by RAM
- **Logarithmic growth**: Tree height grows slowly with keys
- Tested with billions of accounts in production (Diem)

### Throughput Scalability
- **Append-only writes**: No compaction bottleneck
- **Batch operations**: Multiple updates per version
- **Parallel read support**: MVCC via versioning
- Target: Thousands of updates per second

### Storage Scalability
- **Delta storage**: Only changes stored per version
- **Compression friendly**: Sequential keys compress well
- **Shardable**: Can partition by version or path prefix
- **Prunable**: Old versions can be archived/removed

## Key Design Elements

### 16-ary Structure Benefits
- Shallower trees: O(log₁₆ N) depth
- Fewer nodes overall despite larger nodes
- Better read scalability (fewer I/O operations)
- Suitable for distributed caching

### Version-Based Keys Enable
- **Temporal sharding**: Split by version ranges
- **Spatial sharding**: Split by path prefix
- **Read replicas**: Serve different version ranges
- **Incremental backups**: Only new versions

### Memory Efficiency
- Hot path caching (frequently accessed internal nodes)
- Lazy loading (load nodes on demand)
- Bounded memory usage regardless of state size
- LRU eviction for cached nodes

## Operational Considerations

### Horizontal Scaling
- Read replicas for query load
- Version-based partitioning for storage
- Geographic distribution possible
- Eventually consistent read replicas

### Performance Characteristics
For 1 billion key database:
- Tree depth: ~8 levels
- Lookup time: ~8 disk reads worst case
- Update time: ~8 nodes written
- Proof size: ~8 hash values

### Bottleneck Mitigation
- **Storage I/O**: Addressed by append-only design
- **Network bandwidth**: Compact proofs and keys
- **CPU**: Parallelizable hash computations
- **Memory**: On-demand loading and caching

## Future Scaling Options
- Range proofs for multiple keys
- Subtree pruning for archived data
- Compressed node formats
- Hardware acceleration for hashing