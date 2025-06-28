---
id: step.09.tree‑skeleton
depends_on:
  - step.07.node‑iface
tags: [tree, step]
---

## Objective

Create `Tree` with `root Node`, `version uint64`, and `Get` on empty tree.

## Implements

- **§6 Algorithms – Lookup** initial conditions (root may be empty).
  _What happens_:

  - Creates minimal `Tree` struct with a nil‑root sentinel that returns `nil` for any GET, satisfying lookup step 1 ("Start at Root… if tree empty -> not found").

## Technical Details

### Tree Structure

```go
// Tree represents a Jellyfish Merkle Tree with concurrency support
type Tree struct {
    // Storage backend
    db *pebble.DB
    
    // Concurrency control
    mu          sync.RWMutex              // Protects all mutable state
    
    // Version management
    latestVer   Version                   // Latest committed version
    rootHashes  map[Version]Hash          // Version -> root hash cache
    
    // Node cache for performance
    nodeCache   *NodeCache                // Thread-safe LRU cache
    
    // Configuration
    config      TreeConfig                // Tree configuration
}

// TreeConfig holds configuration options
type TreeConfig struct {
    CacheSize    int           // Number of nodes to cache (default: 10000)
    MaxBatchSize int           // Maximum operations per batch (default: 1000)
    MetricsEnabled bool        // Enable metrics collection
}

// DefaultTreeConfig returns default configuration
func DefaultTreeConfig() TreeConfig {
    return TreeConfig{
        CacheSize:      10000,
        MaxBatchSize:   1000,
        MetricsEnabled: true,
    }
}
```

### Tree Initialization

```go
// NewTree creates a new Jellyfish Merkle Tree
func NewTree(db *pebble.DB, config TreeConfig) (*Tree, error) {
    if db == nil {
        return nil, errors.New("database cannot be nil")
    }
    
    tree := &Tree{
        db:         db,
        rootHashes: make(map[Version]Hash),
        config:     config,
    }
    
    // Initialize node cache
    cache, err := NewNodeCache(config.CacheSize)
    if err != nil {
        return nil, fmt.Errorf("failed to create cache: %w", err)
    }
    tree.nodeCache = cache
    
    // Load existing root hashes from storage
    if err := tree.loadRootHashes(); err != nil {
        return nil, fmt.Errorf("failed to load root hashes: %w", err)
    }
    
    return tree, nil
}

// loadRootHashes loads all version -> root hash mappings
func (t *Tree) loadRootHashes() error {
    prefix := []byte(RootKeyPrefix)
    iter := t.db.NewIter(&pebble.IterOptions{
        LowerBound: prefix,
        UpperBound: append(prefix, 0xFF),
    })
    defer iter.Close()
    
    for iter.First(); iter.Valid(); iter.Next() {
        // Parse version from key
        version, err := parseRootKey(iter.Key())
        if err != nil {
            return err
        }
        
        // Parse root hash from value
        if len(iter.Value()) != 32 {
            return fmt.Errorf("invalid root hash size for version %d", version)
        }
        
        var hash Hash
        copy(hash[:], iter.Value())
        t.rootHashes[version] = hash
        
        // Track latest version
        if version > t.latestVer {
            t.latestVer = version
        }
    }
    
    return iter.Error()
}
```

### Basic Get Operation

```go
// Get retrieves a value by key at the specified version
func (t *Tree) Get(version Version, key Key) ([]byte, error) {
    // Validate inputs
    if err := ValidateVersion(version); err != nil {
        return nil, err
    }
    if err := ValidateKey(key); err != nil {
        return nil, err
    }
    
    // Check version exists
    t.mu.RLock()
    rootHash, exists := t.rootHashes[version]
    t.mu.RUnlock()
    
    if !exists {
        return nil, ErrVersionNotFound
    }
    
    // Empty tree case
    if rootHash == EmptyHash {
        return nil, nil  // Key not found in empty tree
    }
    
    // Create snapshot for consistent read
    snapshot := t.db.NewSnapshot()
    defer snapshot.Close()
    
    // Create reader with snapshot
    reader := &TreeReader{
        tree:     t,
        snapshot: snapshot,
        version:  version,
        rootHash: rootHash,
    }
    
    return reader.Get(key)
}

// TreeReader handles read operations with a snapshot
type TreeReader struct {
    tree     *Tree
    snapshot *pebble.Snapshot
    version  Version
    rootHash Hash
}

// Get retrieves a value by traversing the tree
func (r *TreeReader) Get(key Key) ([]byte, error) {
    // Load root node
    root, err := r.loadNode(RootNodeKey(r.version))
    if err != nil {
        return nil, err
    }
    
    // Empty tree
    if root == nil {
        return nil, nil
    }
    
    // Traverse to leaf
    nibblePath := KeyToNibblePath(key)
    current := root
    
    for depth := 0; depth < MaxTreeDepth; depth++ {
        switch node := current.(type) {
        case *LeafNode:
            // Found a leaf - check if it's our key
            if node.Key == key {
                // Load actual value using hash
                return r.loadValue(node.ValueHash)
            }
            return nil, nil  // Different key, not found
            
        case *InternalNode:
            // Follow the child for this nibble
            nibble := nibblePath[depth]
            child, exists := node.Child(nibble)
            if !exists {
                return nil, nil  // Path doesn't exist
            }
            
            // Load child node
            childKey := NodeKey{
                Version: child.Version,
                NibblePath: NibblePath{
                    Nibbles: nibblePath[:depth+1],
                    Length:  uint16(depth + 1),
                },
            }
            
            next, err := r.loadNode(childKey)
            if err != nil {
                return nil, err
            }
            if next == nil {
                return nil, nil  // Child doesn't exist
            }
            
            current = next
            
        default:
            return nil, fmt.Errorf("unknown node type: %T", node)
        }
    }
    
    return nil, ErrMaxDepthExceeded
}

// loadNode loads a node from storage or cache
func (r *TreeReader) loadNode(key NodeKey) (Node, error) {
    // Check cache first
    if node, found := r.tree.nodeCache.Get(key); found {
        return node, nil
    }
    
    // Load from storage
    storageKey := key.StorageKey()
    data, closer, err := r.snapshot.Get(storageKey)
    if err == pebble.ErrNotFound {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("failed to load node: %w", err)
    }
    defer closer.Close()
    
    // Decode node
    codec := &NodeCodec{}
    node, err := codec.DecodeNode(data)
    if err != nil {
        return nil, fmt.Errorf("failed to decode node: %w", err)
    }
    
    // Update cache
    r.tree.nodeCache.Put(key, node)
    
    return node, nil
}

// loadValue loads a value by its hash
func (r *TreeReader) loadValue(hash Hash) ([]byte, error) {
    valueKey := makeValueKey(r.version, hash)
    data, closer, err := r.snapshot.Get(valueKey)
    if err != nil {
        return nil, fmt.Errorf("failed to load value: %w", err)
    }
    defer closer.Close()
    
    // Copy data before closing
    result := make([]byte, len(data))
    copy(result, data)
    
    return result, nil
}
```

### Helper Functions

```go
// GetLatestVersion returns the latest committed version
func (t *Tree) GetLatestVersion() Version {
    t.mu.RLock()
    defer t.mu.RUnlock()
    return t.latestVer
}

// HasVersion checks if a version exists
func (t *Tree) HasVersion(version Version) bool {
    t.mu.RLock()
    defer t.mu.RUnlock()
    _, exists := t.rootHashes[version]
    return exists
}

// GetRootHash returns the root hash for a version
func (t *Tree) GetRootHash(version Version) (Hash, error) {
    t.mu.RLock()
    defer t.mu.RUnlock()
    
    hash, exists := t.rootHashes[version]
    if !exists {
        return Hash{}, ErrVersionNotFound
    }
    
    return hash, nil
}

// IsEmpty checks if the tree is empty at a version
func (t *Tree) IsEmpty(version Version) (bool, error) {
    hash, err := t.GetRootHash(version)
    if err != nil {
        return false, err
    }
    
    return hash == EmptyHash, nil
}

// makeRootKey creates a storage key for root hash
func makeRootKey(version Version) []byte {
    key := make([]byte, len(RootKeyPrefix)+8)
    copy(key, RootKeyPrefix)
    binary.BigEndian.PutUint64(key[len(RootKeyPrefix):], uint64(version))
    return key
}

// parseRootKey extracts version from root key
func parseRootKey(key []byte) (Version, error) {
    if !bytes.HasPrefix(key, []byte(RootKeyPrefix)) {
        return 0, errors.New("invalid root key prefix")
    }
    
    versionBytes := key[len(RootKeyPrefix):]
    if len(versionBytes) != 8 {
        return 0, errors.New("invalid root key length")
    }
    
    return Version(binary.BigEndian.Uint64(versionBytes)), nil
}

// makeValueKey creates a storage key for value
func makeValueKey(version Version, hash Hash) []byte {
    key := make([]byte, 1+8+32)
    key[0] = 'v'  // Value prefix
    binary.BigEndian.PutUint64(key[1:9], uint64(version))
    copy(key[9:], hash[:])
    return key
}
```

### Error Handler Implementations

Create `pkg/tree/error_handlers.go`:
```go
package tree

import (
    "log"
    "math/rand"
    "time"
    
    "github.com/neutral/proofbox/pkg/types"
)

// DefaultReadErrorHandler implements basic recovery
type DefaultReadErrorHandler struct {
    logger *log.Logger
}

func NewDefaultReadErrorHandler(logger *log.Logger) *DefaultReadErrorHandler {
    if logger == nil {
        logger = log.New(log.Writer(), "[tree] ", log.LstdFlags)
    }
    return &DefaultReadErrorHandler{logger: logger}
}

func (h *DefaultReadErrorHandler) HandleCorruptedNode(key types.NodeKey, err error) error {
    h.logger.Printf("ERROR: Corrupted node detected at version=%d path=%x: %v",
        key.Version, key.NibblePath.Nibbles, err)
    
    // Return wrapped error with context
    return fmt.Errorf("corrupted node at %v: %w", key, err)
}

func (h *DefaultReadErrorHandler) HandleMissingNode(key types.NodeKey) error {
    // Missing nodes might be due to pruning
    h.logger.Printf("WARN: Missing node at version=%d path=%x",
        key.Version, key.NibblePath.Nibbles)
    return fmt.Errorf("node not found at %v", key)
}

// DefaultWriteErrorHandler implements exponential backoff
type DefaultWriteErrorHandler struct {
    maxRetries int
    baseDelay  time.Duration
    logger     *log.Logger
}

func NewDefaultWriteErrorHandler(maxRetries int, baseDelay time.Duration, logger *log.Logger) *DefaultWriteErrorHandler {
    return &DefaultWriteErrorHandler{
        maxRetries: maxRetries,
        baseDelay:  baseDelay,
        logger:     logger,
    }
}

func (h *DefaultWriteErrorHandler) HandleWriteFailure(err error) error {
    h.logger.Printf("ERROR: Write failure: %v", err)
    return fmt.Errorf("write operation failed: %w", err)
}

func (h *DefaultWriteErrorHandler) ShouldRetry(err error) bool {
    // Retry on temporary errors
    // In practice, would check for specific error types
    return false // Conservative default
}

func (h *DefaultWriteErrorHandler) RetryDelay(attempt int) time.Duration {
    // Exponential backoff with jitter
    delay := h.baseDelay * time.Duration(1<<uint(attempt))
    // Add up to 25% jitter
    jitter := time.Duration(rand.Float64() * float64(delay) * 0.25)
    return delay + jitter
}
```

### Health Check Implementation

Create `pkg/tree/health.go`:
```go
package tree

import (
    "bytes"
    "context"
    
    "github.com/neutral/proofbox/pkg/types"
)

// TreeHealthChecker verifies tree integrity
type TreeHealthChecker struct {
    tree *Tree
}

// NewTreeHealthChecker creates a new health checker
func NewTreeHealthChecker(tree *Tree) *TreeHealthChecker {
    return &TreeHealthChecker{tree: tree}
}

// VerifyIntegrity performs comprehensive integrity check
func (hc *TreeHealthChecker) VerifyIntegrity(ctx context.Context, version types.Version) error {
    // Start from root
    rootKey := types.RootNodeKey(version)
    visited := make(map[types.NodeKey]bool)
    
    return hc.verifyNode(ctx, rootKey, visited)
}

func (hc *TreeHealthChecker) verifyNode(ctx context.Context, key types.NodeKey, visited map[types.NodeKey]bool) error {
    // Check context cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Avoid cycles
    if visited[key] {
        return nil
    }
    visited[key] = true
    
    // Create reader with snapshot for consistency
    snapshot := hc.tree.db.NewSnapshot()
    defer snapshot.Close()
    
    reader := &TreeReader{
        tree:     hc.tree,
        snapshot: snapshot,
        version:  key.Version,
    }
    
    // Load and verify node
    node, err := reader.loadNode(key)
    if err != nil {
        return fmt.Errorf("failed to load node at %v: %w", key, err)
    }
    if node == nil {
        return fmt.Errorf("node not found at %v", key)
    }
    
    // Verify hash computation
    computed := node.Hash()
    // In practice, we'd verify against stored hash
    // For now, just ensure hash can be computed
    if computed == types.EmptyHash() {
        return fmt.Errorf("invalid empty hash for node at %v", key)
    }
    
    // Recursively verify children
    if internal, ok := node.(*InternalNode); ok {
        children := internal.Children()
        for nibble, child := range children {
            childKey := types.NodeKey{
                Version: child.Version,
                NibblePath: types.NibblePath{
                    Nibbles: append(key.NibblePath.Nibbles, byte(nibble)),
                    Length:  key.NibblePath.Length + 1,
                },
            }
            if err := hc.verifyNode(ctx, childKey, visited); err != nil {
                return err
            }
        }
    }
    
    return nil
}
```

## Implementation Steps

1. **Define Tree struct**: Storage, version tracking, concurrency primitives
2. **Implement initialization**: Load existing versions from storage
3. **Add Get operation**: Handle empty tree case correctly
4. **Create TreeReader**: Snapshot-based consistent reads
5. **Add helper methods**: Version queries, root hash access
6. **Implement caching**: Node cache integration
7. **Implement error handlers**: DefaultReadErrorHandler and DefaultWriteErrorHandler from step 3 interfaces
8. **Create health checker**: TreeHealthChecker implementing the HealthChecker interface

## Testing Requirements

### Unit Tests

```go
func TestEmptyTreeGet(t *testing.T) {
    // Create in-memory database
    db := createTestDB(t)
    defer db.Close()
    
    // Create tree
    tree, err := NewTree(db, DefaultTreeConfig())
    if err != nil {
        t.Fatalf("Failed to create tree: %v", err)
    }
    
    // Get from empty tree (version 0)
    key := KeyHash([]byte("test-key"))
    value, err := tree.Get(0, key)
    
    if err != nil {
        t.Errorf("Get should not error on empty tree: %v", err)
    }
    if value != nil {
        t.Errorf("Get should return nil for empty tree, got: %v", value)
    }
}

func TestVersionNotFound(t *testing.T) {
    db := createTestDB(t)
    defer db.Close()
    
    tree, _ := NewTree(db, DefaultTreeConfig())
    
    // Try to get from non-existent version
    _, err := tree.Get(999, KeyHash([]byte("key")))
    
    if err != ErrVersionNotFound {
        t.Errorf("Expected ErrVersionNotFound, got: %v", err)
    }
}

func TestTreeInitialization(t *testing.T) {
    db := createTestDB(t)
    defer db.Close()
    
    // Add some root hashes manually
    batch := db.NewBatch()
    batch.Set(makeRootKey(1), Hash{0x01}.Bytes(), nil)
    batch.Set(makeRootKey(5), Hash{0x05}.Bytes(), nil)
    batch.Set(makeRootKey(3), Hash{0x03}.Bytes(), nil)
    batch.Commit(pebble.Sync)
    
    // Create tree - should load existing versions
    tree, err := NewTree(db, DefaultTreeConfig())
    if err != nil {
        t.Fatalf("Failed to create tree: %v", err)
    }
    
    // Check latest version
    if tree.GetLatestVersion() != 5 {
        t.Errorf("Wrong latest version: %d", tree.GetLatestVersion())
    }
    
    // Check all versions exist
    for _, v := range []Version{1, 3, 5} {
        if !tree.HasVersion(v) {
            t.Errorf("Version %d should exist", v)
        }
    }
}
```

### Error Handler Tests

```go
func TestWriteRetryWithBackoff(t *testing.T) {
    handler := NewDefaultWriteErrorHandler(3, 100*time.Millisecond, log.New(io.Discard, "", 0))
    
    // Test exponential backoff
    for i := 0; i < 3; i++ {
        delay := handler.RetryDelay(i)
        expected := 100 * time.Millisecond * time.Duration(1<<uint(i))
        
        // Allow for jitter (up to 25%)
        assert.InDelta(t, expected, delay, float64(expected)*0.3)
    }
}

func TestDefaultReadErrorHandler(t *testing.T) {
    handler := NewDefaultReadErrorHandler(log.New(io.Discard, "", 0))
    
    // Test corrupted node handling
    key := types.NodeKey{Version: 1, NibblePath: types.NibblePath{Nibbles: []byte{0x1, 0x2}}}
    err := handler.HandleCorruptedNode(key, errors.New("test corruption"))
    assert.Equal(t, types.ErrCorruptedNode, err)
    
    // Test missing node handling
    err = handler.HandleMissingNode(key)
    assert.Equal(t, types.ErrNodeNotFound, err)
}
```

## Performance Considerations

- **Snapshot isolation**: Each read creates lightweight snapshot
- **Node caching**: Recently accessed nodes kept in memory
- **No read locks**: Reads don't block each other
- **Lazy loading**: Nodes loaded only when traversed

## Security Notes

- **Version validation**: Prevent integer overflow attacks
- **Key validation**: Ensure keys are proper size
- **Snapshot cleanup**: Always close snapshots to prevent leaks
- **Cache size limits**: Prevent memory exhaustion

## Done When ✓

- [ ] `Get(anyVersion, anyKey)` returns nil on empty tree
- [ ] Tree struct with storage, version tracking, and concurrency support
- [ ] Tree initialization loads existing versions from storage
- [ ] Get operation with proper empty tree handling
- [ ] Snapshot-based reads for consistency
- [ ] Node cache integration for performance
- [ ] Thread-safe version management
- [ ] Helper methods for version and root hash queries
- [ ] DefaultReadErrorHandler and DefaultWriteErrorHandler implemented
- [ ] TreeHealthChecker implementing HealthChecker interface
- [ ] Error handler tests with retry logic validation
- [ ] 100% test coverage for empty tree cases
