package main

import (
	"fmt"
	"os"

	"github.com/neutral/proofbox/pkg/types"
)

// This example demonstrates batch deduplication behavior
func main() {
	// Create temporary database
	db, cleanup, err := CreateTempDB("batch-dedup")
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	// Create tree with initial data
	jmt, err := CreateExampleTree(db)
	if err != nil {
		fmt.Printf("Failed to create tree: %v\n", err)
		os.Exit(1)
	}

	PrintSection("Batch Deduplication Example")
	
	initialVersion := jmt.GetLatestVersion()
	PrintKeyValue("Initial version", initialVersion)

	// Demonstrate deduplication
	PrintSection("Creating Batch with Duplicate Operations")
	batch := jmt.NewBatchTransaction()
	
	key := types.KeyHash([]byte("test-key"))
	
	// Add multiple operations for the same key
	PrintInfo("Adding multiple operations for 'test-key':")
	
	batch.BatchPut(key, []byte("value1"))
	PrintBatchOperation("PUT", []byte("test-key"), []byte("value1"))
	
	batch.BatchPut(key, []byte("value2"))
	PrintBatchOperation("PUT", []byte("test-key"), []byte("value2"))
	
	batch.BatchPut(key, []byte("value3"))
	PrintBatchOperation("PUT", []byte("test-key"), []byte("value3"))
	
	batch.BatchDelete(key)
	PrintBatchOperation("DELETE", []byte("test-key"), nil)
	
	batch.BatchPut(key, []byte("final-value"))
	PrintBatchOperation("PUT", []byte("test-key"), []byte("final-value"))
	
	PrintKeyValue("\nTotal operations added", batch.Size())
	
	// Show optimized operations
	PrintSection("Optimized Operations")
	optimized := batch.OptimizeOperations()
	PrintKeyValue("Operations after deduplication", len(optimized))
	
	PrintInfo("Final operation for 'test-key':")
	for _, op := range optimized {
		if op.Type == 0 { // BatchOpPut
			PrintBatchOperation("PUT", op.Key[:], op.Value)
		} else {
			PrintBatchOperation("DELETE", op.Key[:], nil)
		}
	}
	
	// Execute batch
	version, err := batch.Execute()
	if CheckError(err, "Failed to execute batch") {
		os.Exit(1)
	}
	
	PrintSuccess("\nBatch executed with version %d", version)
	
	// Verify final state
	PrintSection("Verification")
	value, err := jmt.GetAtVersion(version, key)
	if err != nil {
		PrintError("Failed to get key: %v", err)
	} else {
		PrintSuccess("Final value for 'test-key': %s", value)
	}
	
	// Demonstrate multiple key deduplication
	PrintSection("Multiple Key Deduplication")
	batch2 := jmt.NewBatchTransaction()
	
	keys := []string{"alpha", "beta", "gamma"}
	
	PrintInfo("Adding interleaved operations:")
	// Interleave operations for different keys
	for i := 1; i <= 3; i++ {
		for _, k := range keys {
			key := types.KeyHash([]byte(k))
			value := []byte(fmt.Sprintf("%s-v%d", k, i))
			batch2.BatchPut(key, value)
			PrintBatchOperation("PUT", []byte(k), value)
		}
	}
	
	// Add some deletes
	batch2.BatchDelete(types.KeyHash([]byte("beta")))
	PrintBatchOperation("DELETE", []byte("beta"), nil)
	
	// Add beta back
	batch2.BatchPut(types.KeyHash([]byte("beta")), []byte("beta-final"))
	PrintBatchOperation("PUT", []byte("beta"), []byte("beta-final"))
	
	PrintKeyValue("\nTotal operations", batch2.Size())
	
	optimized2 := batch2.OptimizeOperations()
	PrintKeyValue("After deduplication", len(optimized2))
	
	// Execute and verify
	version2, err := batch2.Execute()
	if CheckError(err, "Failed to execute batch 2") {
		os.Exit(1)
	}
	
	PrintSuccess("\nBatch executed with version %d", version2)
	
	PrintSubSection("Final Values")
	for _, k := range keys {
		key := types.KeyHash([]byte(k))
		value, err := jmt.GetAtVersion(version2, key)
		if err != nil {
			PrintError("%s: not found", k)
		} else {
			PrintSuccess("%s = %s", k, value)
		}
	}
	
	// Show statistics
	PrintSection("Deduplication Statistics")
	PrintInfo("Batch 1: %d operations → %d after deduplication (%.0f%% reduction)", 
		5, 1, (1.0 - 1.0/5.0) * 100)
	PrintInfo("Batch 2: %d operations → %d after deduplication (%.0f%% reduction)", 
		11, 3, (1.0 - 3.0/11.0) * 100)
	
	PrintSuccess("\nDeduplication ensures only the final operation per key is executed!")
}

// To run this example:
// cd examples && go run example_batch_deduplication.go examples_utils.go
// Or: cd examples && ./run_example.sh batch_deduplication