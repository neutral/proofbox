package main

import (
	"fmt"
	"os"

	"github.com/neutral/proofbox/docs/examples/utils"

	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
)

// This example demonstrates the internal structure of UpdateBatch
func main() {
	// Create temporary database
	store, cleanup, err := utils.CreateTempStorage("batch-internals")
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

	utils.PrintSection("Update Batch Internals Example")

	// Create initial data
	utils.PrintSubSection("Setting Up Initial Tree")

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
			utils.PrintError("Failed to put %s: %v", kv.key, err)
		} else {
			utils.PrintSuccess("v%d: %s = %s", v, kv.key, kv.value)
			initialVersion = v
		}
	}

	// Create a version with multiple operations
	utils.PrintSection("Creating Version with Multiple Operations")

	newVersion, err := jmt.BeginVersion()
	if err != nil {
		utils.PrintError("Failed to begin version: %v", err)
		os.Exit(1)
	}

	utils.PrintKeyValue("New version", newVersion)
	utils.PrintKeyValue("Parent version", initialVersion)

	// Perform various operations
	utils.PrintSubSection("Adding Operations")

	// Update existing key
	err = jmt.PutVersioned(newVersion, types.KeyHash([]byte("key-001")), []byte("updated-value-1"))
	if err == nil {
		utils.PrintBatchOperation("UPDATE", []byte("key-001"), []byte("updated-value-1"))
	}

	// Add new keys
	err = jmt.PutVersioned(newVersion, types.KeyHash([]byte("key-004")), []byte("new-value-4"))
	if err == nil {
		utils.PrintBatchOperation("ADD", []byte("key-004"), []byte("new-value-4"))
	}

	err = jmt.PutVersioned(newVersion, types.KeyHash([]byte("key-005")), []byte("new-value-5"))
	if err == nil {
		utils.PrintBatchOperation("ADD", []byte("key-005"), []byte("new-value-5"))
	}

	// Delete a key
	err = jmt.DeleteVersioned(newVersion, types.KeyHash([]byte("key-002")))
	if err == nil {
		utils.PrintBatchOperation("DELETE", []byte("key-002"), nil)
	}

	// Get the pending version info (internal access for demonstration)
	// In real usage, this would be internal to the tree
	pending := jmt.GetVersionManager().GetPending(newVersion)
	if pending == nil {
		utils.PrintError("No pending version found")
		os.Exit(1)
	}

	// Build the update batch
	utils.PrintSection("Building Update Batch")

	batch, err := pending.GetUpdater().BuildUpdateBatch()
	if err != nil {
		utils.PrintError("Failed to build batch: %v", err)
		os.Exit(1)
	}

	// Display batch contents
	utils.PrintSubSection("Batch Structure")

	utils.PrintKeyValue("Root Hash", fmt.Sprintf("%x", batch.NewRootHash[:16])+"...")
	utils.PrintKeyValue("New Nodes Count", len(batch.NewNodes))
	utils.PrintKeyValue("Stale Nodes Count", len(batch.StaleNodes))

	// Show new nodes
	utils.PrintSubSection("New Nodes in Batch")
	utils.PrintInfo("These nodes will be written to storage:")

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
		fmt.Printf("  %s[%s]%s v%d:%s\n", utils.ColorPurple, nodeType, utils.ColorReset,
			nodeWrite.Key.Version, nibbleStr)
	}

	utils.PrintKeyValue("  Leaf nodes", nodeTypes["leaf"])
	utils.PrintKeyValue("  Internal nodes", nodeTypes["internal"])

	// Show stale nodes
	if len(batch.StaleNodes) > 0 {
		utils.PrintSubSection("Stale Nodes")
		utils.PrintInfo("These nodes are no longer referenced:")

		for _, staleKey := range batch.StaleNodes {
			nibbleStr := "root"
			if len(staleKey.NibblePath.Nibbles) > 0 {
				displayLen := 4
				if len(staleKey.NibblePath.Nibbles) < displayLen {
					displayLen = len(staleKey.NibblePath.Nibbles)
				}
				nibbleStr = fmt.Sprintf("%x", staleKey.NibblePath.Nibbles[:displayLen])
			}
			fmt.Printf("  %sv%d:%s%s\n", utils.ColorGray, staleKey.Version,
				nibbleStr, utils.ColorReset)
		}
	}

	// Demonstrate batch optimization
	utils.PrintSection("Batch Optimization")

	optimizer := tree.NewBatchOptimizer(tree.DefaultBatchOptimizerConfig())

	// Get batch statistics
	stats, err := optimizer.GetBatchStats(batch)
	if err != nil {
		utils.PrintError("Failed to get batch stats: %v", err)
	} else {
		utils.PrintSubSection("Batch Statistics")
		utils.PrintKeyValue("Total Nodes", stats.TotalNodes)
		utils.PrintKeyValue("Leaf Nodes", stats.LeafNodes)
		utils.PrintKeyValue("Internal Nodes", stats.InternalNodes)
		utils.PrintKeyValue("Stale Nodes", stats.StaleNodes)

		if stats.UncompressedSize > 0 {
			utils.PrintKeyValue("Uncompressed Size", utils.FormatBytes(stats.UncompressedSize))
			utils.PrintKeyValue("Compressed Size", utils.FormatBytes(stats.CompressedSize))
			utils.PrintKeyValue("Compression Ratio", fmt.Sprintf("%.1f%%", stats.CompressionRatio*100))
		}
	}

	// Optimize the batch
	optimizedBatch, err := optimizer.OptimizeBatch(batch)
	if err != nil {
		utils.PrintError("Failed to optimize batch: %v", err)
	} else {
		utils.PrintSubSection("After Optimization")
		utils.PrintKeyValue("New Nodes", len(optimizedBatch.NewNodes))
		utils.PrintKeyValue("Stale Nodes", len(optimizedBatch.StaleNodes))

		reduction := (1.0 - float64(len(optimizedBatch.StaleNodes))/float64(len(batch.StaleNodes))) * 100
		if reduction > 0 {
			utils.PrintSuccess("Stale nodes reduced by %.0f%%", reduction)
		}
	}

	// Commit the version
	utils.PrintSection("Committing Batch")

	err = jmt.CommitVersion(newVersion)
	if err != nil {
		utils.PrintError("Failed to commit: %v", err)
		os.Exit(1)
	}

	utils.PrintSuccess("Batch committed successfully!")

	// Verify the changes
	utils.PrintSection("Verification")

	keysToCheck := []string{"key-001", "key-002", "key-003", "key-004", "key-005"}
	for _, k := range keysToCheck {
		key := types.KeyHash([]byte(k))
		value, err := jmt.GetAtVersion(newVersion, key)
		if err != nil {
			utils.PrintInfo("%s: (deleted)", k)
		} else {
			utils.PrintSuccess("%s: %s", k, value)
		}
	}

	// Summary
	utils.PrintSection("Update Batch Summary")
	utils.PrintInfo("• UpdateBatch tracks all changes for a version")
	utils.PrintInfo("• NewNodes contains all nodes created/modified")
	utils.PrintInfo("• StaleNodes lists nodes no longer referenced")
	utils.PrintInfo("• Path cloning creates new internal nodes along update paths")
	utils.PrintInfo("• Optimization can reduce redundancy in batches")
	utils.PrintInfo("• All changes are written atomically to storage")
}

// To run this example:
// cd docs/examples/batch_internals && go run .
// Or: cd docs/examples cd examples && ./run_example.shcd examples && ./run_example.sh ./run_example.sh batch_internals
