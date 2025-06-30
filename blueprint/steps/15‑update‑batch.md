---
id: step.15.update‑batch
depends_on:
  - step.14.versioning
tags: [batch, step]
---

## Objective

Return `UpdateBatch{NewNodes, StaleNodeKeys}` from each commit.

## Implements

- **§6.2** end paragraph ("TreeUpdateBatch containing node\*batch + stale*node_index_batch").
  \_What happens*:

  - Counts newly minted vs stale NodeKeys, setting up later Pebble persistence.

## Technical Details

### Update Batch Architecture

The UpdateBatch encapsulates all changes made during a tree update operation. It tracks both new nodes created and old nodes that became stale, enabling atomic commits and efficient storage operations.

## Implementation Steps

1. **Create batch structure**: Define UpdateBatch and builder
2. **Implement transaction support**: Atomic multi-operation updates
3. **Add batch optimization**: Deduplication and compression
4. **Implement parallel processing**: Handle large batches efficiently
5. **Add validation**: Ensure batch integrity
6. **Integrate with storage**: Atomic batch commits

## Testing Requirements

### Basic Batch Tests

### Transaction Tests

### Parallel Batch Tests

## Performance Considerations

- **Batch size optimization**: Balance memory usage vs throughput
- **Parallel processing**: Utilize multiple cores for large batches
- **Deduplication**: Reduce storage for common patterns
- **Compression**: Minimize disk I/O
- **Write coalescing**: Combine small operations

## Security Considerations

- **Atomic commits**: All-or-nothing batch application
- **Validation**: Ensure batch integrity before commit
- **Isolation**: Transactions don't see partial updates
- **Rollback safety**: Clean abort without side effects

## Done When ✓

- [ ] Batch counts equal nodes touched in commit test
- [ ] UpdateBatch tracks all new and stale nodes
- [ ] Transactions provide atomic multi-operation updates
- [ ] Batch optimization reduces storage overhead
- [ ] Parallel processing improves throughput for large batches
- [ ] Validation catches malformed batches
- [ ] Compression significantly reduces batch size
- [ ] 100% test coverage for batch operations
- [ ] gore REPL testing instructions added here to this file
