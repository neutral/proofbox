# Versioned Node Keys Specification

## Overview
Every node in the JMT is identified by a versioned key system that enables append-only storage and efficient historical queries. This is a critical innovation that drastically improves performance on LSM-tree storage engines.

## NodeKey Structure

In the persistent key-value store, each node is stored with a key called `NodeKey`. A `NodeKey` is a tuple `(version, nibble_path)`:

- **version** – The version number (typically a 64-bit integer) when this node was _created_
- **nibble_path** – The sequence of nibbles from the root to this node's position

### Implementation
```rust
pub struct NodeKey {
    version: Version,
    nibble_path: NibblePath,
}
```

## Key Schema
JMT adopts a version-based node key schema by splicing version and nibble path:
```
NodeKey = version || nibble_path
```
Where:
- The node was created at `version`
- `nibble_path` is the sequence of nibbles from root to this node

See [SN-008] for the original specification.

## Examples

1. **Root node** of version 42:
   - `NodeKey(version=42, nibble_path="")`
   - Empty path since it's the root

2. **Child of root** at nibble `A` created in version 42:
   - `NodeKey(version=42, nibble_path="A")`

3. **Deeper node** at path `AB7` created in version 43:
   - `NodeKey(version=43, nibble_path="AB7")`

## Global Uniqueness
At any version, a nibble path itself can uniquely identify a node. Combining version and nibble path pinpoints any node across versions from the whole blockchain history (see [SN-009]).

Key properties:
- No two different nodes share the same NodeKey
- Given a NodeKey, one can determine exactly which version and path the node belongs to

## Storage Benefits

### Append-Only Writes
Because keys are prefixed with version:
- All nodes of version N have keys lexicographically less than nodes of version N+1
- New versions append their nodes rather than overwriting
- This matches LSM-tree's append-friendly nature

### Performance Impact
This schema saves IOPS and disk bandwidth by more than 90% compared to hash-based node keys (see [SN-015]):
- Sequential writes eliminate compaction overhead
- Keys are much shorter (version + few nibbles vs 32-byte hash)
- Sorted order enables efficient range scans

## Persistence Example
When key K2 is inserted at version 2 into a tree that already has K1 from version 1:
- Existing nodes from version 1 remain untouched
- New nodes for version 2 are created with `version=2` in their NodeKey
- The tree at version 2 reuses unchanged branches from version 1
- Both version 1 and version 2 states can be reconstructed