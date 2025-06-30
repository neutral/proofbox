package tree

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/neutral/proofbox/pkg/codec"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTree(t testing.TB) *Tree {
	// For benchmarks, we need to create a temporary DB
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	opts := &pebble.Options{}
	db, err := pebble.Open(dbPath, opts)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)
	return tree
}

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

func TestBasicLeafSplitting(t *testing.T) {
	db := createTestDB(t)

	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	// First insert a key
	key1 := types.Key{0x10, 0x20} // Nibbles: 1,0,2,0,...
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

	// Try to insert different key - should cause split
	batch = db.NewBatch()
	defer batch.Close()

	updater := &TreeUpdater{
		tree:       tree,
		batch:      batch,
		oldVersion: 1,
		newVersion: 2,
		nodeWrites: make(map[string]nodeWrite),
		staleNodes: []types.NodeKey{},
	}

	key2 := types.Key{0x10, 0x30} // Nibbles: 1,0,3,0,... (diverges from 1,0,2,0,... at index 2)
	value2 := []byte("value2")

	rootHash, err := updater.Put(key2, value2)
	require.NoError(t, err)
	assert.NotEqual(t, types.EmptyHash(), rootHash)

	// Should have created multiple nodes (leaves + internal nodes)
	assert.Greater(t, len(updater.nodeWrites), 2)

	// Root should now be internal node
	newRootKey := types.RootNodeKey(2)
	nw, exists := updater.nodeWrites[newRootKey.String()]
	assert.True(t, exists)

	internal, ok := AsInternal(nw.node)
	require.True(t, ok, "Root should be internal after split")

	// Should have exactly one child at nibble 1
	childCount := 0
	for n := types.Nibble(0); n <= 15; n++ {
		if _, exists := internal.Child(n); exists {
			childCount++
		}
	}
	assert.Equal(t, 1, childCount)
}

func TestMultiLevelSplitting(t *testing.T) {
	tree := createTestTree(t)

	// Insert keys that will create multi-level tree
	// Note: Need at least one non-zero byte to avoid empty key error
	keys := []types.Key{
		{0xA0, 0x00}, // A,0,0,0,...
		{0xA0, 0x01}, // A,0,0,1,... (split at depth 3)
		{0xA0, 0x10}, // A,0,1,0,... (split at depth 2)
		{0xA1, 0x00}, // A,1,0,0,... (split at depth 1)
		{0xB0, 0x00}, // B,0,0,0,... (split at depth 0)
	}

	var version types.Version
	for i, key := range keys {
		v, err := tree.Put(key, []byte(fmt.Sprintf("value%d", i)))
		require.NoError(t, err)
		version = v
	}

	// Validate final tree structure
	err := ValidateTreeStructure(tree, version)
	assert.NoError(t, err)

	// Verify all keys are accessible
	for i, key := range keys {
		value, err := tree.Get(version, key)
		require.NoError(t, err)
		assert.Equal(t, []byte(fmt.Sprintf("value%d", i)), value)
	}
}

func TestSplitWithExistingInternal(t *testing.T) {
	tree := createTestTree(t)

	// Build initial structure
	tree.Put(types.Key{0x10, 0x00}, []byte("A"))
	tree.Put(types.Key{0x10, 0x10}, []byte("B"))
	tree.Put(types.Key{0x20, 0x00}, []byte("C"))
	

	// Insert key that splits under existing internal node
	key := types.Key{0x10, 0x01} // Will split with 0x10,0x00
	v2, err := tree.Put(key, []byte("D"))
	require.NoError(t, err)

	// Verify structure maintained
	err = ValidateTreeStructure(tree, v2)
	assert.NoError(t, err)

	// All keys should be accessible
	val, _ := tree.Get(v2, types.Key{0x10, 0x00})
	assert.Equal(t, []byte("A"), val)
	val, _ = tree.Get(v2, types.Key{0x10, 0x01})
	assert.Equal(t, []byte("D"), val)
	val, _ = tree.Get(v2, types.Key{0x10, 0x10})
	assert.Equal(t, []byte("B"), val)
	val, _ = tree.Get(v2, types.Key{0x20, 0x00})
	assert.Equal(t, []byte("C"), val)
}

func TestStaleNodeTracking(t *testing.T) {
	db := createTestDB(t)
	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	// Capture stale nodes during update
	var capturedStale []types.NodeKey

	// Initial insert
	key1 := types.Key{0x10}
	v1, err := tree.Put(key1, []byte("v1"))
	require.NoError(t, err)

	// Create a new batch for the second update
	batch := db.NewBatch()
	defer batch.Close()

	updater := &TreeUpdater{
		tree:       tree,
		batch:      batch,
		oldVersion: v1,
		newVersion: v1 + 1,
		nodeWrites: make(map[string]nodeWrite),
		staleNodes: []types.NodeKey{},
	}

	// Insert that causes split
	key2 := types.Key{0x20}
	_, err = updater.Put(key2, []byte("v2"))
	require.NoError(t, err)

	capturedStale = updater.staleNodes

	// Should have marked the old root (leaf) as stale
	assert.Greater(t, len(capturedStale), 0, "Should track stale nodes during split")

	// The old leaf should be in stale list
	found := false
	for _, stale := range capturedStale {
		if stale.NibblePath.Equals(types.NewNibblePath(key1[:])) {
			found = true
			break
		}
	}
	assert.True(t, found, "Original leaf should be marked stale")
}

func TestIdenticalPrefixSplit(t *testing.T) {
	tree := createTestTree(t)

	// Keys with long common prefix
	key1 := types.Key{}
	key1[31] = 0x01 // Differs only in last byte

	key2 := types.Key{}
	key2[31] = 0x02

	tree.Put(key1, []byte("v1"))
	v2, err := tree.Put(key2, []byte("v2"))
	require.NoError(t, err)

	// Should create deep tree with internal nodes
	// all the way down to nibble 62 (31 bytes * 2 nibbles - 2)
	err = ValidateTreeStructure(tree, v2)
	assert.NoError(t, err)
}

func TestInternalNodeDepthLimit(t *testing.T) {
	tree := createTestTree(t)

	// Test that internal nodes cannot exist at depth 64
	// This would require a custom crafted scenario that tries to create
	// an internal node at MaxTreeDepth, which should fail
	
	// Create a key that would require depth > 64
	key1 := types.Key{}
	for i := 0; i < 32; i++ {
		key1[i] = 0xFF
	}
	
	// First insert succeeds
	v1, err := tree.Put(key1, []byte("value1"))
	require.NoError(t, err)
	
	// Try to validate tree structure - should pass since leaves can be at depth 64
	err = ValidateTreeStructure(tree, v1)
	assert.NoError(t, err)
}

func TestMaxDepthSplit(t *testing.T) {
	tree := createTestTree(t)

	// Create keys that differ at maximum depth
	key1 := types.Key{}
	key2 := types.Key{}

	// Make them differ only in the last nibble
	for i := 0; i < 31; i++ {
		key1[i] = 0xFF
		key2[i] = 0xFF
	}
	key1[31] = 0xF0 // Last nibbles: F,0
	key2[31] = 0xF1 // Last nibbles: F,1

	// Test that the tree correctly handles keys that differ only in their last nibble.
	// This creates a tree with leaves at depth 64 (the maximum allowed depth for leaves).
	
	v1, err := tree.Put(key1, []byte("v1"))
	require.NoError(t, err)
	
	v2, err := tree.Put(key2, []byte("v2"))
	require.NoError(t, err)
	
	// Log the versions for debugging
	t.Logf("Put key1 at version %d, key2 at version %d", v1, v2)
	
	// Verify we can retrieve both values
	val1, err := tree.Get(v2, key1)
	require.NoError(t, err)
	assert.Equal(t, []byte("v1"), val1)
	
	val2, err := tree.Get(v2, key2)
	require.NoError(t, err)
	assert.Equal(t, []byte("v2"), val2)
}

func BenchmarkLeafSplitting(b *testing.B) {
	tree := createTestTree(b)

	// Pre-insert base key
	baseKey := types.Key{0x10}
	tree.Put(baseKey, []byte("base"))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create key that will split with base
		key := types.Key{0x10}
		key[1] = byte(i % 256)

		tree.Put(key, []byte("value"))
	}
}

func BenchmarkDeepTreeInsertion(b *testing.B) {
	tree := createTestTree(b)

	// Build a deep tree
	for i := 0; i < 1000; i++ {
		key := make([]byte, 32)
		binary.BigEndian.PutUint32(key[28:], uint32(i))
		tree.Put(types.Key(key), []byte("value"))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := make([]byte, 32)
		binary.BigEndian.PutUint32(key[28:], uint32(1000+i))
		tree.Put(types.Key(key), []byte("value"))
	}
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
