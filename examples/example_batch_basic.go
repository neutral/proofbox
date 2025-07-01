package main

import (
	"fmt"
	"os"

	"github.com/neutral/proofbox/pkg/types"
)

// This example demonstrates basic batch transaction operations
func main() {
	// Create temporary database
	store, cleanup, err := CreateTempStorage("batch-basic")
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	// Create tree with initial data
	jmt, err := CreateExampleTree(store)
	if err != nil {
		fmt.Printf("Failed to create tree: %v\n", err)
		os.Exit(1)
	}

	PrintSection("Basic Batch Transaction Example")
	
	// Show initial state
	PrintTreeStats(jmt)
	initialVersion := jmt.GetLatestVersion()

	// Create a batch transaction
	PrintSection("Creating Batch Transaction")
	batch := jmt.NewBatchTransaction()
	
	// Add multiple operations
	PrintInfo("Adding operations to batch:")
	
	// Put operations
	batch.BatchPut(types.KeyHash([]byte("fig")), []byte("fruit"))
	PrintBatchOperation("PUT", []byte("fig"), []byte("fruit"))
	
	batch.BatchPut(types.KeyHash([]byte("grape")), []byte("fruit"))
	PrintBatchOperation("PUT", []byte("grape"), []byte("fruit"))
	
	batch.BatchPut(types.KeyHash([]byte("horse")), []byte("animal"))
	PrintBatchOperation("PUT", []byte("horse"), []byte("animal"))
	
	// Update existing key
	batch.BatchPut(types.KeyHash([]byte("carrot")), []byte("root vegetable"))
	PrintBatchOperation("PUT", []byte("carrot"), []byte("root vegetable"))
	
	// Delete operation
	batch.BatchDelete(types.KeyHash([]byte("banana")))
	PrintBatchOperation("DELETE", []byte("banana"), nil)
	
	PrintKeyValue("Total operations in batch", batch.Size())
	
	// Execute the batch
	PrintSection("Executing Batch")
	newVersion, err := batch.Execute()
	if CheckError(err, "Failed to execute batch") {
		os.Exit(1)
	}
	
	PrintSuccess("Batch executed successfully!")
	PrintKeyValue("New version", newVersion)
	PrintKeyValue("Version increment", newVersion - initialVersion)
	
	// Show what changed
	keysToCheck := []string{"apple", "banana", "carrot", "dog", "eagle", "fig", "grape", "horse"}
	PrintVersionComparison(jmt, initialVersion, newVersion, keysToCheck)
	
	// Demonstrate batch atomicity
	PrintSection("Demonstrating Batch Atomicity")
	
	// Create another batch with an operation that will succeed and one that might fail
	batch2 := jmt.NewBatchTransaction()
	
	PrintInfo("Creating batch with multiple operations:")
	batch2.BatchPut(types.KeyHash([]byte("iguana")), []byte("reptile"))
	PrintBatchOperation("PUT", []byte("iguana"), []byte("reptile"))
	
	batch2.BatchPut(types.KeyHash([]byte("jaguar")), []byte("cat"))
	PrintBatchOperation("PUT", []byte("jaguar"), []byte("cat"))
	
	// Execute second batch
	version2, err := batch2.Execute()
	if CheckError(err, "Failed to execute batch 2") {
		os.Exit(1)
	}
	
	PrintSuccess("Second batch executed successfully!")
	PrintKeyValue("Final version", version2)
	
	// Show final state
	PrintSection("Final Tree State")
	PrintTreeStats(jmt)
	
	// Verify all operations were applied
	PrintSubSection("Verifying Final State")
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
				PrintError("%s should be deleted but has value: %s", k, value)
				allCorrect = false
			} else {
				PrintSuccess("%s correctly deleted", k)
			}
		} else {
			if err != nil || string(value) != expectedValue {
				PrintError("%s has incorrect value: got %s, want %s", k, value, expectedValue)
				allCorrect = false
			} else {
				PrintSuccess("%s = %s ✓", k, value)
			}
		}
	}
	
	if allCorrect {
		PrintSuccess("\nAll batch operations completed successfully!")
	}
}

// To run this example:
// cd examples && go run example_batch_basic.go examples_utils.go
// Or: cd examples && ./run_example.sh batch_basic