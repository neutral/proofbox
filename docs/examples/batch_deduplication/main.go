package main

import (
	"fmt"
	"os"

	"github.com/neutral/proofbox/docs/examples/utils"

	"github.com/neutral/proofbox/pkg/types"
)

// This example demonstrates batch deduplication behavior
func main() {
	// Create temporary database
	store, cleanup, err := utils.CreateTempStorage("batch-dedup")
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	// Create tree with initial data
	jmt, err := utils.CreateExampleTree(store)
	if err != nil {
		fmt.Printf("Failed to create tree: %v\n", err)
		os.Exit(1)
	}

	utils.PrintSection("Batch Deduplication Example")

	initialVersion := jmt.GetLatestVersion()
	utils.PrintKeyValue("Initial version", initialVersion)

	// Demonstrate deduplication
	utils.PrintSection("Creating Batch with Duplicate Operations")
	batch := jmt.NewBatchTransaction()

	key := types.KeyHash([]byte("test-key"))

	// Add multiple operations for the same key
	utils.PrintInfo("Adding multiple operations for 'test-key':")

	batch.BatchPut(key, []byte("value1"))
	utils.PrintBatchOperation("PUT", []byte("test-key"), []byte("value1"))

	batch.BatchPut(key, []byte("value2"))
	utils.PrintBatchOperation("PUT", []byte("test-key"), []byte("value2"))

	batch.BatchPut(key, []byte("value3"))
	utils.PrintBatchOperation("PUT", []byte("test-key"), []byte("value3"))

	batch.BatchDelete(key)
	utils.PrintBatchOperation("DELETE", []byte("test-key"), nil)

	batch.BatchPut(key, []byte("final-value"))
	utils.PrintBatchOperation("PUT", []byte("test-key"), []byte("final-value"))

	utils.PrintKeyValue("\nTotal operations added", batch.Size())

	// Show optimized operations
	utils.PrintSection("Optimized Operations")
	optimized := batch.OptimizeOperations()
	utils.PrintKeyValue("Operations after deduplication", len(optimized))

	utils.PrintInfo("Final operation for 'test-key':")
	for _, op := range optimized {
		if op.Type == 0 { // BatchOpPut
			utils.PrintBatchOperation("PUT", op.Key[:], op.Value)
		} else {
			utils.PrintBatchOperation("DELETE", op.Key[:], nil)
		}
	}

	// Execute batch
	version, err := batch.Execute()
	if utils.CheckError(err, "Failed to execute batch") {
		os.Exit(1)
	}

	utils.PrintSuccess("\nBatch executed with version %d", version)

	// Verify final state
	utils.PrintSection("Verification")
	value, err := jmt.GetAtVersion(version, key)
	if err != nil {
		utils.PrintError("Failed to get key: %v", err)
	} else {
		utils.PrintSuccess("Final value for 'test-key': %s", value)
	}

	// Demonstrate multiple key deduplication
	utils.PrintSection("Multiple Key Deduplication")
	batch2 := jmt.NewBatchTransaction()

	keys := []string{"alpha", "beta", "gamma"}

	utils.PrintInfo("Adding interleaved operations:")
	// Interleave operations for different keys
	for i := 1; i <= 3; i++ {
		for _, k := range keys {
			key := types.KeyHash([]byte(k))
			value := []byte(fmt.Sprintf("%s-v%d", k, i))
			batch2.BatchPut(key, value)
			utils.PrintBatchOperation("PUT", []byte(k), value)
		}
	}

	// Add some deletes
	batch2.BatchDelete(types.KeyHash([]byte("beta")))
	utils.PrintBatchOperation("DELETE", []byte("beta"), nil)

	// Add beta back
	batch2.BatchPut(types.KeyHash([]byte("beta")), []byte("beta-final"))
	utils.PrintBatchOperation("PUT", []byte("beta"), []byte("beta-final"))

	utils.PrintKeyValue("\nTotal operations", batch2.Size())

	optimized2 := batch2.OptimizeOperations()
	utils.PrintKeyValue("After deduplication", len(optimized2))

	// Execute and verify
	version2, err := batch2.Execute()
	if utils.CheckError(err, "Failed to execute batch 2") {
		os.Exit(1)
	}

	utils.PrintSuccess("\nBatch executed with version %d", version2)

	utils.PrintSubSection("Final Values")
	for _, k := range keys {
		key := types.KeyHash([]byte(k))
		value, err := jmt.GetAtVersion(version2, key)
		if err != nil {
			utils.PrintError("%s: not found", k)
		} else {
			utils.PrintSuccess("%s = %s", k, value)
		}
	}

	// Show statistics
	utils.PrintSection("Deduplication Statistics")
	utils.PrintInfo("Batch 1: %d operations → %d after deduplication (%.0f%% reduction)",
		5, 1, (1.0-1.0/5.0)*100)
	utils.PrintInfo("Batch 2: %d operations → %d after deduplication (%.0f%% reduction)",
		11, 3, (1.0-3.0/11.0)*100)

	utils.PrintSuccess("\nDeduplication ensures only the final operation per key is executed!")
}

// To run this example:
// cd docs/examples/batch_deduplication && go run .
// Or: cd docs/examples cd examples && ./run_example.shcd examples && ./run_example.sh ./run_example.sh batch_deduplication
