---
id: step.14.storage-layer
depends_on:
  - step.13.update-batch
tags: [storage, persistence, step]
---

## Objective

Implement complete storage layer with PebbleDB driver, multi-version persistence, and efficient key-value management for the Jellyfish Merkle Tree.

## Implements

- **Storage Abstraction** - Interface for pluggable storage backends
- **PebbleDB Integration** - LSM-optimized storage implementation
- **Multi-Version Persistence** - Historical state queries
- **Storage Efficiency Specification** (`/blueprint/global/storage/efficiency/specs/storage-efficiency.md`)

## Technical Details

### Storage Interface

Create `pkg/storage/interface.go`:
```go
package storage

import (
    "context"
    "io"
    
    "github.com/acme/jmt/pkg/types"
)

// Storage defines the interface for JMT storage backends
type Storage interface {
    // Get retrieves a value by key
    Get(key []byte) ([]byte, error)
    
    // Put stores a key-value pair
    Put(key, value []byte) error
    
    // Delete removes a key
    Delete(key []byte) error
    
    // NewBatch creates a new batch for atomic operations
    NewBatch() Batch
    
    // NewSnapshot creates a point-in-time snapshot
    NewSnapshot() Snapshot
    
    // Close closes the storage
    Close() error
    
    // Compact triggers storage compaction
    Compact(start, end []byte) error
    
    // EstimateSize estimates size of key range
    EstimateSize(start, end []byte) (uint64, error)
}

// Batch represents a batch of atomic operations
type Batch interface {
    // Put adds a put operation to the batch
    Put(key, value []byte) error
    
    // Delete adds a delete operation to the batch
    Delete(key []byte) error
    
    // Commit atomically applies all operations
    Commit(sync bool) error
    
    // Close discards the batch
    Close() error
    
    // Size returns the current batch size
    Size() int
}

// Snapshot represents a point-in-time view
type Snapshot interface {
    // Get retrieves a value from the snapshot
    Get(key []byte) ([]byte, error)
    
    // NewIterator creates an iterator over the snapshot
    NewIterator(opts *IteratorOptions) Iterator
    
    // Close releases the snapshot
    Close() error
}

// Iterator iterates over key-value pairs
type Iterator interface {
    // Valid returns true if iterator is positioned at valid entry
    Valid() bool
    
    // Next advances to the next entry
    Next() bool
    
    // Prev moves to the previous entry
    Prev() bool
    
    // Seek positions at the first key >= target
    Seek(key []byte) bool
    
    // First positions at the first entry
    First() bool
    
    // Last positions at the last entry
    Last() bool
    
    // Key returns the current key
    Key() []byte
    
    // Value returns the current value
    Value() []byte
    
    // Error returns any error encountered
    Error() error
    
    // Close releases the iterator
    Close() error
}

// IteratorOptions configures iteration
type IteratorOptions struct {
    // Start and end keys (inclusive)
    LowerBound []byte
    UpperBound []byte
    
    // Iteration direction
    Reverse bool
    
    // Prefix iteration
    Prefix []byte
}
```

### PebbleDB Implementation

Create `pkg/storage/pebble/pebble.go`:
```go
package pebble

import (
    "sync"
    "sync/atomic"
    
    "github.com/cockroachdb/pebble"
    "github.com/acme/jmt/pkg/storage"
    "github.com/acme/jmt/pkg/types"
)

// DB wraps a PebbleDB instance
type DB struct {
    db      *pebble.DB
    opts    *Options
    metrics *Metrics
    
    // Stats
    reads   atomic.Uint64
    writes  atomic.Uint64
}

// Options configures PebbleDB
type Options struct {
    // Cache size in bytes
    CacheSize int64
    
    // Max open files
    MaxOpenFiles int
    
    // Write buffer size
    MemTableSize int
    
    // Compression
    Compression pebble.Compression
    
    // Sync writes
    SyncWrites bool
    
    // Custom comparator
    Comparer *pebble.Comparer
}

// DefaultOptions returns recommended options
func DefaultOptions() *Options {
    return &Options{
        CacheSize:    256 << 20, // 256 MB
        MaxOpenFiles: 1000,
        MemTableSize: 64 << 20,  // 64 MB
        Compression:  pebble.SnappyCompression,
        SyncWrites:   true,
        Comparer:     VersionedKeyComparer(),
    }
}

// VersionedKeyComparer optimizes for version-prefixed keys
func VersionedKeyComparer() *pebble.Comparer {
    return &pebble.Comparer{
        Compare: func(a, b []byte) int {
            // Custom comparison for NodeKey format
            // Ensures version ordering within same path
            return bytes.Compare(a, b)
        },
        Name: "jmt.version-key.v1",
    }
}

// Open creates or opens a PebbleDB instance
func Open(path string, opts *Options) (*DB, error) {
    if opts == nil {
        opts = DefaultOptions()
    }
    
    pebbleOpts := &pebble.Options{
        Cache:          pebble.NewCache(opts.CacheSize),
        MaxOpenFiles:   opts.MaxOpenFiles,
        MemTableSize:   opts.MemTableSize,
        Comparer:       opts.Comparer,
        FormatMajorVersion: pebble.FormatNewest,
    }
    
    // Configure compression
    pebbleOpts.Levels = make([]pebble.LevelOptions, 7)
    for i := range pebbleOpts.Levels {
        pebbleOpts.Levels[i].Compression = opts.Compression
    }
    
    db, err := pebble.Open(path, pebbleOpts)
    if err != nil {
        return nil, types.WrapError(err, "failed to open pebble database")
    }
    
    return &DB{
        db:      db,
        opts:    opts,
        metrics: NewMetrics(),
    }, nil
}

// Get retrieves a value by key
func (d *DB) Get(key []byte) ([]byte, error) {
    d.reads.Add(1)
    
    value, closer, err := d.db.Get(key)
    if err != nil {
        if err == pebble.ErrNotFound {
            return nil, types.ErrKeyNotFound
        }
        return nil, types.WrapError(err, "get failed")
    }
    defer closer.Close()
    
    // Copy value since it's only valid until closer is closed
    result := make([]byte, len(value))
    copy(result, value)
    
    return result, nil
}

// Put stores a key-value pair
func (d *DB) Put(key, value []byte) error {
    d.writes.Add(1)
    
    opts := pebble.NoSync
    if d.opts.SyncWrites {
        opts = pebble.Sync
    }
    
    if err := d.db.Set(key, value, opts); err != nil {
        return types.WrapError(err, "put failed")
    }
    
    return nil
}

// Delete removes a key
func (d *DB) Delete(key []byte) error {
    d.writes.Add(1)
    
    opts := pebble.NoSync
    if d.opts.SyncWrites {
        opts = pebble.Sync
    }
    
    if err := d.db.Delete(key, opts); err != nil {
        return types.WrapError(err, "delete failed")
    }
    
    return nil
}

// NewBatch creates a new batch
func (d *DB) NewBatch() storage.Batch {
    return &Batch{
        batch: d.db.NewBatch(),
        db:    d,
    }
}

// NewSnapshot creates a snapshot
func (d *DB) NewSnapshot() storage.Snapshot {
    return &Snapshot{
        snapshot: d.db.NewSnapshot(),
    }
}

// Close closes the database
func (d *DB) Close() error {
    return d.db.Close()
}

// Compact triggers manual compaction
func (d *DB) Compact(start, end []byte) error {
    return d.db.Compact(start, end, true)
}

// EstimateSize estimates size of key range
func (d *DB) EstimateSize(start, end []byte) (uint64, error) {
    sizes, err := d.db.EstimateDiskUsage(start, end)
    if err != nil {
        return 0, err
    }
    return sizes, nil
}
```

### Batch Implementation

Add to `pkg/storage/pebble/batch.go`:
```go
package pebble

import (
    "github.com/cockroachdb/pebble"
    "github.com/acme/jmt/pkg/storage"
)

// Batch implements storage.Batch
type Batch struct {
    batch *pebble.Batch
    db    *DB
    size  int
}

// Put adds a put operation
func (b *Batch) Put(key, value []byte) error {
    if err := b.batch.Set(key, value, nil); err != nil {
        return types.WrapError(err, "batch put failed")
    }
    b.size++
    return nil
}

// Delete adds a delete operation
func (b *Batch) Delete(key []byte) error {
    if err := b.batch.Delete(key, nil); err != nil {
        return types.WrapError(err, "batch delete failed")
    }
    b.size++
    return nil
}

// Commit applies all operations atomically
func (b *Batch) Commit(sync bool) error {
    opts := pebble.NoSync
    if sync {
        opts = pebble.Sync
    }
    
    if err := b.batch.Commit(opts); err != nil {
        return types.WrapError(err, "batch commit failed")
    }
    
    b.db.writes.Add(uint64(b.size))
    return nil
}

// Close discards the batch
func (b *Batch) Close() error {
    return b.batch.Close()
}

// Size returns operation count
func (b *Batch) Size() int {
    return b.size
}
```

### Multi-Version Storage

Create `pkg/storage/versioned.go`:
```go
package storage

import (
    "encoding/binary"
    "sync"
    
    "github.com/acme/jmt/pkg/types"
)

// VersionedStorage manages multi-version state
type VersionedStorage struct {
    storage Storage
    
    mu            sync.RWMutex
    latestVersion types.Version
    rootHashes    map[types.Version]types.Hash // Version -> root hash cache
    pruneHeight   int                           // Number of versions to keep
}

// NewVersionedStorage creates versioned storage wrapper
func NewVersionedStorage(storage Storage, pruneHeight int) *VersionedStorage {
    return &VersionedStorage{
        storage:     storage,
        rootHashes:  make(map[types.Version]types.Hash),
        pruneHeight: pruneHeight,
    }
}

// Initialize loads existing versions from storage
func (vs *VersionedStorage) Initialize() error {
    // Scan for existing root hashes
    prefix := []byte("root:")
    iter := vs.storage.NewSnapshot().NewIterator(&IteratorOptions{
        Prefix: prefix,
    })
    defer iter.Close()
    
    vs.mu.Lock()
    defer vs.mu.Unlock()
    
    for iter.First(); iter.Valid(); iter.Next() {
        // Parse version from key
        key := iter.Key()
        versionBytes := key[len(prefix):]
        version := types.Version(binary.BigEndian.Uint64(versionBytes))
        
        // Parse root hash
        var rootHash types.Hash
        copy(rootHash[:], iter.Value())
        
        vs.rootHashes[version] = rootHash
        if version > vs.latestVersion {
            vs.latestVersion = version
        }
    }
    
    return iter.Error()
}

// GetLatestVersion returns the most recent version
func (vs *VersionedStorage) GetLatestVersion() types.Version {
    vs.mu.RLock()
    defer vs.mu.RUnlock()
    return vs.latestVersion
}

// GetRootHash retrieves root hash for a version
func (vs *VersionedStorage) GetRootHash(version types.Version) (types.Hash, error) {
    vs.mu.RLock()
    defer vs.mu.RUnlock()
    
    hash, exists := vs.rootHashes[version]
    if !exists {
        return types.Hash{}, types.ErrVersionNotFound
    }
    
    return hash, nil
}

// CommitVersion stores a new version
func (vs *VersionedStorage) CommitVersion(version types.Version, rootHash types.Hash, batch Batch) error {
    vs.mu.Lock()
    defer vs.mu.Unlock()
    
    if version <= vs.latestVersion {
        return types.WrapError(types.ErrInvalidVersion, "version must be greater than %d", vs.latestVersion)
    }
    
    // Store root hash
    rootKey := vs.rootKey(version)
    if err := batch.Put(rootKey, rootHash[:]); err != nil {
        return err
    }
    
    // Commit batch
    if err := batch.Commit(true); err != nil {
        return err
    }
    
    // Update cache
    vs.rootHashes[version] = rootHash
    vs.latestVersion = version
    
    // Trigger pruning if needed
    if vs.pruneHeight > 0 && len(vs.rootHashes) > vs.pruneHeight {
        go vs.pruneOldVersions()
    }
    
    return nil
}

// GetNode retrieves a node at specific version
func (vs *VersionedStorage) GetNode(nodeKey types.NodeKey) ([]byte, error) {
    storageKey := nodeKey.StorageKey()
    return vs.storage.Get(storageKey)
}

// rootKey generates storage key for root hash
func (vs *VersionedStorage) rootKey(version types.Version) []byte {
    key := make([]byte, 13) // "root:" + 8 bytes
    copy(key, "root:")
    binary.BigEndian.PutUint64(key[5:], uint64(version))
    return key
}

// pruneOldVersions removes old versions
func (vs *VersionedStorage) pruneOldVersions() {
    vs.mu.Lock()
    defer vs.mu.Unlock()
    
    if len(vs.rootHashes) <= vs.pruneHeight {
        return
    }
    
    // Find versions to prune
    versions := make([]types.Version, 0, len(vs.rootHashes))
    for v := range vs.rootHashes {
        versions = append(versions, v)
    }
    sort.Slice(versions, func(i, j int) bool {
        return versions[i] < versions[j]
    })
    
    // Prune oldest versions
    toPrune := versions[:len(versions)-vs.pruneHeight]
    
    batch := vs.storage.NewBatch()
    defer batch.Close()
    
    for _, version := range toPrune {
        // Delete root hash
        batch.Delete(vs.rootKey(version))
        delete(vs.rootHashes, version)
        
        // TODO: Delete all nodes for this version
        // This requires scanning all nodes with this version prefix
    }
    
    batch.Commit(false) // Async commit for pruning
}

// QueryAtVersion creates a reader for specific version
func (vs *VersionedStorage) QueryAtVersion(version types.Version) (*VersionReader, error) {
    rootHash, err := vs.GetRootHash(version)
    if err != nil {
        return nil, err
    }
    
    return &VersionReader{
        storage:  vs,
        version:  version,
        rootHash: rootHash,
        snapshot: vs.storage.NewSnapshot(),
    }, nil
}

// VersionReader reads tree at specific version
type VersionReader struct {
    storage  *VersionedStorage
    version  types.Version
    rootHash types.Hash
    snapshot Snapshot
}

// GetNode retrieves a node
func (vr *VersionReader) GetNode(nodeKey types.NodeKey) ([]byte, error) {
    // For historical queries, we need to find the right version
    // This is where the version-based key ordering helps
    
    storageKey := nodeKey.StorageKey()
    return vr.snapshot.Get(storageKey)
}

// Close releases the reader
func (vr *VersionReader) Close() error {
    return vr.snapshot.Close()
}
```

## Testing Requirements

### Storage Interface Tests
```go
func TestStorageInterface(t *testing.T) {
    tmpDir := t.TempDir()
    db, err := pebble.Open(tmpDir, nil)
    assert.NoError(t, err)
    defer db.Close()
    
    // Test basic operations
    key := []byte("test-key")
    value := []byte("test-value")
    
    // Put
    err = db.Put(key, value)
    assert.NoError(t, err)
    
    // Get
    retrieved, err := db.Get(key)
    assert.NoError(t, err)
    assert.Equal(t, value, retrieved)
    
    // Delete
    err = db.Delete(key)
    assert.NoError(t, err)
    
    // Get deleted
    _, err = db.Get(key)
    assert.ErrorIs(t, err, types.ErrKeyNotFound)
}

func TestBatchOperations(t *testing.T) {
    db := setupTestDB(t)
    
    batch := db.NewBatch()
    
    // Add operations
    for i := 0; i < 100; i++ {
        key := []byte(fmt.Sprintf("key-%d", i))
        value := []byte(fmt.Sprintf("value-%d", i))
        err := batch.Put(key, value)
        assert.NoError(t, err)
    }
    
    // Commit
    err := batch.Commit(true)
    assert.NoError(t, err)
    
    // Verify all written
    for i := 0; i < 100; i++ {
        key := []byte(fmt.Sprintf("key-%d", i))
        value, err := db.Get(key)
        assert.NoError(t, err)
        assert.Equal(t, fmt.Sprintf("value-%d", i), string(value))
    }
}

func TestVersionedStorage(t *testing.T) {
    db := setupTestDB(t)
    vs := NewVersionedStorage(db, 10)
    
    // Initialize
    err := vs.Initialize()
    assert.NoError(t, err)
    
    // Commit versions
    for v := types.Version(1); v <= 5; v++ {
        batch := db.NewBatch()
        rootHash := types.Hash{byte(v)} // Dummy hash
        
        err := vs.CommitVersion(v, rootHash, batch)
        assert.NoError(t, err)
    }
    
    // Query historical version
    reader, err := vs.QueryAtVersion(3)
    assert.NoError(t, err)
    defer reader.Close()
    
    assert.Equal(t, types.Version(3), reader.version)
    assert.Equal(t, types.Hash{3}, reader.rootHash)
}
```

### Performance Benchmarks
```go
func BenchmarkPebbleWrites(b *testing.B) {
    db := setupBenchDB(b)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        key := make([]byte, 32)
        binary.BigEndian.PutUint64(key, uint64(i))
        value := make([]byte, 100)
        
        if err := db.Put(key, value); err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkBatchWrites(b *testing.B) {
    db := setupBenchDB(b)
    batchSize := 1000
    
    b.ResetTimer()
    for i := 0; i < b.N; i += batchSize {
        batch := db.NewBatch()
        
        for j := 0; j < batchSize && i+j < b.N; j++ {
            key := make([]byte, 32)
            binary.BigEndian.PutUint64(key, uint64(i+j))
            value := make([]byte, 100)
            
            batch.Put(key, value)
        }
        
        if err := batch.Commit(false); err != nil {
            b.Fatal(err)
        }
    }
}
```

## Implementation Steps

1. Create storage interface in `pkg/storage/interface.go`
2. Implement PebbleDB wrapper in `pkg/storage/pebble/`
3. Add batch and snapshot implementations
4. Create versioned storage layer
5. Implement version-based key comparator
6. Add pruning mechanism for old versions
7. Write comprehensive tests
8. Add storage metrics and monitoring
9. Benchmark and optimize performance

## Performance Considerations

- Use batch operations for multiple writes
- Configure appropriate cache and buffer sizes
- Implement key prefix compression
- Use async writes where durability permits
- Monitor and tune compaction settings

## Security Notes

- Validate all keys and values
- Set appropriate file permissions on database
- Implement storage encryption if needed
- Add integrity checks for critical data

## Done When ✓

- [ ] Storage interface defined
- [ ] PebbleDB implementation complete
- [ ] Batch operations working atomically
- [ ] Snapshot isolation implemented
- [ ] Multi-version storage with root tracking
- [ ] Historical queries functional
- [ ] Version pruning mechanism
- [ ] Storage metrics collection
- [ ] Comprehensive test coverage
- [ ] Performance benchmarks passing
- [ ] Documentation complete