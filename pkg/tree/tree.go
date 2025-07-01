package tree

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/neutral/proofbox/pkg/codec"
	"github.com/neutral/proofbox/pkg/metrics"
	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/types"
)

// Tree represents a Jellyfish Merkle Tree with concurrency support
type Tree struct {
	// Storage backend
	db storage.Storage

	// Key encoder for storage keys
	keyEncoder storage.KeyEncoder

	// Version management (protected by mu)
	mu         sync.RWMutex                 // Protects version metadata
	latestVer  types.Version                // Latest committed version
	rootHashes map[types.Version]types.Hash // Version -> root hash cache

	// Write coordination
	writeMu sync.Mutex // Serializes write operations

	// Node cache for performance
	nodeCache *NodeCache // Thread-safe LRU cache

	// Configuration
	config TreeConfig // Tree configuration

	// Version manager
	versionManager *VersionManager // Manages version lifecycle

	// Metrics
	metrics metrics.JMTMetrics // Metrics collector
	
	// Statistics
	stats *TreeStats // Atomic tree statistics
}

// TreeConfig holds configuration options
type TreeConfig struct {
	CacheSize           int                  // Number of nodes to cache (default: 10000)
	MaxBatchSize        int                  // Maximum operations per batch (default: 1000)
	MetricsEnabled      bool                 // Enable metrics collection
	BatchOptimizer      *BatchOptimizer      // Batch optimization settings
	UseParallelBatching bool                 // Use parallel batch processing for large batches
	Metrics             metrics.JMTMetrics   // Custom metrics implementation (optional)
}

// DefaultTreeConfig returns default configuration
func DefaultTreeConfig() TreeConfig {
	return TreeConfig{
		CacheSize:           10000,
		MaxBatchSize:        1000,
		MetricsEnabled:      true,
		BatchOptimizer:      NewBatchOptimizer(DefaultBatchOptimizerConfig()),
		UseParallelBatching: true,
	}
}

// loadNodeFromStorage loads a node directly from storage (internal use)
func (t *Tree) loadNodeFromStorage(key types.NodeKey) (types.Node, error) {
	// Check cache first
	if node, found := t.nodeCache.Get(key); found {
		return node, nil
	}

	// Load from storage
	storageKey := t.keyEncoder.NodeKey(key)
	t.metrics.RecordDBRead()
	data, err := t.db.Get(storageKey)
	if err != nil {
		t.metrics.RecordError("storage")
		return nil, fmt.Errorf("failed to load node: %w", err)
	}
	if data == nil {
		t.metrics.RecordError("storage")
		return nil, fmt.Errorf("node not found")
	}

	// Decode node
	node, err := codec.DecodeNode(data, key.Version)
	if err != nil {
		t.metrics.RecordError("codec")
		return nil, fmt.Errorf("failed to decode node: %w", err)
	}

	// Cache the node
	t.nodeCache.Put(key, node)

	return node, nil
}

// NewTree creates a new Jellyfish Merkle Tree
func NewTree(db storage.Storage, keyEncoder storage.KeyEncoder, config TreeConfig) (*Tree, error) {
	if db == nil {
		return nil, errors.New("storage cannot be nil")
	}
	if keyEncoder == nil {
		keyEncoder = storage.NewDefaultKeyEncoder()
	}

	// Initialize metrics
	var metricsCollector metrics.JMTMetrics
	if config.Metrics != nil {
		// Use provided metrics implementation
		metricsCollector = config.Metrics
	} else if config.MetricsEnabled {
		// Create Prometheus metrics only if explicitly enabled
		metricsCollector = metrics.NewPrometheusMetrics()
	} else {
		// Default to no-op metrics
		metricsCollector = metrics.NoOpMetrics{}
	}

	tree := &Tree{
		db:             db,
		keyEncoder:     keyEncoder,
		rootHashes:     make(map[types.Version]types.Hash),
		config:         config,
		versionManager: NewVersionManager(),
		metrics:        metricsCollector,
		stats:          NewTreeStats(),
	}

	// Initialize node cache
	cache, err := NewNodeCache(config.CacheSize)
	if err != nil {
		return nil, WrapError(err, "failed to create cache")
	}
	tree.nodeCache = cache

	// Load existing root hashes from storage
	if err := tree.loadRootHashes(); err != nil {
		return nil, WrapError(err, "failed to load root hashes")
	}

	// If this is a new tree (no versions loaded), initialize with version 0
	if tree.latestVer == 0 && len(tree.rootHashes) == 0 {
		tree.rootHashes[0] = types.EmptyHash()
		// Initialize version manager with version 0
		tree.versionManager.versions[0] = &VersionInfo{
			Version:       0,
			Status:        VersionStatusCommitted,
			RootHash:      types.EmptyHash(),
			ParentVersion: 0,
			CreatedAt:     time.Now(),
			NodeCount:     0,
		}
		tree.versionManager.latestVersion = 0
		tree.versionManager.committedVersion = 0
	}

	return tree, nil
}

// GetLatestVersion returns the latest committed version
func (t *Tree) GetLatestVersion() types.Version {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.latestVer
}

// HasVersion checks if a version exists
func (t *Tree) HasVersion(version types.Version) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, exists := t.rootHashes[version]
	return exists
}

// GetRootHash returns the root hash for a version
func (t *Tree) GetRootHash(version types.Version) (types.Hash, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	hash, exists := t.rootHashes[version]
	if !exists {
		t.metrics.RecordError("version")
		return types.Hash{}, types.ErrVersionNotFound
	}

	return hash, nil
}

// IsEmpty checks if the tree is empty at a version
func (t *Tree) IsEmpty(version types.Version) (bool, error) {
	hash, err := t.GetRootHash(version)
	if err != nil {
		return false, err
	}

	return hash == types.EmptyHash(), nil
}

// loadRootHashes loads all version -> root hash mappings from storage
func (t *Tree) loadRootHashes() error {
	prefix := t.keyEncoder.RootKeyPrefix()
	iter := t.db.NewIterator(&storage.IteratorOptions{
		Prefix: prefix,
	})
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		// Parse version from key
		version, err := t.keyEncoder.ParseRootKey(iter.Key())
		if err != nil {
			return err
		}

		// Parse root hash from value
		if len(iter.Value()) != 32 {
			return fmt.Errorf("invalid root hash size for version %d", version)
		}

		var hash types.Hash
		copy(hash[:], iter.Value())
		t.rootHashes[version] = hash

		// Track latest version
		if version > t.latestVer {
			t.latestVer = version
		}

		// Reconstruct version info in version manager
		if version != 0 { // Version 0 is already initialized
			parentVersion := version - 1
			if version == 1 {
				parentVersion = 0
			}
			t.versionManager.versions[version] = &VersionInfo{
				Version:       version,
				RootHash:      hash,
				ParentVersion: parentVersion,
				CreatedAt:     time.Now(), // We don't persist timestamp, use current time
				Status:        VersionStatusCommitted,
				NodeCount:     0, // We don't persist node count
			}
		}
	}

	// Update version manager's latest version
	t.versionManager.latestVersion = t.latestVer
	t.versionManager.committedVersion = t.latestVer

	return iter.Error()
}

// Reader creates a new TreeReader for the specified version
func (t *Tree) Reader(version types.Version) (TreeReaderInterface, error) {
	// Validate version exists
	t.mu.RLock()
	rootHash, exists := t.rootHashes[version]
	t.mu.RUnlock()

	if !exists {
		return nil, types.ErrVersionNotFound
	}

	// Create snapshot for consistent read
	snapshot := t.db.NewSnapshot()

	return &TreeReader{
		tree:     t,
		snapshot: snapshot,
		version:  version,
		rootHash: rootHash,
	}, nil
}

// BeginVersion starts a new version for updates
func (t *Tree) BeginVersion() (types.Version, error) {
	current := t.versionManager.GetLatestCommitted()
	return t.versionManager.Begin(current)
}

// PutVersioned inserts or updates a key-value pair in a specific version
func (t *Tree) PutVersioned(version types.Version, key types.Key, value []byte) error {
	// Validate inputs
	if err := t.validateVersion(version); err != nil {
		t.metrics.RecordError("validation")
		return fmt.Errorf("invalid version: %w", err)
	}
	if err := t.validateKey(key); err != nil {
		t.metrics.RecordError("validation")
		return fmt.Errorf("invalid key: %w", err)
	}
	if err := t.validateValue(value); err != nil {
		t.metrics.RecordError("validation")
		return fmt.Errorf("invalid value: %w", err)
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	// Get pending version
	pending := t.versionManager.GetPending(version)
	if pending == nil {
		t.metrics.RecordError("version")
		return fmt.Errorf("version %d not pending", version)
	}

	// Initialize updater if needed
	if pending.updater == nil {
		updater := NewTreeUpdater(t, pending.parent, version)
		if err := t.versionManager.SetPendingUpdater(version, updater); err != nil {
			return err
		}
		pending = t.versionManager.GetPending(version)
	}

	// Perform update
	_, err := pending.updater.Put(key, value)
	if err != nil {
		return fmt.Errorf("put failed: %w", err)
	}

	// Track operation
	t.versionManager.IncrementOperations(version)

	return nil
}

// DeleteVersioned removes a key in a specific version
func (t *Tree) DeleteVersioned(version types.Version, key types.Key) error {
	// Validate inputs
	if err := t.validateVersion(version); err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}
	if err := t.validateKey(key); err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	// Get pending version
	pending := t.versionManager.GetPending(version)
	if pending == nil {
		t.metrics.RecordError("version")
		return fmt.Errorf("version %d not pending", version)
	}

	// Initialize updater if needed
	if pending.updater == nil {
		updater := NewTreeUpdater(t, pending.parent, version)
		if err := t.versionManager.SetPendingUpdater(version, updater); err != nil {
			return err
		}
		pending = t.versionManager.GetPending(version)
	}

	// Perform delete
	err := pending.updater.Delete(key)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	// Track operation
	t.versionManager.IncrementOperations(version)

	return nil
}

// GetAtVersion retrieves a value at a specific version
func (t *Tree) GetAtVersion(version types.Version, key types.Key) ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Get version info
	info, err := t.versionManager.GetVersion(version)
	if err != nil {
		return nil, err
	}

	if info.Status != VersionStatusCommitted {
		return nil, fmt.Errorf("version %d not committed", version)
	}

	// Use existing Get method
	return t.Get(version, key)
}

// CommitVersion finalizes all changes in a version
func (t *Tree) CommitVersion(version types.Version) error {
	// Start timing for metrics
	start := time.Now()
	
	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	pending := t.versionManager.GetPending(version)
	if pending == nil {
		t.metrics.RecordError("version")
		return fmt.Errorf("version %d not pending", version)
	}

	// Handle empty version (no operations performed)
	if pending.updater == nil {
		// No changes made, just update version metadata
		rootHash := types.EmptyHash()
		if pending.parent > 0 {
			// Use parent's root hash
			parentHash, err := t.GetRootHash(pending.parent)
			if err == nil {
				rootHash = parentHash
			}
		}

		// Update in-memory state
		t.mu.Lock()
		t.rootHashes[version] = rootHash
		if version > t.latestVer {
			t.latestVer = version
		}
		t.mu.Unlock()

		// Update version manager
		err := t.versionManager.Commit(version, rootHash, 0)
		
		// Record metrics for empty commit
		duration := time.Since(start).Seconds()
		t.metrics.RecordCommit(duration, 0)
		
		return err
	}

	// Build update batch
	var batch *UpdateBatch
	var err error
	if t.config.UseParallelBatching {
		batch, err = pending.updater.BuildUpdateBatchParallel()
	} else {
		batch, err = pending.updater.BuildUpdateBatch()
	}
	if err != nil {
		return fmt.Errorf("failed to build batch: %w", err)
	}

	// Apply batch optimization if configured
	if t.config.BatchOptimizer != nil {
		batch, err = t.config.BatchOptimizer.OptimizeBatch(batch)
		if err != nil {
			return fmt.Errorf("failed to optimize batch: %w", err)
		}
	}

	// Write to storage atomically
	writeBatch := t.db.NewBatch()
	defer writeBatch.Close()

	// Create codec
	nodeCodec := &codec.NodeCodec{}

	// Write all new nodes
	dbWrites := 0
	for _, nodeWrite := range batch.NewNodes {
		data, err := nodeCodec.EncodeNode(nodeWrite.Node)
		if err != nil {
			t.metrics.RecordError("codec")
			return fmt.Errorf("failed to encode node: %w", err)
		}

		storageKey := t.keyEncoder.NodeKey(nodeWrite.Key)
		if err := writeBatch.Put(storageKey, data); err != nil {
			t.metrics.RecordError("storage")
			return fmt.Errorf("failed to write node: %w", err)
		}
		dbWrites++
		t.metrics.RecordDBWrite()

		// If it's a leaf node, also store the value
		if leaf, ok := nodeWrite.Node.(*LeafNode); ok && leaf.Value() != nil {
			valueKey := t.keyEncoder.ValueKey(leaf.ValueHash())
			if err := writeBatch.Put(valueKey, leaf.Value()); err != nil {
				t.metrics.RecordError("storage")
				return fmt.Errorf("failed to write value: %w", err)
			}
			dbWrites++
			t.metrics.RecordDBWrite()
		}
	}

	// Write root hash
	rootKey := t.keyEncoder.RootKey(version)
	if err := writeBatch.Put(rootKey, batch.NewRootHash[:]); err != nil {
		t.metrics.RecordError("storage")
		return fmt.Errorf("failed to write root: %w", err)
	}
	dbWrites++
	t.metrics.RecordDBWrite()

	// Prepare in-memory state updates (but don't apply yet)
	t.mu.RLock()
	newRootHashes := make(map[types.Version]types.Hash, len(t.rootHashes)+1)
	for k, v := range t.rootHashes {
		newRootHashes[k] = v
	}
	currentLatestVer := t.latestVer
	t.mu.RUnlock()

	newRootHashes[version] = batch.NewRootHash

	newLatestVer := currentLatestVer
	if version > newLatestVer {
		newLatestVer = version
	}

	// Commit to storage
	if err := writeBatch.Commit(storage.CommitOptions{Sync: true}); err != nil {
		t.metrics.RecordError("storage")
		return fmt.Errorf("storage commit failed: %w", err)
	}

	// Storage commit succeeded, now update in-memory state atomically
	t.mu.Lock()
	t.rootHashes = newRootHashes
	t.latestVer = newLatestVer
	t.mu.Unlock()

	// Update version manager (critical - must succeed)
	nodeCount := int64(len(batch.NewNodes))
	if err := t.versionManager.Commit(version, batch.NewRootHash, nodeCount); err != nil {
		// This is a critical error - storage committed but version manager failed
		// The tree is now in an inconsistent state
		// We should panic here as continuing would lead to data corruption
		panic(fmt.Sprintf("version manager commit failed after storage commit: %v", err))
	}

	// Record commit metrics
	duration := time.Since(start).Seconds()
	t.metrics.RecordCommit(duration, len(batch.NewNodes))
	
	// Update tree statistics atomically
	t.stats.UpdateNodeCount(nodeCount)
	t.stats.UpdateVersion(int64(version))
	
	// Update metrics from atomic stats
	t.metrics.UpdateTreeNodeCount(int(t.stats.GetNodeCount()))
	t.metrics.UpdateVersionCount(int(t.stats.GetVersion()))

	return nil
}

// AbortVersion cancels a pending version
func (t *Tree) AbortVersion(version types.Version) error {
	return t.versionManager.Abort(version)
}

// GetStats returns a snapshot of tree statistics
func (t *Tree) GetStats() (height, nodeCount, version int64) {
	return t.stats.GetHeight(), t.stats.GetNodeCount(), t.stats.GetVersion()
}

// CollectVersionGarbage removes old versions based on retention policy
// Returns the list of versions that were removed
func (t *Tree) CollectVersionGarbage() ([]types.Version, error) {
	// Get versions to remove from version manager
	removed, err := t.versionManager.CollectGarbage()
	if err != nil {
		return nil, fmt.Errorf("version GC failed: %w", err)
	}

	// Remove from in-memory root hash cache
	t.mu.Lock()
	for _, v := range removed {
		delete(t.rootHashes, v)
	}
	t.mu.Unlock()

	// Note: We don't remove the actual tree nodes from storage
	// That would require a more complex pruning operation
	// For now, we just remove the version metadata

	return removed, nil
}

// SetVersionRetentionPolicy updates how versions are retained
func (t *Tree) SetVersionRetentionPolicy(policy VersionRetentionPolicy, minVersions int, maxAge time.Duration) {
	t.versionManager.SetRetentionPolicy(policy, minVersions, maxAge)
}

// NewBatchTransaction creates a new batch transaction for atomic multi-operation updates
func (t *Tree) NewBatchTransaction() *BatchTransaction {
	return NewBatchTransaction(t)
}

// GetVersionManager returns the version manager (for internal/testing use)
func (t *Tree) GetVersionManager() *VersionManager {
	return t.versionManager
}
