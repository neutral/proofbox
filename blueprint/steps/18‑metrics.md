---
id: step.18.metrics
depends_on:
tags: [metrics, step]
---

## Objective

Expose Prometheus counter `jmt_commits_total`.

## Implements

- **Non‑Functional Goals – Flexibility & Observability** (wasn't named but implied).
  _What happens_:

  - Real‑time counter supports ops dashboards and mirrors spec's idea of "prometheus metrics appear at /metrics" (original 10‑step plan, step 09).

## Technical Details

### Comprehensive Metrics Collection

Per `nfr.observability.metrics` requirement, implement Prometheus-compatible metrics:

1. **Core JMT Metrics**

   ```
   # Counter metrics
   jmt_commits_total                 # Total number of commits
   jmt_operations_total{type="..."}  # Operations by type (insert/update/delete)
   jmt_proof_generations_total{type="..."} # Proofs generated (inclusion/exclusion)

   # Histogram metrics
   jmt_commit_latency_seconds        # Commit operation duration
   jmt_lookup_latency_seconds        # Lookup operation duration
   jmt_proof_generation_latency_seconds # Proof generation time
   jmt_batch_size                   # Number of operations per batch

   # Gauge metrics
   jmt_tree_height                   # Current tree height
   jmt_tree_nodes_total             # Total nodes in tree
   jmt_versions_stored              # Number of versions retained
   jmt_db_live_bytes               # Storage size in bytes
   ```

2. **Performance Metrics**

   ```
   jmt_hash_operations_total        # Hash computations
   jmt_db_reads_total              # Database read operations
   jmt_db_writes_total             # Database write operations
   jmt_proof_size_bytes            # Size of generated proofs
   ```

3. **Error Tracking**
   ```
   jmt_errors_total{type="..."}    # Errors by type
   jmt_validation_failures_total   # Proof validation failures
   ```

### Implementation Architecture

```go
// metrics/collector.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // Counters
    CommitsTotal = promauto.NewCounter(prometheus.CounterOpts{
        Name: "jmt_commits_total",
        Help: "Total number of tree commits",
    })

    OperationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "jmt_operations_total",
        Help: "Total operations by type",
    }, []string{"type"})

    // Histograms
    CommitLatency = promauto.NewHistogram(prometheus.HistogramOpts{
        Name:    "jmt_commit_latency_seconds",
        Help:    "Commit operation latency",
        Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
    })

    ProofSize = promauto.NewHistogram(prometheus.HistogramOpts{
        Name:    "jmt_proof_size_bytes",
        Help:    "Size of generated proofs",
        Buckets: prometheus.ExponentialBuckets(100, 2, 10),
    })

    // Gauges
    TreeHeight = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "jmt_tree_height",
        Help: "Current height of the tree",
    })

    DBLiveBytes = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "jmt_db_live_bytes",
        Help: "Live data size in database",
    })
)

// Instrumented tree operations
func InstrumentedCommit(tree *Tree, batch *UpdateBatch) error {
    timer := prometheus.NewTimer(CommitLatency)
    defer timer.ObserveDuration()

    err := tree.Commit(batch)
    if err == nil {
        CommitsTotal.Inc()
        OperationsTotal.WithLabelValues("commit").Add(float64(len(batch.Updates)))
    }

    return err
}
```

### HTTP Endpoint Setup

```go
// metrics/server.go
package metrics

import (
    "net/http"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartMetricsServer(addr string) error {
    http.Handle("/metrics", promhttp.Handler())

    // Add custom metrics with database stats
    prometheus.MustRegister(prometheus.NewGaugeFunc(
        prometheus.GaugeOpts{
            Name: "jmt_db_live_bytes",
            Help: "Database live data size",
        },
        func() float64 {
            stats := GetDBStats()
            return float64(stats.LiveBytes)
        },
    ))

    return http.ListenAndServe(addr, nil)
}
```

## Implementation Steps

1. **Add Prometheus Dependencies**

   ```bash
   go get github.com/prometheus/client_golang/prometheus
   go get github.com/prometheus/client_golang/prometheus/promhttp
   ```

2. **Create Metrics Package**

   - Define all metric collectors
   - Implement instrumentation helpers
   - Add metric registration

3. **Instrument Core Operations**

   - Wrap tree operations with metrics
   - Add timing measurements
   - Track operation counts

4. **Implement Custom Collectors**

   - Database statistics collector
   - Tree statistics collector
   - Version history collector

5. **Add Metrics HTTP Server**
   - Configure `/metrics` endpoint
   - Optional metrics port configuration
   - Graceful shutdown handling

## Testing Requirements

### Functional Tests

- [ ] Metrics endpoint responds with 200 OK
- [ ] All defined metrics appear in output
- [ ] Metrics increment correctly during operations
- [ ] Histograms record accurate latencies

### Integration Tests

- [ ] Run operations and verify counters
- [ ] Check histogram buckets populated
- [ ] Gauge values update correctly
- [ ] Labels properly applied to metrics

### Load Tests

- [ ] Metrics collection doesn't impact performance
- [ ] No memory leaks from metrics
- [ ] Cardinality stays reasonable

### Monitoring Validation

- [ ] Prometheus can scrape endpoint
- [ ] Grafana dashboards display correctly
- [ ] Alerts can be configured on metrics

## Done When ✓

- [ ] `/metrics` shows counter > 0 after tests.
- [ ] All core JMT operations instrumented
- [ ] Latency histograms tracking p50/p95/p99
- [ ] Database size metrics exposed
- [ ] Error metrics tracking failures
- [ ] Documentation includes metrics reference
- [ ] Example Grafana dashboard provided
