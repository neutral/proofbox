# Transaction Boundaries Specification

## Overview
This specification defines transaction boundaries for atomic operations in the Jellyfish Merkle Tree, ensuring consistency during batch operations and handling failures gracefully.

## Transaction Model

### ACID Properties

1. **Atomicity**: All operations in a transaction succeed or all fail
2. **Consistency**: Tree maintains valid state across transactions
3. **Isolation**: Transactions don't interfere with concurrent reads
4. **Durability**: Committed transactions persist to storage

### Transaction Lifecycle

```go
// Transaction represents an atomic set of operations
type Transaction struct {
    id          uint64           // Unique transaction ID
    tree        *Tree            // Parent tree
    version     Version          // Target version
    operations  []TxOperation    // Pending operations
    writeBatch  *pebble.Batch    // Storage batch
    nodeWrites  map[NodeKey]Node // Modified nodes
    rootHash    Hash             // New root after operations
    state       TxState          // Current state
    startTime   time.Time        // Transaction start time
}

type TxState uint8

const (
    TxStateActive TxState = iota
    TxStateCommitting
    TxStateCommitted
    TxStateAborted
)

type TxOperation struct {
    Type      OpType
    Key       Key
    Value     []byte
    Timestamp time.Time
}

type OpType uint8

const (
    OpTypePut OpType = iota
    OpTypeDelete
)
```

## Transaction Operations

### Creating Transactions

```go
// BeginTransaction starts a new transaction
func (t *Tree) BeginTransaction() (*Transaction, error) {
    // Acquire transaction ID
    txID := atomic.AddUint64(&t.txCounter, 1)
    
    // Get current version under read lock
    t.mu.RLock()
    currentVersion := t.latestVer
    currentRoot := t.rootHashes[currentVersion]
    t.mu.RUnlock()
    
    tx := &Transaction{
        id:         txID,
        tree:       t,
        version:    currentVersion + 1,
        operations: make([]TxOperation, 0),
        writeBatch: t.db.NewBatch(),
        nodeWrites: make(map[NodeKey]Node),
        rootHash:   currentRoot,
        state:      TxStateActive,
        startTime:  time.Now(),
    }
    
    // Register active transaction
    t.registerTransaction(tx)
    
    return tx, nil
}

// Put adds a put operation to the transaction
func (tx *Transaction) Put(key Key, value []byte) error {
    if err := tx.checkState(); err != nil {
        return err
    }
    
    // Validate input
    if err := ValidateKey(key); err != nil {
        return WrapError(err, "invalid key in transaction")
    }
    
    if len(value) > MaxValueSize {
        return ErrValueTooLarge
    }
    
    // Add to pending operations
    tx.operations = append(tx.operations, TxOperation{
        Type:      OpTypePut,
        Key:       key,
        Value:     value,
        Timestamp: time.Now(),
    })
    
    return nil
}

// Delete adds a delete operation to the transaction
func (tx *Transaction) Delete(key Key) error {
    if err := tx.checkState(); err != nil {
        return err
    }
    
    // Validate input
    if err := ValidateKey(key); err != nil {
        return WrapError(err, "invalid key in transaction")
    }
    
    // Add to pending operations
    tx.operations = append(tx.operations, TxOperation{
        Type:      OpTypeDelete,
        Key:       key,
        Value:     nil,
        Timestamp: time.Now(),
    })
    
    return nil
}
```

### Committing Transactions

```go
// Commit atomically applies all transaction operations
func (tx *Transaction) Commit() error {
    // Check state
    if err := tx.checkState(); err != nil {
        return err
    }
    
    // Mark as committing
    if !tx.compareAndSwapState(TxStateActive, TxStateCommitting) {
        return ErrTxStateInvalid
    }
    
    // Acquire write lock
    tx.tree.writeMu.Lock()
    defer tx.tree.writeMu.Unlock()
    
    // Verify version hasn't changed
    if tx.tree.getLatestVersion() != tx.version-1 {
        tx.state = TxStateAborted
        return ErrTxConflict
    }
    
    // Apply all operations
    if err := tx.applyOperations(); err != nil {
        tx.state = TxStateAborted
        return WrapError(err, "failed to apply operations")
    }
    
    // Write all nodes to batch
    if err := tx.writeNodes(); err != nil {
        tx.state = TxStateAborted
        return WrapError(err, "failed to write nodes")
    }
    
    // Write new root
    if err := tx.writeRoot(); err != nil {
        tx.state = TxStateAborted
        return WrapError(err, "failed to write root")
    }
    
    // Commit to storage
    if err := tx.writeBatch.Commit(pebble.Sync); err != nil {
        tx.state = TxStateAborted
        return WrapError(ErrStorageFailure, "failed to commit to storage")
    }
    
    // Update tree metadata
    tx.tree.updateLatestVersion(tx.version, tx.rootHash)
    
    // Mark as committed
    tx.state = TxStateCommitted
    
    // Cleanup
    tx.tree.unregisterTransaction(tx)
    
    return nil
}

// applyOperations executes all pending operations
func (tx *Transaction) applyOperations() error {
    // Create tree updater
    updater := &TreeUpdater{
        oldVersion: tx.version - 1,
        newVersion: tx.version,
        nodes:      tx.nodeWrites,
        rootHash:   tx.rootHash,
    }
    
    // Apply each operation
    for i, op := range tx.operations {
        var err error
        
        switch op.Type {
        case OpTypePut:
            tx.rootHash, err = updater.Put(op.Key, op.Value)
        case OpTypeDelete:
            tx.rootHash, err = updater.Delete(op.Key)
        default:
            return fmt.Errorf("unknown operation type: %v", op.Type)
        }
        
        if err != nil {
            return WrapError(err, "operation %d failed", i)
        }
    }
    
    return nil
}

// writeNodes writes all modified nodes to the batch
func (tx *Transaction) writeNodes() error {
    for nodeKey, node := range tx.nodeWrites {
        // Encode node
        data, err := EncodeNode(node)
        if err != nil {
            return WrapError(err, "failed to encode node")
        }
        
        // Add to batch
        storageKey := nodeKey.StorageKey()
        if err := tx.writeBatch.Set(storageKey, data, pebble.Sync); err != nil {
            return WrapError(err, "failed to write node")
        }
    }
    
    return nil
}

// writeRoot writes the new root hash
func (tx *Transaction) writeRoot() error {
    rootKey := append([]byte(RootKeyPrefix), EncodeVersion(tx.version)...)
    rootData := tx.rootHash[:]
    
    if err := tx.writeBatch.Set(rootKey, rootData, pebble.Sync); err != nil {
        return WrapError(err, "failed to write root")
    }
    
    return nil
}
```

### Aborting Transactions

```go
// Abort cancels the transaction and discards all changes
func (tx *Transaction) Abort() error {
    // Check if already terminated
    if tx.state == TxStateCommitted || tx.state == TxStateAborted {
        return nil
    }
    
    // Mark as aborted
    tx.state = TxStateAborted
    
    // Close write batch without committing
    if tx.writeBatch != nil {
        if err := tx.writeBatch.Close(); err != nil {
            return WrapError(err, "failed to close write batch")
        }
    }
    
    // Clear pending operations
    tx.operations = nil
    tx.nodeWrites = nil
    
    // Unregister from tree
    tx.tree.unregisterTransaction(tx)
    
    return nil
}
```

## Isolation Levels

### Read Committed (Default)

```go
// Reads see only committed data
func (t *Tree) ReadCommitted(key Key) ([]byte, error) {
    // Always read from latest committed version
    t.mu.RLock()
    version := t.latestVer
    t.mu.RUnlock()
    
    return t.Get(version, key)
}
```

### Snapshot Isolation

```go
// SnapshotReader provides consistent reads at a specific version
type SnapshotReader struct {
    tree     *Tree
    version  Version
    snapshot *pebble.Snapshot
}

// NewSnapshotReader creates a reader at current version
func (t *Tree) NewSnapshotReader() (*SnapshotReader, error) {
    t.mu.RLock()
    version := t.latestVer
    t.mu.RUnlock()
    
    snapshot := t.db.NewSnapshot()
    
    return &SnapshotReader{
        tree:     t,
        version:  version,
        snapshot: snapshot,
    }, nil
}

// Get reads from the snapshot
func (sr *SnapshotReader) Get(key Key) ([]byte, error) {
    reader := &TreeReader{
        snapshot: sr.snapshot,
        version:  sr.version,
    }
    return reader.Get(key)
}

// Close releases the snapshot
func (sr *SnapshotReader) Close() error {
    return sr.snapshot.Close()
}
```

## Conflict Resolution

### Write-Write Conflicts

```go
// Transaction conflict detection
func (tx *Transaction) detectConflicts(other *Transaction) bool {
    // Check if transactions modify same keys
    txKeys := make(map[Key]bool)
    for _, op := range tx.operations {
        txKeys[op.Key] = true
    }
    
    for _, op := range other.operations {
        if txKeys[op.Key] {
            return true // Conflict detected
        }
    }
    
    return false
}

// Optimistic concurrency control
type OptimisticTransaction struct {
    *Transaction
    readSet  map[Key]Hash // Keys read and their hashes
    writeSet map[Key]bool // Keys written
}

// ValidateReadSet checks if read values are still current
func (tx *OptimisticTransaction) ValidateReadSet() error {
    current := tx.tree.getLatestVersion()
    
    for key, expectedHash := range tx.readSet {
        // Read current value
        value, err := tx.tree.Get(current, key)
        if err != nil && !errors.Is(err, ErrKeyNotFound) {
            return err
        }
        
        // Compute current hash
        var currentHash Hash
        if value != nil {
            currentHash = HashFunction(value)
        } else {
            currentHash = EmptyHash
        }
        
        // Check if changed
        if currentHash != expectedHash {
            return ErrTxConflict
        }
    }
    
    return nil
}
```

## Recovery and Rollback

### Crash Recovery

```go
// RecoverTransactions handles incomplete transactions after crash
func (t *Tree) RecoverTransactions() error {
    // Scan for incomplete transaction markers
    iter := t.db.NewIter(&pebble.IterOptions{
        LowerBound: []byte("tx_"),
        UpperBound: []byte("tx~"),
    })
    defer iter.Close()
    
    for iter.First(); iter.Valid(); iter.Next() {
        txID := parseTxID(iter.Key())
        txData := iter.Value()
        
        // Reconstruct transaction
        tx, err := decodeTxData(txData)
        if err != nil {
            continue // Skip corrupted data
        }
        
        // Check if committed
        if tx.state == TxStateCommitting {
            // Transaction was committing - check if succeeded
            if t.hasVersion(tx.version) {
                // Committed successfully, clean up marker
                t.db.Delete(iter.Key(), pebble.Sync)
            } else {
                // Failed to commit, rollback
                tx.Abort()
            }
        } else if tx.state == TxStateActive {
            // Active transaction, abort it
            tx.Abort()
        }
    }
    
    return iter.Error()
}
```

### Savepoints

```go
// Savepoint allows partial rollback within a transaction
type Savepoint struct {
    id         uint64
    tx         *Transaction
    opCount    int
    nodeCount  int
}

// CreateSavepoint creates a point to rollback to
func (tx *Transaction) CreateSavepoint(name string) *Savepoint {
    return &Savepoint{
        id:        atomic.AddUint64(&tx.tree.savepointCounter, 1),
        tx:        tx,
        opCount:   len(tx.operations),
        nodeCount: len(tx.nodeWrites),
    }
}

// RollbackToSavepoint undoes operations after savepoint
func (tx *Transaction) RollbackToSavepoint(sp *Savepoint) error {
    if sp.tx != tx {
        return errors.New("savepoint from different transaction")
    }
    
    // Truncate operations
    tx.operations = tx.operations[:sp.opCount]
    
    // Remove nodes added after savepoint
    for key := range tx.nodeWrites {
        if len(tx.nodeWrites) > sp.nodeCount {
            delete(tx.nodeWrites, key)
        }
    }
    
    return nil
}
```

## Performance Optimizations

### Batch Commit

```go
// BatchCommit commits multiple transactions together
func (t *Tree) BatchCommit(txs []*Transaction) error {
    // Validate all transactions first
    for _, tx := range txs {
        if err := tx.validate(); err != nil {
            return WrapError(err, "transaction validation failed")
        }
    }
    
    // Single write lock for all
    t.writeMu.Lock()
    defer t.writeMu.Unlock()
    
    // Create mega-batch
    megaBatch := t.db.NewBatch()
    defer megaBatch.Close()
    
    // Apply all transactions
    for _, tx := range txs {
        if err := tx.applyToBatch(megaBatch); err != nil {
            return err
        }
    }
    
    // Single commit
    return megaBatch.Commit(pebble.Sync)
}
```

### Write-Ahead Logging

```go
// WAL provides durability before commit
type WAL struct {
    file *os.File
    mu   sync.Mutex
}

// LogTransaction writes tx to WAL before execution
func (w *WAL) LogTransaction(tx *Transaction) error {
    w.mu.Lock()
    defer w.mu.Unlock()
    
    // Encode transaction
    data, err := encodeTx(tx)
    if err != nil {
        return err
    }
    
    // Write to WAL with sync
    if _, err := w.file.Write(data); err != nil {
        return err
    }
    
    return w.file.Sync()
}
```

## Best Practices

1. **Keep Transactions Short**: Minimize lock hold time
2. **Batch Related Operations**: Group updates to same keys
3. **Use Appropriate Isolation**: Snapshot for long reads
4. **Handle Conflicts**: Implement retry logic for conflicts
5. **Monitor Transaction Times**: Set timeouts for long transactions
6. **Clean Up Resources**: Always close transactions (commit or abort)