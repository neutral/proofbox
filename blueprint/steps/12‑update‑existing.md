---
id: step.12.update‑existing
depends_on:
  - step.11.leaf‑splitting
tags: [update, step]
---

## Objective

Handle insertions that update an existing key's value.

## Implements

- §6.2 Insertion – "Existing leaf, same key → update" path.
  _What happens_:

  - Tests that digest changes but topology is stable; this ensures clients can detect updates via root‑hash change.

## Technical Details

### Update Existing Key Algorithm

When inserting a key that already exists, the tree must:

1. Navigate to the existing leaf node
2. Create a new leaf with updated value (maintaining immutability)
3. Update all ancestors up to root with new version
4. Ensure tree structure remains unchanged (only hashes update)

### Enhanced TreeUpdater Implementation

```go
// insertIntoTree handles insertion into existing tree 
// Note: Leaf splitting is implemented in step 11
func (u *TreeUpdater) insertIntoTree(root Node, key Key, valueHash Hash) (Hash, error) {
    switch node := root.(type) {
    case *LeafNode:
        if node.Key == key {
            // Update existing key - same structure, new value
            return u.updateLeaf(node, valueHash)
        }
        // Different key - delegate to leaf splitting (step 11)
        return u.splitLeafNode(node, key, valueHash)

    case *InternalNode:
        // Navigate down the tree
        return u.insertIntoInternalNode(node, key, valueHash, 0)

    default:
        return Hash{}, fmt.Errorf("unexpected node type: %T", node)
    }
}

// updateLeaf creates new leaf version preserving tree structure
func (u *TreeUpdater) updateLeaf(oldLeaf *LeafNode, newValueHash Hash) (Hash, error) {
    // Check if value actually changed
    if oldLeaf.ValueHash == newValueHash {
        // No change needed - return existing hash
        return oldLeaf.Hash(), nil
    }

    // Create new leaf with updated value
    newLeaf := &LeafNode{
        Key:       oldLeaf.Key,
        ValueHash: newValueHash,
    }
    newLeaf.SetVersion(u.newVersion)

    // Determine node key based on path
    nodeKey := NodeKey{
        Version: u.newVersion,
        Path:    oldLeaf.GetPath(),
    }

    // Mark old node as stale
    u.staleNodes = append(u.staleNodes, NodeKey{
        Version: oldLeaf.Version(),
        Path:    oldLeaf.GetPath(),
    })

    // Store new leaf
    u.nodeWrites[nodeKey] = newLeaf

    // Return new hash (structure unchanged, only hash differs)
    return newLeaf.Hash(), nil
}

// Path copying for version management

// insertIntoInternalNode navigates internal nodes for insertion
func (u *TreeUpdater) insertIntoInternalNode(node *InternalNode, key Key,
    valueHash Hash, path NibblePath, depth int) (Hash, error) {

    if depth >= MaxKeyNibbles {
        return Hash{}, errors.New("maximum tree depth exceeded")
    }

    // Get next nibble in path
    nibble := path.GetNibble(depth)

    // Check if child exists
    child, exists := node.GetChild(nibble)
    if !exists {
        // Empty slot - create new leaf
        return u.createLeafInSlot(node, key, valueHash, path, depth, nibble)
    }

    // Load child node
    childNode, err := u.loadNode(NodeKey{
        Version: child.Version,
        Path:    path.Prefix(depth + 1),
    })
    if err != nil {
        return Hash{}, fmt.Errorf("failed to load child: %w", err)
    }

    // Recursively insert into child
    childHash, err := u.insertIntoTree(childNode, key, valueHash)
    if err != nil {
        return Hash{}, err
    }

    // Create new version of internal node with updated child
    return u.updateInternalNode(node, nibble, childHash, path.Prefix(depth))
}

// updateInternalNode creates new version with updated child
func (u *TreeUpdater) updateInternalNode(oldNode *InternalNode,
    childNibble Nibble, childHash Hash, nodePath NibblePath) (Hash, error) {

    // Create new internal node
    newNode := &InternalNode{}
    newNode.SetVersion(u.newVersion)

    // Copy all children
    for nibble := Nibble(0); nibble < 16; nibble++ {
        if child, exists := oldNode.GetChild(nibble); exists {
            if nibble == childNibble {
                // Update the modified child
                newNode.SetChild(nibble, nil, u.newVersion)
                newNode.children[nibble].Hash = childHash
            } else {
                // Copy existing child
                newNode.SetChild(nibble, nil, child.Version)
                newNode.children[nibble].Hash = child.Hash
                newNode.children[nibble].IsLeaf = child.IsLeaf
            }
        }
    }

    // Store new node
    nodeKey := NodeKey{
        Version: u.newVersion,
        Path:    nodePath,
    }
    u.nodeWrites[nodeKey] = newNode

    // Mark old node as stale
    u.staleNodes = append(u.staleNodes, NodeKey{
        Version: oldNode.Version(),
        Path:    nodePath,
    })

    return newNode.Hash(), nil
}

// createLeafInSlot creates new leaf in empty child slot
func (u *TreeUpdater) createLeafInSlot(parent *InternalNode, key Key,
    valueHash Hash, path NibblePath, depth int, nibble Nibble) (Hash, error) {

    // Create new leaf
    leaf := &LeafNode{
        Key:       key,
        ValueHash: valueHash,
    }
    leaf.SetVersion(u.newVersion)

    // Store leaf
    leafPath := path.Prefix(depth + 1)
    leafKey := NodeKey{
        Version: u.newVersion,
        Path:    leafPath,
    }
    u.nodeWrites[leafKey] = leaf

    // Update parent to include new leaf
    return u.updateInternalNode(parent, nibble, leaf.Hash(), path.Prefix(depth))
}
```

### Path Copying and Version Management

When updating an existing key, the tree must maintain immutability by creating new versions of all nodes along the path from the updated leaf to the root. This ensures previous versions remain accessible.

### Tree Restructuring

```go
// UpdateBatch tracks all changes in a single update
type UpdateBatch struct {
    NewNodes      map[NodeKey]Node
    StaleNodeKeys []NodeKey
    NewRootHash   Hash
    OldVersion    Version
    NewVersion    Version
}

// BuildUpdateBatch constructs batch with all tree changes
func (u *TreeUpdater) BuildUpdateBatch() (*UpdateBatch, error) {
    batch := &UpdateBatch{
        NewNodes:      u.nodeWrites,
        StaleNodeKeys: u.staleNodes,
        OldVersion:    u.oldVersion,
        NewVersion:    u.newVersion,
    }

    // Find root hash
    rootKey := NodeKey{Version: u.newVersion, Path: NibblePath{}}
    if root, exists := u.nodeWrites[rootKey]; exists {
        batch.NewRootHash = root.Hash()
    } else {
        return nil, errors.New("no root found in update batch")
    }

    return batch, nil
}

// ValidateStructuralIntegrity ensures tree structure is valid
func (t *Tree) ValidateStructuralIntegrity(version Version) error {
    rootHash, err := t.GetRootHash(version)
    if err != nil {
        return fmt.Errorf("failed to get root hash: %w", err)
    }

    if rootHash == EmptyHash {
        // Empty tree is valid
        return nil
    }

    // Load and validate root
    root, err := t.loadNode(RootNodeKey(version))
    if err != nil {
        return fmt.Errorf("failed to load root: %w", err)
    }

    visited := make(map[NodeKey]bool)
    return t.validateNode(root, version, NibblePath{}, visited)
}

func (t *Tree) validateNode(node Node, version Version,
    path NibblePath, visited map[NodeKey]bool) error {

    nodeKey := NodeKey{Version: version, Path: path}
    if visited[nodeKey] {
        return fmt.Errorf("cycle detected at %v", nodeKey)
    }
    visited[nodeKey] = true

    switch n := node.(type) {
    case *LeafNode:
        // Verify leaf path matches key
        keyPath := NewNibblePath(n.Key[:])
        if !path.Equals(keyPath.Prefix(len(path.nibbles))) {
            return fmt.Errorf("leaf path mismatch at %v", nodeKey)
        }

    case *InternalNode:
        childCount := 0
        for nibble := Nibble(0); nibble < 16; nibble++ {
            if child, exists := n.GetChild(nibble); exists {
                childCount++

                // Load and validate child
                childPath := append(path.nibbles, nibble)
                childKey := NodeKey{
                    Version: child.Version,
                    Path:    NibblePath{nibbles: childPath},
                }

                childNode, err := t.loadNode(childKey)
                if err != nil {
                    return fmt.Errorf("failed to load child %v: %w", childKey, err)
                }

                if err := t.validateNode(childNode, child.Version,
                    childKey.Path, visited); err != nil {
                    return err
                }
            }
        }

        // Internal nodes should have at least 2 children
        if childCount < 2 && len(path.nibbles) > 0 {
            return fmt.Errorf("internal node has < 2 children at %v", nodeKey)
        }
    }

    return nil
}
```

## Implementation Steps

1. **Enhance update detection**: Identify when inserting existing key
2. **Implement path copying**: Create new nodes along update path
3. **Handle collision cases**: Split leaves with different keys
4. **Manage internal nodes**: Update children while preserving structure
5. **Track stale nodes**: Identify nodes replaced in new version
6. **Validate tree integrity**: Ensure structure remains valid

## Testing Requirements

### Update Detection Tests

```go
func TestUpdateExistingKeyDetection(t *testing.T) {
    tree := createTestTree(t)

    key := KeyHash([]byte("update-key"))

    // Initial insert
    v1, _ := tree.Put(key, []byte("value1"))

    // Track tree structure before update
    structure1 := captureTreeStructure(tree, v1)

    // Update same key
    v2, _ := tree.Put(key, []byte("value2"))

    // Track tree structure after update
    structure2 := captureTreeStructure(tree, v2)

    // Verify structure unchanged
    if !structure1.Equals(structure2) {
        t.Error("Tree structure should remain identical for key update")
    }

    // Verify root hash changed
    hash1, _ := tree.GetRootHash(v1)
    hash2, _ := tree.GetRootHash(v2)

    if hash1 == hash2 {
        t.Error("Root hash must change when value updated")
    }
}

func TestCollisionHandling(t *testing.T) {
    tree := createTestTree(t)

    // Insert keys that will collide
    key1 := Key{0x12, 0x34} // Nibbles: 1,2,3,4
    key2 := Key{0x12, 0x35} // Nibbles: 1,2,3,5 (diverge at position 3)

    v1, _ := tree.Put(key1, []byte("value1"))
    v2, _ := tree.Put(key2, []byte("value2"))

    // Verify both keys exist
    val1, _ := tree.Get(v2, key1)
    val2, _ := tree.Get(v2, key2)

    if !bytes.Equal(val1, []byte("value1")) {
        t.Error("First key value incorrect after collision")
    }
    if !bytes.Equal(val2, []byte("value2")) {
        t.Error("Second key value incorrect")
    }

    // Verify internal node created at divergence
    root, _ := tree.loadNode(RootNodeKey(v2))
    internal, ok := root.(*InternalNode)
    if !ok {
        t.Fatal("Root should be internal node after collision")
    }

    // Should have exactly one internal node with 2 children
    childCount := 0
    for i := Nibble(0); i < 16; i++ {
        if _, exists := internal.GetChild(i); exists {
            childCount++
        }
    }

    if childCount < 2 {
        t.Errorf("Internal node should have at least 2 children, got %d", childCount)
    }
}

func TestBatchUpdateTracking(t *testing.T) {
    tree := createTestTree(t)

    // Insert initial data
    key := KeyHash([]byte("batch-key"))
    v1, _ := tree.Put(key, []byte("value1"))

    // Capture update batch for second version
    var capturedBatch *UpdateBatch
    tree.beforeCommit = func(batch *UpdateBatch) {
        capturedBatch = batch
    }

    v2, _ := tree.Put(key, []byte("value2"))

    // Verify batch contents
    if capturedBatch == nil {
        t.Fatal("Update batch not captured")
    }

    if len(capturedBatch.NewNodes) == 0 {
        t.Error("Batch should contain new nodes")
    }

    if len(capturedBatch.StaleNodeKeys) == 0 {
        t.Error("Batch should mark old nodes as stale")
    }

    if capturedBatch.OldVersion != v1 {
        t.Errorf("Old version incorrect: got %d, want %d", capturedBatch.OldVersion, v1)
    }

    if capturedBatch.NewVersion != v2 {
        t.Errorf("New version incorrect: got %d, want %d", capturedBatch.NewVersion, v2)
    }
}
```

### Structural Integrity Tests

```go
func TestTreeStructuralIntegrity(t *testing.T) {
    tree := createTestTree(t)

    // Build tree with multiple collisions
    keys := []Key{
        {0x00, 0x00},
        {0x00, 0x01},
        {0x00, 0x10},
        {0x00, 0x11},
        {0x01, 0x00},
        {0x01, 0x01},
    }

    var version Version
    for i, key := range keys {
        v, err := tree.Put(key, []byte(fmt.Sprintf("value%d", i)))
        if err != nil {
            t.Fatalf("Failed to insert key %d: %v", i, err)
        }
        version = v
    }

    // Validate final structure
    if err := tree.ValidateStructuralIntegrity(version); err != nil {
        t.Errorf("Tree structure invalid: %v", err)
    }

    // Update middle key and revalidate
    v2, _ := tree.Put(keys[3], []byte("updated"))

    if err := tree.ValidateStructuralIntegrity(v2); err != nil {
        t.Errorf("Tree structure invalid after update: %v", err)
    }
}

func TestConcurrentUpdates(t *testing.T) {
    tree := createTestTree(t)

    // Initial state
    keys := make([]Key, 100)
    for i := range keys {
        keys[i] = KeyHash([]byte(fmt.Sprintf("key-%d", i)))
        tree.Put(keys[i], []byte(fmt.Sprintf("value-%d", i)))
    }

    // Concurrent updates to same keys
    var wg sync.WaitGroup
    errors := make(chan error, 10)

    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(worker int) {
            defer wg.Done()

            // Each worker updates subset of keys
            for j := worker; j < len(keys); j += 10 {
                _, err := tree.Put(keys[j], []byte(fmt.Sprintf("updated-%d-%d", worker, j)))
                if err != nil {
                    errors <- fmt.Errorf("worker %d: %w", worker, err)
                    return
                }
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    // Check for errors
    for err := range errors {
        t.Errorf("Concurrent update error: %v", err)
    }

    // Validate final tree
    finalVersion := tree.GetLatestVersion()
    if err := tree.ValidateStructuralIntegrity(finalVersion); err != nil {
        t.Errorf("Tree corrupted by concurrent updates: %v", err)
    }
}
```

## Performance Considerations

- **Path copying overhead**: Only nodes along update path are duplicated
- **Structure preservation**: No rebalancing needed for simple updates
- **Cache efficiency**: Updated nodes immediately cached
- **Batch optimization**: Multiple updates can share path copying
- **Memory usage**: Old versions retained until pruned

## Security Considerations

- **Version isolation**: Updates don't affect previous versions
- **Atomic updates**: All changes committed together
- **Hash verification**: Each node's hash validates children
- **Collision resistance**: SHA-256 prevents malicious collisions

## Done When ✓

- [ ] Root digest changes while tree shape stays identical for key updates
- [ ] Collision handling creates proper internal node structure
- [ ] All ancestor nodes updated with new version
- [ ] Update batch correctly tracks new and stale nodes
- [ ] Tree structural integrity maintained after updates
- [ ] Concurrent updates handled safely with proper locking
- [ ] Performance tests show minimal overhead for updates
- [ ] 100% test coverage for update scenarios
