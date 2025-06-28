---
id: step.10.insert‑basic
depends_on:
  - step.09.tree‑skeleton
  - step.03.error-handling
tags: [insert, step]
---

## Objective

Support inserting first key/value into an empty tree.

## Implements

- **§6.2 Insertion & Update** case "Empty Slot → create new leaf" as the very first insertion.
  _What happens_:

  - Adds ability to store first `(key,value)` and compute a non‑default root digest, preparing for later branching logic.

## Technical Details

### Basic Put Operation

```go
// Put inserts or updates a key-value pair, creating a new version
func (t *Tree) Put(key Key, value []byte) (Version, error) {
    // Validate inputs
    if err := ValidateKey(key); err != nil {
        return 0, fmt.Errorf("invalid key: %w", err)
    }
    if len(value) > MaxValueSize {
        return 0, fmt.Errorf("value too large: %d bytes", len(value))
    }
    
    // Serialize write operations
    t.writeMu.Lock()
    defer t.writeMu.Unlock()
    
    // Get current version
    currentVersion := t.GetLatestVersion()
    newVersion := currentVersion + 1
    
    // Create write batch
    batch := t.db.NewBatch()
    defer batch.Close()
    
    // Create updater for this operation
    updater := &TreeUpdater{
        tree:       t,
        batch:      batch,
        oldVersion: currentVersion,
        newVersion: newVersion,
        nodeWrites: make(map[NodeKey]Node),
    }
    
    // Perform the update
    newRootHash, err := updater.Put(key, value)
    if err != nil {
        return 0, fmt.Errorf("failed to put: %w", err)
    }
    
    // Write all nodes to batch
    if err := updater.writeToBatch(); err != nil {
        return 0, fmt.Errorf("failed to write nodes: %w", err)
    }
    
    // Store new root hash
    rootKey := makeRootKey(newVersion)
    if err := batch.Set(rootKey, newRootHash[:], nil); err != nil {
        return 0, fmt.Errorf("failed to write root hash: %w", err)
    }
    
    // Commit the batch
    if err := batch.Commit(pebble.Sync); err != nil {
        return 0, fmt.Errorf("failed to commit: %w", err)
    }
    
    // Update in-memory state
    t.mu.Lock()
    t.rootHashes[newVersion] = newRootHash
    t.latestVer = newVersion
    t.mu.Unlock()
    
    return newVersion, nil
}
```

### TreeUpdater Implementation

```go
// TreeUpdater handles the logic for updating the tree
type TreeUpdater struct {
    tree       *Tree
    batch      *pebble.Batch
    oldVersion Version
    newVersion Version
    nodeWrites map[NodeKey]Node  // Nodes to write in this update
}

// Put inserts or updates a key-value pair
func (u *TreeUpdater) Put(key Key, value []byte) (Hash, error) {
    // Compute value hash
    valueHash := sha256.Sum256(value)
    
    // Store value
    valueKey := makeValueKey(u.newVersion, Hash(valueHash))
    if err := u.batch.Set(valueKey, value, nil); err != nil {
        return Hash{}, fmt.Errorf("failed to store value: %w", err)
    }
    
    // Get current root
    var currentRoot Node
    if u.oldVersion > 0 {
        rootHash, exists := u.tree.rootHashes[u.oldVersion]
        if exists && rootHash != EmptyHash {
            // Load existing root
            root, err := u.loadNode(RootNodeKey(u.oldVersion))
            if err != nil {
                return Hash{}, fmt.Errorf("failed to load root: %w", err)
            }
            currentRoot = root
        }
    }
    
    // Handle empty tree case
    if currentRoot == nil {
        return u.insertIntoEmptyTree(key, Hash(valueHash))
    }
    
    // Handle non-empty tree (will be implemented in later steps)
    return u.insertIntoTree(currentRoot, key, Hash(valueHash))
}

// insertIntoEmptyTree handles insertion into an empty tree
func (u *TreeUpdater) insertIntoEmptyTree(key Key, valueHash Hash) (Hash, error) {
    // Create new leaf node as root
    leaf := &LeafNode{
        Key:       key,
        ValueHash: valueHash,
    }
    leaf.SetVersion(u.newVersion)
    
    // Store the leaf
    leafKey := RootNodeKey(u.newVersion)
    u.nodeWrites[leafKey] = leaf
    
    // Return leaf's hash as new root hash
    return leaf.Hash(), nil
}

// insertIntoTree handles insertion into existing tree
func (u *TreeUpdater) insertIntoTree(root Node, key Key, valueHash Hash) (Hash, error) {
    // For now, only handle the case where root is a leaf
    if leaf, ok := root.(*LeafNode); ok {
        if leaf.Key == key {
            // Update existing key
            return u.updateLeaf(leaf, valueHash)
        } else {
            // Split into internal node (will be implemented later)
            return Hash{}, errors.New("leaf splitting not yet implemented")
        }
    }
    
    // Internal node case will be implemented in later steps
    return Hash{}, errors.New("internal node insertion not yet implemented")
}

// updateLeaf creates a new version of a leaf with updated value
func (u *TreeUpdater) updateLeaf(oldLeaf *LeafNode, newValueHash Hash) (Hash, error) {
    // Create new leaf with same key but new value
    newLeaf := &LeafNode{
        Key:       oldLeaf.Key,
        ValueHash: newValueHash,
    }
    newLeaf.SetVersion(u.newVersion)
    
    // Store the new leaf
    leafKey := RootNodeKey(u.newVersion)
    u.nodeWrites[leafKey] = newLeaf
    
    return newLeaf.Hash(), nil
}

// loadNode loads a node from the previous version
func (u *TreeUpdater) loadNode(key NodeKey) (Node, error) {
    // First check if we're loading from the current update
    if node, exists := u.nodeWrites[key]; exists {
        return node, nil
    }
    
    // Load from storage
    storageKey := key.StorageKey()
    data, closer, err := u.tree.db.Get(storageKey)
    if err == pebble.ErrNotFound {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    defer closer.Close()
    
    // Decode node
    codec := &NodeCodec{}
    return codec.DecodeNode(data)
}

// writeToBatch writes all nodes to the batch
func (u *TreeUpdater) writeToBatch() error {
    codec := &NodeCodec{}
    
    for nodeKey, node := range u.nodeWrites {
        // Encode node
        data, err := codec.EncodeNode(node)
        if err != nil {
            return fmt.Errorf("failed to encode node: %w", err)
        }
        
        // Write to batch
        storageKey := nodeKey.StorageKey()
        if err := u.batch.Set(storageKey, data, nil); err != nil {
            return fmt.Errorf("failed to write node: %w", err)
        }
        
        // Update cache
        u.tree.nodeCache.Put(nodeKey, node)
    }
    
    return nil
}
```

### Simple Update Example

```go
// Example: Basic key update in tree with single leaf
func ExampleBasicUpdate() {
    tree := createTestTree()
    
    // Insert first key
    v1, _ := tree.Put(KeyHash([]byte("key1")), []byte("value1"))
    
    // Update same key
    v2, _ := tree.Put(KeyHash([]byte("key1")), []byte("value2"))
    
    // Get at different versions
    val1, _ := tree.Get(v1, KeyHash([]byte("key1")))  // Returns "value1"
    val2, _ := tree.Get(v2, KeyHash([]byte("key1")))  // Returns "value2"
}
```

## Implementation Steps

1. **Add Put method**: Basic single key-value insertion
2. **Create TreeUpdater**: Handles update logic and batching
3. **Handle empty tree**: Create leaf as root
4. **Handle simple update**: Update existing leaf with same key
5. **Write nodes to storage**: Persist changes atomically
6. **Update version tracking**: Maintain root hashes

## Testing Requirements

### Unit Tests

```go
func TestInsertIntoEmptyTree(t *testing.T) {
    db := createTestDB(t)
    defer db.Close()
    
    tree, _ := NewTree(db, DefaultTreeConfig())
    
    // Insert into empty tree
    key := KeyHash([]byte("test-key"))
    value := []byte("test-value")
    
    version, err := tree.Put(key, value)
    if err != nil {
        t.Fatalf("Put failed: %v", err)
    }
    
    // Verify version incremented
    if version != 1 {
        t.Errorf("Expected version 1, got %d", version)
    }
    
    // Verify we can retrieve the value
    retrieved, err := tree.Get(version, key)
    if err != nil {
        t.Fatalf("Get failed: %v", err)
    }
    
    if !bytes.Equal(retrieved, value) {
        t.Errorf("Value mismatch: got %v, want %v", retrieved, value)
    }
    
    // Verify root hash changed
    rootHash, _ := tree.GetRootHash(version)
    if rootHash == EmptyHash {
        t.Error("Root hash should not be empty after insert")
    }
}

func TestUpdateExistingKey(t *testing.T) {
    db := createTestDB(t)
    defer db.Close()
    
    tree, _ := NewTree(db, DefaultTreeConfig())
    key := KeyHash([]byte("key"))
    
    // Insert initial value
    v1, _ := tree.Put(key, []byte("value1"))
    
    // Update with new value
    v2, err := tree.Put(key, []byte("value2"))
    if err != nil {
        t.Fatalf("Update failed: %v", err)
    }
    
    // Check both versions
    val1, _ := tree.Get(v1, key)
    val2, _ := tree.Get(v2, key)
    
    if !bytes.Equal(val1, []byte("value1")) {
        t.Error("Version 1 value incorrect")
    }
    if !bytes.Equal(val2, []byte("value2")) {
        t.Error("Version 2 value incorrect")
    }
    
    // Verify different root hashes
    hash1, _ := tree.GetRootHash(v1)
    hash2, _ := tree.GetRootHash(v2)
    
    if hash1 == hash2 {
        t.Error("Root hashes should differ after update")
    }
}

func TestPersistence(t *testing.T) {
    db := createTestDB(t)
    defer db.Close()
    
    key := KeyHash([]byte("persistent-key"))
    value := []byte("persistent-value")
    var version Version
    
    // Insert with first tree instance
    {
        tree, _ := NewTree(db, DefaultTreeConfig())
        v, err := tree.Put(key, value)
        if err != nil {
            t.Fatalf("Put failed: %v", err)
        }
        version = v
    }
    
    // Create new tree instance
    tree2, err := NewTree(db, DefaultTreeConfig())
    if err != nil {
        t.Fatalf("Failed to create second tree: %v", err)
    }
    
    // Should be able to read the value
    retrieved, err := tree2.Get(version, key)
    if err != nil {
        t.Fatalf("Get failed: %v", err)
    }
    
    if !bytes.Equal(retrieved, value) {
        t.Error("Value not persisted correctly")
    }
}
```

## Performance Considerations

- **Batch writes**: All changes in single atomic batch
- **Value deduplication**: Same value hash can be reused
- **Write locking**: Serialized writes prevent conflicts
- **Cache updates**: New nodes added to cache immediately

## Security Notes

- **Version overflow**: Check version doesn't exceed MaxVersion
- **Value size limits**: Enforce MaxValueSize to prevent DoS
- **Atomic commits**: All-or-nothing updates maintain consistency
- **Hash verification**: Value hash computed before storage

## Done When ✓

- [ ] After insert, `Get` returns stored value
- [ ] Put method creates new version with incremented number
- [ ] Empty tree insertion creates leaf as root
- [ ] Simple key update creates new leaf version
- [ ] All changes written atomically in batch
- [ ] Root hash updated for new version
- [ ] Persistence verified across tree instances
- [ ] Thread-safe with proper locking
- [ ] 100% test coverage for basic insert cases
