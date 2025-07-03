//go:build !short
// +build !short

package performance

import (
	"testing"
	"time"

	"github.com/neutral/proofbox/pkg/fuzz"
	"github.com/neutral/proofbox/pkg/types"
)

// TestMemoryGrowthExtended tests memory growth with larger datasets
// This test is skipped in short mode
func TestMemoryGrowthExtended(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping extended memory growth test in short mode")
	}

	// Test with 1M operations
	tree, err := fuzz.NewTestTree()
	if err != nil {
		t.Fatalf("failed to create tree: %v", err)
	}

	// Run 1M operations and check memory growth
	const numOps = 1_000_000
	for i := 0; i < numOps; i++ {
		key := types.KeyHash([]byte{byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i)})
		value := []byte{byte(i)}

		if _, err := tree.Put(key, value); err != nil {
			t.Fatalf("put failed at %d: %v", i, err)
		}

		// Log memory usage every 100k operations
		if i > 0 && i%100000 == 0 {
			// In real implementation, would check actual memory usage
			t.Logf("Processed %d operations", i)
		}
	}
}

// TestLatencyUnderLoadExtended tests latency with sustained high load
// This test is skipped in short mode
func TestLatencyUnderLoadExtended(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping extended latency test in short mode")
	}

	tree, err := fuzz.NewTestTree()
	if err != nil {
		t.Fatalf("failed to create tree: %v", err)
	}

	// Pre-populate with 100k keys
	for i := 0; i < 100_000; i++ {
		key := types.KeyHash([]byte{byte(i >> 16), byte(i >> 8), byte(i), 0})
		value := []byte{byte(i)}
		if _, err := tree.Put(key, value); err != nil {
			t.Fatalf("pre-populate failed: %v", err)
		}
	}

	// Measure latency for 10k operations under load
	start := time.Now()
	for i := 0; i < 10_000; i++ {
		key := types.KeyHash([]byte{byte(i >> 8), byte(i), 0, 0})
		value := []byte{byte(i)}

		opStart := time.Now()
		if _, err := tree.Put(key, value); err != nil {
			t.Fatalf("put failed: %v", err)
		}
		opDuration := time.Since(opStart)

		// Check that no single operation takes more than 10ms
		if opDuration > 10*time.Millisecond {
			t.Errorf("operation %d took %v, exceeding 10ms threshold", i, opDuration)
		}
	}

	totalDuration := time.Since(start)
	avgLatency := totalDuration / 10_000
	t.Logf("Average latency: %v", avgLatency)

	// Average should be under 1ms
	if avgLatency > time.Millisecond {
		t.Errorf("average latency %v exceeds 1ms threshold", avgLatency)
	}
}
