---
id: step.16.update‑batch
depends_on:
  - step.15.versioning
tags: [batch, step]
---

## Objective

Return `UpdateBatch{NewNodes, StaleNodeKeys}` from each commit.

## Implements

- **§6.2** end paragraph ("TreeUpdateBatch containing node\*batch + stale*node_index_batch").
  \_What happens*:

  - Counts newly minted vs stale NodeKeys, setting up later Pebble persistence.

## Technical Details

### Update Batch Architecture

The UpdateBatch encapsulates all changes made during a tree update operation. It tracks both new nodes created and old nodes that became stale, enabling atomic commits and efficient storage operations.

### Core Batch Implementation

```go
// UpdateBatch represents all changes in a single version update
type UpdateBatch struct {
    // New nodes to write
    NewNodes map[NodeKey]Node
    
    // Nodes that became stale
    StaleNodeKeys []NodeKey
    
    // Root hash after update
    NewRootHash Hash
    
    // Version information
    OldVersion Version
    NewVersion Version
    
    // Statistics
    Stats BatchStats
}

// BatchStats tracks operation metrics
type BatchStats struct {
    NodesCreated   int
    NodesModified  int
    NodesReused    int
    LeafNodes      int
    InternalNodes  int
    BytesWritten   int64
    StartTime      time.Time
    EndTime        time.Time
}

// BatchBuilder constructs update batches efficiently
type BatchBuilder struct {
    oldVersion Version
    newVersion Version
    newNodes   map[NodeKey]Node
    staleNodes map[NodeKey]bool
    nodeCodec  *NodeCodec
    stats      BatchStats
}

// NewBatchBuilder creates a batch builder
func NewBatchBuilder(oldVersion, newVersion Version) *BatchBuilder {
    return &BatchBuilder{
        oldVersion: oldVersion,
        newVersion: newVersion,
        newNodes:   make(map[NodeKey]Node),
        staleNodes: make(map[NodeKey]bool),
        nodeCodec:  &NodeCodec{},
        stats: BatchStats{
            StartTime: time.Now(),
        },
    }
}

// AddNode adds a new node to the batch
func (bb *BatchBuilder) AddNode(key NodeKey, node Node) error {
    // Validate node
    if node == nil {
        return errors.New("cannot add nil node")
    }
    
    if key.Version != bb.newVersion {
        return fmt.Errorf("node version %d doesn't match batch version %d", 
            key.Version, bb.newVersion)
    }
    
    // Check for duplicates
    if _, exists := bb.newNodes[key]; exists {
        return fmt.Errorf("duplicate node key: %v", key)
    }
    
    // Add to batch
    bb.newNodes[key] = node
    
    // Update statistics
    switch node.(type) {
    case *LeafNode:
        bb.stats.LeafNodes++
    case *InternalNode:
        bb.stats.InternalNodes++
    }
    bb.stats.NodesCreated++
    
    // Calculate size
    encoded, err := bb.nodeCodec.EncodeNode(node)
    if err == nil {
        bb.stats.BytesWritten += int64(len(encoded))
    }
    
    return nil
}

// MarkStale marks a node as stale
func (bb *BatchBuilder) MarkStale(key NodeKey) {
    if !bb.staleNodes[key] {
        bb.staleNodes[key] = true
        bb.stats.NodesModified++
    }
}

// Build finalizes the batch
func (bb *BatchBuilder) Build() (*UpdateBatch, error) {
    bb.stats.EndTime = time.Now()
    
    // Convert stale nodes map to slice
    staleKeys := make([]NodeKey, 0, len(bb.staleNodes))
    for key := range bb.staleNodes {
        staleKeys = append(staleKeys, key)
    }
    
    // Sort for deterministic order
    sort.Slice(staleKeys, func(i, j int) bool {
        return staleKeys[i].Less(staleKeys[j])
    })
    
    // Calculate root hash
    rootKey := NodeKey{Version: bb.newVersion, Path: NibblePath{}}
    rootNode, exists := bb.newNodes[rootKey]
    if !exists {
        return nil, errors.New("no root node in batch")
    }
    
    batch := &UpdateBatch{
        NewNodes:      bb.newNodes,
        StaleNodeKeys: staleKeys,
        NewRootHash:   rootNode.Hash(),
        OldVersion:    bb.oldVersion,
        NewVersion:    bb.newVersion,
        Stats:         bb.stats,
    }
    
    return batch, nil
}
```

### Transaction Support

```go
// Transaction represents an atomic set of operations
type Transaction struct {
    tree      *Tree
    version   Version
    ops       []Operation
    isolation IsolationLevel
    readCache map[Key][]byte
    mu        sync.Mutex
}

// Operation represents a single tree operation
type Operation struct {
    Type  OpType
    Key   Key
    Value []byte
}

type OpType int

const (
    OpTypePut OpType = iota
    OpTypeDelete
)

type IsolationLevel int

const (
    IsolationSnapshot IsolationLevel = iota
    IsolationSerializable
)

// BeginTransaction starts a new transaction
func (t *Tree) BeginTransaction(isolation IsolationLevel) (*Transaction, error) {
    version, err := t.BeginVersion()
    if err != nil {
        return nil, fmt.Errorf("failed to begin version: %w", err)
    }
    
    return &Transaction{
        tree:      t,
        version:   version,
        isolation: isolation,
        readCache: make(map[Key][]byte),
    }, nil
}

// Put adds a put operation to the transaction
func (tx *Transaction) Put(key Key, value []byte) error {
    tx.mu.Lock()
    defer tx.mu.Unlock()
    
    tx.ops = append(tx.ops, Operation{
        Type:  OpTypePut,
        Key:   key,
        Value: value,
    })
    
    // Update read cache for read-your-writes
    tx.readCache[key] = value
    
    return nil
}

// Get reads a value within the transaction
func (tx *Transaction) Get(key Key) ([]byte, error) {
    tx.mu.Lock()
    defer tx.mu.Unlock()
    
    // Check read cache first
    if value, exists := tx.readCache[key]; exists {
        return value, nil
    }
    
    // Read from tree at transaction start version
    return tx.tree.GetAtVersion(tx.version-1, key)
}

// Commit applies all operations atomically
func (tx *Transaction) Commit() error {
    tx.mu.Lock()
    defer tx.mu.Unlock()
    
    // Build batch from operations
    builder := NewBatchBuilder(tx.version-1, tx.version)
    updater := &TreeUpdater{
        tree:       tx.tree,
        batch:      builder,
        oldVersion: tx.version - 1,
        newVersion: tx.version,
    }
    
    // Apply all operations
    for _, op := range tx.ops {
        switch op.Type {
        case OpTypePut:
            if err := updater.Put(op.Key, op.Value); err != nil {
                return fmt.Errorf("put failed: %w", err)
            }
        case OpTypeDelete:
            if err := updater.Delete(op.Key); err != nil {
                return fmt.Errorf("delete failed: %w", err)
            }
        }
    }
    
    // Build final batch
    batch, err := builder.Build()
    if err != nil {
        return fmt.Errorf("failed to build batch: %w", err)
    }
    
    // Commit to tree
    return tx.tree.CommitBatch(batch)
}

// Rollback cancels the transaction
func (tx *Transaction) Rollback() error {
    return tx.tree.versionManager.Abort(tx.version)
}
```

### Batch Optimization

```go
// BatchOptimizer reduces batch size and improves performance
type BatchOptimizer struct {
    deduplicator *NodeDeduplicator
    compressor   *BatchCompressor
}

// NodeDeduplicator eliminates duplicate nodes
type NodeDeduplicator struct {
    nodeHashes map[Hash]NodeKey
}

// Deduplicate removes duplicate nodes from batch
func (nd *NodeDeduplicator) Deduplicate(batch *UpdateBatch) (*UpdateBatch, error) {
    nd.nodeHashes = make(map[Hash]NodeKey)
    optimized := make(map[NodeKey]Node)
    
    // First pass: build hash index
    for key, node := range batch.NewNodes {
        hash := node.Hash()
        if existingKey, exists := nd.nodeHashes[hash]; exists {
            // Duplicate found - reuse existing
            log.Printf("Deduped node %v -> %v", key, existingKey)
        } else {
            nd.nodeHashes[hash] = key
            optimized[key] = node
        }
    }
    
    // Create optimized batch
    return &UpdateBatch{
        NewNodes:      optimized,
        StaleNodeKeys: batch.StaleNodeKeys,
        NewRootHash:   batch.NewRootHash,
        OldVersion:    batch.OldVersion,
        NewVersion:    batch.NewVersion,
        Stats:         batch.Stats,
    }, nil
}

// BatchCompressor compresses batch data
type BatchCompressor struct {
    compressionLevel int
}

// CompressBatch reduces batch size for storage
func (bc *BatchCompressor) CompressBatch(batch *UpdateBatch) ([]byte, error) {
    // Serialize batch
    var buf bytes.Buffer
    encoder := gob.NewEncoder(&buf)
    
    if err := encoder.Encode(batch); err != nil {
        return nil, fmt.Errorf("failed to encode batch: %w", err)
    }
    
    // Compress with zstd
    compressed := make([]byte, 0, buf.Len()/2)
    encoder, _ := zstd.NewWriter(nil,
        zstd.WithEncoderLevel(zstd.EncoderLevel(bc.compressionLevel)))
    
    compressed = encoder.EncodeAll(buf.Bytes(), compressed)
    
    // Log compression ratio
    ratio := float64(len(compressed)) / float64(buf.Len())
    log.Printf("Batch compressed: %d -> %d bytes (%.1f%%)", 
        buf.Len(), len(compressed), ratio*100)
    
    return compressed, nil
}
```

### Parallel Batch Processing

```go
// ParallelBatchProcessor handles large batches efficiently
type ParallelBatchProcessor struct {
    workers    int
    bufferSize int
}

// ProcessBatch applies batch operations in parallel
func (pbp *ParallelBatchProcessor) ProcessBatch(tree *Tree, ops []Operation) (*UpdateBatch, error) {
    if pbp.workers <= 0 {
        pbp.workers = runtime.NumCPU()
    }
    
    // Start version
    version, err := tree.BeginVersion()
    if err != nil {
        return nil, err
    }
    
    // Partition operations by key prefix
    partitions := pbp.partitionOps(ops, pbp.workers)
    
    // Process partitions in parallel
    results := make(chan partitionResult, pbp.workers)
    var wg sync.WaitGroup
    
    for i, partition := range partitions {
        wg.Add(1)
        go func(id int, ops []Operation) {
            defer wg.Done()
            
            result := pbp.processPartition(tree, version, id, ops)
            results <- result
        }(i, partition)
    }
    
    // Wait for completion
    wg.Wait()
    close(results)
    
    // Merge results
    return pbp.mergeResults(results, version)
}

// partitionOps divides operations by key prefix
func (pbp *ParallelBatchProcessor) partitionOps(ops []Operation, numPartitions int) [][]Operation {
    partitions := make([][]Operation, numPartitions)
    
    for _, op := range ops {
        // Hash key to determine partition
        h := fnv.New32a()
        h.Write(op.Key[:])
        partition := h.Sum32() % uint32(numPartitions)
        
        partitions[partition] = append(partitions[partition], op)
    }
    
    return partitions
}

type partitionResult struct {
    id       int
    newNodes map[NodeKey]Node
    stale    []NodeKey
    err      error
}

// processPartition handles operations for one partition
func (pbp *ParallelBatchProcessor) processPartition(tree *Tree, version Version, 
    id int, ops []Operation) partitionResult {
    
    result := partitionResult{
        id:       id,
        newNodes: make(map[NodeKey]Node),
    }
    
    // Create local updater
    updater := &TreeUpdater{
        tree:       tree,
        oldVersion: version - 1,
        newVersion: version,
        nodeWrites: make(map[NodeKey]Node),
    }
    
    // Process operations
    for _, op := range ops {
        var err error
        switch op.Type {
        case OpTypePut:
            _, err = updater.Put(op.Key, op.Value)
        case OpTypeDelete:
            err = updater.Delete(op.Key)
        }
        
        if err != nil {
            result.err = err
            return result
        }
    }
    
    // Collect results
    result.newNodes = updater.nodeWrites
    result.stale = updater.staleNodes
    
    return result
}

// mergeResults combines partition results
func (pbp *ParallelBatchProcessor) mergeResults(results <-chan partitionResult, 
    version Version) (*UpdateBatch, error) {
    
    builder := NewBatchBuilder(version-1, version)
    
    // Merge all partition results
    for result := range results {
        if result.err != nil {
            return nil, fmt.Errorf("partition %d failed: %w", result.id, result.err)
        }
        
        // Add nodes
        for key, node := range result.newNodes {
            if err := builder.AddNode(key, node); err != nil {
                return nil, err
            }
        }
        
        // Mark stale
        for _, key := range result.stale {
            builder.MarkStale(key)
        }
    }
    
    return builder.Build()
}
```

### Batch Validation

```go
// BatchValidator ensures batch integrity
type BatchValidator struct {
    tree *Tree
}

// ValidateBatch checks batch consistency
func (bv *BatchValidator) ValidateBatch(batch *UpdateBatch) error {
    // Check version continuity
    if batch.NewVersion != batch.OldVersion+1 {
        return fmt.Errorf("non-sequential versions: %d -> %d", 
            batch.OldVersion, batch.NewVersion)
    }
    
    // Verify root exists
    rootKey := NodeKey{Version: batch.NewVersion, Path: NibblePath{}}
    if _, exists := batch.NewNodes[rootKey]; !exists {
        return errors.New("batch missing root node")
    }
    
    // Check node relationships
    if err := bv.validateNodeReferences(batch); err != nil {
        return fmt.Errorf("invalid node references: %w", err)
    }
    
    // Verify no cycles
    if err := bv.checkCycles(batch); err != nil {
        return fmt.Errorf("cycle detected: %w", err)
    }
    
    // Validate statistics
    if err := bv.validateStats(batch); err != nil {
        return fmt.Errorf("invalid statistics: %w", err)
    }
    
    return nil
}

// validateNodeReferences ensures all child references are valid
func (bv *BatchValidator) validateNodeReferences(batch *UpdateBatch) error {
    for key, node := range batch.NewNodes {
        if internal, ok := node.(*InternalNode); ok {
            for nibble, child := range internal.children {
                childPath := key.Path.Append(nibble)
                childKey := NodeKey{
                    Version: child.Version,
                    Path:    childPath,
                }
                
                // Child must exist in batch or storage
                if child.Version == batch.NewVersion {
                    if _, exists := batch.NewNodes[childKey]; !exists {
                        return fmt.Errorf("missing child node: %v", childKey)
                    }
                }
            }
        }
    }
    return nil
}

// checkCycles detects circular references
func (bv *BatchValidator) checkCycles(batch *UpdateBatch) error {
    visited := make(map[NodeKey]bool)
    visiting := make(map[NodeKey]bool)
    
    var visit func(NodeKey) error
    visit = func(key NodeKey) error {
        if visiting[key] {
            return fmt.Errorf("cycle at node %v", key)
        }
        if visited[key] {
            return nil
        }
        
        visiting[key] = true
        
        // Check children
        if node, exists := batch.NewNodes[key]; exists {
            if internal, ok := node.(*InternalNode); ok {
                for nibble, child := range internal.children {
                    childKey := NodeKey{
                        Version: child.Version,
                        Path:    key.Path.Append(nibble),
                    }
                    if err := visit(childKey); err != nil {
                        return err
                    }
                }
            }
        }
        
        visiting[key] = false
        visited[key] = true
        return nil
    }
    
    // Start from root
    rootKey := NodeKey{Version: batch.NewVersion, Path: NibblePath{}}
    return visit(rootKey)
}
```

## Implementation Steps

1. **Create batch structure**: Define UpdateBatch and builder
2. **Implement transaction support**: Atomic multi-operation updates
3. **Add batch optimization**: Deduplication and compression
4. **Implement parallel processing**: Handle large batches efficiently
5. **Add validation**: Ensure batch integrity
6. **Integrate with storage**: Atomic batch commits

## Testing Requirements

### Basic Batch Tests

```go
func TestBasicBatch(t *testing.T) {
    tree := createTestTree(t)
    
    // Start transaction
    tx, err := tree.BeginTransaction(IsolationSnapshot)
    if err != nil {
        t.Fatalf("Failed to begin transaction: %v", err)
    }
    
    // Add operations
    keys := []Key{
        KeyHash([]byte("key1")),
        KeyHash([]byte("key2")),
        KeyHash([]byte("key3")),
    }
    
    for i, key := range keys {
        err := tx.Put(key, []byte(fmt.Sprintf("value%d", i)))
        if err != nil {
            t.Fatalf("Put failed: %v", err)
        }
    }
    
    // Commit transaction
    err = tx.Commit()
    if err != nil {
        t.Fatalf("Commit failed: %v", err)
    }
    
    // Verify all keys exist
    version := tree.GetLatestVersion()
    for i, key := range keys {
        value, err := tree.GetAtVersion(version, key)
        if err != nil {
            t.Errorf("Failed to get key %d: %v", i, err)
        }
        
        expected := []byte(fmt.Sprintf("value%d", i))
        if !bytes.Equal(value, expected) {
            t.Errorf("Value mismatch for key %d", i)
        }
    }
}

func TestBatchNodeCounting(t *testing.T) {
    tree := createTestTree(t)
    
    // Track batch statistics
    var capturedBatch *UpdateBatch
    tree.beforeCommit = func(batch *UpdateBatch) {
        capturedBatch = batch
    }
    
    // Insert keys that will create internal nodes
    tx, _ := tree.BeginTransaction(IsolationSnapshot)
    
    keys := []Key{
        {0x10, 0x00},
        {0x10, 0x01},
        {0x20, 0x00},
    }
    
    for _, key := range keys {
        tx.Put(key, []byte("value"))
    }
    
    tx.Commit()
    
    // Verify batch statistics
    if capturedBatch == nil {
        t.Fatal("Batch not captured")
    }
    
    // Should have leaves + internal nodes
    totalNodes := capturedBatch.Stats.LeafNodes + capturedBatch.Stats.InternalNodes
    if totalNodes != len(capturedBatch.NewNodes) {
        t.Errorf("Node count mismatch: stats=%d, actual=%d", 
            totalNodes, len(capturedBatch.NewNodes))
    }
    
    t.Logf("Batch stats: %d leaves, %d internal, %d bytes",
        capturedBatch.Stats.LeafNodes,
        capturedBatch.Stats.InternalNodes,
        capturedBatch.Stats.BytesWritten)
}

func TestBatchDeduplication(t *testing.T) {
    tree := createTestTree(t)
    
    // Insert duplicate values
    tx, _ := tree.BeginTransaction(IsolationSnapshot)
    
    duplicateValue := []byte("same-value")
    for i := 0; i < 10; i++ {
        key := KeyHash([]byte(fmt.Sprintf("key%d", i)))
        tx.Put(key, duplicateValue)
    }
    
    // Capture batch before optimization
    var originalBatch *UpdateBatch
    tree.beforeOptimization = func(batch *UpdateBatch) {
        originalBatch = batch
    }
    
    tx.Commit()
    
    // Apply deduplication
    dedup := &NodeDeduplicator{}
    optimized, err := dedup.Deduplicate(originalBatch)
    if err != nil {
        t.Fatalf("Deduplication failed: %v", err)
    }
    
    // Should have fewer nodes after dedup
    if len(optimized.NewNodes) >= len(originalBatch.NewNodes) {
        t.Error("Deduplication didn't reduce node count")
    }
    
    t.Logf("Deduplication: %d -> %d nodes",
        len(originalBatch.NewNodes), len(optimized.NewNodes))
}
```

### Transaction Tests

```go
func TestTransactionIsolation(t *testing.T) {
    tree := createTestTree(t)
    
    // Insert initial data
    key := KeyHash([]byte("shared-key"))
    tree.Put(key, []byte("initial"))
    
    // Start two transactions
    tx1, _ := tree.BeginTransaction(IsolationSnapshot)
    tx2, _ := tree.BeginTransaction(IsolationSnapshot)
    
    // Both should see initial value
    val1, _ := tx1.Get(key)
    val2, _ := tx2.Get(key)
    
    if !bytes.Equal(val1, []byte("initial")) || !bytes.Equal(val2, []byte("initial")) {
        t.Error("Transactions don't see initial value")
    }
    
    // Update in tx1
    tx1.Put(key, []byte("updated-by-tx1"))
    
    // tx2 shouldn't see update yet
    val2, _ = tx2.Get(key)
    if !bytes.Equal(val2, []byte("initial")) {
        t.Error("tx2 sees uncommitted change")
    }
    
    // Commit tx1
    tx1.Commit()
    
    // tx2 still shouldn't see (snapshot isolation)
    val2, _ = tx2.Get(key)
    if !bytes.Equal(val2, []byte("initial")) {
        t.Error("tx2 sees change after commit (broken isolation)")
    }
    
    // New transaction should see update
    tx3, _ := tree.BeginTransaction(IsolationSnapshot)
    val3, _ := tx3.Get(key)
    if !bytes.Equal(val3, []byte("updated-by-tx1")) {
        t.Error("New transaction doesn't see committed change")
    }
}

func TestTransactionRollback(t *testing.T) {
    tree := createTestTree(t)
    
    // Start transaction
    tx, _ := tree.BeginTransaction(IsolationSnapshot)
    
    // Make changes
    keys := make([]Key, 5)
    for i := range keys {
        keys[i] = KeyHash([]byte(fmt.Sprintf("rollback-key-%d", i)))
        tx.Put(keys[i], []byte("should-not-persist"))
    }
    
    // Rollback
    err := tx.Rollback()
    if err != nil {
        t.Fatalf("Rollback failed: %v", err)
    }
    
    // Verify changes not persisted
    version := tree.GetLatestVersion()
    for _, key := range keys {
        _, err := tree.GetAtVersion(version, key)
        if err == nil {
            t.Error("Rolled back changes persisted")
        }
    }
}
```

### Parallel Batch Tests

```go
func TestParallelBatchProcessing(t *testing.T) {
    tree := createTestTree(t)
    
    // Create large batch of operations
    numOps := 10000
    ops := make([]Operation, numOps)
    
    for i := 0; i < numOps; i++ {
        ops[i] = Operation{
            Type:  OpTypePut,
            Key:   KeyHash([]byte(fmt.Sprintf("parallel-key-%d", i))),
            Value: []byte(fmt.Sprintf("value-%d", i)),
        }
    }
    
    // Process in parallel
    processor := &ParallelBatchProcessor{
        workers: 8,
    }
    
    start := time.Now()
    batch, err := processor.ProcessBatch(tree, ops)
    elapsed := time.Since(start)
    
    if err != nil {
        t.Fatalf("Parallel processing failed: %v", err)
    }
    
    // Verify all operations applied
    if batch.Stats.NodesCreated < numOps {
        t.Errorf("Not all nodes created: %d < %d", 
            batch.Stats.NodesCreated, numOps)
    }
    
    t.Logf("Processed %d ops in %v (%.0f ops/sec)",
        numOps, elapsed, float64(numOps)/elapsed.Seconds())
}

func TestBatchValidation(t *testing.T) {
    validator := &BatchValidator{}
    
    // Test valid batch
    validBatch := &UpdateBatch{
        NewNodes: map[NodeKey]Node{
            {Version: 2, Path: NibblePath{}}: &LeafNode{
                Key:       KeyHash([]byte("test")),
                ValueHash: Hash{1, 2, 3},
            },
        },
        OldVersion: 1,
        NewVersion: 2,
    }
    
    err := validator.ValidateBatch(validBatch)
    if err != nil {
        t.Errorf("Valid batch failed validation: %v", err)
    }
    
    // Test invalid batch (missing root)
    invalidBatch := &UpdateBatch{
        NewNodes:   map[NodeKey]Node{},
        OldVersion: 1,
        NewVersion: 2,
    }
    
    err = validator.ValidateBatch(invalidBatch)
    if err == nil {
        t.Error("Invalid batch passed validation")
    }
}

func TestBatchCompression(t *testing.T) {
    // Create large batch
    builder := NewBatchBuilder(1, 2)
    
    for i := 0; i < 1000; i++ {
        key := NodeKey{
            Version: 2,
            Path:    NibblePath{nibbles: []Nibble{byte(i % 16), byte(i / 16 % 16)}},
        }
        
        node := &LeafNode{
            Key:       KeyHash([]byte(fmt.Sprintf("key-%d", i))),
            ValueHash: Hash{byte(i), byte(i >> 8)},
        }
        
        builder.AddNode(key, node)
    }
    
    batch, _ := builder.Build()
    
    // Compress batch
    compressor := &BatchCompressor{compressionLevel: 3}
    compressed, err := compressor.CompressBatch(batch)
    if err != nil {
        t.Fatalf("Compression failed: %v", err)
    }
    
    // Should achieve significant compression
    originalSize := batch.Stats.BytesWritten
    compressedSize := int64(len(compressed))
    ratio := float64(compressedSize) / float64(originalSize)
    
    t.Logf("Compression: %d -> %d bytes (%.1f%%)",
        originalSize, compressedSize, ratio*100)
    
    if ratio > 0.5 {
        t.Error("Poor compression ratio")
    }
}
```

## Performance Considerations

- **Batch size optimization**: Balance memory usage vs throughput
- **Parallel processing**: Utilize multiple cores for large batches
- **Deduplication**: Reduce storage for common patterns
- **Compression**: Minimize disk I/O
- **Write coalescing**: Combine small operations

## Security Considerations

- **Atomic commits**: All-or-nothing batch application
- **Validation**: Ensure batch integrity before commit
- **Isolation**: Transactions don't see partial updates
- **Rollback safety**: Clean abort without side effects

## Done When ✓

- [ ] Batch counts equal nodes touched in commit test
- [ ] UpdateBatch tracks all new and stale nodes
- [ ] Transactions provide atomic multi-operation updates
- [ ] Batch optimization reduces storage overhead
- [ ] Parallel processing improves throughput for large batches
- [ ] Validation catches malformed batches
- [ ] Compression significantly reduces batch size
- [ ] 100% test coverage for batch operations
