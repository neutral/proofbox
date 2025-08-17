# Concurrency Model Specification

## Overview
This specification defines the concurrency model for the Jellyfish Merkle Tree implementation, including thread safety guarantees, locking strategies, and concurrent operation handling.

## Concurrency Goals

1. **Multiple Readers**: Support concurrent read operations without blocking
2. **Single Writer**: Ensure write operations are serialized for consistency  
3. **Snapshot Isolation**: Readers see consistent snapshots during updates
4. **Lock-Free Reads**: Minimize read path contention
5. **Atomic Updates**: All modifications appear atomic to readers

## Architecture

### Tree Structure

```go
// Tree is the main JMT structure with concurrency control
type Tree struct {
    db         *pebble.DB
    
    // Version management
    mu          sync.RWMutex     // Protects version metadata
    latestVer   Version          // Latest committed version
    rootHashes  map[Version]Hash // Version -> root hash cache
    
    // Write coordination
    writeMu     sync.Mutex       // Serializes write operations
    writeSeq    uint64          // Write sequence number
    
    // Read optimization
    nodeCache   *NodeCache      // Thread-safe LRU cache
    
    // Metrics
    metrics     *Metrics        // Thread-safe metrics
}
```

### Version Management

```go
// VersionManager handles concurrent version access
type VersionManager struct {
    mu         sync.RWMutex
    versions   map[Version]*VersionInfo
    latest     Version
}

type VersionInfo struct {
    Version    Version
    RootHash   Hash
    Timestamp  time.Time
    TxCount    uint64
}

// GetLatestVersion returns the latest committed version
func (vm *VersionManager) GetLatestVersion() Version {
    vm.mu.RLock()
    defer vm.mu.RUnlock()
    return vm.latest
}

// CommitVersion atomically updates to a new version
func (vm *VersionManager) CommitVersion(ver Version, rootHash Hash) error {
    vm.mu.Lock()
    defer vm.mu.Unlock()
    
    if ver <= vm.latest {
        return ErrInvalidVersion
    }
    
    vm.versions[ver] = &VersionInfo{
        Version:   ver,
        RootHash:  rootHash,
        Timestamp: time.Now(),
    }
    vm.latest = ver
    return nil
}
```

## Read Operations

### Concurrent Reads

```go
// Get performs a read operation with snapshot isolation
func (t *Tree) Get(version Version, key Key) ([]byte, error) {
    // No locking needed - reads are naturally isolated by version
    
    // Validate version exists
    if !t.hasVersion(version) {
        return nil, ErrVersionNotFound
    }
    
    // Create read-only snapshot
    snapshot := t.db.NewSnapshot()
    defer snapshot.Close()
    
    // Perform lookup using snapshot
    reader := &TreeReader{
        snapshot: snapshot,
        version:  version,
        cache:    t.nodeCache,
    }
    
    return reader.Get(key)
}

// TreeReader encapsulates read-only operations
type TreeReader struct {
    snapshot *pebble.Snapshot
    version  Version
    cache    *NodeCache
}

// Get reads a value without locks
func (r *TreeReader) Get(key Key) ([]byte, error) {
    nibblePath := KeyToNibblePath(key)
    node, err := r.traverseToLeaf(nibblePath)
    if err != nil {
        return nil, err
    }
    
    if node == nil {
        return nil, ErrKeyNotFound
    }
    
    return node.Value, nil
}
```

### Batch Reads

```go
// BatchGet performs multiple reads efficiently
func (t *Tree) BatchGet(version Version, keys []Key) ([][]byte, error) {
    // Single snapshot for consistency
    snapshot := t.db.NewSnapshot()
    defer snapshot.Close()
    
    reader := &TreeReader{
        snapshot: snapshot,
        version:  version,
        cache:    t.nodeCache,
    }
    
    // Parallel reads using goroutines
    results := make([][]byte, len(keys))
    errors := make([]error, len(keys))
    
    var wg sync.WaitGroup
    for i, key := range keys {
        wg.Add(1)
        go func(idx int, k Key) {
            defer wg.Done()
            results[idx], errors[idx] = reader.Get(k)
        }(i, key)
    }
    
    wg.Wait()
    
    // Check for any errors
    for i, err := range errors {
        if err != nil && !errors.Is(err, ErrKeyNotFound) {
            return nil, WrapError(err, "batch get failed at key %d", i)
        }
    }
    
    return results, nil
}
```

## Write Operations

### Write Serialization

```go
// Put performs a write operation with proper locking
func (t *Tree) Put(key Key, value []byte) (Version, error) {
    // Serialize all write operations
    t.writeMu.Lock()
    defer t.writeMu.Unlock()
    
    // Get next version
    currentVersion := t.getLatestVersion()
    newVersion := currentVersion + 1
    
    // Create write batch
    batch := t.db.NewBatch()
    defer batch.Close()
    
    // Perform update
    updater := &TreeUpdater{
        batch:      batch,
        oldVersion: currentVersion,
        newVersion: newVersion,
        cache:      t.nodeCache,
    }
    
    newRoot, err := updater.Put(key, value)
    if err != nil {
        return 0, err
    }
    
    // Commit atomically
    if err := batch.Commit(pebble.Sync); err != nil {
        return 0, WrapError(err, "failed to commit put operation")
    }
    
    // Update version metadata
    t.updateLatestVersion(newVersion, newRoot)
    
    return newVersion, nil
}
```

### Batch Writes

```go
// BatchWrite performs multiple operations atomically
type BatchWrite struct {
    tree       *Tree
    operations []Operation
    mu         sync.Mutex
}

type Operation struct {
    Type  OperationType
    Key   Key
    Value []byte
}

type OperationType uint8

const (
    OpPut OperationType = iota
    OpDelete
)

// Execute commits all operations atomically
func (b *BatchWrite) Execute() (Version, error) {
    // Single write lock for entire batch
    b.tree.writeMu.Lock()
    defer b.tree.writeMu.Unlock()
    
    currentVersion := b.tree.getLatestVersion()
    newVersion := currentVersion + 1
    
    // Create single write batch
    batch := b.tree.db.NewBatch()
    defer batch.Close()
    
    updater := &TreeUpdater{
        batch:      batch,
        oldVersion: currentVersion,
        newVersion: newVersion,
        cache:      b.tree.nodeCache,
    }
    
    // Apply all operations
    var newRoot Hash
    var err error
    
    for _, op := range b.operations {
        switch op.Type {
        case OpPut:
            newRoot, err = updater.Put(op.Key, op.Value)
        case OpDelete:
            newRoot, err = updater.Delete(op.Key)
        }
        
        if err != nil {
            return 0, WrapError(err, "batch operation failed")
        }
    }
    
    // Commit atomically
    if err := batch.Commit(pebble.Sync); err != nil {
        return 0, WrapError(err, "failed to commit batch")
    }
    
    // Update version
    b.tree.updateLatestVersion(newVersion, newRoot)
    
    return newVersion, nil
}
```

## Node Cache

### Thread-Safe LRU Cache

```go
// NodeCache provides thread-safe caching of nodes
type NodeCache struct {
    mu       sync.RWMutex
    cache    *lru.Cache
    hits     uint64
    misses   uint64
    maxSize  int
}

// CacheKey combines version and nibble path
type CacheKey struct {
    Version Version
    Path    string // Encoded nibble path
}

// Get retrieves a node from cache
func (c *NodeCache) Get(key NodeKey) (Node, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    cacheKey := CacheKey{
        Version: key.Version,
        Path:    encodePath(key.NibblePath),
    }
    
    if val, ok := c.cache.Get(cacheKey); ok {
        atomic.AddUint64(&c.hits, 1)
        return val.(Node), true
    }
    
    atomic.AddUint64(&c.misses, 1)
    return nil, false
}

// Put adds a node to cache
func (c *NodeCache) Put(key NodeKey, node Node) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    cacheKey := CacheKey{
        Version: key.Version,
        Path:    encodePath(key.NibblePath),
    }
    
    c.cache.Add(cacheKey, node)
}

// Clear removes all entries from cache
func (c *NodeCache) Clear() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.cache.Purge()
}
```

## Proof Generation

### Concurrent Proof Generation

```go
// GenerateProof creates a proof with read consistency
func (t *Tree) GenerateProof(version Version, key Key) (*Proof, error) {
    // Read lock for version check
    t.mu.RLock()
    rootHash, exists := t.rootHashes[version]
    t.mu.RUnlock()
    
    if !exists {
        return nil, ErrVersionNotFound
    }
    
    // Create snapshot for consistent read
    snapshot := t.db.NewSnapshot()
    defer snapshot.Close()
    
    prover := &ProofGenerator{
        snapshot: snapshot,
        version:  version,
        rootHash: rootHash,
    }
    
    return prover.Generate(key)
}

// BatchGenerateProofs generates multiple proofs concurrently
func (t *Tree) BatchGenerateProofs(version Version, keys []Key) ([]*Proof, error) {
    // Validate version once
    t.mu.RLock()
    rootHash, exists := t.rootHashes[version]
    t.mu.RUnlock()
    
    if !exists {
        return nil, ErrVersionNotFound
    }
    
    // Single snapshot for all proofs
    snapshot := t.db.NewSnapshot()
    defer snapshot.Close()
    
    // Generate proofs in parallel
    proofs := make([]*Proof, len(keys))
    errors := make([]error, len(keys))
    
    var wg sync.WaitGroup
    sem := make(chan struct{}, runtime.NumCPU()) // Limit parallelism
    
    for i, key := range keys {
        wg.Add(1)
        go func(idx int, k Key) {
            defer wg.Done()
            
            sem <- struct{}{} // Acquire semaphore
            defer func() { <-sem }() // Release
            
            prover := &ProofGenerator{
                snapshot: snapshot,
                version:  version,
                rootHash: rootHash,
            }
            
            proofs[idx], errors[idx] = prover.Generate(k)
        }(i, key)
    }
    
    wg.Wait()
    
    // Check errors
    for i, err := range errors {
        if err != nil {
            return nil, WrapError(err, "proof generation failed for key %d", i)
        }
    }
    
    return proofs, nil
}
```

## Memory Ordering

### Atomic Operations

```go
// AtomicVersion provides atomic version access
type AtomicVersion struct {
    value uint64
}

func (v *AtomicVersion) Load() Version {
    return Version(atomic.LoadUint64(&v.value))
}

func (v *AtomicVersion) Store(ver Version) {
    atomic.StoreUint64(&v.value, uint64(ver))
}

func (v *AtomicVersion) CompareAndSwap(old, new Version) bool {
    return atomic.CompareAndSwapUint64(&v.value, uint64(old), uint64(new))
}
```

## Deadlock Prevention

### Lock Ordering

Always acquire locks in this order to prevent deadlocks:
1. `writeMu` (write operations)
2. `mu` (version metadata)
3. `cache.mu` (node cache)

```go
// Example: Safe lock ordering
func (t *Tree) UpdateWithCache(key Key, value []byte) error {
    // 1. Acquire write lock first
    t.writeMu.Lock()
    defer t.writeMu.Unlock()
    
    // 2. Then version lock
    t.mu.Lock()
    version := t.latestVer
    t.mu.Unlock()
    
    // 3. Finally cache operations
    t.nodeCache.Clear()
    
    // Perform update...
    return nil
}
```

## Performance Considerations

### Read Scaling

1. **Snapshot Isolation**: Each read creates a lightweight snapshot
2. **No Read Locks**: Reads don't block each other or writes
3. **Cache Sharing**: All readers share the same node cache
4. **Parallel Reads**: Batch operations use goroutines

### Write Optimization

1. **Batch Writes**: Group multiple operations in one transaction
2. **Async Commit**: Use `pebble.NoSync` for non-critical writes
3. **Write Pipelining**: Prepare next batch while current commits

```go
// Example: Pipelined writes
type PipelinedWriter struct {
    tree     *Tree
    pipeline chan *BatchWrite
    results  chan error
}

func (pw *PipelinedWriter) Start() {
    go func() {
        for batch := range pw.pipeline {
            _, err := batch.Execute()
            pw.results <- err
        }
    }()
}
```

## Best Practices

1. **Use Batch Operations**: Group related operations for better performance
2. **Cache Appropriately**: Size cache based on working set
3. **Monitor Contention**: Track lock wait times and cache hit rates
4. **Snapshot Cleanup**: Always close snapshots to prevent resource leaks
5. **Error Handling**: Don't hold locks during error handling
6. **Graceful Shutdown**: Ensure all operations complete before closing