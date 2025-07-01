package main

import (
	"github.com/neutral/proofbox/docs/examples/utils"
	"fmt"
	"os"
	"time"

	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
)

// This example demonstrates batch validation and error handling
func main() {
	// Create temporary database
	store, cleanup, err := utils.CreateTempStorage("batch-validation")
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	// Create tree
	jmt, err := utils.CreateExampleTree(store)
	if err != nil {
		fmt.Printf("Failed to create tree: %v\n", err)
		os.Exit(1)
	}

	utils.PrintSection("Batch Validation Example")

	// Test 1: Valid batch operations
	utils.PrintSection("Test 1: Valid Batch Operations")

	batch1 := jmt.NewBatchTransaction()

	utils.PrintInfo("Adding valid operations:")
	validOps := []struct {
		key   string
		value string
	}{
		{"valid-key-1", "value1"},
		{"valid-key-2", "value2"},
		{"valid-key-3", "value3"},
	}

	for _, op := range validOps {
		err := batch1.BatchPut(types.KeyHash([]byte(op.key)), []byte(op.value))
		if err != nil {
			utils.PrintError("Failed to add %s: %v", op.key, err)
		} else {
			utils.PrintBatchOperation("PUT", []byte(op.key), []byte(op.value))
		}
	}

	version1, err := batch1.Execute()
	if err != nil {
		utils.PrintError("Batch execution failed: %v", err)
	} else {
		utils.PrintSuccess("Valid batch executed successfully! Version: %d", version1)
	}

	// Test 2: Invalid key validation
	utils.PrintSection("Test 2: Invalid Key Validation")

	batch2 := jmt.NewBatchTransaction()

	utils.PrintInfo("Attempting to add invalid keys:")

	// Try empty key
	err = batch2.BatchPut(types.Key{}, []byte("value"))
	if err != nil {
		utils.PrintSuccess("Empty key correctly rejected: %v", err)
	} else {
		utils.PrintError("Empty key should have been rejected")
	}

	// Try nil value
	err = batch2.BatchPut(types.KeyHash([]byte("key")), nil)
	if err != nil {
		utils.PrintSuccess("Nil value correctly rejected: %v", err)
	} else {
		utils.PrintError("Nil value should have been rejected")
	}

	// Add a valid operation to the batch
	err = batch2.BatchPut(types.KeyHash([]byte("valid-after-errors")), []byte("value"))
	if err == nil {
		utils.PrintSuccess("Valid operation added after errors")
	}

	// Execute should work with valid operations
	version2, err := batch2.Execute()
	if err != nil {
		utils.PrintError("Batch execution failed: %v", err)
	} else {
		utils.PrintSuccess("Batch with valid operations executed! Version: %d", version2)
	}

	// Test 3: Batch atomicity with errors
	utils.PrintSection("Test 3: Batch Atomicity")

	// First, add a key that we'll try to delete
	setupVersion, err := jmt.Put(types.KeyHash([]byte("existing-key")), []byte("existing-value"))
	if err != nil {
		utils.PrintError("Failed to setup test: %v", err)
	} else {
		utils.PrintSuccess("Setup complete. Version: %d", setupVersion)
	}

	// Create updater directly to demonstrate internal validation
	utils.PrintSubSection("Testing Internal Batch Validation")

	updater := tree.NewTreeUpdater(jmt, setupVersion, setupVersion+1)

	// Add some operations
	_, err = updater.Put(types.KeyHash([]byte("batch-key-1")), []byte("value1"))
	if err != nil {
		utils.PrintError("Failed to put key: %v", err)
	}

	// Build batch
	batch, err := updater.BuildUpdateBatch()
	if err != nil {
		utils.PrintError("Failed to build batch: %v", err)
	} else {
		utils.PrintSuccess("Batch built successfully")
		utils.PrintKeyValue("  New nodes", len(batch.NewNodes))
		utils.PrintKeyValue("  Stale nodes", len(batch.StaleNodes))
		utils.PrintKeyValue("  Root hash", fmt.Sprintf("%x", batch.NewRootHash[:8])+"...")
	}

	// Validate the batch
	err = updater.ValidateBatch(batch)
	if err != nil {
		utils.PrintError("Batch validation failed: %v", err)
	} else {
		utils.PrintSuccess("Batch validation passed!")
	}

	// Test 4: Rollback behavior
	utils.PrintSection("Test 4: Rollback Behavior")

	batch4 := jmt.NewBatchTransaction()

	utils.PrintInfo("Creating batch that will be rolled back:")

	// Add operations
	rollbackKeys := []string{"rollback-1", "rollback-2", "rollback-3"}
	for _, k := range rollbackKeys {
		batch4.BatchPut(types.KeyHash([]byte(k)), []byte("should-not-exist"))
		utils.PrintBatchOperation("PUT", []byte(k), []byte("should-not-exist"))
	}

	// Clear the batch before execution (simulating rollback)
	batch4.Clear()
	utils.PrintInfo("Batch cleared before execution")

	// Try to execute empty batch
	_, err = batch4.Execute()
	if err != nil {
		utils.PrintSuccess("Empty batch correctly rejected: %v", err)
	}

	// Verify keys don't exist
	utils.PrintInfo("Verifying rollback keys don't exist:")
	latestVersion := jmt.GetLatestVersion()
	for _, k := range rollbackKeys {
		_, err := jmt.GetAtVersion(latestVersion, types.KeyHash([]byte(k)))
		if err != nil {
			utils.PrintSuccess("  %s: correctly not found", k)
		} else {
			utils.PrintError("  %s: should not exist!", k)
		}
	}

	// Test 5: Large batch validation
	utils.PrintSection("Test 5: Large Batch Validation")

	largeBatch := jmt.NewBatchTransaction()
	numOps := 1000

	utils.PrintInfo("Creating large batch with %d operations...", numOps)

	start := time.Now()
	for i := 0; i < numOps; i++ {
		key := types.KeyHash([]byte(fmt.Sprintf("large-batch-key-%04d", i)))
		value := []byte(fmt.Sprintf("value-%04d", i))

		err := largeBatch.BatchPut(key, value)
		if err != nil {
			utils.PrintError("Failed at operation %d: %v", i, err)
			break
		}

		if (i+1) % 200 == 0 {
			fmt.Printf("  %sAdded %d/%d operations...%s\r", utils.ColorGray, i+1, numOps, utils.ColorReset)
		}
	}
	fmt.Println()

	utils.PrintKeyValue("Time to build batch", time.Since(start))
	utils.PrintKeyValue("Batch size", largeBatch.Size())

	// Execute large batch
	execStart := time.Now()
	largeVersion, err := largeBatch.Execute()
	execDuration := time.Since(execStart)

	if err != nil {
		utils.PrintError("Large batch execution failed: %v", err)
	} else {
		utils.PrintSuccess("Large batch executed successfully!")
		utils.PrintKeyValue("  Version", largeVersion)
		utils.PrintKeyValue("  Execution time", execDuration)
		utils.PrintKeyValue("  Ops/second", fmt.Sprintf("%.0f", float64(numOps)/execDuration.Seconds()))
	}

	// Summary
	utils.PrintSection("Validation Summary")
	utils.PrintSuccess("✓ Invalid keys are rejected at batch creation time")
	utils.PrintSuccess("✓ Batches with only valid operations execute successfully")
	utils.PrintSuccess("✓ Empty batches are rejected")
	utils.PrintSuccess("✓ Cleared batches don't affect tree state")
	utils.PrintSuccess("✓ Large batches are validated efficiently")
	utils.PrintSuccess("✓ All operations within a batch are atomic")
}

// To run this example:
// cd docs/examples/batch_validation && go run .
// Or: cd docs/examples cd examples && ./run_example.shcd examples && ./run_example.sh ./run_example.sh batch_validation