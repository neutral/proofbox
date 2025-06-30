# Batch Optimizer

This file implements optimization strategies for update batches, including deduplication, compression, and future support for write coalescing.

## Why Batch Optimization?

Large update batches can consume significant memory and I/O bandwidth. The batch optimizer addresses these challenges by:

1. **Reducing Redundancy**: Deduplicating operations that write to the same nodes multiple times
2. **Minimizing I/O**: Compressing batch data before storage operations
3. **Improving Throughput**: Enabling efficient batch transmission and storage

## Optimization Strategies

### Deduplication
Removes duplicate node writes and stale node entries. This is particularly important when:
- Multiple operations affect the same tree paths
- Path cloning creates intermediate nodes that are later replaced
- The same node appears multiple times in the stale nodes list

### Compression
Uses gzip compression to reduce batch size. This is beneficial for:
- Large batches with many nodes
- Network transmission of batches (future use)
- Reducing storage I/O bandwidth

### Write Coalescing (Future)
Currently a placeholder for grouping writes by storage locality to improve LSM-tree performance.

## Design Decisions

### Configurable Optimizations
All optimizations can be individually enabled/disabled through BatchOptimizerConfig, allowing users to tune for their specific workload.

### Compression Level
Defaults to gzip.DefaultCompression (level 6) as a balance between compression ratio and CPU usage. Can be adjusted from 1 (fastest) to 9 (best compression).

### Statistics Collection
The GetBatchStats method provides insights into batch composition and compression effectiveness, useful for monitoring and tuning.

## Performance Considerations

- Deduplication has minimal overhead and should generally be enabled
- Compression trades CPU for I/O bandwidth - beneficial for large batches
- Statistics collection is lightweight and can be used for monitoring