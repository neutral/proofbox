# Tree Health Checker

## Purpose

Provides comprehensive integrity verification for Merkle trees, ensuring data consistency and structural correctness across all nodes at a specific version.

## Design Philosophy

The health checker performs deep structural validation rather than just surface-level checks. It verifies:
1. **Node Accessibility**: All referenced nodes can be loaded from storage
2. **Hash Integrity**: Node hashes can be computed (future: verify against stored hashes)
3. **Structural Consistency**: Tree structure follows JMT rules
4. **Reference Validity**: All child references point to valid nodes

## Key Features

### Cycle Detection
Uses a visited map to prevent infinite loops in case of corrupted data creating cycles. While a properly functioning tree should never have cycles, storage corruption could theoretically create them.

### Context Cancellation
Supports context-based cancellation for long-running integrity checks on large trees. This is crucial for production systems where health checks might need to be aborted.

### Snapshot Isolation
Creates a storage snapshot to ensure consistent reads throughout the verification process, preventing issues from concurrent modifications.

## Use Cases

1. **Post-Recovery Validation**: Verify tree integrity after database recovery
2. **Periodic Health Checks**: Scheduled integrity verification in production
3. **Debugging**: Diagnose tree corruption or inconsistencies
4. **Migration Verification**: Ensure tree integrity after data migrations

## Performance Considerations

- Uses depth-first traversal to minimize memory usage
- Context cancellation allows early termination
- Could be enhanced with:
  - Parallel verification of subtrees
  - Sampling-based verification for large trees
  - Progress reporting for long-running checks

## Future Enhancements

The current implementation verifies basic structural integrity. Future versions could:
- Verify hash chains against stored root hashes
- Check value integrity for leaf nodes
- Validate version consistency across nodes
- Report detailed corruption information for repair tools