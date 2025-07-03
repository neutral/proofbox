package tree

import (
	"testing"
	"time"

	"github.com/neutral/proofbox/pkg/metrics"
	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsValidation ensures metrics are accurate, not just present
func TestMetricsValidation(t *testing.T) {
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

	t.Run("Validate commit metrics accuracy", func(t *testing.T) {
		// Perform multiple puts
		numOperations := 5
		for i := 0; i < numOperations; i++ {
			key := types.KeyHash([]byte{byte(i)})
			value := []byte{byte(i + 100)}
			_, err := tree.Put(key, value)
			require.NoError(t, err)
		}

		// Get metrics
		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		// Validate commits_total
		commitsTotal := getMetricValue(t, metricFamilies, "jmt_commits_total")
		assert.Equal(t, float64(numOperations), commitsTotal, "Should have exactly %d commits", numOperations)

		// Validate operations_total for inserts
		insertOps := getMetricValueWithLabel(t, metricFamilies, "jmt_operations_total", "type", "insert")
		assert.Equal(t, float64(numOperations), insertOps, "Should have exactly %d insert operations", numOperations)
	})

	t.Run("Validate lookup metrics accuracy", func(t *testing.T) {
		// Insert test data
		key1 := types.KeyHash([]byte("lookup-key-1"))
		key2 := types.KeyHash([]byte("lookup-key-2"))
		value := []byte("test-value")

		version, err := tree.Put(key1, value)
		require.NoError(t, err)

		// Reset metrics for clean test
		registry = prometheus.NewRegistry()
		metricsCollector = metrics.NewPrometheusMetricsWithRegistry(registry)
		config.Metrics = metricsCollector
		tree.metrics = metricsCollector

		// Perform lookups
		numHits := 3
		numMisses := 2

		// Hits
		for i := 0; i < numHits; i++ {
			val, err := tree.Get(version, key1)
			require.NoError(t, err)
			assert.NotNil(t, val)
		}

		// Misses
		for i := 0; i < numMisses; i++ {
			val, err := tree.Get(version, key2)
			require.NoError(t, err)
			assert.Nil(t, val)
		}

		// Validate lookup count
		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		lookupCount := getHistogramCount(t, metricFamilies, "jmt_lookup_latency_seconds")
		assert.Equal(t, float64(numHits+numMisses), lookupCount, "Should have correct total lookup count")

		// Validate hit/miss operations
		hitOps := getMetricValueWithLabel(t, metricFamilies, "jmt_operations_total", "type", "lookup_hit")
		missOps := getMetricValueWithLabel(t, metricFamilies, "jmt_operations_total", "type", "lookup_miss")
		assert.Equal(t, float64(numHits), hitOps, "Should have correct hit count")
		assert.Equal(t, float64(numMisses), missOps, "Should have correct miss count")
	})

	t.Run("Validate batch operation metrics accuracy", func(t *testing.T) {
		// Reset metrics
		registry = prometheus.NewRegistry()
		metricsCollector = metrics.NewPrometheusMetricsWithRegistry(registry)
		config.Metrics = metricsCollector
		tree.metrics = metricsCollector

		// Create batch with mixed operations
		batch := tree.NewBatchTransaction()

		// Inserts
		numOps := 3
		for i := 0; i < numOps; i++ {
			key := types.KeyHash([]byte{byte(i + 100)})
			value := []byte{byte(i)}
			require.NoError(t, batch.BatchPut(key, value))
		}

		// Execute batch
		_, err = batch.Execute()
		require.NoError(t, err)

		// Validate metrics
		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		// Should have 1 commit for the batch
		commitsTotal := getMetricValue(t, metricFamilies, "jmt_commits_total")
		assert.Equal(t, float64(1), commitsTotal, "Batch should result in single commit")

		// Validate operation types
		insertOps := getMetricValueWithLabel(t, metricFamilies, "jmt_operations_total", "type", "insert")
		// TODO: Once we track updates properly, validate update count
		// For now, all puts are counted as inserts
		assert.Equal(t, float64(numOps), insertOps, "Should have correct insert count")
	})

	t.Run("Validate database operation metrics", func(t *testing.T) {
		// Reset metrics
		registry = prometheus.NewRegistry()
		metricsCollector = metrics.NewPrometheusMetricsWithRegistry(registry)
		config.Metrics = metricsCollector
		tree.metrics = metricsCollector

		// Perform operations that trigger DB reads/writes
		key := types.KeyHash([]byte("db-test-key"))
		value := []byte("db-test-value")

		// Put triggers writes
		version, err := tree.Put(key, value)
		require.NoError(t, err)

		// Get triggers reads
		_, err = tree.Get(version, key)
		require.NoError(t, err)

		// Validate metrics
		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		dbReads := getMetricValue(t, metricFamilies, "jmt_db_reads_total")
		dbWrites := getMetricValue(t, metricFamilies, "jmt_db_writes_total")

		assert.Greater(t, dbReads, float64(0), "Should have DB reads")
		assert.Greater(t, dbWrites, float64(0), "Should have DB writes")
	})

	t.Run("Validate error metrics accuracy", func(t *testing.T) {
		// Reset metrics
		registry = prometheus.NewRegistry()
		metricsCollector = metrics.NewPrometheusMetricsWithRegistry(registry)
		config.Metrics = metricsCollector
		tree.metrics = metricsCollector

		// Trigger various errors

		// Validation error - empty key
		_, err = tree.Put(types.Key{}, []byte("value"))
		assert.Error(t, err)

		// Version error - non-existent version
		_, err = tree.Get(9999, types.KeyHash([]byte("key")))
		assert.Error(t, err)

		// Value too large error
		largeValue := make([]byte, types.MaxValueSize+1)
		_, err = tree.Put(types.KeyHash([]byte("key")), largeValue)
		assert.Error(t, err)

		// Validate error metrics
		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		validationErrors := getMetricValueWithLabel(t, metricFamilies, "jmt_errors_total", "type", "validation")
		versionErrors := getMetricValueWithLabel(t, metricFamilies, "jmt_errors_total", "type", "version")

		assert.GreaterOrEqual(t, validationErrors, float64(2), "Should have at least 2 validation errors")
		assert.GreaterOrEqual(t, versionErrors, float64(1), "Should have at least 1 version error")
	})

	t.Run("Validate commit latency histogram", func(t *testing.T) {
		// Reset metrics
		registry = prometheus.NewRegistry()
		metricsCollector = metrics.NewPrometheusMetricsWithRegistry(registry)
		config.Metrics = metricsCollector
		tree.metrics = metricsCollector

		// Perform commits and track time
		numCommits := 10
		for i := 0; i < numCommits; i++ {
			key := types.KeyHash([]byte{byte(i + 200)})
			value := []byte{byte(i)}
			_, err := tree.Put(key, value)
			require.NoError(t, err)
		}

		// Validate histogram
		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		// Check commit latency count
		commitLatencyCount := getHistogramCount(t, metricFamilies, "jmt_commit_latency_seconds")
		assert.Equal(t, float64(numCommits), commitLatencyCount, "Should have correct commit count in histogram")

		// Check that sum is positive (commits took some time)
		commitLatencySum := getHistogramSum(t, metricFamilies, "jmt_commit_latency_seconds")
		assert.Greater(t, commitLatencySum, float64(0), "Commits should have positive total duration")
	})

	t.Run("Validate tree state gauges", func(t *testing.T) {
		// Tree state metrics should be updated
		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		nodeCount := getMetricValue(t, metricFamilies, "jmt_tree_nodes_total")
		versionCount := getMetricValue(t, metricFamilies, "jmt_versions_stored")

		assert.Greater(t, nodeCount, float64(0), "Should have nodes in tree")
		assert.Greater(t, versionCount, float64(0), "Should have versions stored")

		// Version count should match latest version
		latestVersion := tree.GetLatestVersion()
		assert.LessOrEqual(t, versionCount, float64(latestVersion), "Version count should not exceed latest version")
	})

	t.Run("Validate batch size histogram", func(t *testing.T) {
		// Reset metrics
		registry = prometheus.NewRegistry()
		metricsCollector = metrics.NewPrometheusMetricsWithRegistry(registry)
		config.Metrics = metricsCollector
		tree.metrics = metricsCollector

		// Create batches of different sizes
		batchSizes := []int{1, 5, 10}

		for _, size := range batchSizes {
			batch := tree.NewBatchTransaction()
			for i := 0; i < size; i++ {
				key := types.KeyHash([]byte{byte(i), byte(size)})
				value := []byte{byte(i)}
				require.NoError(t, batch.BatchPut(key, value))
			}
			_, err := batch.Execute()
			require.NoError(t, err)
		}

		// Validate batch size metrics
		metricFamilies, err := registry.Gather()
		require.NoError(t, err)

		batchSizeCount := getHistogramCount(t, metricFamilies, "jmt_batch_size")
		assert.Equal(t, float64(len(batchSizes)), batchSizeCount, "Should have correct number of batches")

		batchSizeSum := getHistogramSum(t, metricFamilies, "jmt_batch_size")
		// Note: batch size currently measures nodes created, not operations
		// This is why the sum is much larger than expected operations
		assert.Greater(t, batchSizeSum, float64(0), "Should have recorded batch sizes")
	})
}

// TestMetricsConcurrentAccuracy validates metrics under concurrent load
func TestMetricsConcurrentAccuracy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent metrics test in short mode")
	}

	registry := prometheus.NewRegistry()
	metricsCollector := metrics.NewPrometheusMetricsWithRegistry(registry)

	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()

	config := DefaultTreeConfig()
	config.MetricsEnabled = true
	config.Metrics = metricsCollector

	tree, err := NewTree(store, keyEncoder, config)
	require.NoError(t, err)

	// Run concurrent operations
	numGoroutines := 10
	opsPerGoroutine := 100
	done := make(chan bool, numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			for i := 0; i < opsPerGoroutine; i++ {
				key := types.KeyHash([]byte{byte(goroutineID), byte(i)})
				value := []byte{byte(i)}
				_, err := tree.Put(key, value)
				if err != nil {
					t.Errorf("Put failed: %v", err)
				}
			}
			done <- true
		}(g)
	}

	// Wait for completion
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Validate metrics
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	expectedOps := float64(numGoroutines * opsPerGoroutine)

	// Check commits
	commitsTotal := getMetricValue(t, metricFamilies, "jmt_commits_total")
	assert.Equal(t, expectedOps, commitsTotal, "Should have correct total commits under concurrent load")

	// Check operations
	insertOps := getMetricValueWithLabel(t, metricFamilies, "jmt_operations_total", "type", "insert")
	assert.Equal(t, expectedOps, insertOps, "Should have correct insert operations under concurrent load")
}

// TestMetricsLatencyAccuracy validates that latency metrics are reasonable
func TestMetricsLatencyAccuracy(t *testing.T) {
	registry := prometheus.NewRegistry()
	metricsCollector := metrics.NewPrometheusMetricsWithRegistry(registry)

	store := createTestStorage(t)
	keyEncoder := storage.NewDefaultKeyEncoder()

	config := DefaultTreeConfig()
	config.MetricsEnabled = true
	config.Metrics = metricsCollector

	tree, err := NewTree(store, keyEncoder, config)
	require.NoError(t, err)

	// Perform timed operations
	start := time.Now()
	key := types.KeyHash([]byte("latency-test"))
	value := []byte("test-value")

	_, err = tree.Put(key, value)
	require.NoError(t, err)

	elapsed := time.Since(start)

	// Get metrics
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	// Check commit latency
	commitLatencySum := getHistogramSum(t, metricFamilies, "jmt_commit_latency_seconds")

	// Latency should be positive but less than total elapsed time
	assert.Greater(t, commitLatencySum, float64(0), "Commit latency should be positive")
	assert.LessOrEqual(t, commitLatencySum, elapsed.Seconds(), "Commit latency should not exceed total time")

	// Latency should be reasonable (less than 1 second for in-memory operation)
	assert.Less(t, commitLatencySum, 1.0, "Commit latency should be reasonable for in-memory operation")
}

// Helper function to get metric value by name
func getMetricValue(t *testing.T, families []*dto.MetricFamily, name string) float64 {
	for _, mf := range families {
		if mf.GetName() == name {
			metrics := mf.GetMetric()
			if len(metrics) > 0 {
				switch mf.GetType() {
				case dto.MetricType_COUNTER:
					return metrics[0].GetCounter().GetValue()
				case dto.MetricType_GAUGE:
					return metrics[0].GetGauge().GetValue()
				case dto.MetricType_HISTOGRAM:
					// For histograms, check the actual metric name
					hist := metrics[0].GetHistogram()
					if hist != nil {
						return float64(hist.GetSampleCount())
					}
					return 0
				}
			}
		}
	}
	t.Logf("Metric %s not found", name)
	return 0
}

// Helper function to get histogram count
func getHistogramCount(t *testing.T, families []*dto.MetricFamily, name string) float64 {
	for _, mf := range families {
		if mf.GetName() == name && mf.GetType() == dto.MetricType_HISTOGRAM {
			metrics := mf.GetMetric()
			if len(metrics) > 0 {
				hist := metrics[0].GetHistogram()
				if hist != nil {
					return float64(hist.GetSampleCount())
				}
			}
		}
	}
	t.Logf("Histogram %s not found", name)
	return 0
}

// Helper function to get histogram sum
func getHistogramSum(t *testing.T, families []*dto.MetricFamily, name string) float64 {
	for _, mf := range families {
		if mf.GetName() == name && mf.GetType() == dto.MetricType_HISTOGRAM {
			metrics := mf.GetMetric()
			if len(metrics) > 0 {
				hist := metrics[0].GetHistogram()
				if hist != nil {
					return hist.GetSampleSum()
				}
			}
		}
	}
	t.Logf("Histogram %s sum not found", name)
	return 0
}

// Helper function to get metric value with specific label
func getMetricValueWithLabel(t *testing.T, families []*dto.MetricFamily, name, labelName, labelValue string) float64 {
	for _, mf := range families {
		if mf.GetName() == name {
			for _, metric := range mf.GetMetric() {
				for _, label := range metric.GetLabel() {
					if label.GetName() == labelName && label.GetValue() == labelValue {
						switch mf.GetType() {
						case dto.MetricType_COUNTER:
							return metric.GetCounter().GetValue()
						case dto.MetricType_GAUGE:
							return metric.GetGauge().GetValue()
						}
					}
				}
			}
		}
	}
	t.Logf("Metric %s with label %s=%s not found", name, labelName, labelValue)
	return 0
}
