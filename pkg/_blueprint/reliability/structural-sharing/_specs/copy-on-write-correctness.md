---
id: copy-on-write-correctness-spec
title: Copy-on-Write Correctness Specification
status: active
created: 2024-01-03
updated: 2024-01-03
tags: [specs, reliability, correctness, copy-on-write, structural-sharing]
---

# Copy-on-Write Correctness Specification

## Overview

This specification details how structural sharing must be correctly implemented in the Jellyfish Merkle Tree to ensure unchanged subtrees share node instances across versions.

## Structural Sharing Mechanics

### Node Storage Model

Each node in the tree is identified by a `NodeKey`:
```go
type NodeKey struct {
    Version    Version    // Version when node was created
    NibblePath NibblePath // Path from root to this node
}
```

### Child References

Internal nodes store references to children:
```go
type Child struct {
    Hash    Hash    // Cryptographic hash of child
    Version Version // Version when child was created
    IsLeaf  bool    // Node type indicator
}
```

### Copy-on-Write Process

When modifying the tree at version V:

1. **Path Identification**: Identify the path from root to the target key
2. **Node Cloning**: Clone only nodes along this path with new version V
3. **Reference Updates**: Update child references in cloned nodes
4. **Unchanged References**: Maintain original version numbers for unchanged children

## Correctness Invariants

### 1. Node Instance Sharing

**Invariant**: For any NodeKey K, all reads of K within the same cache lifetime must return the same node instance.

**Verification**:
```go
func verifyNodeSharing(tree *Tree, key NodeKey) bool {
    node1 := tree.loadNode(key)
    node2 := tree.loadNode(key)
    return &node1 == &node2  // Same memory address
}
```

### 2. Version Monotonicity

**Invariant**: In any path from root to leaf, child.Version ≤ parent.Version

**Rationale**: A parent created at version V can only reference children created at version V or earlier.

### 3. Minimal Cloning

**Invariant**: When updating a single key, exactly log₁₆(N) nodes are cloned (the path from root to leaf).

**Verification**:
```go
func countClonedNodes(oldVersion, newVersion Version) int {
    // Compare node sets between versions
    // Count nodes that exist in both with different versions
}
```

### 4. Reference Preservation

**Invariant**: Unchanged subtrees maintain their original NodeKey references.

**Example**:
```
Version 1: Root(v1) → A(v1) → B(v1) → Leaf1(v1)
                   ↘ C(v1) → Leaf2(v1)

After modifying Leaf1 in Version 2:
Version 2: Root(v2) → A(v2) → B(v2) → Leaf1(v2)
                   ↘ C(v1) → Leaf2(v1)  // C subtree unchanged
```

## Implementation Requirements

### Node Loading

The node loading mechanism MUST:
1. Check cache before storage
2. Return cached instance if present
3. Cache newly loaded nodes
4. Preserve instance identity

### Tree Updater

The TreeUpdater MUST:
1. Track nodes requiring cloning
2. Create new versions only for modified paths
3. Preserve child references to unchanged subtrees
4. Update version numbers correctly

### Cache Behavior

The cache MUST:
1. Use NodeKey as the cache key
2. Return same instance for repeated lookups
3. Handle concurrent access safely
4. Maintain instance identity during lifetime

## Testing Strategy

### Invariant Testing

Test structural sharing by:

1. **Setup**: Create tree with multiple keys
2. **Modify**: Update subset of keys
3. **Verify**:
   - Unchanged paths share node instances
   - Modified paths have new instances
   - Child versions are consistent
   - Cache returns same instances

### Test Implementation

```go
func TestStructuralSharingCorrectness(t *testing.T) {
    tree := NewTree()
    
    // Add initial data
    v1 := tree.Put(Key("A"), Value("1"))
    v1 = tree.Put(Key("B"), Value("2"))
    v1 = tree.Put(Key("C"), Value("3"))
    tree.Commit(v1)
    
    // Modify only key A
    v2 := tree.Put(Key("A"), Value("4"))
    tree.Commit(v2)
    
    // Load nodes for unchanged key B from both versions
    reader1 := tree.Reader(v1)
    reader2 := tree.Reader(v2)
    
    // Traverse to key B and collect nodes
    nodes1 := collectNodesOnPath(reader1, Key("B"))
    nodes2 := collectNodesOnPath(reader2, Key("B"))
    
    // Verify unchanged nodes are same instances
    for i, node1 := range nodes1 {
        node2 := nodes2[i]
        if node1.Version == node2.Version {
            assert.Same(t, node1, node2) // Same instance
        }
    }
}
```

## Performance Implications

Correct structural sharing provides:
1. **Memory Efficiency**: O(log N) new nodes per update
2. **Cache Efficiency**: Shared nodes increase cache hit rate
3. **I/O Reduction**: Fewer unique nodes to load

## Common Pitfalls

1. **Incorrect Cloning**: Cloning unchanged nodes breaks sharing
2. **Cache Misses**: Different instances for same NodeKey violates invariant
3. **Version Confusion**: Incorrect version assignment breaks consistency
4. **Concurrent Modification**: Modifying "immutable" nodes causes corruption

## References

- [Structural Sharing Requirement](../requirement.md)
- [ADR: Structural Sharing Strategy](../../../../tree/_blueprint/_decisions/structural-sharing-strategy.md)
- [Storage Architecture](../../../../specs/storage-architecture.md)