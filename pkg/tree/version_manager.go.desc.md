# Version Manager

The `version_manager.go` file implements the core version lifecycle management for the Jellyfish Merkle Tree.

## Purpose

This component manages the state transitions of tree versions from creation through commit or abort, ensuring version consistency and preventing conflicts in concurrent scenarios.

## Key Design Decisions

1. **Version States**: Uses explicit state tracking (pending, committed, aborted) to enforce proper lifecycle transitions and prevent invalid operations.

2. **Concurrent Version Support**: Multiple versions can be in pending state simultaneously, allowing for optimistic concurrency patterns.

3. **Parent Version Tracking**: Each version tracks its parent to enable structural sharing and ensure versions build upon committed states.

4. **Thread Safety**: All operations are protected by mutex to support concurrent access patterns.

5. **Garbage Collection**: Configurable retention policies prevent unbounded memory growth:
   - `RetentionPolicyCount`: Keep last N versions
   - `RetentionPolicyTime`: Keep versions newer than specified duration
   - `RetentionPolicyNone`: Keep all versions (no GC)
   - Default retention: Last 100 versions

## Why This Design

The version manager acts as a central coordinator for multi-version operations, preventing common issues like:
- Creating versions from uncommitted parents
- Double commits or aborts
- Version number conflicts
- Lost updates in concurrent scenarios
- Unbounded memory growth from version accumulation

By centralizing version state management, the tree implementation can focus on the actual data structure operations while relying on the version manager for lifecycle enforcement.

## Locking Hierarchy

The version manager's mutex must be acquired after the tree's writeMu but before the cache mutex to prevent deadlocks. See `locking.md` for the complete hierarchy.