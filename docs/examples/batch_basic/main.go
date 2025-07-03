package main

import (
	"fmt"
	"os"

	"github.com/neutral/proofbox/docs/examples/utils"
	"github.com/neutral/proofbox/pkg/types"
)

// This example demonstrates basic batch transaction operations
func main() {
	// Create temporary database
	store, cleanup, err := utils.CreateTempStorage("batch-basic")
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

	utils.PrintSection("Basic Batch Transaction Example")

	// Show initial state
	utils.PrintTreeStats(jmt)
	initialVersion := jmt.GetLatestVersion()

	// Create a batch transaction
	utils.PrintSection("Creating Batch Transaction")
	batch := jmt.NewBatchTransaction()

	// Add multiple operations
	utils.PrintInfo("Adding operations to batch:")

	// Put operations
	batch.BatchPut(types.KeyHash([]byte("fig")), []byte("fruit"))
	utils.PrintBatchOperation("PUT", []byte("fig"), []byte("fruit"))

	batch.BatchPut(types.KeyHash([]byte("grape")), []byte("fruit"))
	utils.PrintBatchOperation("PUT", []byte("grape"), []byte("fruit"))

	batch.BatchPut(types.KeyHash([]byte("horse")), []byte("animal"))
	utils.PrintBatchOperation("PUT", []byte("horse"), []byte("animal"))

	// Update existing key
	batch.BatchPut(types.KeyHash([]byte("carrot")), []byte("root vegetable"))
	utils.PrintBatchOperation("PUT", []byte("carrot"), []byte("root vegetable"))

	// Delete operation
	batch.BatchDelete(types.KeyHash([]byte("banana")))
	utils.PrintBatchOperation("DELETE", []byte("banana"), nil)

	utils.PrintKeyValue("Total operations in batch", batch.Size())

	// Execute the batch
	utils.PrintSection("Executing Batch")
	newVersion, err := batch.Execute()
	if utils.CheckError(err, "Failed to execute batch") {
		os.Exit(1)
	}

	utils.PrintSuccess("Batch executed successfully!")
	utils.PrintKeyValue("New version", newVersion)
	utils.PrintKeyValue("Version increment", newVersion-initialVersion)

	// Show what changed
	keysToCheck := []string{"apple", "banana", "carrot", "dog", "eagle", "fig", "grape", "horse"}
	utils.PrintVersionComparison(jmt, initialVersion, newVersion, keysToCheck)

	// Demonstrate batch atomicity
	utils.PrintSection("Demonstrating Batch Atomicity")

	// Create another batch with an operation that will succeed and one that might fail
	batch2 := jmt.NewBatchTransaction()

	utils.PrintInfo("Creating batch with multiple operations:")
	batch2.BatchPut(types.KeyHash([]byte("iguana")), []byte("reptile"))
	utils.PrintBatchOperation("PUT", []byte("iguana"), []byte("reptile"))

	batch2.BatchPut(types.KeyHash([]byte("jaguar")), []byte("cat"))
	utils.PrintBatchOperation("PUT", []byte("jaguar"), []byte("cat"))

	// Execute second batch
	version2, err := batch2.Execute()
	if utils.CheckError(err, "Failed to execute batch 2") {
		os.Exit(1)
	}

	utils.PrintSuccess("Second batch executed successfully!")
	utils.PrintKeyValue("Final version", version2)

	// Show final state
	utils.PrintSection("Final Tree State")
	utils.PrintTreeStats(jmt)

	// Verify all operations were applied
	utils.PrintSubSection("Verifying Final State")
	verifyKeys := map[string]string{
		"apple":  "fruit",          // unchanged
		"banana": "",               // deleted
		"carrot": "root vegetable", // updated
		"dog":    "animal",         // unchanged
		"eagle":  "bird",           // unchanged
		"fig":    "fruit",          // added
		"grape":  "fruit",          // added
		"horse":  "animal",         // added
		"iguana": "reptile",        // added in batch 2
		"jaguar": "cat",            // added in batch 2
	}

	allCorrect := true
	for k, expectedValue := range verifyKeys {
		key := types.KeyHash([]byte(k))
		value, err := jmt.GetAtVersion(version2, key)

		if expectedValue == "" {
			if err == nil && value != nil {
				utils.PrintError("%s should be deleted but has value: %s", k, value)
				allCorrect = false
			} else {
				utils.PrintSuccess("%s correctly deleted", k)
			}
		} else {
			if err != nil || string(value) != expectedValue {
				utils.PrintError("%s has incorrect value: got %s, want %s", k, value, expectedValue)
				allCorrect = false
			} else {
				utils.PrintSuccess("%s = %s ✓", k, value)
			}
		}
	}

	if allCorrect {
		utils.PrintSuccess("\nAll batch operations completed successfully!")
	}
}

// To run this example:
// cd docs/examples/batch_basic && go run .
// Or: cd docs/examples cd examples && ./run_example.shcd examples && ./run_example.sh ./run_example.sh batch_basic
