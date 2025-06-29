package tree

import (
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValueDeduplication(t *testing.T) {
	// This test will be more meaningful after step 10 (leaf splitting)
	// For now, we can only test deduplication with the same key
	t.Skip("Requires leaf splitting (step 10) to test different keys with same value")
}

func TestCrossVersionDeduplication(t *testing.T) {
	db := createTestDB(t)

	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	key := types.KeyHash([]byte("test-key"))
	value := []byte("constant-value")

	// Insert same value multiple times (simulating no change)
	for i := 1; i <= 5; i++ {
		version, err := tree.Put(key, value)
		require.NoError(t, err)
		assert.Equal(t, types.Version(i), version)
	}

	// All versions should retrieve the same value
	for i := 1; i <= 5; i++ {
		retrieved, err := tree.Get(types.Version(i), key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)
	}

	// Verify storage efficiency - count value keys
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{'v'},
		UpperBound: []byte{'w'},
	})
	require.NoError(t, err)
	defer iter.Close()

	valueCount := 0
	for iter.First(); iter.Valid(); iter.Next() {
		valueCount++
	}
	require.NoError(t, iter.Error())

	// Should only have one value stored despite 5 versions
	assert.Equal(t, 1, valueCount, "Expected only one value in storage")
}

func TestDifferentValuesNotDeduplicated(t *testing.T) {
	db := createTestDB(t)

	tree, err := NewTree(db, DefaultTreeConfig())
	require.NoError(t, err)

	key := types.KeyHash([]byte("test-key"))

	// Insert different values
	values := []string{"value1", "value2", "value3"}
	for i, val := range values {
		version, err := tree.Put(key, []byte(val))
		require.NoError(t, err)
		assert.Equal(t, types.Version(i+1), version)
	}

	// Count stored values
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{'v'},
		UpperBound: []byte{'w'},
	})
	require.NoError(t, err)
	defer iter.Close()

	valueCount := 0
	for iter.First(); iter.Valid(); iter.Next() {
		valueCount++
	}
	require.NoError(t, iter.Error())

	// Should have 3 different values stored
	assert.Equal(t, 3, valueCount, "Expected three different values in storage")
}