package main

import (
	"fmt"
	"os"

	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
)

// This example demonstrates batch optimization features
func main() {
	// Create temporary database
	db, cleanup, err := CreateTempDB("batch-optimization")
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	PrintSection("Batch Optimization Example")
	
	// Create tree with optimization enabled
	config := tree.DefaultTreeConfig()
	config.BatchOptimizer = tree.NewBatchOptimizer(tree.BatchOptimizerConfig{
		EnableCompression:   true,
		CompressionLevel:    6,
		EnableDeduplication: true,
		EnableCoalescing:    true,
	})
	
	jmt, err := tree.NewTree(db, config)
	if err != nil {
		PrintError("Failed to create tree: %v", err)
		os.Exit(1)
	}
	
	// Create a batch with many operations
	PrintSection("Creating Large Batch")
	version, err := jmt.BeginVersion()
	if err != nil {
		PrintError("Failed to begin version: %v", err)
		os.Exit(1)
	}
	
	// Add operations that will benefit from optimization
	numOperations := 150
	PrintInfo("Adding %d operations...", numOperations)
	
	// Add some keys multiple times (for deduplication)
	for i := 0; i < numOperations; i++ {
		var key types.Key
		var value []byte
		
		if i % 10 < 3 {
			// Reuse some keys to demonstrate deduplication
			key = types.KeyHash([]byte(fmt.Sprintf("duplicate-key-%d", i%10)))
			value = []byte(fmt.Sprintf("value-iteration-%d", i))
		} else {
			// Unique keys
			key = types.KeyHash([]byte(fmt.Sprintf("unique-key-%d", i)))
			value = []byte(fmt.Sprintf("This is a longer value to demonstrate compression benefits. Iteration: %d", i))
		}
		
		err := jmt.PutVersioned(version, key, value)
		if err != nil {
			PrintError("Failed to put key: %v", err)
			continue
		}
		
		// Show progress
		if (i+1) % (numOperations/5) == 0 {
			fmt.Printf("  %sProgress: %d/%d operations%s\r", ColorGray, i+1, numOperations, ColorReset)
		}
	}
	fmt.Println() // Clear progress line
	
	// Get batch statistics before commit
	PrintSection("Batch Statistics")
	
	// To demonstrate, we'll create our own batch optimizer and test it
	optimizer := tree.NewBatchOptimizer(tree.DefaultBatchOptimizerConfig())
	
	// Create a sample batch for statistics
	sampleBatch := &tree.UpdateBatch{
		NewRootHash: types.Hash{},
		NewNodes:    make(map[string]tree.NodeWrite),
		StaleNodes:  make([]types.NodeKey, 0),
	}
	
	// Add sample nodes
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("node%d", i)
		// Create a proper leaf node for the sample
		leafNode, _ := tree.NewLeafNode(
			types.Key{byte(i)},
			[]byte(fmt.Sprintf("sample-value-%d", i)),
			types.Version(1),
		)
		sampleBatch.NewNodes[key] = tree.NodeWrite{
			Key:  types.NodeKey{Version: 1},
			Node: leafNode,
		}
	}
	
	// Get statistics
	stats, err := optimizer.GetBatchStats(sampleBatch)
	if err != nil {
		PrintError("Failed to get stats: %v", err)
	} else {
		PrintKeyValue("Total Nodes", stats.TotalNodes)
		PrintKeyValue("Leaf Nodes", stats.LeafNodes)
		PrintKeyValue("Internal Nodes", stats.InternalNodes)
		PrintKeyValue("Stale Nodes", stats.StaleNodes)
		
		if stats.CompressionRatio > 0 {
			PrintKeyValue("Uncompressed Size", FormatBytes(stats.UncompressedSize))
			PrintKeyValue("Compressed Size", FormatBytes(stats.CompressedSize))
			PrintKeyValue("Compression Ratio", fmt.Sprintf("%.1f%%", stats.CompressionRatio*100))
		}
	}
	
	// Commit the batch
	PrintSection("Committing Optimized Batch")
	err = jmt.CommitVersion(version)
	if err != nil {
		PrintError("Failed to commit: %v", err)
		os.Exit(1)
	}
	
	PrintSuccess("Batch committed successfully!")
	
	// Demonstrate compression
	PrintSection("Compression Demonstration")
	
	// Create test data
	testBatch := &tree.UpdateBatch{
		NewRootHash: types.Hash{1, 2, 3, 4, 5},
		NewNodes:    make(map[string]tree.NodeWrite),
		StaleNodes:  make([]types.NodeKey, 50),
	}
	
	// Add nodes with repetitive data (compresses well)
	for i := 0; i < 200; i++ {
		key := fmt.Sprintf("compress-test-%d", i)
		// Create a proper leaf node with repetitive data
		leafNode, _ := tree.NewLeafNode(
			types.Key{byte(i % 256)},
			[]byte("This is repetitive data that compresses well. " +
				"The same pattern repeated many times. " +
				"This helps demonstrate compression benefits."),
			types.Version(1),
		)
		testBatch.NewNodes[key] = tree.NodeWrite{
			Key:  types.NodeKey{Version: types.Version(i)},
			Node: leafNode,
		}
	}
	
	// Test compression
	compressed, err := optimizer.CompressBatch(testBatch)
	if err != nil {
		PrintError("Compression failed: %v", err)
	} else {
		uncompressed, _ := optimizer.GetBatchStats(testBatch)
		
		PrintKeyValue("Original size", FormatBytes(uncompressed.UncompressedSize))
		PrintKeyValue("Compressed size", FormatBytes(int64(len(compressed))))
		PrintKeyValue("Compression ratio", fmt.Sprintf("%.1f%%", 
			(1.0 - float64(len(compressed))/float64(uncompressed.UncompressedSize)) * 100))
		
		// Test decompression
		decompressed, err := optimizer.DecompressBatch(compressed)
		if err != nil {
			PrintError("Decompression failed: %v", err)
		} else {
			PrintSuccess("Compression/decompression cycle successful!")
			PrintKeyValue("Nodes after decompression", len(decompressed.NewNodes))
		}
	}
	
	// Show optimization benefits
	PrintSection("Optimization Benefits")
	PrintSuccess("✓ Deduplication removes redundant operations")
	PrintSuccess("✓ Compression reduces storage I/O")
	PrintSuccess("✓ Batch validation ensures integrity")
	PrintSuccess("✓ Atomic commits prevent partial updates")
	
	// Verify some keys
	PrintSection("Verification")
	keysToCheck := []string{"duplicate-key-0", "duplicate-key-1", "unique-key-50"}
	
	for _, k := range keysToCheck {
		key := types.KeyHash([]byte(k))
		value, err := jmt.GetAtVersion(version, key)
		if err != nil {
			PrintError("%s: not found", k)
		} else {
			PrintSuccess("%s: %d bytes stored", k, len(value))
		}
	}
	
	PrintSuccess("\nBatch optimization demonstration complete!")
}

// To run this example:
// cd examples && go run example_batch_optimization.go examples_utils.go
// Or: cd examples && ./run_example.sh batch_optimization