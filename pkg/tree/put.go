package tree

import (
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/neutral/proofbox/pkg/types"
)

// Put inserts or updates a key-value pair, creating a new version
func (t *Tree) Put(key types.Key, value []byte) (types.Version, error) {
	// Validate inputs
	if err := types.ValidateKey(key); err != nil {
		return 0, fmt.Errorf("invalid key: %w", err)
	}
	if len(value) > types.MaxValueSize {
		return 0, fmt.Errorf("value too large: %d bytes (max %d)", len(value), types.MaxValueSize)
	}

	// Serialize write operations
	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	// Get current version
	currentVersion := t.GetLatestVersion()
	newVersion := currentVersion + 1

	// Check for version overflow
	if err := types.ValidateVersion(newVersion); err != nil {
		return 0, fmt.Errorf("version overflow: %w", err)
	}

	// Create write batch
	batch := t.db.NewBatch()
	defer batch.Close()

	// Create updater for this operation
	updater := &TreeUpdater{
		tree:       t,
		batch:      batch,
		oldVersion: currentVersion,
		newVersion: newVersion,
		nodeWrites: make(map[string]nodeWrite),
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
