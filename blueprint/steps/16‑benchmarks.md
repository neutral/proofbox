---
id: step.20.benchmarks
depends_on:
  - step.18.persist‑multi‑ver
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

### Benchmark Implementation

```go
// bench/hasher_bench.go
func BenchmarkHasher(b *testing.B) {
    hasher := NewHasher()
    data := make([]byte, 32)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        hasher.Hash(data)
    }
    b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "hashes/sec")
}

// bench/tree_bench.go
func BenchmarkTreeOperations(b *testing.B) {
    sizes := []int{1000, 10000, 100000, 1000000}
    
    for _, size := range sizes {
        b.Run(fmt.Sprintf("Insert-%d", size), func(b *testing.B) {
            tree := setupTree(b, size)
            benchmarkInsert(b, tree)
        })
        
        b.Run(fmt.Sprintf("Lookup-%d", size), func(b *testing.B) {
            tree := setupTree(b, size)
            benchmarkLookup(b, tree)
        })
        
        b.Run(fmt.Sprintf("Proof-%d", size), func(b *testing.B) {
            tree := setupTree(b, size)
            benchmarkProof(b, tree)
        })
    }
}

// bench/batch_bench.go
func BenchmarkBatchCommit(b *testing.B) {
    batchSizes := []int{1, 10, 100, 1000, 10000}
    
    for _, batchSize := range batchSizes {
        b.Run(fmt.Sprintf("Batch-%d", batchSize), func(b *testing.B) {
            tree := NewTree()
            updates := generateUpdates(batchSize)
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                batch := tree.NewBatch()
                for _, update := range updates {
                    batch.Put(update.Key, update.Value)
                }
                batch.Commit()
            }
            
            b.ReportMetric(float64(batchSize*b.N)/b.Elapsed().Seconds(), "updates/sec")
        })
    }
}
```

### Result Collection

```go
// bench/results.go
type BenchmarkResult struct {
    Name       string
    Operations int
    Duration   time.Duration
    Throughput float64
    Latency    Percentiles
    Memory     MemoryStats
}

func SaveResults(results []BenchmarkResult) error {
    timestamp := time.Now().Format("20060102-150405")
    
    // Save raw data
    rawPath := fmt.Sprintf("bench/results/raw-%s.json", timestamp)
    // Save human-readable report
    reportPath := fmt.Sprintf("bench/results/report-%s.txt", timestamp)
    // Save CSV for trend analysis
    csvPath := fmt.Sprintf("bench/results/data-%s.csv", timestamp)
    
    return nil
}
```

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
