package tree

import (
	"fmt"
	"path/filepath"
	"testing"

	pebblestorage "github.com/neutral/proofbox/pkg/storage/pebble"
	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/require"
)

// BenchmarkVersionCreation measures the cost of creating new versions
func BenchmarkVersionCreation(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	opts := &pebblestorage.Options{
		EnableMetrics: false,
	}
	store, err := pebblestorage.NewStorage(dbPath, opts)
	require.NoError(b, err)
	defer store.Close()
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(b, err)

	// Pre-populate with some data
	for i := 0; i < 1000; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i)))
		value := []byte(fmt.Sprintf("value-%d", i))
		_, err := tree.Put(key, value)
		require.NoError(b, err)
	}

	b.ResetTimer()

	// Benchmark version creation
	for i := 0; i < b.N; i++ {
		v, err := tree.BeginVersion()
		if err != nil {
			b.Fatal(err)
		}

		// Minimal operation
		key := types.KeyHash([]byte(fmt.Sprintf("bench-key-%d", i)))
		err = tree.PutVersioned(v, key, []byte("bench-value"))
		if err != nil {
			b.Fatal(err)
		}

		err = tree.CommitVersion(v)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStructuralSharingOverhead measures overhead of path cloning
func BenchmarkStructuralSharingOverhead(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	opts := &pebblestorage.Options{
		EnableMetrics: false,
	}
	store, err := pebblestorage.NewStorage(dbPath, opts)
	require.NoError(b, err)
	defer store.Close()
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(b, err)

	// Create a tree with many nodes
	numKeys := 10000
	for i := 0; i < numKeys; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("key-%08d", i)))
		value := []byte(fmt.Sprintf("value-%d", i))
		_, err := tree.Put(key, value)
		require.NoError(b, err)
	}

	b.ResetTimer()

	// Benchmark updating a single key in a large tree
	for i := 0; i < b.N; i++ {
		// Update a key in the middle
		key := types.KeyHash([]byte(fmt.Sprintf("key-%08d", numKeys/2)))
		value := []byte(fmt.Sprintf("updated-%d", i))

		_, err := tree.Put(key, value)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVersionedGet measures read performance across versions
func BenchmarkVersionedGet(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	opts := &pebblestorage.Options{
		EnableMetrics: false,
	}
	store, err := pebblestorage.NewStorage(dbPath, opts)
	require.NoError(b, err)
	defer store.Close()
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(b, err)

	// Create multiple versions
	versions := make([]types.Version, 100)
	key := types.KeyHash([]byte("benchmark-key"))

	for i := 0; i < len(versions); i++ {
		value := []byte(fmt.Sprintf("value-v%d", i))
		v, err := tree.Put(key, value)
		require.NoError(b, err)
		versions[i] = v
	}

	b.ResetTimer()

	// Benchmark reading from different versions
	for i := 0; i < b.N; i++ {
		version := versions[i%len(versions)]
		_, err := tree.GetAtVersion(version, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGarbageCollection measures GC performance
func BenchmarkGarbageCollection(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	opts := &pebblestorage.Options{
		EnableMetrics: false,
	}
	store, err := pebblestorage.NewStorage(dbPath, opts)
	require.NoError(b, err)
	defer store.Close()
	keyEncoder := storage.NewDefaultKeyEncoder()

	tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
	require.NoError(b, err)

	// Set retention policy
	tree.SetVersionRetentionPolicy(RetentionPolicyCount, 50, 0)

	// Create many versions
	for i := 0; i < 1000; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i)))
		value := []byte(fmt.Sprintf("value-%d", i))
		_, err := tree.Put(key, value)
		require.NoError(b, err)
	}

	b.ResetTimer()

	// Benchmark GC
	for i := 0; i < b.N; i++ {
		removed, err := tree.CollectVersionGarbage()
		if err != nil {
			b.Fatal(err)
		}
		_ = removed

		// Create more versions to ensure GC has work to do
		if i%10 == 0 {
			for j := 0; j < 20; j++ {
				key := types.KeyHash([]byte(fmt.Sprintf("gc-key-%d-%d", i, j)))
				_, err := tree.Put(key, []byte("value"))
				require.NoError(b, err)
			}
		}
	}
}

// BenchmarkMemoryUsageWithVersions measures memory growth with versions
func BenchmarkMemoryUsageWithVersions(b *testing.B) {
	scenarios := []struct {
		name      string
		retention VersionRetentionPolicy
		keep      int
	}{
		{"NoGC", RetentionPolicyNone, 0},
		{"Keep100", RetentionPolicyCount, 100},
		{"Keep10", RetentionPolicyCount, 10},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			tmpDir := b.TempDir()
			dbPath := filepath.Join(tmpDir, "bench.db")

			opts := &pebblestorage.Options{
				EnableMetrics: false,
			}
			store, err := pebblestorage.NewStorage(dbPath, opts)
			require.NoError(b, err)
			defer store.Close()

			keyEncoder := storage.NewDefaultKeyEncoder()

			tree, err := NewTree(store, keyEncoder, DefaultTreeConfig())
			require.NoError(b, err)

			tree.SetVersionRetentionPolicy(scenario.retention, scenario.keep, 0)

			b.ResetTimer()

			// Create versions and periodically run GC
			for i := 0; i < b.N; i++ {
				key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i)))
				value := []byte(fmt.Sprintf("value-%d", i))

				_, err := tree.Put(key, value)
				if err != nil {
					b.Fatal(err)
				}

				// Run GC every 50 versions
				if i%50 == 0 && scenario.retention != RetentionPolicyNone {
					_, err := tree.CollectVersionGarbage()
					if err != nil {
						b.Fatal(err)
					}
				}
			}

			// Report final version count
			versions := tree.versionManager.GetAllVersions()
			b.ReportMetric(float64(len(versions)), "versions")
		})
	}
}
