# Put Operation

The Put method implements key-value insertion and updates for the Jellyfish Merkle Tree.

## Purpose

Put is the primary write operation for the tree, handling:
- Input validation
- Version management
- Atomic batch commits
- In-memory state updates

## Design Principles

1. **Write Serialization**: All Put operations are serialized using `writeMu` to ensure consistency and prevent concurrent modifications.

2. **Atomic Operations**: Each Put creates a new version with all changes committed atomically via PebbleDB batch operations.

3. **Fail-Safe**: If any part of the operation fails, the entire operation is rolled back, maintaining tree consistency.

4. **Version Monotonicity**: Versions always increment by 1, with overflow protection.

## Operation Flow

1. Validate inputs (key non-empty, value size within limits)
2. Acquire write lock
3. Create new version number
4. Delegate to TreeUpdater for tree modification logic
5. Write all changes to batch
6. Store new root hash
7. Commit batch atomically
8. Update in-memory version tracking

## Error Handling

- Input validation errors return immediately without side effects
- Storage errors cause rollback via batch.Close() without commit
- Version overflow is detected and prevented
- All errors follow patterns from step 03 error handling