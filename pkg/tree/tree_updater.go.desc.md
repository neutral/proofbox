# Tree Updater

The `tree_updater.go` file implements transactional update operations for pending tree versions.

## Purpose

This component accumulates tree modifications (inserts, updates, deletes) in memory before committing them atomically to storage, providing transaction-like semantics for version creation.

## Key Design Decisions

1. **Write Accumulation**: All modifications are collected in memory as NodeWrite entries before being written to storage in a single batch.

2. **Current Root Tracking**: Maintains the evolving root as operations are applied, ensuring each operation builds on previous ones within the same version.

3. **Path Cloner Integration**: Uses the path cloner for structural sharing when modifying existing nodes.

4. **Atomic Batch Building**: Produces an UpdateBatch that can be committed atomically to storage.

5. **Parallel Processing**: BuildUpdateBatchParallel() automatically activates for batches with 100+ nodes, using concurrent hash computation.

6. **Batch Validation**: ValidateBatch() ensures version consistency and referential integrity before commit.

## New Features (Step 15)

- **Parallel Batch Building**: Automatically uses goroutines for large batches to compute hashes concurrently
- **Enhanced Validation**: Comprehensive checks for batch integrity including:
  - Version consistency across all nodes
  - Stale node version validation
  - Circular reference detection
  - New/stale node overlap prevention

## Why This Design

The tree updater serves several critical purposes:
- **Atomicity**: All changes for a version are committed together or not at all
- **Isolation**: Pending changes are invisible to readers until commit
- **Efficiency**: Batching reduces storage write operations
- **Performance**: Parallel processing improves throughput for large batches
- **Consistency**: The current root tracking ensures operations within a version see each other's effects
- **Reliability**: Validation catches errors before they reach storage

The bug fix applied (using currentRoot instead of oldVersion for each operation) was crucial for maintaining consistency when multiple operations occur within a single version.