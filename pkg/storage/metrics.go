package storage

import (
	"sync/atomic"
	"time"
)

// MetricsCollector collects storage metrics for performance monitoring.
type MetricsCollector struct {
	// Operation counts
	getOps    atomic.Uint64
	putOps    atomic.Uint64
	deleteOps atomic.Uint64
	batchOps  atomic.Uint64

	// Latency tracking (using histograms)
	getLatencies *latencyHistogram
	putLatencies *latencyHistogram

	// Size metrics
	dbSize       atomic.Uint64
	liveDataSize atomic.Uint64

	// Cache metrics
	cacheHits   atomic.Uint64
	cacheMisses atomic.Uint64
	cacheSize   atomic.Uint64
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		getLatencies: newLatencyHistogram(),
		putLatencies: newLatencyHistogram(),
	}
}

// RecordGet records a get operation with its latency.
func (m *MetricsCollector) RecordGet(latency time.Duration) {
	m.getOps.Add(1)
	m.getLatencies.record(latency)
}

// RecordPut records a put operation with its latency.
func (m *MetricsCollector) RecordPut(latency time.Duration) {
	m.putOps.Add(1)
	m.putLatencies.record(latency)
}

// RecordDelete records a delete operation.
func (m *MetricsCollector) RecordDelete() {
	m.deleteOps.Add(1)
}

// RecordBatchCommit records a batch commit.
func (m *MetricsCollector) RecordBatchCommit() {
	m.batchOps.Add(1)
}

// RecordCacheHit records a cache hit.
func (m *MetricsCollector) RecordCacheHit() {
	m.cacheHits.Add(1)
}

// RecordCacheMiss records a cache miss.
func (m *MetricsCollector) RecordCacheMiss() {
	m.cacheMisses.Add(1)
}

// UpdateDatabaseSize updates the database size metric.
func (m *MetricsCollector) UpdateDatabaseSize(size uint64) {
	m.dbSize.Store(size)
}

// UpdateLiveDataSize updates the live data size metric.
func (m *MetricsCollector) UpdateLiveDataSize(size uint64) {
	m.liveDataSize.Store(size)
}

// UpdateCacheSize updates the cache size metric.
func (m *MetricsCollector) UpdateCacheSize(size uint64) {
	m.cacheSize.Store(size)
}

// GetOperations returns the total number of get operations.
func (m *MetricsCollector) GetOperations() uint64 {
	return m.getOps.Load()
}

// PutOperations returns the total number of put operations.
func (m *MetricsCollector) PutOperations() uint64 {
	return m.putOps.Load()
}

// DeleteOperations returns the total number of delete operations.
func (m *MetricsCollector) DeleteOperations() uint64 {
	return m.deleteOps.Load()
}

// BatchCommits returns the total number of batch commits.
func (m *MetricsCollector) BatchCommits() uint64 {
	return m.batchOps.Load()
}

// GetLatencyP50 returns the 50th percentile get latency in nanoseconds.
func (m *MetricsCollector) GetLatencyP50() uint64 {
	return uint64(m.getLatencies.percentile(0.50))
}

// GetLatencyP99 returns the 99th percentile get latency in nanoseconds.
func (m *MetricsCollector) GetLatencyP99() uint64 {
	return uint64(m.getLatencies.percentile(0.99))
}

// PutLatencyP50 returns the 50th percentile put latency in nanoseconds.
func (m *MetricsCollector) PutLatencyP50() uint64 {
	return uint64(m.putLatencies.percentile(0.50))
}

// PutLatencyP99 returns the 99th percentile put latency in nanoseconds.
func (m *MetricsCollector) PutLatencyP99() uint64 {
	return uint64(m.putLatencies.percentile(0.99))
}

// DatabaseSize returns the total database size in bytes.
func (m *MetricsCollector) DatabaseSize() uint64 {
	return m.dbSize.Load()
}

// LiveDataSize returns the live data size in bytes.
func (m *MetricsCollector) LiveDataSize() uint64 {
	return m.liveDataSize.Load()
}

// CacheHitRate returns the cache hit rate as a percentage.
func (m *MetricsCollector) CacheHitRate() float64 {
	hits := float64(m.cacheHits.Load())
	misses := float64(m.cacheMisses.Load())
	total := hits + misses
	if total == 0 {
		return 0
	}
	return hits / total
}

// CacheSize returns the current cache size in bytes.
func (m *MetricsCollector) CacheSize() uint64 {
	return m.cacheSize.Load()
}

// latencyHistogram tracks latency distributions using a simple histogram.
type latencyHistogram struct {
	buckets []atomic.Uint64
	bounds  []time.Duration
	total   atomic.Uint64
}

func newLatencyHistogram() *latencyHistogram {
	// Define latency buckets (in nanoseconds)
	bounds := []time.Duration{
		100 * time.Nanosecond,
		500 * time.Nanosecond,
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
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}

	h := &latencyHistogram{
		buckets: make([]atomic.Uint64, len(bounds)+1),
		bounds:  bounds,
	}
	return h
}

func (h *latencyHistogram) record(d time.Duration) {
	h.total.Add(1)
	
	// Find the appropriate bucket
	for i, bound := range h.bounds {
		if d <= bound {
			h.buckets[i].Add(1)
			return
		}
	}
	// If greater than all bounds, add to the last bucket
	h.buckets[len(h.buckets)-1].Add(1)
}

func (h *latencyHistogram) percentile(p float64) time.Duration {
	total := h.total.Load()
	if total == 0 {
		return 0
	}

	target := uint64(float64(total) * p)
	var count uint64

	for i, bucket := range h.buckets {
		count += bucket.Load()
		if count >= target {
			if i == 0 {
				return h.bounds[0] / 2
			}
			if i < len(h.bounds) {
				return h.bounds[i-1] + (h.bounds[i]-h.bounds[i-1])/2
			}
			return h.bounds[len(h.bounds)-1]
		}
	}

	return h.bounds[len(h.bounds)-1]
}

// NullMetrics is a no-op implementation of the Metrics interface.
type NullMetrics struct{}

func (n NullMetrics) GetOperations() uint64     { return 0 }
func (n NullMetrics) PutOperations() uint64     { return 0 }
func (n NullMetrics) DeleteOperations() uint64  { return 0 }
func (n NullMetrics) BatchCommits() uint64      { return 0 }
func (n NullMetrics) GetLatencyP50() uint64     { return 0 }
func (n NullMetrics) GetLatencyP99() uint64     { return 0 }
func (n NullMetrics) PutLatencyP50() uint64     { return 0 }
func (n NullMetrics) PutLatencyP99() uint64     { return 0 }
func (n NullMetrics) DatabaseSize() uint64      { return 0 }
func (n NullMetrics) LiveDataSize() uint64      { return 0 }
func (n NullMetrics) CacheHitRate() float64     { return 0 }
func (n NullMetrics) CacheSize() uint64         { return 0 }

// Ensure implementations satisfy the Metrics interface
var (
	_ Metrics = (*MetricsCollector)(nil)
	_ Metrics = NullMetrics{}
)