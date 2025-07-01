# LSM Storage Optimization Specification

## Overview
The JMT's versioned node key design achieves exceptional performance on Log-Structured Merge (LSM) tree storage engines like RocksDB/PebbleDB.

## Key Innovation: Sequential Writes
Given the JMT node key schema `version || nibble_path`:
- Keys of version N are lexicographically less than version N+1
- New nodes append to current key set in storage
- Write pattern matches LSM's append-friendly nature

## Performance Impact
Experimental results show this schema saves IOPS and disk bandwidth by **more than 90%** compared to hash-based node keys (see [SN-015]).

### Why Hash-Based Keys Fail
Traditional Merkle trees using node hash as storage key:
- Random 32-byte keys scatter across key space
- Triggers frequent compaction in LSM stores
- High write amplification
- Poor locality of reference

### Why Version-Based Keys Succeed
JMT's approach:
- Sequential append pattern
- **Compaction becomes unnecessary** - keys already ordered
- Minimal write amplification
- Excellent locality for version-based queries

## Storage Pattern Example
```
Version 1 nodes:
(1, "")      -> root_v1
(1, "9")     -> internal_v1
(1, "9A")    -> leaf_K1

Version 2 nodes (append after v1):
(2, "")      -> root_v2
(2, "9")     -> internal_v2_updated
(2, "9F")    -> leaf_K2_new
```

## Benefits for Distributed Systems
- High-frequency state updates (thousands per batch)
- Sustained write throughput with minimal I/O
- Target: <10 MB/s sustained write on commodity SSDs
- Enables high transaction throughput

## Additional Optimizations
1. **Version-based sharding**: Partition by version prefix
2. **Batch writes**: Group all nodes for version atomically
3. **Read caching**: Hot internal nodes stay in memory
4. **Pruning**: Drop old version prefixes entirely