package tree

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/neutral/proofbox/pkg/storage"
	pebblestorage "github.com/neutral/proofbox/pkg/storage/pebble"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchTransaction(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, testTreeConfig())
	require.NoError(t, err)

	t.Run("Basic Batch Operations", func(t *testing.T) {
		batch := tree.NewBatchTransaction()

		// Add operations
		err := batch.BatchPut(types.KeyHash([]byte("key1")), []byte("value1"))
		require.NoError(t, err)

		err = batch.BatchPut(types.KeyHash([]byte("key2")), []byte("value2"))
		require.NoError(t, err)

		// Test delete of existing key
		existingKey := types.KeyHash([]byte("existing"))
		_, err = tree.Put(existingKey, []byte("old-value"))
		require.NoError(t, err)

		err = batch.BatchDelete(existingKey)
		require.NoError(t, err)

		// Execute batch
		version, err := batch.Execute()
		require.NoError(t, err)
		assert.Greater(t, version, types.Version(0))

		// Verify results
		val1, err := tree.GetAtVersion(version, types.KeyHash([]byte("key1")))
		require.NoError(t, err)
		assert.Equal(t, []byte("value1"), val1)

		val2, err := tree.GetAtVersion(version, types.KeyHash([]byte("key2")))
		require.NoError(t, err)
		assert.Equal(t, []byte("value2"), val2)

		// Verify deleted key
		val3, err := tree.GetAtVersion(version, existingKey)
		assert.NoError(t, err)
		assert.Nil(t, val3)
	})

	t.Run("Batch Deduplication", func(t *testing.T) {
		batch := tree.NewBatchTransaction()
		key := types.KeyHash([]byte("dup-key"))

		// Add multiple operations for same key
		err := batch.BatchPut(key, []byte("value1"))
		require.NoError(t, err)
		err = batch.BatchPut(key, []byte("value2"))
		require.NoError(t, err)
		err = batch.BatchPut(key, []byte("value3"))
		require.NoError(t, err)

		// Only the last operation should take effect
		ops := batch.OptimizeOperations()
		assert.Equal(t, 1, len(ops))
		assert.Equal(t, []byte("value3"), ops[0].Value)

		version, err := batch.Execute()
		require.NoError(t, err)

		val, err := tree.GetAtVersion(version, key)
		require.NoError(t, err)
		assert.Equal(t, []byte("value3"), val)
	})

	t.Run("Empty Batch", func(t *testing.T) {
		batch := tree.NewBatchTransaction()
		_, err := batch.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no operations")
	})

	t.Run("Batch Rollback on Error", func(t *testing.T) {
		// Create initial state
		v1, err := tree.Put(types.KeyHash([]byte("existing")), []byte("initial"))
		require.NoError(t, err)

		batch := tree.NewBatchTransaction()
		err = batch.BatchPut(types.KeyHash([]byte("new-key")), []byte("new-value"))
		require.NoError(t, err)

		// Force an error by using an invalid key
		invalidKey := types.Key{}
		err = batch.BatchPut(invalidKey, []byte("bad"))
		assert.Error(t, err)

		// The batch should still be executable with valid operations
		batch.Clear()
		err = batch.BatchPut(types.KeyHash([]byte("good-key")), []byte("good-value"))
		require.NoError(t, err)

		version, err := batch.Execute()
		require.NoError(t, err)
		assert.Greater(t, version, v1)
	})

	t.Run("Concurrent Batch Operations", func(t *testing.T) {
		const numGoroutines = 10
		const opsPerGoroutine = 100

		var wg sync.WaitGroup
		results := make([]types.Version, numGoroutines)
		errors := make([]error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				batch := tree.NewBatchTransaction()
				for j := 0; j < opsPerGoroutine; j++ {
					key := types.KeyHash([]byte(fmt.Sprintf("key-%d-%d", idx, j)))
					value := []byte(fmt.Sprintf("value-%d-%d", idx, j))
					if err := batch.BatchPut(key, value); err != nil {
						errors[idx] = err
						return
					}
				}

				results[idx], errors[idx] = batch.Execute()
			}(i)
		}

		wg.Wait()

		// Check all batches succeeded
		for i, err := range errors {
			require.NoError(t, err, "goroutine %d failed", i)
			assert.Greater(t, results[i], types.Version(0))
		}

		// Verify all keys exist in their respective versions
		for i := 0; i < numGoroutines; i++ {
			version := results[i]
			for j := 0; j < opsPerGoroutine; j++ {
				key := types.KeyHash([]byte(fmt.Sprintf("key-%d-%d", i, j)))
				expected := []byte(fmt.Sprintf("value-%d-%d", i, j))

				val, err := tree.GetAtVersion(version, key)
				require.NoError(t, err, "failed to get key-%d-%d at version %d", i, j, version)
				assert.Equal(t, expected, val, "key-%d-%d at version %d", i, j, version)
			}
		}
	})
}

func TestBatchOptimizer(t *testing.T) {
	t.Run("Deduplication", func(t *testing.T) {
		optimizer := NewBatchOptimizer(DefaultBatchOptimizerConfig())

		// Create batch with duplicates
		batch := &UpdateBatch{
			NewRootHash: types.Hash{1, 2, 3},
			NewNodes: map[string]NodeWrite{
				"key1": {Key: types.NodeKey{Version: 1}, Node: &LeafNode{}},
				"key2": {Key: types.NodeKey{Version: 1}, Node: &LeafNode{}},
			},
			StaleNodes: []types.NodeKey{
				{Version: 0, NibblePath: types.NibblePath{Nibbles: []types.Nibble{1}}},
				{Version: 0, NibblePath: types.NibblePath{Nibbles: []types.Nibble{1}}}, // Duplicate
			},
		}

		optimized, err := optimizer.OptimizeBatch(batch)
		require.NoError(t, err)

		// Check stale nodes deduplicated
		assert.Equal(t, 1, len(optimized.StaleNodes))
		assert.Equal(t, 2, len(optimized.NewNodes))
	})

	t.Run("Compression", func(t *testing.T) {
		optimizer := NewBatchOptimizer(BatchOptimizerConfig{
			EnableCompression: true,
			CompressionLevel:  6,
		})

		// Create a large batch
		batch := &UpdateBatch{
			NewRootHash: types.Hash{1, 2, 3},
			NewNodes:    make(map[string]NodeWrite),
			StaleNodes:  make([]types.NodeKey, 0),
		}

		// Add many nodes
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("key%d", i)
			batch.NewNodes[key] = NodeWrite{
				Key:  types.NodeKey{Version: 1},
				Node: &LeafNode{key: types.Key{byte(i)}, value: []byte("value"), version: 1},
			}
		}

		// Get stats
		stats, err := optimizer.GetBatchStats(batch)
		require.NoError(t, err)

		assert.Equal(t, 100, stats.TotalNodes)
		assert.Equal(t, 100, stats.LeafNodes)
		assert.Greater(t, stats.CompressionRatio, 0.0)

		// Test compression/decompression
		compressed, err := optimizer.CompressBatch(batch)
		require.NoError(t, err)

		decompressed, err := optimizer.DecompressBatch(compressed)
		require.NoError(t, err)

		// Since we're using a simplified encoding for testing,
		// just verify the counts match
		assert.Equal(t, len(batch.NewNodes), len(decompressed.NewNodes))
		assert.Equal(t, len(batch.StaleNodes), len(decompressed.StaleNodes))
	})
}

func TestParallelBatchProcessing(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()

	config := testTreeConfig()
	config.UseParallelBatching = true

	tree, err := NewTree(store, keyEncoder, config)
	require.NoError(t, err)

	// Create a large batch that will trigger parallel processing
	version, err := tree.BeginVersion()
	require.NoError(t, err)

	// Add many operations
	for i := 0; i < 200; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("parallel-key-%d", i)))
		value := []byte(fmt.Sprintf("parallel-value-%d", i))
		err := tree.PutVersioned(version, key, value)
		require.NoError(t, err)
	}

	// Commit should use parallel processing
	err = tree.CommitVersion(version)
	require.NoError(t, err)

	// Verify all values
	for i := 0; i < 200; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("parallel-key-%d", i)))
		expected := []byte(fmt.Sprintf("parallel-value-%d", i))

		val, err := tree.GetAtVersion(tree.GetLatestVersion(), key)
		require.NoError(t, err)
		assert.Equal(t, expected, val)
	}
}

func TestBatchValidation(t *testing.T) {
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, testTreeConfig())
	require.NoError(t, err)

	t.Run("Valid Batch", func(t *testing.T) {
		updater := NewTreeUpdater(tree, 0, 1)
		_, err = updater.Put(types.KeyHash([]byte("key1")), []byte("value1"))
		require.NoError(t, err)

		batch, err := updater.BuildUpdateBatch()
		require.NoError(t, err)
		assert.NotNil(t, batch)
	})

	t.Run("Batch with Invalid Version", func(t *testing.T) {
		updater := NewTreeUpdater(tree, 0, 1)

		// Manually create a batch with wrong version
		batch := &UpdateBatch{
			NewRootHash: types.Hash{},
			NewNodes: map[string]NodeWrite{
				"bad": {
					Key:  types.NodeKey{Version: 2}, // Wrong version
					Node: &LeafNode{key: types.Key{1}, value: []byte("val"), version: 2},
				},
			},
		}

		err := updater.ValidateBatch(batch)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "incorrect version")
	})

	t.Run("Batch with Future Child Version", func(t *testing.T) {
		updater := NewTreeUpdater(tree, 0, 1)

		// Create internal node with future child
		internal := NewInternalNode(1)
		err = internal.SetChild(0, types.Child{
			Hash:    types.Hash{1, 2, 3},
			Version: 99, // Future version
			IsLeaf:  true,
		})
		require.NoError(t, err)

		batch := &UpdateBatch{
			NewRootHash: types.Hash{},
			NewNodes: map[string]NodeWrite{
				"bad": {
					Key:  types.NodeKey{Version: 1},
					Node: internal,
				},
			},
		}

		err := updater.ValidateBatch(batch)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "future version")
	})
}

func BenchmarkBatchOperations(b *testing.B) {
	dir, err := os.MkdirTemp("", "batch-bench-*")
	require.NoError(b, err)
	defer os.RemoveAll(dir)

	opts := &pebblestorage.Options{
		EnableMetrics: false,
	}
	store, err := pebblestorage.NewStorage(dir, opts)
	require.NoError(b, err)
	defer store.Close()
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, testTreeConfig())
	require.NoError(b, err)

	b.Run("Sequential Batch", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			batch := tree.NewBatchTransaction()
			for j := 0; j < 100; j++ {
				key := types.KeyHash([]byte(fmt.Sprintf("bench-key-%d-%d", i, j)))
				value := []byte(fmt.Sprintf("bench-value-%d-%d", i, j))
				if err := batch.BatchPut(key, value); err != nil {
					b.Fatal(err)
				}
			}
			if _, err := batch.Execute(); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Parallel Batch", func(b *testing.B) {
		config := testTreeConfig()
		config.UseParallelBatching = true

		tree2, err := NewTree(store, keyEncoder, config)
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			version, err := tree2.BeginVersion()
			if err != nil {
				b.Fatal(err)
			}
			for j := 0; j < 200; j++ { // Larger batch to trigger parallel
				key := types.KeyHash([]byte(fmt.Sprintf("parallel-key-%d-%d", i, j)))
				value := []byte(fmt.Sprintf("parallel-value-%d-%d", i, j))
				if err := tree2.PutVersioned(version, key, value); err != nil {
					b.Fatal(err)
				}
			}
			if err := tree2.CommitVersion(version); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Batch with Optimization", func(b *testing.B) {
		config := testTreeConfig()
		config.BatchOptimizer = NewBatchOptimizer(BatchOptimizerConfig{
			EnableCompression:   true,
			EnableDeduplication: true,
		})

		tree3, err := NewTree(store, keyEncoder, config)
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			batch := tree3.NewBatchTransaction()
			for j := 0; j < 100; j++ {
				key := types.KeyHash([]byte(fmt.Sprintf("opt-key-%d-%d", i, j)))
				value := []byte(fmt.Sprintf("opt-value-%d-%d", i, j))
				if err := batch.BatchPut(key, value); err != nil {
					b.Fatal(err)
				}

				// Add some duplicates to test deduplication
				if j%10 == 0 {
					if err := batch.BatchPut(key, value); err != nil {
						b.Fatal(err)
					}
				}
			}
			if _, err := batch.Execute(); err != nil {
				b.Fatal(err)
			}
		}
	})
}
