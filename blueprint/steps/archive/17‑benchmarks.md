---
id: step.17.benchmarks
depends_on:
tags: [benchmark, step]
---

## Objective

Add benchmarks for hasher, lookup, and commit; save under `bench/`.

## Implements

- **Non‑Functional Goals – Performance & Scalability**.
  _What happens_:

  - Encodes spec's benchmark aspirations ("bench throughput ≥ 20 k inserts/s") into repeatable Go benchmarks and spills raw numbers to files for trend tracking.

## Technical Details

### Comprehensive Benchmark Suite

Per `proof-performance.md` and throughput requirements, the benchmark suite must validate:

1. **Core Operation Benchmarks**

   - Hash computation (SHA-256 throughput)
   - Key encoding/decoding (nibble operations)
   - Tree traversal (lookup paths)
   - Proof generation (inclusion/exclusion)
   - Batch commit (write amplification)

2. **Performance Targets**

   - Insert throughput: ≥ 20,000 ops/sec
   - Lookup latency: p95 ≤ 1ms
   - Proof generation: p95 ≤ 5ms
   - Proof verification: p95 ≤ 300μs
   - Commit latency: scales linearly with batch size

3. **Scalability Tests**
   - Tree sizes: 1K, 10K, 100K, 1M, 10M keys
   - Batch sizes: 1, 10, 100, 1000, 10000 updates
   - Concurrent operations: 1, 4, 16, 64 goroutines
   - Memory usage tracking per operation

## Implementation Steps

1. **Create Benchmark Package Structure**

   ```
   bench/
   ├── hasher_bench.go      # Hash function performance
   ├── tree_bench.go        # Core tree operations
   ├── batch_bench.go       # Batch commit performance
   ├── proof_bench.go       # Proof generation/verification
   ├── memory_bench.go      # Memory allocation tracking
   ├── concurrent_bench.go  # Concurrency stress tests
   ├── results.go           # Result collection/formatting
   └── results/             # Benchmark output directory
   ```

2. **Implement Core Benchmarks**

   - Hash computation throughput
   - Single key insert/lookup/delete
   - Batch operations at various sizes
   - Proof generation for different tree depths

3. **Add Latency Percentile Tracking**

   - Use histogram for latency distribution
   - Report p50, p95, p99, p99.9
   - Track outliers and max latency

4. **Memory Profiling Integration**

   - Track allocations per operation
   - Monitor heap growth over time
   - Identify memory leaks or inefficiencies

5. **Result Persistence**
   - JSON for machine parsing
   - Human-readable reports
   - CSV for Excel/trend analysis
   - Markdown for documentation

## Testing Requirements

### Benchmark Validity

- [ ] Benchmarks use realistic data distributions
- [ ] Tree pre-populated to target size
- [ ] Warmup iterations before timing
- [ ] Results reproducible across runs

### Performance Targets Met

- [ ] Insert throughput ≥ 20K ops/sec at 1M keys
- [ ] Lookup latency p95 ≤ 1ms at 10M keys
- [ ] Proof verification ≤ 300μs consistently
- [ ] Linear scaling with batch size

### Result Quality

- [ ] Results include environment info (CPU, RAM, Go version)
- [ ] Statistical significance reported
- [ ] Comparison with previous runs
- [ ] Regression detection automated

## Done When ✓

- [ ] `go test -bench ./...` writes three result files.
- [ ] Benchmark suite covers all core operations
- [ ] Results show performance meeting targets
- [ ] Memory usage tracked and acceptable
- [ ] Latency percentiles calculated correctly
- [ ] Results persisted in bench/results/
- [ ] Trend analysis possible from saved data
