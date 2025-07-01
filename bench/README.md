# ProofBox Benchmarks

This directory contains system-level benchmarks and performance validation for the Jellyfish Merkle Tree implementation.

## Overview

ProofBox benchmarks are organized into two categories:

1. **Package-level benchmarks** (in `pkg/*/..._test.go`): Micro-benchmarks for specific functions and components
2. **System-level benchmarks** (in this directory): End-to-end performance tests and requirement validation

## Performance Targets

Based on the requirements, ProofBox must meet these performance targets:

- **Insert throughput**: ≥ 50,000 ops/sec on 8-core 2024 laptop
- **Lookup latency**: p95 ≤ 1ms at 10M keys
- **Proof generation**: p95 ≤ 5ms
- **Proof verification**: p95 ≤ 300μs
- **Commit latency**: Scales linearly with batch size

## Running Benchmarks

### Quick Start

```bash
# Run all benchmarks (system + package level)
make bench-all

# Run only system benchmarks
make bench-system

# Run only package benchmarks
make bench-pkg

# Validate performance targets are met
make bench-validate

# Generate performance report
make bench-report
```

### Manual Execution

```bash
# Run specific benchmark
go test -bench=BenchmarkRealWorldInsert -benchmem ./bench/...

# Run with custom duration
go test -bench=. -benchtime=30s ./bench/...

# Run with CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./bench/...

# Quick mode for development
BENCH_QUICK=1 go test -bench=. -benchtime=1s ./bench/...

# Save results
go test -bench=. ./bench/... | tee bench/results/$(date +%Y%m%d_%H%M%S).txt
```

## Benchmark Organization

### System Benchmarks (`system_bench_test.go`)
End-to-end benchmarks that simulate real-world usage:
- `BenchmarkRealWorldInsert`: Realistic key distribution and values
- `BenchmarkBlockProcessing`: Simulates blockchain state updates
- `BenchmarkLightClientProofPath`: Full proof generation and verification flow
- `BenchmarkConcurrentReads`: Multi-reader performance


### Requirements Tests (`requirements_test.go`)
Validates that performance targets are met:
- `TestInsertThroughputRequirement`
- `TestLookupLatencyRequirement`
- `TestProofGenerationLatency`
- `TestProofVerificationLatency`

## Interpreting Results

### Benchmark Output
```
BenchmarkRealWorldInsert-8    50000    23456 ns/op    4096 B/op    45 allocs/op
```
- `-8`: Number of CPU cores
- `50000`: Number of iterations
- `23456 ns/op`: Nanoseconds per operation
- `4096 B/op`: Bytes allocated per operation
- `45 allocs/op`: Number of allocations per operation

### Performance Metrics
The benchrunner tool calculates:
- **Throughput**: Operations per second
- **Latency percentiles**: p50, p95, p99, p99.9
- **Memory efficiency**: Allocations and bytes per operation
- **Scalability**: Performance at different tree sizes

### Results Storage
Results are saved in `bench/results/` with timestamps:
- Raw output: `YYYYMMDD_HHMMSS.txt`
- JSON format: `YYYYMMDD_HHMMSS.json`
- Comparison reports: `comparison_*.md`

## Adding New Benchmarks

### System Benchmark
Add to `system_bench_test.go`:
```go
func BenchmarkYourScenario(b *testing.B) {
    // Setup
    tree := setupBenchmarkTree(b, 1_000_000) // 1M keys
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Your benchmark code
    }
}
```

### Package Benchmark
Add to the relevant `pkg/*/..._test.go` file:
```go
func BenchmarkYourFunction(b *testing.B) {
    // Setup
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Benchmark the specific function
    }
}
```

## Continuous Performance Monitoring

The CI pipeline runs benchmarks on each commit to:
1. Detect performance regressions
2. Track performance trends over time
3. Validate that targets are still met
4. Generate performance reports

## Tips for Accurate Benchmarks

1. **Use realistic data**: Don't just use sequential keys
2. **Pre-populate trees**: Test operations on already-populated trees
3. **Reset timer after setup**: Use `b.ResetTimer()` after initialization
4. **Run multiple times**: Results can vary, run several times for stability
5. **Control environment**: Close other applications, use consistent hardware
6. **Profile when needed**: Use `-cpuprofile` and `-memprofile` for deep analysis

## Benchmark Development Guidelines

1. **Isolation**: Each benchmark should be independent
2. **Repeatability**: Results should be consistent across runs
3. **Relevance**: Measure what matters for real usage
4. **Clarity**: Use descriptive names and document what's measured
5. **Efficiency**: Don't measure setup/teardown time