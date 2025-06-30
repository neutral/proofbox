# Batch Transaction

This file implements atomic batch operations for the Jellyfish Merkle Tree, allowing multiple Put and Delete operations to be executed as a single atomic transaction.

## Why Batch Transactions?

Batch transactions provide several critical benefits:

1. **Atomicity**: All operations in a batch either succeed together or fail together, preventing partial updates that could leave the tree in an inconsistent state

2. **Performance**: By accumulating operations and executing them in a single version, we reduce overhead from version management and enable optimizations like deduplication

3. **Consistency**: Operations within a batch see each other's changes, providing a consistent view during execution

4. **Optimization Opportunities**: The batch layer enables deduplication of redundant operations and future optimizations like write coalescing

## Design Decisions

### Deduplication Strategy
When the same key is modified multiple times within a batch, only the last operation is retained. This is tracked using the `keyOps` map which stores the index of the latest operation for each key.

### Thread Safety
The BatchTransaction is thread-safe, allowing concurrent calls to BatchPut and BatchDelete. This enables applications to build batches from multiple goroutines safely.

### Execution Model
Batches are executed by:
1. Beginning a new version
2. Optimizing operations (deduplication) before execution
3. Applying all operations to that version
4. Committing the version atomically
5. Clearing the batch on success

If any operation fails, the entire version is aborted, ensuring atomicity.

### Operation Optimization
The OptimizeOperations method (called automatically during Execute) performs:
- Deduplication: Only the final operation per key is retained
- Order preservation: Operations maintain their relative order after deduplication
- Thread-safe optimization: Uses optimizeOperationsLocked to avoid double-locking

## Integration Points

- Uses the tree's existing versioning infrastructure (BeginVersion, PutVersioned, DeleteVersioned, CommitVersion)
- Leverages version abort mechanism for rollback on errors
- Works with the batch optimizer for additional optimizations