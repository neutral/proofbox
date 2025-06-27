---
id: step.15.versioning
depends_on:
  - step.11.update‑existing
tags: [versioning, step]
---

## Objective

Implement `Begin(version)` and path‑cloning for new versions.

## Implements

- **§4.3 NodeKey**, **§6.2 Persistent Structure** ("path cloning"), and **§5 S2 sequential writes**.
  _What happens_:

  - `Begin(version)` forks the root; ancestor duplication realises the _persistent functional tree_ described under _Persistent Versioning_.

## Technical Details

### Multi-Version Architecture

The JMT implements persistent versioning through structural sharing. Each version maintains its own root, but unchanged subtrees are shared between versions. This enables efficient snapshots and historical queries.

### Version Management Implementation

```go
// VersionManager handles multi-version state
type VersionManager struct {
    mu              sync.RWMutex
    versions        map[Version]*VersionInfo
    latestVersion   Version
    committedVersion Version
    pendingWrites   map[Version]*PendingVersion
}

// VersionInfo stores metadata for a version
type VersionInfo struct {
    Version     Version
    RootHash    Hash
    ParentVersion Version
    CreatedAt   time.Time
    NodeCount   int64
    Status      VersionStatus
}

type VersionStatus int

const (
    VersionStatusPending VersionStatus = iota
    VersionStatusCommitted
    VersionStatusAborted
)

// PendingVersion tracks in-progress version creation
type PendingVersion struct {
    version    Version
    parent     Version
    updater    *TreeUpdater
    startTime  time.Time
    operations int
}

// Begin starts a new version based on parent
func (vm *VersionManager) Begin(parentVersion Version) (Version, error) {
    vm.mu.Lock()
    defer vm.mu.Unlock()
    
    // Validate parent version exists
    parent, exists := vm.versions[parentVersion]
    if !exists {
        return 0, fmt.Errorf("parent version %d not found", parentVersion)
    }
    
    if parent.Status != VersionStatusCommitted {
        return 0, fmt.Errorf("parent version %d not committed", parentVersion)
    }
    
    // Allocate new version number
    newVersion := vm.latestVersion + 1
    
    // Create pending version
    pending := &PendingVersion{
        version:   newVersion,
        parent:    parentVersion,
        startTime: time.Now(),
    }
    
    // Initialize updater for path copying
    pending.updater = &TreeUpdater{
        oldVersion: parentVersion,
        newVersion: newVersion,
        nodeWrites: make(map[NodeKey]Node),
        staleNodes: make([]NodeKey, 0),
    }
    
    vm.pendingWrites[newVersion] = pending
    vm.latestVersion = newVersion
    
    return newVersion, nil
}

// Commit finalizes a pending version
func (vm *VersionManager) Commit(version Version, batch *UpdateBatch) error {
    vm.mu.Lock()
    defer vm.mu.Unlock()
    
    pending, exists := vm.pendingWrites[version]
    if !exists {
        return fmt.Errorf("no pending version %d", version)
    }
    
    // Create version info
    info := &VersionInfo{
        Version:       version,
        RootHash:      batch.NewRootHash,
        ParentVersion: pending.parent,
        CreatedAt:     time.Now(),
        NodeCount:     int64(len(batch.NewNodes)),
        Status:        VersionStatusCommitted,
    }
    
    vm.versions[version] = info
    delete(vm.pendingWrites, version)
    
    // Update committed version if this is latest
    if version > vm.committedVersion {
        vm.committedVersion = version
    }
    
    return nil
}

// Abort cancels a pending version
func (vm *VersionManager) Abort(version Version) error {
    vm.mu.Lock()
    defer vm.mu.Unlock()
    
    pending, exists := vm.pendingWrites[version]
    if !exists {
        return fmt.Errorf("no pending version %d", version)
    }
    
    delete(vm.pendingWrites, version)
    
    // Mark as aborted
    vm.versions[version] = &VersionInfo{
        Version:       version,
        ParentVersion: pending.parent,
        Status:        VersionStatusAborted,
    }
    
    return nil
}
```

### Path Cloning Implementation

```go
// PathCloner handles efficient node copying for new versions
type PathCloner struct {
    sourceVersion Version
    targetVersion Version
    nodeCache     *NodeCache
    clonedNodes   map[NodeKey]NodeKey // Maps old key to new key
}

// ClonePath copies nodes along path for new version
func (pc *PathCloner) ClonePath(path []NodeKey) error {
    // Clone from leaf to root
    for i := len(path) - 1; i >= 0; i-- {
        oldKey := path[i]
        
        // Check if already cloned
        if _, cloned := pc.clonedNodes[oldKey]; cloned {
            continue
        }
        
        // Load node
        node, err := pc.nodeCache.Get(oldKey)
        if err != nil {
            return fmt.Errorf("failed to load node %v: %w", oldKey, err)
        }
        
        // Clone based on type
        var clonedNode Node
        switch n := node.(type) {
        case *LeafNode:
            clonedNode = pc.cloneLeaf(n)
        case *InternalNode:
            clonedNode = pc.cloneInternal(n)
        default:
            return fmt.Errorf("unknown node type: %T", n)
        }
        
        // Store with new version
        newKey := NodeKey{
            Version: pc.targetVersion,
            Path:    oldKey.Path,
        }
        
        pc.clonedNodes[oldKey] = newKey
        pc.nodeCache.Put(newKey, clonedNode)
    }
    
    return nil
}

// cloneLeaf creates copy of leaf for new version
func (pc *PathCloner) cloneLeaf(leaf *LeafNode) *LeafNode {
    cloned := &LeafNode{
        Key:       leaf.Key,
        ValueHash: leaf.ValueHash,
    }
    cloned.SetVersion(pc.targetVersion)
    return cloned
}

// cloneInternal creates copy of internal node with child references
func (pc *PathCloner) cloneInternal(internal *InternalNode) *InternalNode {
    cloned := &InternalNode{}
    cloned.SetVersion(pc.targetVersion)
    
    // Copy child references
    for nibble, child := range internal.children {
        // Check if child was cloned in this version
        childKey := NodeKey{
            Version: child.Version,
            Path:    internal.GetPath().Append(nibble),
        }
        
        if newKey, wasCloned := pc.clonedNodes[childKey]; wasCloned {
            // Reference cloned child
            cloned.children[nibble] = Child{
                Hash:    child.Hash, // Hash remains same
                Version: newKey.Version,
                IsLeaf:  child.IsLeaf,
            }
        } else {
            // Reference original child (structural sharing)
            cloned.children[nibble] = child
        }
    }
    
    return cloned
}
```

### Version-Aware Tree Operations

```go
// Tree with version management
type Tree struct {
    db             *pebble.DB
    versionManager *VersionManager
    nodeCache      *NodeCache
    hasher         Hasher
    
    // Concurrency control
    writeMu sync.Mutex    // Serializes writes
    readMu  sync.RWMutex // Allows concurrent reads
}

// BeginVersion starts a new version for updates
func (t *Tree) BeginVersion() (Version, error) {
    current := t.versionManager.GetLatestCommitted()
    return t.versionManager.Begin(current)
}

// PutVersioned inserts key-value in specific version
func (t *Tree) PutVersioned(version Version, key Key, value []byte) error {
    t.writeMu.Lock()
    defer t.writeMu.Unlock()
    
    // Get pending version
    pending := t.versionManager.GetPending(version)
    if pending == nil {
        return fmt.Errorf("version %d not pending", version)
    }
    
    // Perform update
    updater := pending.updater
    _, err := updater.Put(key, value)
    if err != nil {
        return fmt.Errorf("put failed: %w", err)
    }
    
    pending.operations++
    return nil
}

// CommitVersion finalizes all changes in a version
func (t *Tree) CommitVersion(version Version) error {
    t.writeMu.Lock()
    defer t.writeMu.Unlock()
    
    pending := t.versionManager.GetPending(version)
    if pending == nil {
        return fmt.Errorf("version %d not pending", version)
    }
    
    // Build update batch
    batch, err := pending.updater.BuildUpdateBatch()
    if err != nil {
        return fmt.Errorf("failed to build batch: %w", err)
    }
    
    // Write to storage atomically
    writeBatch := t.db.NewBatch()
    defer writeBatch.Close()
    
    // Write all new nodes
    codec := &NodeCodec{}
    for nodeKey, node := range batch.NewNodes {
        data, err := codec.EncodeNode(node)
        if err != nil {
            return fmt.Errorf("failed to encode node: %w", err)
        }
        
        storageKey := nodeKey.StorageKey()
        if err := writeBatch.Set(storageKey, data, nil); err != nil {
            return fmt.Errorf("failed to write node: %w", err)
        }
    }
    
    // Write root hash
    rootKey := makeRootKey(version)
    if err := writeBatch.Set(rootKey, batch.NewRootHash[:], nil); err != nil {
        return fmt.Errorf("failed to write root: %w", err)
    }
    
    // Commit to storage
    if err := writeBatch.Commit(pebble.Sync); err != nil {
        return fmt.Errorf("storage commit failed: %w", err)
    }
    
    // Update version manager
    return t.versionManager.Commit(version, batch)
}

// GetAtVersion retrieves value at specific version
func (t *Tree) GetAtVersion(version Version, key Key) ([]byte, error) {
    t.readMu.RLock()
    defer t.readMu.RUnlock()
    
    // Get version info
    info := t.versionManager.GetVersion(version)
    if info == nil {
        return nil, fmt.Errorf("version %d not found", version)
    }
    
    if info.Status != VersionStatusCommitted {
        return nil, fmt.Errorf("version %d not committed", version)
    }
    
    // Load root
    root, err := t.loadNode(RootNodeKey(version))
    if err != nil {
        return nil, fmt.Errorf("failed to load root: %w", err)
    }
    
    // Traverse tree at version
    return t.lookupInNode(root, key, version)
}
```

### Version Pruning

```go
// VersionPruner manages removal of old versions
type VersionPruner struct {
    tree           *Tree
    retentionPolicy RetentionPolicy
}

type RetentionPolicy struct {
    KeepVersions   int           // Number of versions to keep
    KeepDuration   time.Duration // How long to keep versions
    KeepCheckpoints bool         // Keep checkpoint versions
}

// PruneVersions removes old versions based on policy
func (vp *VersionPruner) PruneVersions() error {
    versions := vp.tree.versionManager.GetAllVersions()
    
    // Sort by version number
    sort.Slice(versions, func(i, j int) bool {
        return versions[i].Version < versions[j].Version
    })
    
    // Determine versions to keep
    toKeep := make(map[Version]bool)
    
    // Keep recent versions
    if vp.retentionPolicy.KeepVersions > 0 {
        start := len(versions) - vp.retentionPolicy.KeepVersions
        if start < 0 {
            start = 0
        }
        for i := start; i < len(versions); i++ {
            toKeep[versions[i].Version] = true
        }
    }
    
    // Keep by duration
    cutoff := time.Now().Add(-vp.retentionPolicy.KeepDuration)
    for _, v := range versions {
        if v.CreatedAt.After(cutoff) {
            toKeep[v.Version] = true
        }
    }
    
    // Prune versions not in keep set
    batch := vp.tree.db.NewBatch()
    defer batch.Close()
    
    for _, v := range versions {
        if !toKeep[v.Version] {
            if err := vp.pruneVersion(v.Version, batch); err != nil {
                return fmt.Errorf("failed to prune version %d: %w", v.Version, err)
            }
        }
    }
    
    return batch.Commit(pebble.Sync)
}

// pruneVersion removes all nodes specific to a version
func (vp *VersionPruner) pruneVersion(version Version, batch *pebble.Batch) error {
    // Iterate nodes with this version
    iter := vp.tree.db.NewIter(&pebble.IterOptions{
        LowerBound: NodeKeyPrefix(version),
        UpperBound: NodeKeyPrefix(version + 1),
    })
    defer iter.Close()
    
    pruned := 0
    for iter.First(); iter.Valid(); iter.Next() {
        // Check if node is referenced by other versions
        nodeKey := DecodeNodeKey(iter.Key())
        if vp.isNodeShared(nodeKey) {
            continue
        }
        
        // Delete node
        if err := batch.Delete(iter.Key(), nil); err != nil {
            return err
        }
        pruned++
    }
    
    // Delete root hash
    rootKey := makeRootKey(version)
    if err := batch.Delete(rootKey, nil); err != nil {
        return err
    }
    
    // Update version manager
    vp.tree.versionManager.RemoveVersion(version)
    
    return nil
}

// isNodeShared checks if node is referenced by other versions
func (vp *VersionPruner) isNodeShared(nodeKey NodeKey) bool {
    // Load node to check child references
    node, err := vp.tree.loadNode(nodeKey)
    if err != nil {
        return true // Err on side of caution
    }
    
    if internal, ok := node.(*InternalNode); ok {
        // Check if any child is from different version
        for _, child := range internal.children {
            if child.Version != nodeKey.Version {
                return true
            }
        }
    }
    
    return false
}
```

## Implementation Steps

1. **Create version manager**: Track version metadata and state
2. **Implement Begin/Commit**: Version lifecycle management
3. **Add path cloning**: Efficient node copying
4. **Version-aware operations**: Get/Put with versions
5. **Add pruning support**: Remove old versions
6. **Optimize structural sharing**: Minimize duplication

## Testing Requirements

### Basic Versioning Tests

```go
func TestBasicVersioning(t *testing.T) {
    tree := createTestTree(t)
    
    // Insert in version 1
    key := KeyHash([]byte("test-key"))
    value1 := []byte("value-v1")
    
    v1, err := tree.BeginVersion()
    if err != nil {
        t.Fatalf("Failed to begin v1: %v", err)
    }
    
    err = tree.PutVersioned(v1, key, value1)
    if err != nil {
        t.Fatalf("Failed to put in v1: %v", err)
    }
    
    err = tree.CommitVersion(v1)
    if err != nil {
        t.Fatalf("Failed to commit v1: %v", err)
    }
    
    // Update in version 2
    value2 := []byte("value-v2")
    
    v2, err := tree.BeginVersion()
    if err != nil {
        t.Fatalf("Failed to begin v2: %v", err)
    }
    
    err = tree.PutVersioned(v2, key, value2)
    if err != nil {
        t.Fatalf("Failed to put in v2: %v", err)
    }
    
    err = tree.CommitVersion(v2)
    if err != nil {
        t.Fatalf("Failed to commit v2: %v", err)
    }
    
    // Verify both versions accessible
    got1, err := tree.GetAtVersion(v1, key)
    if err != nil {
        t.Errorf("Failed to get from v1: %v", err)
    }
    if !bytes.Equal(got1, value1) {
        t.Errorf("V1 value mismatch: got %v, want %v", got1, value1)
    }
    
    got2, err := tree.GetAtVersion(v2, key)
    if err != nil {
        t.Errorf("Failed to get from v2: %v", err)
    }
    if !bytes.Equal(got2, value2) {
        t.Errorf("V2 value mismatch: got %v, want %v", got2, value2)
    }
}

func TestOldRootVerifiable(t *testing.T) {
    tree := createTestTree(t)
    
    // Build tree in v1
    keys := []Key{
        KeyHash([]byte("key1")),
        KeyHash([]byte("key2")),
        KeyHash([]byte("key3")),
    }
    
    v1, _ := tree.BeginVersion()
    for i, key := range keys {
        tree.PutVersioned(v1, key, []byte(fmt.Sprintf("value%d", i)))
    }
    tree.CommitVersion(v1)
    
    // Get v1 root
    root1, err := tree.GetRootHash(v1)
    if err != nil {
        t.Fatalf("Failed to get v1 root: %v", err)
    }
    
    // Modify in v2
    v2, _ := tree.BeginVersion()
    tree.PutVersioned(v2, keys[1], []byte("modified"))
    tree.CommitVersion(v2)
    
    // Get v2 root
    root2, err := tree.GetRootHash(v2)
    if err != nil {
        t.Fatalf("Failed to get v2 root: %v", err)
    }
    
    // Roots should differ
    if root1 == root2 {
        t.Error("Roots should differ between versions")
    }
    
    // Generate and verify proof for v1
    proof1, err := tree.GenerateProof(v1, keys[1])
    if err != nil {
        t.Fatalf("Failed to generate v1 proof: %v", err)
    }
    
    err = VerifyProof(proof1, keys[1], root1)
    if err != nil {
        t.Errorf("V1 proof verification failed: %v", err)
    }
    
    // Proof should fail against v2 root
    err = VerifyProof(proof1, keys[1], root2)
    if err == nil {
        t.Error("V1 proof should not verify against v2 root")
    }
}

func TestStructuralSharing(t *testing.T) {
    tree := createTestTree(t)
    
    // Create large tree
    numKeys := 1000
    keys := make([]Key, numKeys)
    for i := range keys {
        keys[i] = KeyHash([]byte(fmt.Sprintf("key-%d", i)))
    }
    
    v1, _ := tree.BeginVersion()
    for _, key := range keys {
        tree.PutVersioned(v1, key, []byte("value"))
    }
    tree.CommitVersion(v1)
    
    // Count v1 nodes
    v1Nodes := countVersionNodes(tree, v1)
    
    // Modify single key in v2
    v2, _ := tree.BeginVersion()
    tree.PutVersioned(v2, keys[500], []byte("modified"))
    tree.CommitVersion(v2)
    
    // Count v2-specific nodes
    v2Nodes := countVersionNodes(tree, v2)
    
    // Should create far fewer nodes than total
    ratio := float64(v2Nodes) / float64(v1Nodes)
    t.Logf("V1 nodes: %d, V2-specific nodes: %d, ratio: %.2f%%", 
        v1Nodes, v2Nodes, ratio*100)
    
    if ratio > 0.1 {
        t.Errorf("Too many nodes created: %.2f%% (expected < 10%%)", ratio*100)
    }
}
```

### Concurrent Version Tests

```go
func TestConcurrentVersions(t *testing.T) {
    tree := createTestTree(t)
    
    // Create base version
    v0, _ := tree.BeginVersion()
    baseKey := KeyHash([]byte("base"))
    tree.PutVersioned(v0, baseKey, []byte("base-value"))
    tree.CommitVersion(v0)
    
    // Start multiple versions concurrently
    numVersions := 10
    var wg sync.WaitGroup
    errors := make(chan error, numVersions)
    
    for i := 0; i < numVersions; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            
            // Begin new version
            v, err := tree.BeginVersion()
            if err != nil {
                errors <- fmt.Errorf("worker %d: begin failed: %w", idx, err)
                return
            }
            
            // Make unique changes
            key := KeyHash([]byte(fmt.Sprintf("key-%d", idx)))
            value := []byte(fmt.Sprintf("value-%d", idx))
            
            err = tree.PutVersioned(v, key, value)
            if err != nil {
                errors <- fmt.Errorf("worker %d: put failed: %w", idx, err)
                return
            }
            
            // Commit
            err = tree.CommitVersion(v)
            if err != nil {
                errors <- fmt.Errorf("worker %d: commit failed: %w", idx, err)
                return
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    // Check for errors
    for err := range errors {
        t.Error(err)
    }
    
    // Verify all versions committed
    versions := tree.versionManager.GetAllVersions()
    if len(versions) != numVersions+1 { // +1 for base
        t.Errorf("Expected %d versions, got %d", numVersions+1, len(versions))
    }
}

func TestVersionAbort(t *testing.T) {
    tree := createTestTree(t)
    
    // Start version but abort
    v1, _ := tree.BeginVersion()
    key := KeyHash([]byte("test"))
    tree.PutVersioned(v1, key, []byte("should-not-persist"))
    
    // Abort instead of commit
    err := tree.versionManager.Abort(v1)
    if err != nil {
        t.Fatalf("Abort failed: %v", err)
    }
    
    // Key should not exist
    _, err = tree.GetAtVersion(v1, key)
    if err == nil {
        t.Error("Aborted version should not be readable")
    }
    
    // Should be able to reuse version number
    v1New, _ := tree.BeginVersion()
    if v1New != v1 {
        t.Errorf("Expected to reuse version %d, got %d", v1, v1New)
    }
}
```

### Version Pruning Tests

```go
func TestVersionPruning(t *testing.T) {
    tree := createTestTree(t)
    
    // Create multiple versions
    versions := make([]Version, 10)
    for i := range versions {
        v, _ := tree.BeginVersion()
        key := KeyHash([]byte(fmt.Sprintf("key-%d", i)))
        tree.PutVersioned(v, key, []byte("value"))
        tree.CommitVersion(v)
        versions[i] = v
    }
    
    // Set pruning policy
    pruner := &VersionPruner{
        tree: tree,
        retentionPolicy: RetentionPolicy{
            KeepVersions: 3,
        },
    }
    
    // Prune old versions
    err := pruner.PruneVersions()
    if err != nil {
        t.Fatalf("Pruning failed: %v", err)
    }
    
    // Check old versions are gone
    for i := 0; i < 7; i++ {
        _, err := tree.GetRootHash(versions[i])
        if err == nil {
            t.Errorf("Version %d should be pruned", versions[i])
        }
    }
    
    // Check recent versions remain
    for i := 7; i < 10; i++ {
        _, err := tree.GetRootHash(versions[i])
        if err != nil {
            t.Errorf("Version %d should be retained: %v", versions[i], err)
        }
    }
}

func TestPathCloning(t *testing.T) {
    tree := createTestTree(t)
    
    // Build initial tree
    v1, _ := tree.BeginVersion()
    keys := []Key{
        {0x10, 0x20},
        {0x10, 0x21},
        {0x10, 0x30},
    }
    
    for _, key := range keys {
        tree.PutVersioned(v1, key, []byte("value"))
    }
    tree.CommitVersion(v1)
    
    // Start v2 and modify one key
    v2, _ := tree.BeginVersion()
    
    // Track cloned nodes
    cloner := &PathCloner{
        sourceVersion: v1,
        targetVersion: v2,
        nodeCache:     tree.nodeCache,
        clonedNodes:   make(map[NodeKey]NodeKey),
    }
    
    // Simulate path cloning
    path := []NodeKey{
        {Version: v1, Path: NibblePath{}},           // Root
        {Version: v1, Path: NibblePath{nibbles: []Nibble{1}}}, // First internal
        {Version: v1, Path: NibblePath{nibbles: []Nibble{1, 0}}}, // Second internal
    }
    
    err := cloner.ClonePath(path)
    if err != nil {
        t.Fatalf("Path cloning failed: %v", err)
    }
    
    // Verify correct number of nodes cloned
    if len(cloner.clonedNodes) != len(path) {
        t.Errorf("Expected %d cloned nodes, got %d", 
            len(path), len(cloner.clonedNodes))
    }
    
    // Verify cloned nodes have new version
    for oldKey, newKey := range cloner.clonedNodes {
        if newKey.Version != v2 {
            t.Errorf("Cloned node has wrong version: %d", newKey.Version)
        }
        if !oldKey.Path.Equals(newKey.Path) {
            t.Error("Cloned node path changed")
        }
    }
}
```

## Performance Considerations

- **Structural sharing**: Minimize node duplication
- **Sequential writes**: Version-prefixed keys optimize LSM
- **Batch operations**: Amortize version creation overhead
- **Lazy cloning**: Only clone modified paths
- **Version caching**: Keep recent versions in memory

## Security Considerations

- **Version isolation**: Changes don't affect committed versions
- **Atomic commits**: All-or-nothing version updates
- **Concurrent safety**: Serialized writes, parallel reads
- **Pruning safety**: Don't delete shared nodes

## Done When ✓

- [ ] Old root still verifiable after a new commit
- [ ] Begin(version) creates isolated update context
- [ ] Path cloning creates minimal new nodes
- [ ] Structural sharing reduces storage overhead
- [ ] Multiple versions can be read concurrently
- [ ] Version pruning safely removes old data
- [ ] Aborted versions don't persist
- [ ] 100% test coverage for versioning scenarios
