---
id: step.11.leaf‑splitting
depends_on:
  - step.10.nibble‑path‑ops
tags: [leaf, splitting, internal, step]
---

## Objective

Implement leaf splitting and internal node creation when inserting keys that differ from existing leaf nodes.

## Implements

- §6.2 Insertion – "Different keys → split into internal node" path
- Tree restructuring to maintain radix-16 structure
- Creation of internal nodes at divergence points

## Technical Details

### Leaf Splitting Algorithm

When inserting a new key that collides with an existing leaf (different keys), the tree must:

1. Find the common prefix between the two keys
2. Create internal nodes for the common path
3. Create a new internal node at the divergence point
4. Place both leaves as children of the divergence node

### Enhanced TreeUpdater Implementation

```go
// Add to TreeUpdater struct in pkg/tree/updater.go
type TreeUpdater struct {
    // ... existing fields ...
    staleNodes []types.NodeKey // Nodes replaced in this update
}

// insertIntoTree handles insertion into existing tree with full logic
func (u *TreeUpdater) insertIntoTree(root types.Node, key types.Key, valueHash types.Hash) (types.Hash, error) {
    switch root.Type() {
    case types.NodeTypeLeaf:
        leaf := root.(*LeafNode)
        if leaf.Key() == key {
            // Same key - update value
            return u.updateLeaf(leaf, valueHash)
        }
        // Different keys - split required
        return u.splitLeafNode(leaf, key, valueHash)
        
    case types.NodeTypeInternal:
        // Navigate down the tree
        return u.insertIntoInternalNode(root.(*InternalNode), key, valueHash, 0)
        
    default:
        return types.Hash{}, fmt.Errorf("unexpected node type: %v", root.Type())
    }
}

// splitLeafNode handles the case where we need to split a leaf
func (u *TreeUpdater) splitLeafNode(existingLeaf *LeafNode, newKey types.Key, 
    newValueHash types.Hash) (types.Hash, error) {
    
    // Get nibble paths for both keys
    existingPath := types.NewNibblePath(existingLeaf.Key()[:])
    newPath := types.NewNibblePath(newKey[:])
    
    // Find where they diverge
    commonLen := existingPath.CommonPrefixLength(newPath)
    
    // Create new leaf for the new key
    newLeaf := &LeafNode{
        key:       newKey,
        valueHash: newValueHash,
        version:   u.newVersion,
    }
    
    // Mark existing leaf as stale
    u.staleNodes = append(u.staleNodes, types.NodeKey{
        Version:    existingLeaf.Version(),
        NibblePath: existingPath,
    })
    
    // Build the tree structure from leaves up to root
    return u.buildTreeFromSplit(existingLeaf, newLeaf, existingPath, newPath, commonLen)
}

// buildTreeFromSplit creates the tree structure after a split
func (u *TreeUpdater) buildTreeFromSplit(existingLeaf *LeafNode, newLeaf *LeafNode,
    existingPath, newPath types.NibblePath, commonLen int) (types.Hash, error) {
    
    // Get the nibbles at divergence point
    existingNibble, err := existingPath.GetNibble(commonLen)
    if err != nil {
        return types.Hash{}, fmt.Errorf("failed to get existing nibble: %w", err)
    }
    
    newNibble, err := newPath.GetNibble(commonLen)
    if err != nil {
        return types.Hash{}, fmt.Errorf("failed to get new nibble: %w", err)
    }
    
    // Create internal node at divergence point
    divergeNode := NewInternalNode(u.newVersion)
    
    // Add both leaves as children
    divergeNode.SetChild(existingNibble, types.Child{
        Hash:    existingLeaf.Hash(),
        Version: u.newVersion,
        IsLeaf:  true,
    })
    
    divergeNode.SetChild(newNibble, types.Child{
        Hash:    newLeaf.Hash(),
        Version: u.newVersion,
        IsLeaf:  true,
    })
    
    // Store the leaves with their full paths
    existingKey := types.NodeKey{
        Version:    u.newVersion,
        NibblePath: existingPath,
    }
    u.nodeWrites[existingKey.String()] = nodeWrite{key: existingKey, node: existingLeaf}
    
    newKey := types.NodeKey{
        Version:    u.newVersion,
        NibblePath: newPath,
    }
    u.nodeWrites[newKey.String()] = nodeWrite{key: newKey, node: newLeaf}
    
    // Now build internal nodes from divergence point up to root
    currentNode := divergeNode
    currentDepth := commonLen
    
    // Build path from divergence to root
    for depth := currentDepth - 1; depth >= 0; depth-- {
        parentNode := NewInternalNode(u.newVersion)
        
        // Get the nibble for this level
        nibble, err := existingPath.GetNibble(depth)
        if err != nil {
            return types.Hash{}, fmt.Errorf("failed to get nibble at depth %d: %w", depth, err)
        }
        
        // Add current node as child
        parentNode.SetChild(nibble, types.Child{
            Hash:    currentNode.Hash(),
            Version: u.newVersion,
            IsLeaf:  false,
        })
        
        // Store current node
        nodeKey := types.NodeKey{
            Version:    u.newVersion,
            NibblePath: existingPath.Prefix(depth + 1),
        }
        u.nodeWrites[nodeKey.String()] = nodeWrite{key: nodeKey, node: currentNode}
        
        currentNode = parentNode
    }
    
    // Store root node
    rootKey := types.RootNodeKey(u.newVersion)
    u.nodeWrites[rootKey.String()] = nodeWrite{key: rootKey, node: currentNode}
    
    return currentNode.Hash(), nil
}

// insertIntoInternalNode navigates internal nodes for insertion
func (u *TreeUpdater) insertIntoInternalNode(node *InternalNode, key types.Key, 
    valueHash types.Hash, depth int) (types.Hash, error) {
    
    if depth >= types.MaxTreeDepth {
        return types.Hash{}, fmt.Errorf("maximum tree depth exceeded")
    }
    
    // Get nibble at current depth
    nibble, err := key.ExtractNibble(depth)
    if err != nil {
        return types.Hash{}, fmt.Errorf("failed to extract nibble at depth %d: %w", depth, err)
    }
    
    // Check if child exists at this nibble
    child, exists := node.Child(nibble)
    if !exists {
        // Empty slot - create new leaf here
        return u.insertLeafIntoEmptySlot(node, key, valueHash, nibble, depth)
    }
    
    // Load child node
    childPath := types.NewNibblePath(key[:]).Prefix(depth + 1)
    childKey := types.NodeKey{
        Version:    child.Version,
        NibblePath: childPath,
    }
    
    childNode, err := u.loadNode(childKey)
    if err != nil {
        return types.Hash{}, fmt.Errorf("failed to load child node: %w", err)
    }
    
    // Recursively insert into child
    var newChildHash types.Hash
    if child.IsLeaf {
        // Child is a leaf - might need to split
        childLeaf := childNode.(*LeafNode)
        if childLeaf.Key() == key {
            newChildHash, err = u.updateLeaf(childLeaf, valueHash)
        } else {
            newChildHash, err = u.splitLeafNode(childLeaf, key, valueHash)
        }
    } else {
        // Child is internal - continue navigation
        newChildHash, err = u.insertIntoInternalNode(childNode.(*InternalNode), 
            key, valueHash, depth+1)
    }
    
    if err != nil {
        return types.Hash{}, err
    }
    
    // Create new version of current node with updated child
    return u.updateInternalNodeChild(node, nibble, newChildHash, depth)
}

// insertLeafIntoEmptySlot adds a new leaf to an empty child slot
func (u *TreeUpdater) insertLeafIntoEmptySlot(parent *InternalNode, key types.Key,
    valueHash types.Hash, nibble types.Nibble, parentDepth int) (types.Hash, error) {
    
    // Create new leaf
    leaf := &LeafNode{
        key:       key,
        valueHash: valueHash,
        version:   u.newVersion,
    }
    
    // Store leaf
    leafPath := types.NewNibblePath(key[:])
    leafKey := types.NodeKey{
        Version:    u.newVersion,
        NibblePath: leafPath,
    }
    u.nodeWrites[leafKey.String()] = nodeWrite{key: leafKey, node: leaf}
    
    // Create new version of parent with added child
    newParent := parent.Clone(u.newVersion).(*InternalNode)
    newParent.SetChild(nibble, types.Child{
        Hash:    leaf.Hash(),
        Version: u.newVersion,
        IsLeaf:  true,
    })
    
    // Store updated parent
    parentPath := leafPath.Prefix(parentDepth)
    parentKey := types.NodeKey{
        Version:    u.newVersion,
        NibblePath: parentPath,
    }
    u.nodeWrites[parentKey.String()] = nodeWrite{key: parentKey, node: newParent}
    
    // Mark old parent as stale
    u.staleNodes = append(u.staleNodes, types.NodeKey{
        Version:    parent.Version(),
        NibblePath: parentPath,
    })
    
    return newParent.Hash(), nil
}

// updateInternalNodeChild creates new version of internal node with updated child
func (u *TreeUpdater) updateInternalNodeChild(node *InternalNode, childNibble types.Nibble,
    childHash types.Hash, depth int) (types.Hash, error) {
    
    // Clone node with new version
    newNode := node.Clone(u.newVersion).(*InternalNode)
    
    // Update the specific child
    newNode.SetChild(childNibble, types.Child{
        Hash:    childHash,
        Version: u.newVersion,
        IsLeaf:  false, // Will be corrected based on actual child type
    })
    
    // Store new node
    nodePath := types.NewNibblePath(make([]byte, 32)).Prefix(depth)
    nodeKey := types.NodeKey{
        Version:    u.newVersion,
        NibblePath: nodePath,
    }
    u.nodeWrites[nodeKey.String()] = nodeWrite{key: nodeKey, node: newNode}
    
    // Mark old node as stale
    u.staleNodes = append(u.staleNodes, types.NodeKey{
        Version:    node.Version(),
        NibblePath: nodePath,
    })
    
    return newNode.Hash(), nil
}
```

### Tree Structure Validation

```go
// ValidateTreeStructure ensures the tree maintains proper invariants
func ValidateTreeStructure(tree *Tree, version types.Version) error {
    rootHash, err := tree.GetRootHash(version)
    if err != nil {
        return fmt.Errorf("failed to get root hash: %w", err)
    }
    
    if rootHash == types.EmptyHash() {
        return nil // Empty tree is valid
    }
    
    root, err := tree.loadNode(types.RootNodeKey(version))
    if err != nil {
        return fmt.Errorf("failed to load root: %w", err)
    }
    
    visited := make(map[types.NodeKey]bool)
    return validateNode(tree, root, version, types.NibblePath{}, visited)
}

func validateNode(tree *Tree, node types.Node, version types.Version,
    path types.NibblePath, visited map[types.NodeKey]bool) error {
    
    nodeKey := types.NodeKey{Version: version, NibblePath: path}
    if visited[nodeKey] {
        return fmt.Errorf("cycle detected at %v", nodeKey)
    }
    visited[nodeKey] = true
    
    switch n := node.(type) {
    case *LeafNode:
        // Verify leaf path matches its key
        keyPath := types.NewNibblePath(n.Key()[:])
        if !path.Equals(keyPath) {
            return fmt.Errorf("leaf path mismatch: expected %v, got %v", keyPath, path)
        }
        
    case *InternalNode:
        // Check minimum children (except root)
        numChildren := n.NumChildren()
        if path.Length > 0 && numChildren < 2 {
            return fmt.Errorf("internal node at %v has only %d children", path, numChildren)
        }
        
        // Validate all children
        for nibble := types.Nibble(0); nibble <= 15; nibble++ {
            if child, exists := n.Child(nibble); exists {
                childPath := path.Append(nibble)
                childKey := types.NodeKey{
                    Version:    child.Version,
                    NibblePath: childPath,
                }
                
                childNode, err := tree.loadNode(childKey)
                if err != nil {
                    return fmt.Errorf("failed to load child at %v: %w", childPath, err)
                }
                
                if err := validateNode(tree, childNode, child.Version, childPath, visited); err != nil {
                    return err
                }
            }
        }
    }
    
    return nil
}
```

## Testing Requirements

### Leaf Splitting Tests

```go
func TestBasicLeafSplitting(t *testing.T) {
    tree := createTestTree(t)
    
    // Insert first key
    key1 := types.Key{0x10, 0x20} // Nibbles: 1,0,2,0,...
    value1 := []byte("value1")
    v1, err := tree.Put(key1, value1)
    require.NoError(t, err)
    
    // Insert second key that will cause split
    key2 := types.Key{0x10, 0x30} // Nibbles: 1,0,3,0,... (diverges at index 2)
    value2 := []byte("value2")
    v2, err := tree.Put(key2, value2)
    require.NoError(t, err)
    
    // Verify both keys exist
    got1, err := tree.Get(v2, key1)
    require.NoError(t, err)
    assert.Equal(t, value1, got1)
    
    got2, err := tree.Get(v2, key2)
    require.NoError(t, err)
    assert.Equal(t, value2, got2)
    
    // Verify tree structure
    root, err := tree.loadNode(types.RootNodeKey(v2))
    require.NoError(t, err)
    
    // Root should be internal node
    internal, ok := root.(*InternalNode)
    require.True(t, ok, "Root should be internal after split")
    
    // Should have exactly one child at nibble 1
    childCount := 0
    for n := types.Nibble(0); n <= 15; n++ {
        if _, exists := internal.Child(n); exists {
            childCount++
        }
    }
    assert.Equal(t, 1, childCount)
}

func TestMultiLevelSplitting(t *testing.T) {
    tree := createTestTree(t)
    
    // Insert keys that will create multi-level tree
    keys := []types.Key{
        {0x00, 0x00}, // 0,0,0,0,...
        {0x00, 0x01}, // 0,0,0,1,... (split at depth 3)
        {0x00, 0x10}, // 0,0,1,0,... (split at depth 2)
        {0x01, 0x00}, // 0,1,0,0,... (split at depth 1)
        {0x10, 0x00}, // 1,0,0,0,... (split at depth 0)
    }
    
    var version types.Version
    for i, key := range keys {
        v, err := tree.Put(key, []byte(fmt.Sprintf("value%d", i)))
        require.NoError(t, err)
        version = v
    }
    
    // Validate final tree structure
    err := ValidateTreeStructure(tree, version)
    assert.NoError(t, err)
    
    // Verify all keys are accessible
    for i, key := range keys {
        value, err := tree.Get(version, key)
        require.NoError(t, err)
        assert.Equal(t, []byte(fmt.Sprintf("value%d", i)), value)
    }
}

func TestSplitWithExistingInternal(t *testing.T) {
    tree := createTestTree(t)
    
    // Build initial structure
    tree.Put(types.Key{0x10, 0x00}, []byte("A"))
    tree.Put(types.Key{0x10, 0x10}, []byte("B"))
    v1, _ := tree.Put(types.Key{0x20, 0x00}, []byte("C"))
    
    // Insert key that splits under existing internal node
    key := types.Key{0x10, 0x01} // Will split with 0x10,0x00
    v2, err := tree.Put(key, []byte("D"))
    require.NoError(t, err)
    
    // Verify structure maintained
    err = ValidateTreeStructure(tree, v2)
    assert.NoError(t, err)
    
    // All keys should be accessible
    assert.NotNil(t, mustGet(t, tree, v2, types.Key{0x10, 0x00}))
    assert.NotNil(t, mustGet(t, tree, v2, types.Key{0x10, 0x01}))
    assert.NotNil(t, mustGet(t, tree, v2, types.Key{0x10, 0x10}))
    assert.NotNil(t, mustGet(t, tree, v2, types.Key{0x20, 0x00}))
}

func TestStaleNodeTracking(t *testing.T) {
    tree := createTestTree(t)
    
    // Capture stale nodes during update
    var capturedStale []types.NodeKey
    tree.beforeCommit = func(updater *TreeUpdater) {
        capturedStale = updater.staleNodes
    }
    
    // Initial insert
    key1 := types.Key{0x10}
    tree.Put(key1, []byte("v1"))
    
    // Clear captured state
    capturedStale = nil
    
    // Insert that causes split
    key2 := types.Key{0x20}
    tree.Put(key2, []byte("v2"))
    
    // Should have marked the old root (leaf) as stale
    assert.Greater(t, len(capturedStale), 0, "Should track stale nodes during split")
    
    // The old leaf should be in stale list
    found := false
    for _, stale := range capturedStale {
        if stale.NibblePath.Equals(types.NewNibblePath(key1[:])) {
            found = true
            break
        }
    }
    assert.True(t, found, "Original leaf should be marked stale")
}
```

### Edge Case Tests

```go
func TestIdenticalPrefixSplit(t *testing.T) {
    tree := createTestTree(t)
    
    // Keys with long common prefix
    key1 := types.Key{}
    key1[31] = 0x01 // Differs only in last byte
    
    key2 := types.Key{}
    key2[31] = 0x02
    
    tree.Put(key1, []byte("v1"))
    v2, err := tree.Put(key2, []byte("v2"))
    require.NoError(t, err)
    
    // Should create deep tree with internal nodes
    // all the way down to nibble 62 (31 bytes * 2 nibbles - 2)
    err = ValidateTreeStructure(tree, v2)
    assert.NoError(t, err)
}

func TestMaxDepthSplit(t *testing.T) {
    tree := createTestTree(t)
    
    // Create keys that differ at maximum depth
    key1 := types.Key{}
    key2 := types.Key{}
    
    // Make them differ only in the last nibble
    for i := 0; i < 31; i++ {
        key1[i] = 0xFF
        key2[i] = 0xFF
    }
    key1[31] = 0xF0 // Last nibbles: F,0
    key2[31] = 0xF1 // Last nibbles: F,1
    
    tree.Put(key1, []byte("v1"))
    v2, err := tree.Put(key2, []byte("v2"))
    require.NoError(t, err)
    
    // Both should be accessible
    assert.NotNil(t, mustGet(t, tree, v2, key1))
    assert.NotNil(t, mustGet(t, tree, v2, key2))
}
```

### Performance Tests

```go
func BenchmarkLeafSplitting(b *testing.B) {
    tree := createTestTree(b)
    
    // Pre-insert base key
    baseKey := types.Key{0x10}
    tree.Put(baseKey, []byte("base"))
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        // Create key that will split with base
        key := types.Key{0x10}
        key[1] = byte(i % 256)
        
        tree.Put(key, []byte("value"))
    }
}

func BenchmarkDeepTreeInsertion(b *testing.B) {
    tree := createTestTree(b)
    
    // Build a deep tree
    for i := 0; i < 1000; i++ {
        key := make([]byte, 32)
        binary.BigEndian.PutUint32(key[28:], uint32(i))
        tree.Put(types.Key(key), []byte("value"))
    }
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        key := make([]byte, 32)
        binary.BigEndian.PutUint32(key[28:], uint32(1000+i))
        tree.Put(types.Key(key), []byte("value"))
    }
}
```

## Implementation Steps

1. Add `staleNodes` field to TreeUpdater
2. Implement `splitLeafNode` method
3. Implement `buildTreeFromSplit` for creating internal nodes
4. Update `insertIntoTree` to handle splitting
5. Implement `insertIntoInternalNode` for navigation
6. Add comprehensive test coverage
7. Add tree structure validation
8. Performance optimization

## Performance Considerations

- Minimize node creation during splits
- Reuse existing nodes where possible
- Batch node writes for efficiency
- Track stale nodes for garbage collection
- Optimize common prefix calculation

## Done When ✓

- [ ] Leaf nodes split correctly when keys differ
- [ ] Internal nodes created at proper divergence points
- [ ] Tree maintains radix-16 structure after splits
- [ ] All ancestor nodes updated with new version
- [ ] Stale nodes properly tracked
- [ ] Tree structure validation passes
- [ ] No nodes have single children (except root)
- [ ] Performance benchmarks show acceptable overhead
- [ ] 100% test coverage for splitting scenarios