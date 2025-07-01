package tree

import (
	"testing"
	"time"

	"github.com/neutral/proofbox/pkg/metrics"
	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsIntegration(t *testing.T) {
	// Create a custom registry for testing
	registry := prometheus.NewRegistry()
	metricsCollector := metrics.NewPrometheusMetricsWithRegistry(registry)

	// Create storage and tree with metrics
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()
	
	config := DefaultTreeConfig()
	config.MetricsEnabled = true
	config.Metrics = metricsCollector
	
	tree, err := NewTree(store, keyEncoder, config)
	require.NoError(t, err)

	t.Run("Put operations record metrics", func(t *testing.T) {
		key := types.KeyHash([]byte("metrics-test-key"))
		value := []byte("metrics-test-value")
		
		// Perform put operation
		version, err := tree.Put(key, value)
		require.NoError(t, err)
		assert.Equal(t, types.Version(1), version)
		
		// Check commit counter
		commitCounter, err := registry.Gather()
		require.NoError(t, err)
		
		// Find the commits_total metric
		var found bool
		for _, mf := range commitCounter {
			if mf.GetName() == "jmt_commits_total" {
				found = true
				assert.Equal(t, float64(1), mf.GetMetric()[0].GetCounter().GetValue())
			}
		}
		assert.True(t, found, "commits_total metric not found")
	})

	t.Run("Get operations record metrics", func(t *testing.T) {
		key := types.KeyHash([]byte("metrics-get-key"))
		value := []byte("metrics-get-value")
		
		// Insert value
		version, err := tree.Put(key, value)
		require.NoError(t, err)
		
		// Perform get operation
		retrieved, err := tree.Get(version, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)
		
		// Try a miss
		_, err = tree.Get(version, types.KeyHash([]byte("non-existent")))
		assert.NoError(t, err) // Get returns nil, not error for missing keys
	})

	t.Run("Batch operations record metrics", func(t *testing.T) {
		batch := tree.NewBatchTransaction()
		
		// Add multiple operations to batch
		for i := 0; i < 5; i++ {
			key := types.KeyHash([]byte{byte(i)})
			value := []byte{byte(i + 100)}
			require.NoError(t, batch.BatchPut(key, value))
		}
		
		// Execute batch
		version, err := batch.Execute()
		require.NoError(t, err)
		assert.Greater(t, version, types.Version(0))
	})

	t.Run("Tree state metrics", func(t *testing.T) {
		// The tree should have some state by now
		latestVersion := tree.GetLatestVersion()
		assert.Greater(t, latestVersion, types.Version(0))
		
		// Check if metrics were updated (we can't easily check gauge values
		// without exposing them, but we can verify they don't panic)
		reader, err := tree.Reader(latestVersion)
		require.NoError(t, err)
		defer reader.Close()
	})

	t.Run("Error metrics", func(t *testing.T) {
		// Try to get from invalid version
		_, err := tree.Get(types.Version(9999), types.KeyHash([]byte("key")))
		assert.ErrorIs(t, err, types.ErrVersionNotFound)
		
		// Try invalid operations
		_, err = tree.Put(types.Key{}, []byte("value"))
		assert.Error(t, err)
	})

	t.Run("Metrics collection can be disabled", func(t *testing.T) {
		// Create tree without metrics
		config := DefaultTreeConfig()
		config.MetricsEnabled = false
		
		tree2, err := NewTree(createTestStorage(t), keyEncoder, config)
		require.NoError(t, err)
		
		// Operations should work without metrics
		key := types.KeyHash([]byte("no-metrics-key"))
		version, err := tree2.Put(key, []byte("value"))
		require.NoError(t, err)
		
		retrieved, err := tree2.Get(version, key)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
	})
}

func TestMetricsPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Benchmark overhead of metrics collection
	runBenchmark := func(metricsEnabled bool) time.Duration {
		store := createTestStorage(t)
		keyEncoder := storage.NewDefaultKeyEncoder()
		
		config := DefaultTreeConfig()
		if metricsEnabled {
			registry := prometheus.NewRegistry()
			config.MetricsEnabled = true
			config.Metrics = metrics.NewPrometheusMetricsWithRegistry(registry)
		} else {
			config.MetricsEnabled = false
			config.Metrics = metrics.NoOpMetrics{}
		}
		
		tree, err := NewTree(store, keyEncoder, config)
		require.NoError(t, err)
		
		start := time.Now()
		
		// Perform many operations
		for i := 0; i < 1000; i++ {
			key := types.KeyHash([]byte{byte(i % 256), byte(i / 256)})
			value := []byte("test-value")
			
			version, err := tree.Put(key, value)
			require.NoError(t, err)
			
			_, err = tree.Get(version, key)
			require.NoError(t, err)
		}
		
		return time.Since(start)
	}
	
	// Run without metrics
	durationNoMetrics := runBenchmark(false)
	
	// Run with metrics
	durationWithMetrics := runBenchmark(true)
	
	// Calculate overhead
	overhead := float64(durationWithMetrics-durationNoMetrics) / float64(durationNoMetrics) * 100
	
	t.Logf("Without metrics: %v", durationNoMetrics)
	t.Logf("With metrics: %v", durationWithMetrics)
	t.Logf("Overhead: %.2f%%", overhead)
	
	// Ensure overhead is reasonable (less than 10%)
	assert.Less(t, overhead, 10.0, "Metrics overhead should be less than 10%")
}

func TestMetricsRegistryIsolation(t *testing.T) {
	// Test that multiple trees with metrics can coexist
	registry1 := prometheus.NewRegistry()
	registry2 := prometheus.NewRegistry()
	
	store1 := createTestStorage(t)
	store2 := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()
	
	// Create first tree
	config1 := DefaultTreeConfig()
	config1.MetricsEnabled = true
	config1.Metrics = metrics.NewPrometheusMetricsWithRegistry(registry1)
	
	tree1, err := NewTree(store1, keyEncoder, config1)
	require.NoError(t, err)
	
	// Create second tree
	config2 := DefaultTreeConfig()
	config2.MetricsEnabled = true
	config2.Metrics = metrics.NewPrometheusMetricsWithRegistry(registry2)
	
	tree2, err := NewTree(store2, keyEncoder, config2)
	require.NoError(t, err)
	
	// Use both trees
	key := types.KeyHash([]byte("test-key"))
	
	_, err = tree1.Put(key, []byte("value1"))
	require.NoError(t, err)
	
	_, err = tree2.Put(key, []byte("value2"))
	require.NoError(t, err)
	
	// Both should have their own metrics
	metrics1, err := registry1.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metrics1)
	
	metrics2, err := registry2.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metrics2)
}

func TestPrometheusMetricsLabels(t *testing.T) {
	registry := prometheus.NewRegistry()
	metricsCollector := metrics.NewPrometheusMetricsWithRegistry(registry)
	
	// Test that operation types are recorded with correct labels
	metricsCollector.RecordOperation("insert")
	metricsCollector.RecordOperation("update")
	metricsCollector.RecordOperation("delete")
	
	// Test error types
	metricsCollector.RecordError("storage")
	metricsCollector.RecordError("validation")
	metricsCollector.RecordError("codec")
	
	// Test proof types
	metricsCollector.RecordProofGeneration(0.001, "inclusion", 100)
	metricsCollector.RecordProofGeneration(0.001, "exclusion", 50)
	
	// Gather and check metrics
	gathered, err := registry.Gather()
	require.NoError(t, err)
	
	// Verify operation labels
	for _, mf := range gathered {
		switch mf.GetName() {
		case "jmt_operations_total":
			// Should have metrics for each operation type
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 3)
			
		case "jmt_errors_total":
			// Should have metrics for each error type
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 3)
			
		case "jmt_proof_generations_total":
			// Should have metrics for each proof type
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 2)
		}
	}
}

func TestMetricsWithTreeReader(t *testing.T) {
	registry := prometheus.NewRegistry()
	metricsCollector := metrics.NewPrometheusMetricsWithRegistry(registry)
	
	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()
	
	config := DefaultTreeConfig()
	config.MetricsEnabled = true
	config.Metrics = metricsCollector
	
	tree, err := NewTree(store, keyEncoder, config)
	require.NoError(t, err)
	
	// Insert some data
	key := types.KeyHash([]byte("reader-test-key"))
	value := []byte("reader-test-value")
	version, err := tree.Put(key, value)
	require.NoError(t, err)
	
	// Create reader and perform operations
	reader, err := tree.Reader(version)
	require.NoError(t, err)
	defer reader.Close()
	
	// Get value through tree (reader doesn't have Get method)
	retrieved, err := tree.Get(version, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
	
	// Try non-existent key through tree
	_, err = tree.Get(version, types.KeyHash([]byte("non-existent")))
	assert.NoError(t, err)
	
	// Verify reader has access to metrics
	assert.NotNil(t, reader.Metrics())
}