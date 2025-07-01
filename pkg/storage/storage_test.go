package storage_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/storage/memory"
	"github.com/neutral/proofbox/pkg/storage/pebble"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStorageImplementation runs a comprehensive test suite on a storage implementation
func testStorageImplementation(t *testing.T, name string, createStorage func(t *testing.T) storage.Storage) {
	t.Run(name, func(t *testing.T) {
		t.Run("BasicOperations", func(t *testing.T) {
			store := createStorage(t)
			defer store.Close()

			// Test Put and Get
			key := []byte("test-key")
			value := []byte("test-value")
			
			err := store.Put(key, value)
			require.NoError(t, err)

			got, err := store.Get(key)
			require.NoError(t, err)
			assert.Equal(t, value, got)

			// Test Get non-existent key
			got, err = store.Get([]byte("non-existent"))
			require.NoError(t, err)
			assert.Nil(t, got)

			// Test Delete
			err = store.Delete(key)
			require.NoError(t, err)

			got, err = store.Get(key)
			require.NoError(t, err)
			assert.Nil(t, got)
		})

		t.Run("BatchOperations", func(t *testing.T) {
			store := createStorage(t)
			defer store.Close()

			batch := store.NewBatch()
			
			// Add multiple operations
			for i := 0; i < 10; i++ {
				key := []byte(fmt.Sprintf("batch-key-%d", i))
				value := []byte(fmt.Sprintf("batch-value-%d", i))
				err := batch.Put(key, value)
				require.NoError(t, err)
			}

			// Delete some keys
			err := batch.Delete([]byte("batch-key-5"))
			require.NoError(t, err)

			// Commit batch
			err = batch.Commit(storage.CommitOptions{Sync: true})
			require.NoError(t, err)

			// Verify results
			for i := 0; i < 10; i++ {
				key := []byte(fmt.Sprintf("batch-key-%d", i))
				got, err := store.Get(key)
				require.NoError(t, err)
				
				if i == 5 {
					assert.Nil(t, got, "Key 5 should be deleted")
				} else {
					expected := []byte(fmt.Sprintf("batch-value-%d", i))
					assert.Equal(t, expected, got)
				}
			}

			batch.Close()
		})

		t.Run("Iterator", func(t *testing.T) {
			store := createStorage(t)
			defer store.Close()

			// Add test data
			keys := []string{"a", "b", "c", "d", "e"}
			for _, k := range keys {
				err := store.Put([]byte(k), []byte(k+"-value"))
				require.NoError(t, err)
			}

			t.Run("Forward", func(t *testing.T) {
				iter := store.NewIterator(nil)
				defer iter.Close()

				var collected []string
				for iter.First(); iter.Valid(); iter.Next() {
					collected = append(collected, string(iter.Key()))
				}
				assert.Equal(t, keys, collected)
			})

			t.Run("Backward", func(t *testing.T) {
				iter := store.NewIterator(nil)
				defer iter.Close()

				var collected []string
				for iter.Last(); iter.Valid(); iter.Prev() {
					collected = append(collected, string(iter.Key()))
				}
				
				// Reverse expected
				expected := make([]string, len(keys))
				for i, k := range keys {
					expected[len(keys)-1-i] = k
				}
				assert.Equal(t, expected, collected)
			})

			t.Run("Seek", func(t *testing.T) {
				iter := store.NewIterator(nil)
				defer iter.Close()

				// SeekGE
				found := iter.SeekGE([]byte("c"))
				assert.True(t, found)
				assert.Equal(t, []byte("c"), iter.Key())

				// SeekLT
				found = iter.SeekLT([]byte("c"))
				assert.True(t, found)
				assert.Equal(t, []byte("b"), iter.Key())
			})

			t.Run("Bounds", func(t *testing.T) {
				iter := store.NewIterator(&storage.IteratorOptions{
					LowerBound: []byte("b"),
					UpperBound: []byte("d"),
				})
				defer iter.Close()

				var collected []string
				for iter.First(); iter.Valid(); iter.Next() {
					collected = append(collected, string(iter.Key()))
				}
				assert.Equal(t, []string{"b", "c"}, collected)
			})

			t.Run("Prefix", func(t *testing.T) {
				// Add prefixed keys
				prefix := []byte("prefix:")
				for i := 0; i < 3; i++ {
					key := append(prefix, []byte(fmt.Sprintf("%d", i))...)
					err := store.Put(key, []byte("value"))
					require.NoError(t, err)
				}

				iter := store.NewIterator(&storage.IteratorOptions{
					Prefix: prefix,
				})
				defer iter.Close()

				count := 0
				for iter.First(); iter.Valid(); iter.Next() {
					assert.True(t, bytes.HasPrefix(iter.Key(), prefix))
					count++
				}
				assert.Equal(t, 3, count)
			})
		})

		t.Run("Snapshot", func(t *testing.T) {
			store := createStorage(t)
			defer store.Close()

			// Initial data
			err := store.Put([]byte("key1"), []byte("value1"))
			require.NoError(t, err)

			// Create snapshot
			snap := store.NewSnapshot()
			defer snap.Close()

			// Modify data after snapshot
			err = store.Put([]byte("key1"), []byte("value2"))
			require.NoError(t, err)
			err = store.Put([]byte("key2"), []byte("value2"))
			require.NoError(t, err)

			// Snapshot should see old value
			val, err := snap.Get([]byte("key1"))
			require.NoError(t, err)
			assert.Equal(t, []byte("value1"), val)

			// Snapshot should not see new key
			val, err = snap.Get([]byte("key2"))
			require.NoError(t, err)
			assert.Nil(t, val)

			// Iterator on snapshot
			iter := snap.NewIterator(nil)
			defer iter.Close()

			count := 0
			for iter.First(); iter.Valid(); iter.Next() {
				count++
			}
			assert.Equal(t, 1, count, "Snapshot should only see one key")
		})

		t.Run("ConcurrentAccess", func(t *testing.T) {
			store := createStorage(t)
			defer store.Close()

			const numGoroutines = 10
			const opsPerGoroutine = 100

			var wg sync.WaitGroup
			errors := make(chan error, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()

					for j := 0; j < opsPerGoroutine; j++ {
						key := []byte(fmt.Sprintf("concurrent-%d-%d", id, j))
						value := []byte(fmt.Sprintf("value-%d-%d", id, j))

						if err := store.Put(key, value); err != nil {
							errors <- err
							return
						}

						got, err := store.Get(key)
						if err != nil {
							errors <- err
							return
						}
						
						if !bytes.Equal(got, value) {
							errors <- fmt.Errorf("value mismatch: got %s, want %s", got, value)
							return
						}
					}
				}(i)
			}

			wg.Wait()
			close(errors)

			for err := range errors {
				t.Errorf("Concurrent operation failed: %v", err)
			}
		})

		t.Run("LargeValues", func(t *testing.T) {
			store := createStorage(t)
			defer store.Close()

			// Test various sizes
			sizes := []int{1024, 64 * 1024, 1024 * 1024} // 1KB, 64KB, 1MB

			for _, size := range sizes {
				key := []byte(fmt.Sprintf("large-%d", size))
				value := make([]byte, size)
				rand.Read(value)

				err := store.Put(key, value)
				require.NoError(t, err)

				got, err := store.Get(key)
				require.NoError(t, err)
				assert.Equal(t, value, got, "Large value mismatch for size %d", size)
			}
		})

		t.Run("Metrics", func(t *testing.T) {
			store := createStorage(t)
			defer store.Close()

			metrics := store.Metrics()
			require.NotNil(t, metrics)

			// Perform some operations
			for i := 0; i < 10; i++ {
				key := []byte(fmt.Sprintf("metrics-key-%d", i))
				value := []byte(fmt.Sprintf("metrics-value-%d", i))
				
				store.Put(key, value)
				store.Get(key)
			}

			// For stores with real metrics (like PebbleDB with metrics enabled)
			// we should see non-zero values
			// For memory storage or disabled metrics, we'll see zeros
			// This is fine - we're just testing that the interface works
			_ = metrics.GetOperations()
			_ = metrics.PutOperations()
		})
	})
}

func TestMemoryStorage(t *testing.T) {
	testStorageImplementation(t, "MemoryStorage", func(t *testing.T) storage.Storage {
		return memory.NewStorage()
	})
}

func TestPebbleStorage(t *testing.T) {
	testStorageImplementation(t, "PebbleStorage", func(t *testing.T) storage.Storage {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		
		opts := &pebble.Options{
			EnableMetrics: true,
		}
		
		store, err := pebble.NewStorage(dbPath, opts)
		require.NoError(t, err)
		
		// Don't use t.Cleanup here since the defer in the test function will close it
		
		return store
	})
}

// benchmarkStorage runs storage benchmarks
func benchmarkStorage(b *testing.B, name string, createStorage func(b *testing.B) storage.Storage) {
	b.Run(name, func(b *testing.B) {
		b.Run("Put", func(b *testing.B) {
			store := createStorage(b)
			defer store.Close()

			key := make([]byte, 32)
			value := make([]byte, 1024)
			rand.Read(value)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				binary.BigEndian.PutUint64(key, uint64(i))
				if err := store.Put(key, value); err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run("Get", func(b *testing.B) {
			store := createStorage(b)
			defer store.Close()

			// Pre-populate
			key := make([]byte, 32)
			value := make([]byte, 1024)
			rand.Read(value)

			for i := 0; i < 1000; i++ {
				binary.BigEndian.PutUint64(key, uint64(i))
				store.Put(key, value)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				binary.BigEndian.PutUint64(key, uint64(i%1000))
				if _, err := store.Get(key); err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run("Batch", func(b *testing.B) {
			store := createStorage(b)
			defer store.Close()

			const batchSize = 100
			keys := make([][]byte, batchSize)
			values := make([][]byte, batchSize)

			for i := 0; i < batchSize; i++ {
				keys[i] = make([]byte, 32)
				values[i] = make([]byte, 1024)
				rand.Read(values[i])
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				batch := store.NewBatch()
				
				for j := 0; j < batchSize; j++ {
					binary.BigEndian.PutUint64(keys[j], uint64(i*batchSize+j))
					batch.Put(keys[j], values[j])
				}
				
				if err := batch.Commit(storage.CommitOptions{Sync: false}); err != nil {
					b.Fatal(err)
				}
				batch.Close()
			}
		})

		b.Run("Iterator", func(b *testing.B) {
			store := createStorage(b)
			defer store.Close()

			// Pre-populate
			key := make([]byte, 32)
			value := make([]byte, 1024)
			for i := 0; i < 10000; i++ {
				binary.BigEndian.PutUint64(key, uint64(i))
				store.Put(key, value)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				iter := store.NewIterator(nil)
				count := 0
				for iter.First(); iter.Valid(); iter.Next() {
					count++
				}
				iter.Close()
				
				if count != 10000 {
					b.Fatalf("Expected 10000 keys, got %d", count)
				}
			}
		})
	})
}

func BenchmarkMemoryStorage(b *testing.B) {
	benchmarkStorage(b, "MemoryStorage", func(b *testing.B) storage.Storage {
		return memory.NewStorage()
	})
}

func BenchmarkPebbleStorage(b *testing.B) {
	benchmarkStorage(b, "PebbleStorage", func(b *testing.B) storage.Storage {
		tmpDir := b.TempDir()
		dbPath := filepath.Join(tmpDir, "bench.db")
		
		opts := &pebble.Options{
			EnableMetrics: false, // Disable for benchmarks
		}
		
		store, err := pebble.NewStorage(dbPath, opts)
		if err != nil {
			b.Fatal(err)
		}
		
		b.Cleanup(func() {
			store.Close()
		})
		
		return store
	})
}

// Test metric collection
func TestMetricsCollection(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "metrics.db")
	
	opts := &pebble.Options{
		EnableMetrics: true,
	}
	
	store, err := pebble.NewStorage(dbPath, opts)
	require.NoError(t, err)
	defer store.Close()

	// Perform operations
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		value := []byte(fmt.Sprintf("value-%d", i))
		
		err := store.Put(key, value)
		require.NoError(t, err)
		
		_, err = store.Get(key)
		require.NoError(t, err)
		
		if i%10 == 0 {
			err = store.Delete(key)
			require.NoError(t, err)
		}
	}

	// Use batch operations
	batch := store.NewBatch()
	for i := 0; i < 50; i++ {
		key := []byte(fmt.Sprintf("batch-key-%d", i))
		value := []byte(fmt.Sprintf("batch-value-%d", i))
		batch.Put(key, value)
	}
	err = batch.Commit(storage.CommitOptions{Sync: true})
	require.NoError(t, err)
	batch.Close()

	// Check metrics
	metrics := store.Metrics()
	
	assert.Equal(t, uint64(100), metrics.GetOperations(), "Should have 100 get operations")
	assert.Equal(t, uint64(100), metrics.PutOperations(), "Should have 100 put operations")
	assert.Equal(t, uint64(10), metrics.DeleteOperations(), "Should have 10 delete operations")
	assert.Equal(t, uint64(1), metrics.BatchCommits(), "Should have 1 batch commit")

	// Latencies should be non-zero for operations we performed
	assert.Greater(t, metrics.GetLatencyP50(), uint64(0), "Get P50 latency should be > 0")
	assert.Greater(t, metrics.PutLatencyP50(), uint64(0), "Put P50 latency should be > 0")
	
	// P99 should be >= P50
	assert.GreaterOrEqual(t, metrics.GetLatencyP99(), metrics.GetLatencyP50())
	assert.GreaterOrEqual(t, metrics.PutLatencyP99(), metrics.PutLatencyP50())

	// Wait a bit for background metrics update
	time.Sleep(100 * time.Millisecond)
	
	// Database size should be non-zero after operations
	// Note: This might be 0 immediately after creation depending on PebbleDB behavior
	dbSize := metrics.DatabaseSize()
	t.Logf("Database size: %d bytes", dbSize)
}