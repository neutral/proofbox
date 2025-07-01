package main

import (
	"github.com/neutral/proofbox/docs/examples/utils"
	"fmt"
	"os"
	"time"

	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
)

// This example demonstrates parallel batch processing for large batches
func main() {
	// Create temporary database
	store, cleanup, err := utils.CreateTempStorage("batch-parallel")
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	utils.PrintSection("Parallel Batch Processing Example")
	
	// Test both sequential and parallel processing
	runComparison := func(store storage.Storage, useParallel bool, numKeys int) (time.Duration, types.Version) {
		// Create tree with specific config
		config := tree.DefaultTreeConfig()
		config.UseParallelBatching = useParallel
		
		jmt, err := tree.NewTree(store, nil, config)
		if err != nil {
			utils.PrintError("Failed to create tree: %v", err)
			return 0, 0
		}
		
		mode := "Sequential"
		if useParallel {
			mode = "Parallel"
		}
		
		utils.PrintSubSection(fmt.Sprintf("%s Processing (%d keys)", mode, numKeys))
		
		// Create a large batch
		start := time.Now()
		
		version, err := jmt.BeginVersion()
		if err != nil {
			utils.PrintError("Failed to begin version: %v", err)
			return 0, 0
		}
		
		// Add many operations
		for i := 0; i < numKeys; i++ {
			key := types.KeyHash([]byte(fmt.Sprintf("key-%06d", i)))
			value := []byte(fmt.Sprintf("value-%06d-with-some-extra-data-to-make-it-larger", i))
			
			err := jmt.PutVersioned(version, key, value)
			if err != nil {
				utils.PrintError("Failed to put key %d: %v", i, err)
				return 0, 0
			}
			
			// Show progress
			if (i+1) % (numKeys/10) == 0 {
				fmt.Printf("  %sAdded %d/%d keys...%s\r", utils.ColorGray, i+1, numKeys, utils.ColorReset)
			}
		}
		fmt.Println() // Clear progress line
		
		// Commit the batch
		commitStart := time.Now()
		err = jmt.CommitVersion(version)
		if err != nil {
			utils.PrintError("Failed to commit: %v", err)
			return 0, 0
		}
		commitDuration := time.Since(commitStart)
		
		totalDuration := time.Since(start)
		
		utils.PrintSuccess("Completed in %v (commit: %v)", totalDuration, commitDuration)
		
		// Verify a few keys
		sampleKeys := []int{0, numKeys/2, numKeys-1}
		utils.PrintInfo("Verifying sample keys:")
		for _, i := range sampleKeys {
			key := types.KeyHash([]byte(fmt.Sprintf("key-%06d", i)))
			value, err := jmt.GetAtVersion(version, key)
			if err != nil {
				utils.PrintError("  key-%06d: not found", i)
			} else {
				utils.PrintSuccess("  key-%06d: %d bytes", i, len(value))
			}
		}
		
		return totalDuration, version
	}
	
	// Run tests with different batch sizes
	batchSizes := []int{50, 200, 500}
	
	for _, size := range batchSizes {
		utils.PrintSection(fmt.Sprintf("Batch Size: %d Keys", size))
		
		// Note: In the current implementation, parallel processing
		// only activates for batches with 100+ nodes
		threshold := "(below parallel threshold)"
		if size >= 100 {
			threshold = "(above parallel threshold)"
		}
		utils.PrintInfo("Note: %s", threshold)
		
		seqDuration, seqVersion := runComparison(store, false, size)
		parallelDuration, parallelVersion := runComparison(store, true, size)
		
		if seqVersion > 0 && parallelVersion > 0 {
			speedup := float64(seqDuration) / float64(parallelDuration)
			improvement := (1.0 - float64(parallelDuration)/float64(seqDuration)) * 100
			
			utils.PrintSubSection("Performance Comparison")
			utils.PrintKeyValue("Sequential", seqDuration)
			utils.PrintKeyValue("Parallel", parallelDuration)
			if size >= 100 {
				utils.PrintInfo("Speedup: %.2fx (%.1f%% improvement)", speedup, improvement)
			} else {
				utils.PrintInfo("No parallel processing for batches < 100 nodes")
			}
		}
		
		// Clean up and recreate database for next test
		if size < len(batchSizes)-1 {
			cleanup()
			store, cleanup, err = utils.CreateTempStorage(fmt.Sprintf("batch-parallel-%d", size))
			if err != nil {
				utils.PrintError("Failed to create database: %v", err)
				return
			}
		}
	}
	
	utils.PrintSection("Parallel Processing Benefits")
	utils.PrintInfo("• Automatically activates for batches with 100+ nodes")
	utils.PrintInfo("• Distributes hash computation across CPU cores")
	utils.PrintInfo("• Maintains deterministic operation ordering")
	utils.PrintInfo("• Provides performance improvement for large batches")
	utils.PrintInfo("• Falls back to sequential for small batches")
	
	utils.PrintSuccess("\nParallel batch processing demonstration complete!")
}

// To run this example:
// cd docs/examples/batch_parallel && go run .
// Or: cd docs/examples cd examples && ./run_example.shcd examples && ./run_example.sh ./run_example.sh batch_parallel