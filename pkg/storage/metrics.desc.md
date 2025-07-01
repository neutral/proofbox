# Storage Metrics

This module provides comprehensive metrics collection for storage operations, enabling performance monitoring and optimization.

## Purpose

The metrics system tracks key performance indicators for storage operations:
- Operation counts and throughput
- Latency distributions with percentile calculations
- Database and cache size metrics
- Cache hit rates for optimization

## Key Components

### MetricsCollector
The main metrics aggregator that uses atomic operations for thread-safe counting and a histogram-based approach for latency tracking. It maintains sliding windows of latency samples to calculate accurate percentiles without excessive memory usage.

### Latency Histogram
A custom histogram implementation that buckets latencies into predefined ranges, allowing efficient percentile calculations (P50, P99) without storing every individual sample. The buckets range from nanoseconds to seconds to cover all typical storage operation latencies.

### NullMetrics
A no-op implementation of the Metrics interface used when metrics collection is disabled, ensuring zero overhead in production environments where monitoring is not required.

## Design Decisions

- **Atomic Operations**: All counters use atomic operations to avoid lock contention in high-throughput scenarios
- **Histogram-based Percentiles**: Rather than storing all latency samples, we use a histogram approach that provides good accuracy with bounded memory usage
- **Background Updates**: Database size metrics are updated periodically in the background to avoid impacting operation latency
- **Interface-based**: The Metrics interface allows different storage backends to provide their own optimized implementations

## Usage

Metrics are automatically collected when enabled in storage configuration. The PebbleDB driver integrates metrics collection into all operations, recording latencies and updating counters without significant performance impact.