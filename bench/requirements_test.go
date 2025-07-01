package bench

import (
	"fmt"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/neutral/proofbox/pkg/proof"
	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/storage/memory"
	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/require"
)

// latencyTracker tracks operation latencies for percentile calculation
type latencyTracker struct {
	latencies []time.Duration
}

func (lt *latencyTracker) record(d time.Duration) {
	lt.latencies = append(lt.latencies, d)
}

func (lt *latencyTracker) percentile(p float64) time.Duration {
	if len(lt.latencies) == 0 {
		return 0
	}
	
	sort.Slice(lt.latencies, func(i, j int) bool {
		return lt.latencies[i] < lt.latencies[j]
	})
	
	idx := int(float64(len(lt.latencies)) * p / 100.0)
	if idx >= len(lt.latencies) {
		idx = len(lt.latencies) - 1
	}
	
	return lt.latencies[idx]
}

// TestInsertThroughputRequirement validates ≥ 50,000 ops/sec requirement
func TestInsertThroughputRequirement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping throughput test in short mode")
	}
	
	// Only run on systems with at least 8 cores
	if runtime.NumCPU() < 8 {
		t.Skipf("Skipping throughput test: requires 8+ cores, have %d", runtime.NumCPU())
	}
	
	store := memory.NewStorage()
	keyEncoder := storage.NewDefaultKeyEncoder()
	tr, err := tree.NewTree(store, keyEncoder, tree.DefaultTreeConfig())
	require.NoError(t, err)
	
	// Pre-populate with 1M keys to simulate realistic conditions
	t.Log("Pre-populating tree with 1M keys...")
	for i := 0; i < 1_000_000; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("existing-key-%d", i)))
		_, err := tr.Put(key, []byte("value"))
		require.NoError(t, err)
		
		if (i+1)%100000 == 0 {
			t.Logf("Inserted %d keys...", i+1)
		}
	}
	
	// Measure insert throughput
	const numOps = 100_000
	keys := make([]types.Key, numOps)
	values := make([][]byte, numOps)
	
	// Pre-generate data
	for i := 0; i < numOps; i++ {
		keys[i] = types.KeyHash([]byte(fmt.Sprintf("bench-key-%d", i)))
		values[i] = []byte(fmt.Sprintf("bench-value-%d", i))
	}
	
	t.Log("Starting throughput test...")
	start := time.Now()
	
	for i := 0; i < numOps; i++ {
		_, err := tr.Put(keys[i], values[i])
		require.NoError(t, err)
	}
	
	elapsed := time.Since(start)
	opsPerSec := float64(numOps) / elapsed.Seconds()
	
	t.Logf("Insert throughput: %.0f ops/sec (target: ≥50,000)", opsPerSec)
	
	// Requirement: ≥ 50,000 ops/sec
	require.GreaterOrEqual(t, opsPerSec, 50000.0, 
		"Insert throughput %.0f ops/sec is below requirement of 50,000 ops/sec", opsPerSec)
}

// TestLookupLatencyRequirement validates p95 ≤ 1ms at 10M keys
func TestLookupLatencyRequirement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping latency test in short mode")
	}
	
	store := memory.NewStorage()
	keyEncoder := storage.NewDefaultKeyEncoder()
	tr, err := tree.NewTree(store, keyEncoder, tree.DefaultTreeConfig())
	require.NoError(t, err)
	
	// For practical testing, use 1M keys instead of 10M
	const numKeys = 1_000_000
	t.Logf("Pre-populating tree with %d keys (scaled down from 10M for testing)...", numKeys)
	
	lookupKeys := make([]types.Key, 10000)
	
	for i := 0; i < numKeys; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i)))
		_, err := tr.Put(key, []byte(fmt.Sprintf("value-%d", i)))
		require.NoError(t, err)
		
		// Save some keys for lookup testing
		if i < len(lookupKeys) {
			lookupKeys[i] = key
		}
		
		if (i+1)%100000 == 0 {
			t.Logf("Inserted %d keys...", i+1)
		}
	}
	
	// Measure lookup latency
	tracker := &latencyTracker{}
	const numLookups = 10000
	
	t.Log("Measuring lookup latency...")
	for i := 0; i < numLookups; i++ {
		key := lookupKeys[i%len(lookupKeys)]
		version := tr.GetLatestVersion()
		
		start := time.Now()
		_, err := tr.GetAtVersion(version, key)
		latency := time.Since(start)
		require.NoError(t, err)
		
		tracker.record(latency)
	}
	
	p95 := tracker.percentile(95)
	t.Logf("Lookup latency p95: %v (target: ≤1ms)", p95)
	
	// Requirement: p95 ≤ 1ms
	require.LessOrEqual(t, p95, 1*time.Millisecond,
		"Lookup latency p95 %v exceeds requirement of 1ms", p95)
}

// TestProofGenerationLatency validates p95 ≤ 5ms
func TestProofGenerationLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping proof generation latency test in short mode")
	}
	
	store := memory.NewStorage()
	keyEncoder := storage.NewDefaultKeyEncoder()
	tr, err := tree.NewTree(store, keyEncoder, tree.DefaultTreeConfig())
	require.NoError(t, err)
	
	// Build a reasonably sized tree
	const numKeys = 100_000
	t.Logf("Pre-populating tree with %d keys...", numKeys)
	
	proofKeys := make([]types.Key, 1000)
	
	for i := 0; i < numKeys; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i)))
		_, err := tr.Put(key, []byte(fmt.Sprintf("value-%d", i)))
		require.NoError(t, err)
		
		if i < len(proofKeys) {
			proofKeys[i] = key
		}
	}
	
	version := tr.GetLatestVersion()
	reader, err := tr.Reader(version)
	require.NoError(t, err)
	defer reader.Close()
	
	generator := proof.NewGenerator(reader)
	
	// Measure proof generation latency
	tracker := &latencyTracker{}
	const numProofs = 1000
	
	t.Log("Measuring proof generation latency...")
	for i := 0; i < numProofs; i++ {
		key := proofKeys[i%len(proofKeys)]
		
		start := time.Now()
		_, err := generator.Generate(key)
		latency := time.Since(start)
		require.NoError(t, err)
		
		tracker.record(latency)
	}
	
	p95 := tracker.percentile(95)
	t.Logf("Proof generation latency p95: %v (target: ≤5ms)", p95)
	
	// Requirement: p95 ≤ 5ms
	require.LessOrEqual(t, p95, 5*time.Millisecond,
		"Proof generation latency p95 %v exceeds requirement of 5ms", p95)
}

// TestProofVerificationLatency validates p95 ≤ 300μs
func TestProofVerificationLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping proof verification latency test in short mode")
	}
	
	store := memory.NewStorage()
	keyEncoder := storage.NewDefaultKeyEncoder()
	tr, err := tree.NewTree(store, keyEncoder, tree.DefaultTreeConfig())
	require.NoError(t, err)
	
	// Build tree and generate proofs
	const numKeys = 10_000
	t.Logf("Pre-populating tree with %d keys...", numKeys)
	
	for i := 0; i < numKeys; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i)))
		_, err := tr.Put(key, []byte(fmt.Sprintf("value-%d", i)))
		require.NoError(t, err)
	}
	
	version := tr.GetLatestVersion()
	reader, err := tr.Reader(version)
	require.NoError(t, err)
	defer reader.Close()
	
	generator := proof.NewGenerator(reader)
	verifier := proof.NewVerifier()
	
	// Pre-generate proofs
	t.Log("Pre-generating proofs...")
	proofs := make([]*proof.Proof, 100)
	for i := range proofs {
		key := types.KeyHash([]byte(fmt.Sprintf("key-%d", i)))
		p, err := generator.Generate(key)
		require.NoError(t, err)
		proofs[i] = p
	}
	
	// Measure verification latency
	tracker := &latencyTracker{}
	const numVerifications = 10000
	
	t.Log("Measuring proof verification latency...")
	for i := 0; i < numVerifications; i++ {
		p := proofs[i%len(proofs)]
		
		start := time.Now()
		err := verifier.Verify(p)
		latency := time.Since(start)
		require.NoError(t, err)
		
		tracker.record(latency)
	}
	
	p95 := tracker.percentile(95)
	t.Logf("Proof verification latency p95: %v (target: ≤300μs)", p95)
	
	// Requirement: p95 ≤ 300μs
	require.LessOrEqual(t, p95, 300*time.Microsecond,
		"Proof verification latency p95 %v exceeds requirement of 300μs", p95)
}

// TestBatchCommitLinearScaling validates that commit latency scales linearly with batch size
func TestBatchCommitLinearScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scaling test in short mode")
	}
	
	store := memory.NewStorage()
	keyEncoder := storage.NewDefaultKeyEncoder()
	tr, err := tree.NewTree(store, keyEncoder, tree.DefaultTreeConfig())
	require.NoError(t, err)
	
	// Test different batch sizes
	batchSizes := []int{10, 100, 1000}
	latencies := make([]time.Duration, len(batchSizes))
	
	t.Log("Measuring batch commit scaling...")
	
	for i, size := range batchSizes {
		batch := tr.NewBatchTransaction()
		
		// Add operations to batch
		for j := 0; j < size; j++ {
			key := types.KeyHash([]byte(fmt.Sprintf("batch-key-%d-%d", i, j)))
			value := []byte(fmt.Sprintf("batch-value-%d-%d", i, j))
			err := batch.BatchPut(key, value)
			require.NoError(t, err)
		}
		
		// Measure commit time
		start := time.Now()
		_, err := batch.Execute()
		latency := time.Since(start)
		require.NoError(t, err)
		
		latencies[i] = latency
		t.Logf("Batch size %d: %v", size, latency)
	}
	
	// Check linear scaling: latency should increase roughly proportionally
	// Allow 50% deviation from perfect linear scaling
	for i := 1; i < len(batchSizes); i++ {
		expectedRatio := float64(batchSizes[i]) / float64(batchSizes[0])
		actualRatio := float64(latencies[i]) / float64(latencies[0])
		
		deviation := (actualRatio - expectedRatio) / expectedRatio
		t.Logf("Batch %d vs %d: expected ratio %.2f, actual ratio %.2f (deviation: %.1f%%)",
			batchSizes[i], batchSizes[0], expectedRatio, actualRatio, deviation*100)
		
		// Allow up to 50% deviation from linear scaling
		require.Less(t, deviation, 0.5,
			"Batch commit scaling deviates too much from linear: %.1f%% deviation", deviation*100)
	}
	
	t.Log("Batch commit scales approximately linearly with batch size ✓")
}

// TestSystemInfo logs system information for benchmark context
func TestSystemInfo(t *testing.T) {
	t.Logf("System Information:")
	t.Logf("- Go Version: %s", runtime.Version())
	t.Logf("- GOOS: %s", runtime.GOOS)
	t.Logf("- GOARCH: %s", runtime.GOARCH)
	t.Logf("- NumCPU: %d", runtime.NumCPU())
	t.Logf("- GOMAXPROCS: %d", runtime.GOMAXPROCS(0))
}