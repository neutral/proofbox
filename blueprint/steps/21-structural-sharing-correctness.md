---
id: step.21-structural-sharing-correctness
depends_on:
  - step.20-fuzzing
  - spec.structural-sharing-requirement
  - spec.copy-on-write-correctness
tags: [correctness, structural-sharing, immutability]
---

# Step 21: Structural Sharing Correctness Implementation

## Objective

Implement true node immutability and verify structural sharing correctness to ensure unchanged subtrees share the exact same node instances across versions, as specified in the [Structural Sharing Requirement](../global/reliability/structural-sharing/requirement.md).

## Current State Analysis

### What's Already Implemented

1. **Node Caching** (`pkg/tree/cache.go`)
   - LRU cache with thread-safe access
   - Caches nodes by NodeKey
   - Returns cached instances on hits

2. **Copy-on-Write in TreeUpdater**
   - `cloneInternalNode()` creates new versions of nodes
   - Preserves child references to unchanged subtrees
   - Uses PathCloner to track modifications

3. **Version Tracking**
   - Child references include version numbers
   - Enables cross-version sharing

### Critical Issues

1. **Node Mutability Violation [RSS-001]**
   - `InternalNode.SetChild()` and `RemoveChild()` mutate nodes after creation
   - `LeafNode.SetValue()` mutates leaf nodes
   - Violates immutability requirement

2. **Cache Corruption Risk [RSS-003]**
   - Cached nodes can be modified via setter methods
   - Could lead to incorrect tree state across versions

3. **Missing Verification**
   - No tests verify instance sharing
   - No invariant checks for structural sharing

## Tasks

### 1. Make Nodes Immutable

- [ ] Remove `SetChild()` and `RemoveChild()` from InternalNode
- [ ] Remove `SetValue()` from LeafNode
- [ ] Make all node fields private and final
- [ ] Update node creation to set all data at construction time
- [ ] Ensure hash is computed once during construction

### 2. Update TreeUpdater for Immutability

- [ ] Modify `cloneInternalNode()` to create nodes with desired state
- [ ] Update child modification logic to create new nodes
- [ ] Remove any code that mutates nodes after creation
- [ ] Ensure all node updates create new instances

### 3. Fix Node Loading

- [ ] Update codec to create fully-initialized nodes
- [ ] Remove reliance on `SetValue()` when loading from storage
- [ ] Ensure loaded nodes are immediately immutable

### 4. Add Structural Sharing Tests

- [ ] Create `pkg/tree/structural_sharing_test.go`
- [ ] Test that unchanged subtrees share exact instances
- [ ] Verify child.Version <= parent.Version invariant
- [ ] Test cache returns same instance for same NodeKey
- [ ] Test minimal cloning (only modified path)

### 5. Add Fuzzing Invariants

- [ ] Create `pkg/fuzz/invariants/structural_sharing.go`
- [ ] Implement instance tracking during operations
- [ ] Verify unchanged nodes maintain same memory address
- [ ] Add to property test suite

## Implementation Details

### Immutable Node Design

```go
// InternalNode - all fields private and immutable
type InternalNode struct {
    children map[types.Nibble]types.Child // Set at construction
    version  types.Version                // Never changes
    hash     types.Hash                   // Computed once
}

// Constructor creates fully-initialized node
func NewInternalNodeWithChildren(version types.Version, children map[types.Nibble]types.Child) *InternalNode {
    node := &InternalNode{
        children: children,
        version:  version,
    }
    node.hash = node.computeHash() // Compute once
    return node
}
```

### TreeUpdater Changes

```go
// Instead of cloning then mutating:
// cloned := node.Clone(version)
// cloned.SetChild(nibble, child) // WRONG!

// Create new node with desired state:
newChildren := make(map[types.Nibble]types.Child)
for k, v := range node.Children() {
    newChildren[k] = v
}
newChildren[nibble] = child
newNode := NewInternalNodeWithChildren(version, newChildren)
```

## Acceptance Criteria

1. **Immutability Tests Pass**
   - Nodes have no public mutation methods
   - All fields are effectively immutable
   - Hash is computed exactly once

2. **Structural Sharing Tests Pass**
   - Unchanged subtrees share exact same instances
   - Cache returns identical instances for same NodeKey
   - Only modified path nodes are cloned

3. **Fuzzing Invariants Pass**
   - 1000+ iterations without violations
   - Instance tracking confirms sharing
   - Version monotonicity maintained

4. **Performance Maintained**
   - No regression in benchmark results
   - Memory usage remains efficient
   - Cache hit rate > 80% for typical workloads

5. **Backward Compatibility**
   - Existing trees can be loaded
   - No data migration required
   - API changes are minimal

## Testing Strategy

1. **Unit Tests**: Verify individual node immutability
2. **Integration Tests**: Test tree operations with immutable nodes
3. **Property Tests**: Fuzz with structural sharing invariants
4. **Performance Tests**: Ensure no regression

## References

- [Structural Sharing Requirement](../global/reliability/structural-sharing/requirement.md)
- [Copy-on-Write Correctness Spec](../global/reliability/structural-sharing/specs/copy-on-write-correctness.md)
- [ADR-014: Structural Sharing Strategy](../decisions/014-structural-sharing-strategy.md)