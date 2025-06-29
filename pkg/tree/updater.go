package tree

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/neutral/proofbox/pkg/codec"
	"github.com/neutral/proofbox/pkg/types"
)

// TreeUpdater handles the logic for updating the tree
type TreeUpdater struct {
	tree       *Tree
	batch      *pebble.Batch
	oldVersion types.Version
	newVersion types.Version
	nodeWrites map[string]nodeWrite // Nodes to write in this update
}

// nodeWrite contains a node and its key for writing
type nodeWrite struct {
	key  types.NodeKey
	node types.Node
}

// Put inserts or updates a key-value pair
func (u *TreeUpdater) Put(key types.Key, value []byte) (types.Hash, error) {
	// Compute value hash using SHA-256 for content-addressed storage
	valueHash := sha256.Sum256(value)

	// Store value using content-addressed key (no version for deduplication)
	valueKey := makeValueKey(types.Hash(valueHash))
	if err := u.batch.Set(valueKey, value, nil); err != nil {
		return types.Hash{}, fmt.Errorf("failed to store value: %w", err)
	}

	// Get current root
	var currentRoot types.Node
	if u.oldVersion > 0 {
		u.tree.mu.RLock()
		rootHash, exists := u.tree.rootHashes[u.oldVersion]
		u.tree.mu.RUnlock()

		if exists && rootHash != types.EmptyHash() {
			// Load existing root
			root, err := u.loadNode(types.RootNodeKey(u.oldVersion))
			if err != nil {
				return types.Hash{}, fmt.Errorf("failed to load root: %w", err)
			}
			currentRoot = root
		}
	}

	// Handle empty tree case
	if currentRoot == nil {
		return u.insertIntoEmptyTree(key, types.Hash(valueHash))
	}

	// Handle non-empty tree (will be implemented in later steps)
	return u.insertIntoTree(currentRoot, key, types.Hash(valueHash))
}

// insertIntoEmptyTree handles insertion into an empty tree
func (u *TreeUpdater) insertIntoEmptyTree(key types.Key, valueHash types.Hash) (types.Hash, error) {
	// Create new leaf node directly with the LeafNode type
	leaf := &LeafNode{
		key:       key,
		valueHash: valueHash,
		value:     nil,
		version:   u.newVersion,
	}

	// Store the leaf
	leafKey := types.RootNodeKey(u.newVersion)
	u.nodeWrites[leafKey.String()] = nodeWrite{key: leafKey, node: leaf}

	// Return leaf's hash as new root hash
	return leaf.Hash(), nil
}

// insertIntoTree handles insertion into existing tree
func (u *TreeUpdater) insertIntoTree(root types.Node, key types.Key, valueHash types.Hash) (types.Hash, error) {
	// For now, only handle the case where root is a leaf
	if types.IsLeaf(root) {
		leafNode, ok := AsLeaf(root)
		if !ok {
			return types.Hash{}, errors.New("failed to cast root to leaf node")
		}

		if leafNode.Key() == key {
			// Update existing key
			return u.updateLeaf(leafNode, valueHash)
		} else {
			// Split into internal node (will be implemented later)
			return types.Hash{}, errors.New("leaf splitting not implemented in step 09")
		}
	}

	// Internal node case will be implemented in later steps
	return types.Hash{}, errors.New("internal node updates not implemented in step 09")
}

// updateLeaf creates a new version of a leaf with updated value
func (u *TreeUpdater) updateLeaf(oldLeaf types.LeafNodeInterface, newValueHash types.Hash) (types.Hash, error) {
	// Create new leaf with same key but new value
	newLeaf := &LeafNode{
		key:       oldLeaf.Key(),
		valueHash: newValueHash,
		value:     nil,
		version:   u.newVersion,
	}

	// Store the new leaf
	leafKey := types.RootNodeKey(u.newVersion)
	u.nodeWrites[leafKey.String()] = nodeWrite{key: leafKey, node: newLeaf}

	return newLeaf.Hash(), nil
}

// loadNode loads a node from the previous version
func (u *TreeUpdater) loadNode(key types.NodeKey) (types.Node, error) {
	// First check if we're loading from the current update
	if nw, exists := u.nodeWrites[key.String()]; exists {
		return nw.node, nil
	}

	// Load from storage
	storageKey := key.StorageKey()
	data, closer, err := u.tree.db.Get(storageKey)
	if err == pebble.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	// Decode node
	return codec.DecodeNode(data, key.Version)
}

// writeToBatch writes all nodes to the batch
func (u *TreeUpdater) writeToBatch() error {
	nodeCodec := &codec.NodeCodec{}

	for _, nw := range u.nodeWrites {
		// Encode node
		data, err := nodeCodec.EncodeNode(nw.node)
		if err != nil {
			return fmt.Errorf("failed to encode node: %w", err)
		}

		// Write to batch
		storageKey := nw.key.StorageKey()
		if err := u.batch.Set(storageKey, data, nil); err != nil {
			return fmt.Errorf("failed to write node: %w", err)
		}

		// Update cache
		u.tree.nodeCache.Put(nw.key, nw.node)
	}

	return nil
}
