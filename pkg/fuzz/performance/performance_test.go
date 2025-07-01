package performance

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/neutral/proofbox/pkg/fuzz"
	"github.com/neutral/proofbox/pkg/fuzz/generators"
	"github.com/neutral/proofbox/pkg/types"
	"pgregory.net/rapid"
)

// PerformanceMetrics tracks performance characteristics
type PerformanceMetrics struct {
	OperationType    string
	NumOperations    int
	TotalDuration    time.Duration
	AvgLatency       time.Duration
	MaxLatency       time.Duration
	MinLatency       time.Duration
	MemoryUsedMB     float64
	AllocsMB         float64
	TreeDepth        int
	NumNodes         int
	OpsPerSecond     float64
}

func TestMemoryGrowth(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create tree
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}
		
		// Measure initial memory
		var initialMem runtime.MemStats
		runtime.ReadMemStats(&initialMem)
		
		// Generate operations that might cause memory growth
		numOps := rapid.IntRange(100, 500).Draw(t, "num_ops")
		
		// Track memory at intervals
		memSamples := []float64{}
		opsCompleted := 0
		
		for i := 0; i < numOps; i++ {
			op := generators.OperationGen().Draw(t, "op")
			
			switch op.Type {
			case generators.OpPut:
				_, err := tr.Put(op.Key, op.Value)
				if err != nil {
					continue
				}
				opsCompleted++
				
			case generators.OpDelete:
				_, err := tr.Delete(op.Key)
				if err != nil {
					continue
				}
				opsCompleted++
			}
			
			// Sample memory every 10 operations
			if i%10 == 0 {
				var mem runtime.MemStats
				runtime.ReadMemStats(&mem)
				memUsedMB := float64(mem.Alloc-initialMem.Alloc) / 1024 / 1024
				memSamples = append(memSamples, memUsedMB)
			}
		}
		
		// Check for excessive memory growth
		if len(memSamples) > 2 {
			firstSample := memSamples[0]
			lastSample := memSamples[len(memSamples)-1]
			
			// Memory should not grow excessively
			// Allow for some growth but flag if it's more than 100MB or 10x initial
			if lastSample > 100 && lastSample > firstSample*10 {
				t.Logf("Warning: significant memory growth: %0.2f MB -> %0.2f MB", firstSample, lastSample)
			}
		}
		
		t.Logf("Memory usage after %d operations: %0.2f MB", opsCompleted, memSamples[len(memSamples)-1])
	})
}

func TestOperationLatency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create tree
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}
		
		// Pre-populate tree
		numInitialKeys := rapid.IntRange(100, 1000).Draw(t, "initial_keys")
		for i := 0; i < numInitialKeys; i++ {
			key := generators.OperationGen().Draw(t, "key").Key
			value := rapid.SliceOfN(rapid.Byte(), 10, 100).Draw(t, "value")
			tr.Put(key, value)
		}
		
		// Measure operation latencies
		metrics := make(map[string]*PerformanceMetrics)
		metrics["put"] = &PerformanceMetrics{OperationType: "put", MinLatency: time.Hour}
		metrics["get"] = &PerformanceMetrics{OperationType: "get", MinLatency: time.Hour}
		metrics["delete"] = &PerformanceMetrics{OperationType: "delete", MinLatency: time.Hour}
		
		// Run operations and measure
		numOps := rapid.IntRange(50, 200).Draw(t, "num_ops")
		
		for i := 0; i < numOps; i++ {
			op := generators.OperationGen().Draw(t, "op")
			
			var start time.Time
			var latency time.Duration
			var opType string
			
			switch op.Type {
			case generators.OpPut:
				start = time.Now()
				_, err := tr.Put(op.Key, op.Value)
				latency = time.Since(start)
				opType = "put"
				if err != nil {
					continue
				}
				
			case generators.OpGet:
				if op.Version > tr.GetLatestVersion() {
					continue
				}
				start = time.Now()
				_, err := tr.GetAtVersion(op.Version, op.Key)
				latency = time.Since(start)
				opType = "get"
				if err != nil {
					continue
				}
				
			case generators.OpDelete:
				start = time.Now()
				_, err := tr.Delete(op.Key)
				latency = time.Since(start)
				opType = "delete"
				if err != nil {
					continue
				}
			}
			
			// Update metrics
			m := metrics[opType]
			m.NumOperations++
			m.TotalDuration += latency
			if latency > m.MaxLatency {
				m.MaxLatency = latency
			}
			if latency < m.MinLatency {
				m.MinLatency = latency
			}
		}
		
		// Calculate averages and verify performance
		for opType, m := range metrics {
			if m.NumOperations > 0 {
				m.AvgLatency = m.TotalDuration / time.Duration(m.NumOperations)
				m.OpsPerSecond = float64(m.NumOperations) / m.TotalDuration.Seconds()
				
				// Verify latencies are reasonable (< 10ms for in-memory operations)
				if m.AvgLatency > 10*time.Millisecond {
					t.Fatalf("%s operations too slow: avg=%v, max=%v", opType, m.AvgLatency, m.MaxLatency)
				}
				
				t.Logf("%s performance: avg=%v, max=%v, ops/sec=%.0f",
					opType, m.AvgLatency, m.MaxLatency, m.OpsPerSecond)
			}
		}
	})
}

func TestScalability(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create tree
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}
		
		// Test at different scales
		scales := []int{10, 100, 1000}
		latencies := make(map[int]time.Duration)
		
		for _, scale := range scales {
			// Generate keys for this scale
			keys := make([]types.Key, scale)
			for i := 0; i < scale; i++ {
				keys[i] = types.KeyHash([]byte(fmt.Sprintf("key-%d-%d", scale, i)))
			}
			
			// Measure time to insert all keys
			start := time.Now()
			for i, key := range keys {
				value := []byte(fmt.Sprintf("value-%d", i))
				_, err := tr.Put(key, value)
				if err != nil {
					t.Fatalf("put failed at scale %d: %v", scale, err)
				}
			}
			totalTime := time.Since(start)
			
			avgLatency := totalTime / time.Duration(scale)
			latencies[scale] = avgLatency
			
			t.Logf("Scale %d: total=%v, avg=%v", scale, totalTime, avgLatency)
		}
		
		// Verify scalability (latency shouldn't increase more than logarithmically)
		if len(scales) >= 2 {
			// Compare latency growth between scales
			for i := 1; i < len(scales); i++ {
				prevScale := scales[i-1]
				currScale := scales[i]
				
				prevLatency := latencies[prevScale]
				currLatency := latencies[currScale]
				
				// Allow up to 2x latency increase for 10x scale increase
				maxAllowedIncrease := prevLatency * 2
				
				if currLatency > maxAllowedIncrease {
					t.Fatalf("poor scalability: %d->%d scale increased latency from %v to %v",
						prevScale, currScale, prevLatency, currLatency)
				}
			}
		}
	})
}

func TestCachingEffectiveness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create tree
		tr, err := fuzz.NewTestTree()
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}
		
		// Create a working set of keys
		workingSetSize := rapid.IntRange(10, 50).Draw(t, "working_set_size")
		workingSet := make([]types.Key, workingSetSize)
		
		for i := 0; i < workingSetSize; i++ {
			key := generators.OperationGen().Draw(t, "key").Key
			value := rapid.SliceOfN(rapid.Byte(), 10, 100).Draw(t, "value")
			
			_, err := tr.Put(key, value)
			if err != nil {
				t.Fatalf("failed to put key: %v", err)
			}
			workingSet[i] = key
		}
		
		version := tr.GetLatestVersion()
		
		// Measure cold cache performance
		coldLatencies := []time.Duration{}
		for _, key := range workingSet {
			start := time.Now()
			_, err := tr.Get(version, key)
			if err != nil {
				continue
			}
			coldLatencies = append(coldLatencies, time.Since(start))
		}
		
		// Measure warm cache performance (read same keys again)
		warmLatencies := []time.Duration{}
		for _, key := range workingSet {
			start := time.Now()
			_, err := tr.Get(version, key)
			if err != nil {
				continue
			}
			warmLatencies = append(warmLatencies, time.Since(start))
		}
		
		// Calculate averages
		var coldTotal, warmTotal time.Duration
		for i := range coldLatencies {
			coldTotal += coldLatencies[i]
			warmTotal += warmLatencies[i]
		}
		
		coldAvg := coldTotal / time.Duration(len(coldLatencies))
		warmAvg := warmTotal / time.Duration(len(warmLatencies))
		
		// Warm cache should be faster
		if warmAvg >= coldAvg {
			t.Logf("Warning: caching not effective - cold=%v, warm=%v", coldAvg, warmAvg)
		} else {
			speedup := float64(coldAvg) / float64(warmAvg)
			t.Logf("Cache speedup: %.2fx (cold=%v, warm=%v)", speedup, coldAvg, warmAvg)
		}
	})
}