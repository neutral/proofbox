package tree

import (
	"crypto/sha256"
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/neutral/proofbox/pkg/codec"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeUpdater_InsertIntoEmptyTree(t *testing.T) {
	db := createTestDB(t)

	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	batch := db.NewBatch()
	defer batch.Close()

	updater := &TreeUpdater{
		tree:       tree,
		batch:      batch,
		oldVersion: 0,
		newVersion: 1,
		nodeWrites: make(map[string]nodeWrite),
	}

	// Test key and value
	key := types.KeyHash([]byte("test-key"))
	value := []byte("test-value")
	valueHash := sha256.Sum256(value)

	// Insert into empty tree
	rootHash, err := updater.insertIntoEmptyTree(key, types.Hash(valueHash))
	require.NoError(t, err)
	assert.NotEqual(t, types.EmptyHash(), rootHash)

	// Verify node was created
	assert.Len(t, updater.nodeWrites, 1)

	// Get the created node
	rootKey := types.RootNodeKey(1)
	nw, exists := updater.nodeWrites[rootKey.String()]
	assert.True(t, exists)

	// Verify it's a leaf with correct properties
	leaf, ok := AsLeaf(nw.node)
	require.True(t, ok)
	assert.Equal(t, key, leaf.Key())
	assert.Equal(t, types.Hash(valueHash), leaf.valueHash)
	assert.Equal(t, types.Version(1), leaf.Version())
}

func TestTreeUpdater_UpdateLeaf(t *testing.T) {
	db := createTestDB(t)

	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	// Create an existing leaf
	key := types.KeyHash([]byte("update-key"))
	oldValue := []byte("old-value")
	oldLeaf, err := NewLeafNode(key, oldValue, 1)
	require.NoError(t, err)

	batch := db.NewBatch()
	defer batch.Close()

	updater := &TreeUpdater{
		tree:       tree,
		batch:      batch,
		oldVersion: 1,
		newVersion: 2,
		nodeWrites: make(map[string]nodeWrite),
	}

	// Update with new value
	newValue := []byte("new-value")
	newValueHash := sha256.Sum256(newValue)

	rootHash, err := updater.updateLeaf(oldLeaf, types.Hash(newValueHash))
	require.NoError(t, err)
	assert.NotEqual(t, types.EmptyHash(), rootHash)

	// Verify new node was created
	assert.Len(t, updater.nodeWrites, 1)

	// Get the new node
	rootKey := types.RootNodeKey(2)
	nw, exists := updater.nodeWrites[rootKey.String()]
	assert.True(t, exists)

	// Verify it's a leaf with updated value
	newLeaf, ok := AsLeaf(nw.node)
	require.True(t, ok)
	assert.Equal(t, key, newLeaf.Key())
	assert.Equal(t, types.Hash(newValueHash), newLeaf.valueHash)
	assert.Equal(t, types.Version(2), newLeaf.Version())

	// Verify hashes are different
	assert.NotEqual(t, oldLeaf.Hash(), newLeaf.Hash())
}

func TestTreeUpdater_Put_EmptyTree(t *testing.T) {
	db := createTestDB(t)

	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	batch := db.NewBatch()
	defer batch.Close()

	updater := &TreeUpdater{
		tree:       tree,
		batch:      batch,
		oldVersion: 0,
		newVersion: 1,
		nodeWrites: make(map[string]nodeWrite),
	}

	// Put key-value pair
	key := types.KeyHash([]byte("put-key"))
	value := []byte("put-value")

	rootHash, err := updater.Put(key, value)
	require.NoError(t, err)
	assert.NotEqual(t, types.EmptyHash(), rootHash)

	// Verify value was stored
	// Note: Can't actually verify batch contents directly,
	// but we can verify the node was created
	assert.Len(t, updater.nodeWrites, 1)
}

func TestTreeUpdater_Put_UpdateExisting(t *testing.T) {
	db := createTestDB(t)

	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	// First insert a key
	key := types.KeyHash([]byte("existing-key"))
	oldValue := []byte("old-value")

	// Manually create version 1 with the key
	batch := db.NewBatch()

	// Create and store the leaf
	leaf, err := NewLeafNode(key, oldValue, 1)
	require.NoError(t, err)
	leaf.valueHash = types.Hash(sha256.Sum256(oldValue))

	// Encode and store the node
	nodeCodec := &codec.NodeCodec{}
	nodeData, err := nodeCodec.EncodeNode(leaf)
	require.NoError(t, err)

	rootKey := types.RootNodeKey(1)
	err = batch.Set(rootKey.StorageKey(), nodeData, nil)
	require.NoError(t, err)

	// Store root hash
	rootHashKey := makeRootKey(1)
	leafHash := leaf.Hash()
	err = batch.Set(rootHashKey, leafHash[:], nil)
	require.NoError(t, err)

	err = batch.Commit(pebble.Sync)
	require.NoError(t, err)

	// Reload tree to pick up the changes
	tree, err = NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	// Now update with new value
	batch = db.NewBatch()
	defer batch.Close()

	updater := &TreeUpdater{
		tree:       tree,
		batch:      batch,
		oldVersion: 1,
		newVersion: 2,
		nodeWrites: make(map[string]nodeWrite),
	}

	newValue := []byte("new-value")
	rootHash, err := updater.Put(key, newValue)
	require.NoError(t, err)
	assert.NotEqual(t, types.EmptyHash(), rootHash)

	// Verify a new node was created
	assert.Len(t, updater.nodeWrites, 1)

	// Get the new node
	newRootKey := types.RootNodeKey(2)
	nw, exists := updater.nodeWrites[newRootKey.String()]
	assert.True(t, exists)

	// Verify it has the new value hash
	newLeaf, ok := AsLeaf(nw.node)
	require.True(t, ok)
	assert.Equal(t, key, newLeaf.Key())

	newValueHash := sha256.Sum256(newValue)
	assert.Equal(t, types.Hash(newValueHash), newLeaf.valueHash)
}

func TestTreeUpdater_Put_DifferentKeyError(t *testing.T) {
	db := createTestDB(t)

	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	// First insert a key
	key1 := types.KeyHash([]byte("key1"))
	value1 := []byte("value1")

	// Manually create version 1 with key1
	batch := db.NewBatch()

	leaf, err := NewLeafNode(key1, value1, 1)
	require.NoError(t, err)
	leaf.valueHash = types.Hash(sha256.Sum256(value1))

	nodeCodec := &codec.NodeCodec{}
	nodeData, err := nodeCodec.EncodeNode(leaf)
	require.NoError(t, err)

	rootKey := types.RootNodeKey(1)
	err = batch.Set(rootKey.StorageKey(), nodeData, nil)
	require.NoError(t, err)

	rootHashKey := makeRootKey(1)
	leafHash := leaf.Hash()
	err = batch.Set(rootHashKey, leafHash[:], nil)
	require.NoError(t, err)

	err = batch.Commit(pebble.Sync)
	require.NoError(t, err)

	// Reload tree
	tree, err = NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	// Try to insert different key
	batch = db.NewBatch()
	defer batch.Close()

	updater := &TreeUpdater{
		tree:       tree,
		batch:      batch,
		oldVersion: 1,
		newVersion: 2,
		nodeWrites: make(map[string]nodeWrite),
	}

	key2 := types.KeyHash([]byte("key2"))
	value2 := []byte("value2")

	_, err = updater.Put(key2, value2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "leaf splitting not implemented")
}

func TestTreeUpdater_WriteToBatch(t *testing.T) {
	db := createTestDB(t)

	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	batch := db.NewBatch()
	defer batch.Close()

	updater := &TreeUpdater{
		tree:       tree,
		batch:      batch,
		oldVersion: 0,
		newVersion: 1,
		nodeWrites: make(map[string]nodeWrite),
	}

	// Create some nodes
	key1 := types.KeyHash([]byte("key1"))
	leaf1, err := NewLeafNode(key1, []byte("value1"), 1)
	require.NoError(t, err)

	rootKey := types.RootNodeKey(1)
	updater.nodeWrites[rootKey.String()] = nodeWrite{
		key:  rootKey,
		node: leaf1,
	}

	// Write to batch
	err = updater.writeToBatch()
	require.NoError(t, err)

	// Verify cache was updated
	cached, found := tree.nodeCache.Get(rootKey)
	assert.True(t, found)
	assert.Equal(t, leaf1, cached)
}
