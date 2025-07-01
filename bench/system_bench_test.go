package bench

import (
	"crypto/rand"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/neutral/proofbox/pkg/proof"
	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/storage/memory"
	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/require"
)

// setupBenchmarkTree creates a tree pre-populated with the specified number of keys
func setupBenchmarkTree(b *testing.B, numKeys int) *tree.Tree {
	b.Helper()

	// Use memory storage for consistent benchmarks
	store := memory.NewStorage()
	keyEncoder := storage.NewDefaultKeyEncoder()

	tr, err := tree.NewTree(store, keyEncoder, tree.DefaultTreeConfig())
	require.NoError(b, err)

	// Check for quick mode to reduce setup time
	if os.Getenv("BENCH_QUICK") == "1" && numKeys > 1000 {
		numKeys = 1000
		b.Logf("Quick mode: limiting pre-population to %d keys", numKeys)
	}

	// Pre-populate with realistic key distribution
	b.Logf("Pre-populating tree with %d keys...", numKeys)
	startTime := time.Now()
	
	for i := 0; i < numKeys; i++ {
		// Use random keys to simulate realistic distribution
		keyData := make([]byte, 32)
		rand.Read(keyData)
		key := types.Key(keyData)
		
		value := []byte(fmt.Sprintf("value-%d", i))
		_, err := tr.Put(key, value)
		require.NoError(b, err)
		
		if (i+1)%10000 == 0 {
			b.Logf("Inserted %d keys...", i+1)
		}
	}
	
	b.Logf("Tree setup complete in %v", time.Since(startTime))
	return tr
}

// BenchmarkRealWorldInsert measures insert performance with realistic key distribution
func BenchmarkRealWorldInsert(b *testing.B) {
	scenarios := []struct {
		name         string
		existingKeys int
		skipShort    bool
	}{
		{"Empty", 0, false},
		{"1K_Keys", 1_000, false},
		{"10K_Keys", 10_000, false},
		{"100K_Keys", 100_000, true},
		{"1M_Keys", 1_000_000, true},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			if scenario.skipShort && (testing.Short() || os.Getenv("BENCH_QUICK") == "1") {
				b.Skip("Skipping large tree scenario in short/quick mode")
			}
			tree := setupBenchmarkTree(b, scenario.existingKeys)
			
			// Pre-generate random keys and values
			keys := make([]types.Key, b.N)
			values := make([][]byte, b.N)
			for i := 0; i < b.N; i++ {
				keyData := make([]byte, 32)
				rand.Read(keyData)
				keys[i] = types.Key(keyData)
				values[i] = []byte(fmt.Sprintf("bench-value-%d", i))
			}
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_, err := tree.Put(keys[i], values[i])
				if err != nil {
					b.Fatal(err)
				}
			}
			
			// Report throughput
			opsPerSec := float64(b.N) / b.Elapsed().Seconds()
			b.ReportMetric(opsPerSec, "ops/sec")
		})
	}
}

// BenchmarkLookupLatency measures read performance at various tree sizes
func BenchmarkLookupLatency(b *testing.B) {
	scenarios := []struct {
		name     string
		treeSize int
	}{
		{"1K_Keys", 1_000},
		{"10K_Keys", 10_000},
		{"100K_Keys", 100_000},
		{"1M_Keys", 1_000_000},
		{"10M_Keys", 10_000_000},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Skip very large trees in short mode, quick mode, or normal bench runs
			if scenario.treeSize > 100_000 && (testing.Short() || os.Getenv("BENCH_QUICK") == "1" || os.Getenv("BENCH_LARGE") != "1") {
				b.Skip("Skipping large tree scenario (set BENCH_LARGE=1 to run)")
			}
			
			tree := setupBenchmarkTree(b, scenario.treeSize)
			
			// Pre-generate keys that exist in the tree
			existingKeys := make([]types.Key, 1000)
			for i := 0; i < len(existingKeys); i++ {
				keyData := make([]byte, 32)
				rand.Read(keyData)
				key := types.Key(keyData)
				_, err := tree.Put(key, []byte("lookup-value"))
				require.NoError(b, err)
				existingKeys[i] = key
			}
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				key := existingKeys[i%len(existingKeys)]
				version := tree.GetLatestVersion()
				_, err := tree.GetAtVersion(version, key)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkBlockProcessing simulates blockchain block processing
func BenchmarkBlockProcessing(b *testing.B) {
	blockSizes := []int{100, 500, 1000, 5000, 10000}
	
	for _, blockSize := range blockSizes {
		b.Run(fmt.Sprintf("BlockSize_%d", blockSize), func(b *testing.B) {
			treeSize := 1_000_000
			if os.Getenv("BENCH_QUICK") == "1" {
				treeSize = 10_000
			}
			tree := setupBenchmarkTree(b, treeSize) // Existing accounts
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				// Begin new block
				version, err := tree.BeginVersion()
				require.NoError(b, err)
				
				// Process block transactions
				for j := 0; j < blockSize; j++ {
					// Simulate account updates
					accountKey := types.KeyHash([]byte(fmt.Sprintf("account-%d-%d", i, j)))
					accountData := []byte(fmt.Sprintf("balance:%d,nonce:%d", j*1000, j))
					
					err = tree.PutVersioned(version, accountKey, accountData)
					if err != nil {
						b.Fatal(err)
					}
				}
				
				// Commit block
				err = tree.CommitVersion(version)
				if err != nil {
					b.Fatal(err)
				}
			}
			
			blocksPerSec := float64(b.N) / b.Elapsed().Seconds()
			txPerSec := blocksPerSec * float64(blockSize)
			b.ReportMetric(blocksPerSec, "blocks/sec")
			b.ReportMetric(txPerSec, "tx/sec")
		})
	}
}

// BenchmarkBatchCommit measures batch operation performance
func BenchmarkBatchCommit(b *testing.B) {
	batchSizes := []int{1, 10, 100, 1000, 10000}
	
	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			tree := setupBenchmarkTree(b, 100_000)
			
			// Pre-generate operations
			operations := make([]struct {
				key   types.Key
				value []byte
			}, batchSize)
			
			for i := range operations {
				keyData := make([]byte, 32)
				rand.Read(keyData)
				operations[i].key = types.Key(keyData)
				operations[i].value = []byte(fmt.Sprintf("batch-value-%d", i))
			}
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				batch := tree.NewBatchTransaction()
				
				for _, op := range operations {
					err := batch.BatchPut(op.key, op.value)
					if err != nil {
						b.Fatal(err)
					}
				}
				
				_, err := batch.Execute()
				if err != nil {
					b.Fatal(err)
				}
			}
			
			// Report operations per second
			totalOps := float64(b.N * batchSize)
			opsPerSec := totalOps / b.Elapsed().Seconds()
			b.ReportMetric(opsPerSec, "ops/sec")
			b.ReportMetric(float64(batchSize), "ops/batch")
		})
	}
}

// BenchmarkLightClientProofPath measures full proof generation and verification
func BenchmarkLightClientProofPath(b *testing.B) {
	treeSizes := []int{1000, 10000, 100000, 1000000}
	
	for _, treeSize := range treeSizes {
		b.Run(fmt.Sprintf("TreeSize_%d", treeSize), func(b *testing.B) {
			if treeSize > 100000 && (testing.Short() || os.Getenv("BENCH_QUICK") == "1" || os.Getenv("BENCH_LARGE") != "1") {
				b.Skip("Skipping large tree scenario (set BENCH_LARGE=1 to run)")
			}
			
			tree := setupBenchmarkTree(b, treeSize)
			
			// Pre-generate keys to prove
			keysToProve := make([]types.Key, 100)
			for i := range keysToProve {
				keyData := make([]byte, 32)
				rand.Read(keyData)
				key := types.Key(keyData)
				_, err := tree.Put(key, []byte("prove-me"))
				require.NoError(b, err)
				keysToProve[i] = key
			}
			
			version := tree.GetLatestVersion()
			reader, err := tree.Reader(version)
			require.NoError(b, err)
			defer reader.Close()
			
			generator := proof.NewGenerator(reader)
			verifier := proof.NewVerifier()
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				key := keysToProve[i%len(keysToProve)]
				
				// Generate proof
				proof, err := generator.Generate(key)
				if err != nil {
					b.Fatal(err)
				}
				
				// Verify proof (simulating light client)
				err = verifier.Verify(proof)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkConcurrentReads measures multi-reader performance
func BenchmarkConcurrentReads(b *testing.B) {
	concurrencyLevels := []int{1, 4, 16, 64}
	tree := setupBenchmarkTree(b, 1_000_000)
	
	// Pre-generate keys
	keys := make([]types.Key, 10000)
	for i := range keys {
		keyData := make([]byte, 32)
		rand.Read(keyData)
		key := types.Key(keyData)
		_, err := tree.Put(key, []byte("concurrent-value"))
		require.NoError(b, err)
		keys[i] = key
	}
	
	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.ResetTimer()
			b.ReportAllocs()
			
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := keys[i%len(keys)]
					version := tree.GetLatestVersion()
					_, err := tree.GetAtVersion(version, key)
					if err != nil {
						b.Fatal(err)
					}
					i++
				}
			})
			
			opsPerSec := float64(b.N) / b.Elapsed().Seconds()
			b.ReportMetric(opsPerSec, "reads/sec")
		})
	}
}

// BenchmarkMemoryEfficiency tracks memory usage patterns
func BenchmarkMemoryEfficiency(b *testing.B) {
	scenarios := []struct {
		name       string
		numKeys    int
		valueSize  int
	}{
		{"Small_Values", 10000, 100},
		{"Medium_Values", 10000, 1000},
		{"Large_Values", 10000, 10000},
	}
	
	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			store := memory.NewStorage()
			keyEncoder := storage.NewDefaultKeyEncoder()
			tree, err := tree.NewTree(store, keyEncoder, tree.DefaultTreeConfig())
			require.NoError(b, err)
			
			value := make([]byte, scenario.valueSize)
			rand.Read(value)
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N && i < scenario.numKeys; i++ {
				keyData := make([]byte, 32)
				rand.Read(keyData)
				key := types.Key(keyData)
				
				_, err := tree.Put(key, value)
				if err != nil {
					b.Fatal(err)
				}
			}
			
			// Report memory metrics
			if b.N > 0 {
				allocsPerKey := float64(scenario.numKeys) / float64(b.N)
				b.ReportMetric(allocsPerKey, "allocs/key")
			}
		})
	}
}