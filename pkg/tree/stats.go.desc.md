# Tree Statistics

## Purpose

Provides thread-safe, atomic access to key tree metrics without requiring locks, enabling efficient monitoring and debugging of tree operations.

## Design Rationale

### Why Atomic Operations?
The tree statistics are frequently read by:
- Metrics collectors (Prometheus scraping)
- Health check endpoints
- Debug interfaces
- Logging systems

Using atomic operations instead of mutex-protected fields eliminates lock contention between readers and the writer (tree update operations), providing better performance under high read load.

### Key Metrics

1. **Height**: Maximum depth of the tree
   - Indicates tree balance and lookup complexity
   - Important for performance monitoring

2. **Node Count**: Total nodes in the current version
   - Tracks tree growth over time
   - Useful for capacity planning

3. **Version**: Current committed version number
   - Tracks tree evolution
   - Correlates with transaction history

## Usage Patterns

### Bulk Updates
The `UpdateStats` method updates all metrics atomically, but not as a single atomic operation. This is acceptable because:
- Readers might see temporarily inconsistent values
- All values are eventually consistent
- Slight inconsistency is preferable to lock contention

### Individual Updates
Individual update methods support incremental updates during tree modifications, useful for:
- Real-time height tracking during inserts
- Node count adjustments during pruning
- Version updates on commit

## Performance Characteristics

- **Read Cost**: Single atomic load (~1-2 ns)
- **Write Cost**: Single atomic store (~5-10 ns)
- **No Memory Barriers**: Uses relaxed memory ordering
- **Cache-Friendly**: Fields likely in same cache line

## Monitoring Integration

These statistics feed directly into:
- Prometheus metrics (tree_height, tree_nodes, tree_version)
- Health check responses
- CLI status commands
- Performance dashboards