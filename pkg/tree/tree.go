package tree

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/neutral/proofbox/pkg/types"
)

// Tree represents a Jellyfish Merkle Tree with concurrency support
type Tree struct {
	// Storage backend
	db *pebble.DB

	// Version management (protected by mu)
	mu         sync.RWMutex                    // Protects version metadata
	latestVer  types.Version                   // Latest committed version
	rootHashes map[types.Version]types.Hash    // Version -> root hash cache

	// Write coordination
	writeMu sync.Mutex // Serializes write operations

	// Node cache for performance
	nodeCache *NodeCache // Thread-safe LRU cache

	// Configuration
	config TreeConfig // Tree configuration
}

// TreeConfig holds configuration options
type TreeConfig struct {
	CacheSize      int  // Number of nodes to cache (default: 10000)
	MaxBatchSize   int  // Maximum operations per batch (default: 1000)
	MetricsEnabled bool // Enable metrics collection
}

// DefaultTreeConfig returns default configuration
func DefaultTreeConfig() TreeConfig {
	return TreeConfig{
		CacheSize:      10000,
		MaxBatchSize:   1000,
		MetricsEnabled: true,
	}
}

// NewTree creates a new Jellyfish Merkle Tree
func NewTree(db *pebble.DB, config TreeConfig) (*Tree, error) {
	if db == nil {
		return nil, errors.New("database cannot be nil")
	}

	tree := &Tree{
		db:         db,
		rootHashes: make(map[types.Version]types.Hash),
		config:     config,
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
	prefix := []byte(types.RootKeyPrefix)
	iter, err := t.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(append([]byte{}, prefix...), 0xFF),
	})
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
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

		var hash types.Hash
		copy(hash[:], iter.Value())
		t.rootHashes[version] = hash

		// Track latest version
		if version > t.latestVer {
			t.latestVer = version
		}
	}

	return iter.Error()
}