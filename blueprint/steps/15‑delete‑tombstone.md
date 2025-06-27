---
id: step.19.delete‑tombstone
depends_on:
  - step.18.persist‑multi‑ver
  - step.14.proof‑neighbor
tags: [deletion, step]
---

## Objective

Support delete via tombstone and its exclusion proof.

## Implements

- **Open Question – Deletion Algorithm** (spec calls for tombstones).
  _What happens_:

  - Writes a tombstoned leaf and produces proper exclusion proof, exercising both persistent‑versioning and proof logic under deletion.

## Technical Details

### Tombstone-Based Deletion

Per `delete-operation.md`, deletion creates a new version without the key while preserving historical accessibility:

1. **Tombstone Approach**
   - Mark leaf as deleted with special tombstone value
   - Maintain node in tree structure temporarily
   - Enables clean version transitions
   - Supports eventual pruning

2. **Tree Structure Changes**
   - Parent nodes updated with new hashes
   - Single-child optimization deferred (keep sparse)
   - Preserves proof generation simplicity
   - Avoids complex tree rebalancing

3. **Proof Generation**
   - Exclusion proof shows key doesn't exist
   - Include neighbor nodes as witnesses
   - Prove non-membership via gap in ordering
   - Same algorithm as empty proof with tombstone awareness

### Deletion Algorithm

```go
type DeleteOperation struct {
    key     []byte
    version uint64
}

func (t *Tree) Delete(key []byte) error {
    // 1. Find leaf node
    leaf, path := t.findLeaf(key)
    if leaf == nil {
        return nil // Already deleted
    }
    
    // 2. Create tombstone
    tombstone := &LeafNode{
        Key:       key,
        Value:     nil,
        Tombstone: true,
        Version:   t.version,
    }
    
    // 3. Update path to root
    batch := NewUpdateBatch(t.version)
    batch.MarkStale(leaf.NodeKey())
    batch.Update(tombstone.NodeKey(), tombstone)
    
    // 4. Propagate changes
    t.propagateDelete(path, batch)
    
    return batch.Flush(t.db)
}
```

### Exclusion Proof After Delete

```go
func (t *Tree) ProveDeleted(key []byte, version uint64) (*ExclusionProof, error) {
    // Current version: key is tombstoned/absent
    currentProof := t.generateExclusionProof(key, version)
    
    // Historical version: key existed
    historicalValue, _ := t.GetAtVersion(key, version-1)
    
    return &ExclusionProof{
        Key:              key,
        CurrentVersion:   version,
        HistoricalValue:  historicalValue,
        NonMembershipProof: currentProof,
    }, nil
}
```

## Implementation Steps

1. **Add Tombstone Support to LeafNode**
   - Add `tombstone bool` field
   - Modify encoding to include tombstone flag
   - Update hash computation for tombstones

2. **Implement Delete Method**
   - Traverse to find target leaf
   - Create tombstone replacement
   - Update parent internal nodes
   - Handle non-existent key gracefully

3. **Update Proof Generation**
   - Recognize tombstones during traversal
   - Generate proper exclusion proofs
   - Include tombstone in proof if encountered

4. **Add Pruning Markers**
   - Track tombstoned nodes for later cleanup
   - Version-based pruning policy
   - Batch tombstone removal

## Testing Requirements

### Functional Tests
- [ ] Delete existing key, verify absence
- [ ] Delete non-existent key is no-op
- [ ] Sequential delete/insert/delete cycles
- [ ] Bulk deletion of multiple keys

### Proof Verification Tests
- [ ] Exclusion proof after delete validates
- [ ] Historical proof shows previous value
- [ ] Tombstone included in proof path
- [ ] Neighbor proofs remain valid

### Multi-Version Tests
- [ ] Key exists at V1, deleted at V2, absent at V3
- [ ] Can prove historical existence
- [ ] Can prove current non-existence
- [ ] Version boundaries handled correctly

### Edge Cases
- [ ] Delete root's only child
- [ ] Delete causes empty tree
- [ ] Delete/insert same key in one batch
- [ ] Concurrent delete operations

## Done When ✓

- [ ] Deleted key absent in new version but provable in old.
- [ ] Tombstone nodes properly marked and stored
- [ ] Exclusion proofs validate correctly
- [ ] Historical queries show pre-delete values
- [ ] Parent nodes updated with new hashes
- [ ] No orphaned nodes in storage
