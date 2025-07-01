package storage_test

import (
	"testing"
	"time"

	"github.com/neutral/proofbox/pkg/storage"
	"github.com/stretchr/testify/assert"
)

func TestMetricsCollector(t *testing.T) {
	t.Run("Operation Counts", func(t *testing.T) {
		collector := storage.NewMetricsCollector()

		// Record operations
		collector.RecordGet(10 * time.Microsecond)
		collector.RecordGet(20 * time.Microsecond)
		collector.RecordPut(15 * time.Microsecond)
		collector.RecordDelete()
		collector.RecordBatchCommit()

		// Check counts
		assert.Equal(t, uint64(2), collector.GetOperations())
		assert.Equal(t, uint64(1), collector.PutOperations())
		assert.Equal(t, uint64(1), collector.DeleteOperations())
		assert.Equal(t, uint64(1), collector.BatchCommits())
	})

	t.Run("Latency Tracking", func(t *testing.T) {
		collector := storage.NewMetricsCollector()

		// Record various latencies
		latencies := []time.Duration{
			1 * time.Microsecond,
			5 * time.Microsecond,
			10 * time.Microsecond,
			50 * time.Microsecond,
			100 * time.Microsecond,
			500 * time.Microsecond,
			1 * time.Millisecond,
			5 * time.Millisecond,
			10 * time.Millisecond,
			50 * time.Millisecond,
		}

		for _, latency := range latencies {
			collector.RecordGet(latency)
			collector.RecordPut(latency)
		}

		// Check percentiles
		getP50 := collector.GetLatencyP50()
		getP99 := collector.GetLatencyP99()
		putP50 := collector.PutLatencyP50()
		putP99 := collector.PutLatencyP99()

		// P50 should be around the median value
		assert.Greater(t, getP50, uint64(0))
		assert.Greater(t, putP50, uint64(0))

		// P99 should be >= P50
		assert.GreaterOrEqual(t, getP99, getP50)
		assert.GreaterOrEqual(t, putP99, putP50)

		// P99 should be close to the higher values
		assert.Greater(t, getP99, uint64(1*time.Millisecond))
		assert.Greater(t, putP99, uint64(1*time.Millisecond))
	})

	t.Run("Size Metrics", func(t *testing.T) {
		collector := storage.NewMetricsCollector()

		// Update size metrics
		collector.UpdateDatabaseSize(1024 * 1024) // 1MB
		collector.UpdateLiveDataSize(512 * 1024)  // 512KB
		collector.UpdateCacheSize(64 * 1024)      // 64KB

		// Check values
		assert.Equal(t, uint64(1024*1024), collector.DatabaseSize())
		assert.Equal(t, uint64(512*1024), collector.LiveDataSize())
		assert.Equal(t, uint64(64*1024), collector.CacheSize())
	})

	t.Run("Cache Hit Rate", func(t *testing.T) {
		collector := storage.NewMetricsCollector()

		// No operations yet
		assert.Equal(t, float64(0), collector.CacheHitRate())

		// Record some hits and misses
		for i := 0; i < 70; i++ {
			collector.RecordCacheHit()
		}
		for i := 0; i < 30; i++ {
			collector.RecordCacheMiss()
		}

		// Should be 70%
		hitRate := collector.CacheHitRate()
		assert.InDelta(t, 0.7, hitRate, 0.01)
	})

	t.Run("Concurrent Updates", func(t *testing.T) {
		collector := storage.NewMetricsCollector()

		// Run concurrent operations
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					collector.RecordGet(time.Duration(j) * time.Microsecond)
					collector.RecordPut(time.Duration(j) * time.Microsecond)
					collector.RecordDelete()
					collector.RecordCacheHit()
				}
				done <- true
			}()
		}

		// Wait for completion
		for i := 0; i < 10; i++ {
			<-done
		}

		// Check final counts
		assert.Equal(t, uint64(1000), collector.GetOperations())
		assert.Equal(t, uint64(1000), collector.PutOperations())
		assert.Equal(t, uint64(1000), collector.DeleteOperations())
	})
}

func TestNullMetrics(t *testing.T) {
	metrics := storage.NullMetrics{}

	// All operations should return zero/default values
	assert.Equal(t, uint64(0), metrics.GetOperations())
	assert.Equal(t, uint64(0), metrics.PutOperations())
	assert.Equal(t, uint64(0), metrics.DeleteOperations())
	assert.Equal(t, uint64(0), metrics.BatchCommits())
	assert.Equal(t, uint64(0), metrics.GetLatencyP50())
	assert.Equal(t, uint64(0), metrics.GetLatencyP99())
	assert.Equal(t, uint64(0), metrics.PutLatencyP50())
	assert.Equal(t, uint64(0), metrics.PutLatencyP99())
	assert.Equal(t, uint64(0), metrics.DatabaseSize())
	assert.Equal(t, uint64(0), metrics.LiveDataSize())
	assert.Equal(t, float64(0), metrics.CacheHitRate())
	assert.Equal(t, uint64(0), metrics.CacheSize())
}

func TestLatencyHistogram(t *testing.T) {
	collector := storage.NewMetricsCollector()

	// Test edge cases
	t.Run("Empty Histogram", func(t *testing.T) {
		// New collector should have zero latencies
		assert.Equal(t, uint64(0), collector.GetLatencyP50())
		assert.Equal(t, uint64(0), collector.GetLatencyP99())
	})

	t.Run("Single Value", func(t *testing.T) {
		collector := storage.NewMetricsCollector()
		collector.RecordGet(100 * time.Microsecond)

		// Both percentiles should be around the same value
		p50 := collector.GetLatencyP50()
		p99 := collector.GetLatencyP99()
		
		// Should be in the appropriate bucket range
		assert.Greater(t, p50, uint64(0))
		assert.LessOrEqual(t, p50, uint64(500*time.Microsecond))
		assert.Equal(t, p50, p99)
	})

	t.Run("Very Large Values", func(t *testing.T) {
		collector := storage.NewMetricsCollector()
		
		// Record some very large latencies
		collector.RecordPut(2 * time.Second)
		collector.RecordPut(3 * time.Second)
		collector.RecordPut(5 * time.Second)

		p99 := collector.PutLatencyP99()
		// Should be in the highest bucket
		assert.GreaterOrEqual(t, p99, uint64(1*time.Second))
	})
}

func BenchmarkMetricsCollection(b *testing.B) {
	b.Run("RecordGet", func(b *testing.B) {
		collector := storage.NewMetricsCollector()
		latency := 100 * time.Microsecond

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			collector.RecordGet(latency)
		}
	})

	b.Run("RecordPut", func(b *testing.B) {
		collector := storage.NewMetricsCollector()
		latency := 100 * time.Microsecond

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			collector.RecordPut(latency)
		}
	})

	b.Run("GetPercentiles", func(b *testing.B) {
		collector := storage.NewMetricsCollector()
		
		// Pre-populate with data
		for i := 0; i < 1000; i++ {
			collector.RecordGet(time.Duration(i) * time.Microsecond)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = collector.GetLatencyP50()
			_ = collector.GetLatencyP99()
		}
	})

	b.Run("ConcurrentOperations", func(b *testing.B) {
		collector := storage.NewMetricsCollector()

		b.RunParallel(func(pb *testing.PB) {
			latency := 100 * time.Microsecond
			for pb.Next() {
				collector.RecordGet(latency)
				collector.RecordCacheHit()
			}
		})
	})
}