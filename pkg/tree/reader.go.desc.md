# Tree Reader

## Purpose

Provides snapshot-based read operations for the Merkle tree, ensuring consistent reads at specific versions without blocking concurrent writes.

## Design

The TreeReader encapsulates:
- A storage snapshot for consistent reads
- Version-specific root hash
- Reference to the parent tree for shared resources

This design enables:
1. **Read Consistency**: Snapshot isolation prevents mid-read modifications
2. **Concurrent Access**: Multiple readers can operate simultaneously
3. **Version Isolation**: Each reader sees a fixed tree state
4. **Resource Management**: Automatic snapshot cleanup via defer

## Key Operations

### Get Operation
The two-phase Get design:
1. **Tree.Get(version, key)**: Validates inputs, creates snapshot, delegates to reader
2. **TreeReader.Get(key)**: Performs actual traversal with snapshot isolation

This separation allows the Tree to manage metrics and validation while the Reader focuses on traversal logic.

### Tree Traversal
- Handles the full 256-bit key space (64 nibbles)
- Special handling for depth 64 edge case (leaves only at max depth)
- Lazy value loading - leaf values loaded separately to optimize proof generation

## Performance Considerations

- Snapshot creation has minimal overhead in LSM-based storage
- Node loading is optimized for sequential access patterns
- Metrics collection tracks lookup latency and hit rates
- Empty tree checks avoid unnecessary I/O

## Error Handling

Uses the tree's error handlers for retry logic on transient storage failures. This provides resilience against temporary storage issues while maintaining clean error propagation.