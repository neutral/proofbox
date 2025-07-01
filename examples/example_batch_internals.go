package main

import (
	"fmt"
	"os"

	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
)

// This example demonstrates the internal structure of UpdateBatch
func main() {
	// Create temporary database
	store, cleanup, err := CreateTempStorage("batch-internals")
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	// Create tree
	jmt, err := tree.NewTree(store, nil, tree.DefaultTreeConfig())
	if err != nil {
		fmt.Printf("Failed to create tree: %v\n", err)
		os.Exit(1)
	}

	PrintSection("Update Batch Internals Example")
	
	// Create initial data
	PrintSubSection("Setting Up Initial Tree")
	
	initialKeys := []struct {
		key   string
		value string
	}{
		{"key-001", "initial-value-1"},
		{"key-002", "initial-value-2"},
		{"key-003", "initial-value-3"},
	}
	
	var initialVersion types.Version
	for _, kv := range initialKeys {
		v, err := jmt.Put(types.KeyHash([]byte(kv.key)), []byte(kv.value))
		if err != nil {
			PrintError("Failed to put %s: %v", kv.key, err)
		} else {
			PrintSuccess("v%d: %s = %s", v, kv.key, kv.value)
			initialVersion = v
		}
	}
	
	// Create a version with multiple operations
	PrintSection("Creating Version with Multiple Operations")
	
	newVersion, err := jmt.BeginVersion()
	if err != nil {
		PrintError("Failed to begin version: %v", err)
		os.Exit(1)
	}
	
	PrintKeyValue("New version", newVersion)
	PrintKeyValue("Parent version", initialVersion)
	
	// Perform various operations
	PrintSubSection("Adding Operations")
	
	// Update existing key
	err = jmt.PutVersioned(newVersion, types.KeyHash([]byte("key-001")), []byte("updated-value-1"))
	if err == nil {
		PrintBatchOperation("UPDATE", []byte("key-001"), []byte("updated-value-1"))
	}
	
	// Add new keys
	err = jmt.PutVersioned(newVersion, types.KeyHash([]byte("key-004")), []byte("new-value-4"))
	if err == nil {
		PrintBatchOperation("ADD", []byte("key-004"), []byte("new-value-4"))
	}
	
	err = jmt.PutVersioned(newVersion, types.KeyHash([]byte("key-005")), []byte("new-value-5"))
	if err == nil {
		PrintBatchOperation("ADD", []byte("key-005"), []byte("new-value-5"))
	}
	
	// Delete a key
	err = jmt.DeleteVersioned(newVersion, types.KeyHash([]byte("key-002")))
	if err == nil {
		PrintBatchOperation("DELETE", []byte("key-002"), nil)
	}
	
	// Get the pending version info (internal access for demonstration)
	// In real usage, this would be internal to the tree
	pending := jmt.GetVersionManager().GetPending(newVersion)
	if pending == nil {
		PrintError("No pending version found")
		os.Exit(1)
	}
	
	// Build the update batch
	PrintSection("Building Update Batch")
	
	batch, err := pending.GetUpdater().BuildUpdateBatch()
	if err != nil {
		PrintError("Failed to build batch: %v", err)
		os.Exit(1)
	}
	
	// Display batch contents
	PrintSubSection("Batch Structure")
	
	PrintKeyValue("Root Hash", fmt.Sprintf("%x", batch.NewRootHash[:16])+"...")
	PrintKeyValue("New Nodes Count", len(batch.NewNodes))
	PrintKeyValue("Stale Nodes Count", len(batch.StaleNodes))
	
	// Show new nodes
	PrintSubSection("New Nodes in Batch")
	PrintInfo("These nodes will be written to storage:")
	
	nodeTypes := map[string]int{"leaf": 0, "internal": 0}
	for _, nodeWrite := range batch.NewNodes {
		nodeType := "unknown"
		if types.IsLeaf(nodeWrite.Node) {
			nodeType = "leaf"
			nodeTypes["leaf"]++
		} else if types.IsInternal(nodeWrite.Node) {
			nodeType = "internal"
			nodeTypes["internal"]++
		}
		
		nibbleStr := "root"
		if len(nodeWrite.Key.NibblePath.Nibbles) > 0 {
			displayLen := 4
			if len(nodeWrite.Key.NibblePath.Nibbles) < displayLen {
				displayLen = len(nodeWrite.Key.NibblePath.Nibbles)
			}
			nibbleStr = fmt.Sprintf("%x", nodeWrite.Key.NibblePath.Nibbles[:displayLen])
		}
		fmt.Printf("  %s[%s]%s v%d:%s\n", ColorPurple, nodeType, ColorReset, 
			nodeWrite.Key.Version, nibbleStr)
	}
	
	PrintKeyValue("  Leaf nodes", nodeTypes["leaf"])
	PrintKeyValue("  Internal nodes", nodeTypes["internal"])
	
	// Show stale nodes
	if len(batch.StaleNodes) > 0 {
		PrintSubSection("Stale Nodes")
		PrintInfo("These nodes are no longer referenced:")
		
		for _, staleKey := range batch.StaleNodes {
			nibbleStr := "root"
			if len(staleKey.NibblePath.Nibbles) > 0 {
				displayLen := 4
				if len(staleKey.NibblePath.Nibbles) < displayLen {
					displayLen = len(staleKey.NibblePath.Nibbles)
				}
				nibbleStr = fmt.Sprintf("%x", staleKey.NibblePath.Nibbles[:displayLen])
			}
			fmt.Printf("  %sv%d:%s%s\n", ColorGray, staleKey.Version, 
				nibbleStr, ColorReset)
		}
	}
	
	// Demonstrate batch optimization
	PrintSection("Batch Optimization")
	
	optimizer := tree.NewBatchOptimizer(tree.DefaultBatchOptimizerConfig())
	
	// Get batch statistics
	stats, err := optimizer.GetBatchStats(batch)
	if err != nil {
		PrintError("Failed to get batch stats: %v", err)
	} else {
		PrintSubSection("Batch Statistics")
		PrintKeyValue("Total Nodes", stats.TotalNodes)
		PrintKeyValue("Leaf Nodes", stats.LeafNodes)
		PrintKeyValue("Internal Nodes", stats.InternalNodes)
		PrintKeyValue("Stale Nodes", stats.StaleNodes)
		
		if stats.UncompressedSize > 0 {
			PrintKeyValue("Uncompressed Size", FormatBytes(stats.UncompressedSize))
			PrintKeyValue("Compressed Size", FormatBytes(stats.CompressedSize))
			PrintKeyValue("Compression Ratio", fmt.Sprintf("%.1f%%", stats.CompressionRatio*100))
		}
	}
	
	// Optimize the batch
	optimizedBatch, err := optimizer.OptimizeBatch(batch)
	if err != nil {
		PrintError("Failed to optimize batch: %v", err)
	} else {
		PrintSubSection("After Optimization")
		PrintKeyValue("New Nodes", len(optimizedBatch.NewNodes))
		PrintKeyValue("Stale Nodes", len(optimizedBatch.StaleNodes))
		
		reduction := (1.0 - float64(len(optimizedBatch.StaleNodes))/float64(len(batch.StaleNodes))) * 100
		if reduction > 0 {
			PrintSuccess("Stale nodes reduced by %.0f%%", reduction)
		}
	}
	
	// Commit the version
	PrintSection("Committing Batch")
	
	err = jmt.CommitVersion(newVersion)
	if err != nil {
		PrintError("Failed to commit: %v", err)
		os.Exit(1)
	}
	
	PrintSuccess("Batch committed successfully!")
	
	// Verify the changes
	PrintSection("Verification")
	
	keysToCheck := []string{"key-001", "key-002", "key-003", "key-004", "key-005"}
	for _, k := range keysToCheck {
		key := types.KeyHash([]byte(k))
		value, err := jmt.GetAtVersion(newVersion, key)
		if err != nil {
			PrintInfo("%s: (deleted)", k)
		} else {
			PrintSuccess("%s: %s", k, value)
		}
	}
	
	// Summary
	PrintSection("Update Batch Summary")
	PrintInfo("• UpdateBatch tracks all changes for a version")
	PrintInfo("• NewNodes contains all nodes created/modified")
	PrintInfo("• StaleNodes lists nodes no longer referenced")
	PrintInfo("• Path cloning creates new internal nodes along update paths")
	PrintInfo("• Optimization can reduce redundancy in batches")
	PrintInfo("• All changes are written atomically to storage")
}

// To run this example:
// cd examples && go run example_batch_internals.go examples_utils.go
// Or: cd examples && ./run_example.sh batch_internals