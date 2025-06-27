# Versioning System Specification

## Overview
The JMT supports persistent versioning, allowing multiple versions of the state to coexist. Each update produces a new version with its own Merkle root.

## Version Model
- **Version**: Monotonically increasing integer (typically 64-bit)
- **Interpretation**: Often block height or transaction index
- **Append-only**: Never overwrites nodes, only creates new ones

## Persistent Data Structure
The tree implements persistence through structural sharing (see [SN-010]):
- New tree reuses unchanged portions from previous versions
- Only modified paths create new nodes
- Forms a persistent (immutable) data structure

## Version Creation Process
When creating version `v` from version `v-1`:
1. Start with root of version `v-1`
2. Apply all changes (inserts/updates/deletes)
3. Create new nodes only for modified paths
4. New nodes get `version = v` in their NodeKey
5. Result: New root for version `v`

## Storage Layout
Due to version-prefixed keys:
```
Version 1: (1,path) -> node_data
Version 2: (2,path) -> node_data  
Version 3: (3,path) -> node_data
...
```
All nodes of version N stored before version N+1.

## Multi-Version Benefits

### Historic Queries
- Can reconstruct state at any past version
- Useful for:
  - Audit trails
  - State synchronization
  - Time-travel debugging
  - Regulatory compliance

### Efficient Snapshots
- Each version's root hash is complete snapshot
- No need to copy entire state
- Just store root hash + version number

### Concurrent Reads
- Readers can access old versions without locks
- Writers only create new version
- Natural MVCC (Multi-Version Concurrency Control)

## Pruning Considerations
Long-term storage requires pruning strategy:
- Keep last N versions
- Prune at checkpoint boundaries
- Archive old versions separately
- Balance history needs vs storage costs

## Example Usage
```rust
// Version represents blockchain height
let version_100 = tree.commit_version(100, changes);
let version_101 = tree.commit_version(101, more_changes);

// Can still query old versions
let old_value = tree.get(key, version_100)?;
let new_value = tree.get(key, version_101)?;
```