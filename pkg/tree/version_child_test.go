package tree

import (
	"testing"

	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionChildNodeBug tests that child nodes respect version boundaries
func TestVersionChildNodeBug(t *testing.T) {
	db := createTestDB(t)
	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	// Create keys that will share internal nodes (same prefix)
	// These keys have the same first few nibbles to ensure they share internal nodes
	key1 := types.Key{0x10, 0x00} // nibbles: 1,0,0,0...
	key2 := types.Key{0x10, 0x01} // nibbles: 1,0,0,0,0,0,0,1...
	key3 := types.Key{0x10, 0x02} // nibbles: 1,0,0,0,0,0,1,0...

	// Version 1: Only key1 exists
	v1, err := tree.BeginVersion()
	require.NoError(t, err)

	err = tree.PutVersioned(v1, key1, []byte("value1"))
	require.NoError(t, err)

	err = tree.CommitVersion(v1)
	require.NoError(t, err)

	// Version 2: Add key2 (shares internal nodes with key1)
	v2, err := tree.BeginVersion()
	require.NoError(t, err)

	err = tree.PutVersioned(v2, key2, []byte("value2"))
	require.NoError(t, err)

	err = tree.CommitVersion(v2)
	require.NoError(t, err)

	// Version 3: Add key3 (shares internal nodes with key1 and key2)
	v3, err := tree.BeginVersion()
	require.NoError(t, err)

	err = tree.PutVersioned(v3, key3, []byte("value3"))
	require.NoError(t, err)

	err = tree.CommitVersion(v3)
	require.NoError(t, err)

	// Test: key2 and key3 should NOT exist in version 1
	value, err := tree.GetAtVersion(v1, key2)
	require.NoError(t, err)
	assert.Nil(t, value, "key2 should not exist in version 1")

	value, err = tree.GetAtVersion(v1, key3)
	require.NoError(t, err)
	assert.Nil(t, value, "key3 should not exist in version 1")

	// Test: key3 should NOT exist in version 2
	value, err = tree.GetAtVersion(v2, key3)
	require.NoError(t, err)
	assert.Nil(t, value, "key3 should not exist in version 2")

	// Verify correct values in their respective versions
	value, err = tree.GetAtVersion(v1, key1)
	require.NoError(t, err)
	assert.Equal(t, []byte("value1"), value)

	value, err = tree.GetAtVersion(v2, key2)
	require.NoError(t, err)
	assert.Equal(t, []byte("value2"), value)

	value, err = tree.GetAtVersion(v3, key3)
	require.NoError(t, err)
	assert.Equal(t, []byte("value3"), value)
}

// TestComplexVersionBoundaries tests more complex scenarios with many keys
func TestComplexVersionBoundaries(t *testing.T) {
	db := createTestDB(t)
	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	// Create a series of versions with overlapping key sets
	versions := make([]types.Version, 0)

	// Version 1: keys 0-9
	v1, err := tree.BeginVersion()
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		key := types.Key{byte(i + 1), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
		err = tree.PutVersioned(v1, key, []byte{byte(i)})
		require.NoError(t, err)
	}
	err = tree.CommitVersion(v1)
	require.NoError(t, err)
	versions = append(versions, v1)

	// Version 2: add keys 10-19
	v2, err := tree.BeginVersion()
	require.NoError(t, err)
	for i := 10; i < 20; i++ {
		key := types.Key{byte(i + 1), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
		err = tree.PutVersioned(v2, key, []byte{byte(i)})
		require.NoError(t, err)
	}
	err = tree.CommitVersion(v2)
	require.NoError(t, err)
	versions = append(versions, v2)

	// Version 3: add keys 20-29
	v3, err := tree.BeginVersion()
	require.NoError(t, err)
	for i := 20; i < 30; i++ {
		key := types.Key{byte(i + 1), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
		err = tree.PutVersioned(v3, key, []byte{byte(i)})
		require.NoError(t, err)
	}
	err = tree.CommitVersion(v3)
	require.NoError(t, err)
	versions = append(versions, v3)

	// Verify version boundaries are respected
	// Keys 10-29 should NOT exist in version 1
	for i := 10; i < 30; i++ {
		key := types.Key{byte(i + 1), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
		value, err := tree.GetAtVersion(v1, key)
		require.NoError(t, err)
		assert.Nil(t, value, "key %d should not exist in version 1", i)
	}

	// Keys 20-29 should NOT exist in version 2
	for i := 20; i < 30; i++ {
		key := types.Key{byte(i + 1), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
		value, err := tree.GetAtVersion(v2, key)
		require.NoError(t, err)
		assert.Nil(t, value, "key %d should not exist in version 2", i)
	}

	// All keys 0-29 should exist in version 3
	for i := 0; i < 30; i++ {
		key := types.Key{byte(i + 1), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
		value, err := tree.GetAtVersion(v3, key)
		require.NoError(t, err)
		assert.Equal(t, []byte{byte(i)}, value, "key %d should exist in version 3", i)
	}
}
