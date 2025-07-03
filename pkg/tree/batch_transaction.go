package tree

import (
	"fmt"
	"sync"

	"github.com/neutral/proofbox/pkg/types"
)

// BatchOperation represents a single operation in a batch
type BatchOperation struct {
	Type  BatchOpType
	Key   types.Key
	Value []byte
}

// BatchOpType defines the type of batch operation
type BatchOpType int

const (
	BatchOpPut BatchOpType = iota
	BatchOpDelete
)

// BatchTransaction accumulates multiple operations to be executed atomically
type BatchTransaction struct {
	tree       *Tree
	operations []BatchOperation
	mu         sync.Mutex

	// Track keys for deduplication
	keyOps map[string]int // Map key to latest operation index
}

// NewBatchTransaction creates a new batch transaction
func NewBatchTransaction(tree *Tree) *BatchTransaction {
	return &BatchTransaction{
		tree:       tree,
		operations: make([]BatchOperation, 0),
		keyOps:     make(map[string]int),
	}
}

// BatchPut adds a put operation to the batch
func (bt *BatchTransaction) BatchPut(key types.Key, value []byte) error {
	if err := bt.tree.validateKey(key); err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}
	if value == nil {
		return fmt.Errorf("value cannot be nil")
	}

	bt.mu.Lock()
	defer bt.mu.Unlock()

	op := BatchOperation{
		Type:  BatchOpPut,
		Key:   key,
		Value: value,
	}

	// Track for deduplication
	keyStr := string(key[:])
	bt.keyOps[keyStr] = len(bt.operations)
	bt.operations = append(bt.operations, op)

	return nil
}

// BatchDelete adds a delete operation to the batch
func (bt *BatchTransaction) BatchDelete(key types.Key) error {
	if err := bt.tree.validateKey(key); err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	bt.mu.Lock()
	defer bt.mu.Unlock()

	op := BatchOperation{
		Type: BatchOpDelete,
		Key:  key,
	}

	// Track for deduplication
	keyStr := string(key[:])
	bt.keyOps[keyStr] = len(bt.operations)
	bt.operations = append(bt.operations, op)

	return nil
}

// Size returns the number of operations in the batch
func (bt *BatchTransaction) Size() int {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	return len(bt.operations)
}

// GetOperations returns a copy of the operations (for testing/debugging)
func (bt *BatchTransaction) GetOperations() []BatchOperation {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	ops := make([]BatchOperation, len(bt.operations))
	copy(ops, bt.operations)
	return ops
}

// OptimizeOperations returns deduplicated operations (only latest operation per key)
func (bt *BatchTransaction) OptimizeOperations() []BatchOperation {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	return bt.optimizeOperationsLocked()
}

// optimizeOperationsLocked returns deduplicated operations without locking (caller must hold lock)
func (bt *BatchTransaction) optimizeOperationsLocked() []BatchOperation {
	// If no operations, return empty
	if len(bt.operations) == 0 {
		return nil
	}

	// Build optimized list with only the latest operation per key
	optimized := make([]BatchOperation, 0, len(bt.keyOps))

	// Create a map to track which indices we need
	neededIndices := make(map[int]bool)
	for _, idx := range bt.keyOps {
		neededIndices[idx] = true
	}

	// Add operations in original order (preserving sequence for dependencies)
	for i, op := range bt.operations {
		if neededIndices[i] {
			optimized = append(optimized, op)
		}
	}

	return optimized
}

// Execute applies all operations in the batch atomically
func (bt *BatchTransaction) Execute() (types.Version, error) {
	bt.mu.Lock()
	operations := bt.optimizeOperationsLocked()
	bt.mu.Unlock()

	if len(operations) == 0 {
		return 0, fmt.Errorf("no operations in batch")
	}

	// Count operation types for metrics
	opCounts := map[string]int{
		"insert": 0,
		"update": 0,
		"delete": 0,
	}

	// Begin a new version
	version, err := bt.tree.BeginVersion()
	if err != nil {
		return 0, fmt.Errorf("failed to begin version: %w", err)
	}

	// Execute all operations
	for i, op := range operations {
		switch op.Type {
		case BatchOpPut:
			// For now, treat all puts as inserts
			// TODO: Track insert vs update during tree update operation
			opCounts["insert"]++

			if err := bt.tree.PutVersioned(version, op.Key, op.Value); err != nil {
				// Abort on error
				if abortErr := bt.tree.AbortVersion(version); abortErr != nil {
					// Log but don't mask original error
					bt.tree.metrics.RecordError("abort")
				}
				bt.tree.metrics.RecordError("batch")
				return 0, fmt.Errorf("operation %d (put) failed: %w", i, err)
			}
		case BatchOpDelete:
			opCounts["delete"]++
			if err := bt.tree.DeleteVersioned(version, op.Key); err != nil {
				// Abort on error
				if abortErr := bt.tree.AbortVersion(version); abortErr != nil {
					// Log but don't mask original error
					bt.tree.metrics.RecordError("abort")
				}
				bt.tree.metrics.RecordError("batch")
				return 0, fmt.Errorf("operation %d (delete) failed: %w", i, err)
			}
		default:
			bt.tree.AbortVersion(version)
			return 0, fmt.Errorf("unknown operation type: %v", op.Type)
		}
	}

	// Commit the version
	if err := bt.tree.CommitVersion(version); err != nil {
		// Abort on error
		bt.tree.AbortVersion(version)
		bt.tree.metrics.RecordError("batch")
		return 0, fmt.Errorf("failed to commit batch: %w", err)
	}

	// Record operation metrics after successful commit
	for opType, count := range opCounts {
		for i := 0; i < count; i++ {
			bt.tree.metrics.RecordOperation(opType)
		}
	}

	// Clear the batch after successful execution
	bt.mu.Lock()
	bt.operations = bt.operations[:0]
	bt.keyOps = make(map[string]int)
	bt.mu.Unlock()

	return version, nil
}

// Clear removes all operations from the batch
func (bt *BatchTransaction) Clear() {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	bt.operations = bt.operations[:0]
	bt.keyOps = make(map[string]int)
}
