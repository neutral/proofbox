package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/neutral/proofbox/pkg/types"
)

// This example demonstrates concurrent batch operations
func main() {
	// Create temporary database
	db, cleanup, err := CreateTempDB("batch-concurrent")
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	// Create tree
	jmt, err := CreateExampleTree(db)
	if err != nil {
		fmt.Printf("Failed to create tree: %v\n", err)
		os.Exit(1)
	}

	PrintSection("Concurrent Batch Operations Example")
	
	initialVersion := jmt.GetLatestVersion()
	PrintKeyValue("Initial version", initialVersion)
	
	// Run concurrent batch operations
	PrintSection("Running Concurrent Batches")
	
	numGoroutines := 5
	opsPerGoroutine := 20
	
	PrintInfo("Starting %d concurrent goroutines", numGoroutines)
	PrintInfo("Each will create a batch with %d operations", opsPerGoroutine)
	
	var wg sync.WaitGroup
	results := make(chan struct {
		id      int
		version types.Version
		err     error
		duration time.Duration
	}, numGoroutines)
	
	// Start goroutines
	startTime := time.Now()
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			start := time.Now()
			
			// Create batch
			batch := jmt.NewBatchTransaction()
			
			// Add operations specific to this goroutine
			for j := 0; j < opsPerGoroutine; j++ {
				key := types.KeyHash([]byte(fmt.Sprintf("g%d-key-%d", goroutineID, j)))
				value := []byte(fmt.Sprintf("goroutine-%d-value-%d", goroutineID, j))
				
				err := batch.BatchPut(key, value)
				if err != nil {
					results <- struct {
						id      int
						version types.Version
						err     error
						duration time.Duration
					}{goroutineID, 0, err, time.Since(start)}
					return
				}
			}
			
			// Execute batch
			version, err := batch.Execute()
			
			results <- struct {
				id      int
				version types.Version
				err     error
				duration time.Duration
			}{goroutineID, version, err, time.Since(start)}
			
		}(i)
	}
	
	// Wait for completion
	wg.Wait()
	close(results)
	
	totalDuration := time.Since(startTime)
	
	// Collect results
	PrintSubSection("Results")
	
	successCount := 0
	var versions []types.Version
	
	for result := range results {
		if result.err != nil {
			PrintError("Goroutine %d: Failed after %v - %v", 
				result.id, result.duration, result.err)
		} else {
			PrintSuccess("Goroutine %d: Version %d (completed in %v)", 
				result.id, result.version, result.duration)
			successCount++
			versions = append(versions, result.version)
		}
	}
	
	PrintKeyValue("\nTotal duration", totalDuration)
	PrintKeyValue("Successful batches", fmt.Sprintf("%d/%d", successCount, numGoroutines))
	
	// Verify version sequence
	PrintSection("Version Analysis")
	
	if len(versions) > 0 {
		minVersion := versions[0]
		maxVersion := versions[0]
		
		for _, v := range versions {
			if v < minVersion {
				minVersion = v
			}
			if v > maxVersion {
				maxVersion = v
			}
		}
		
		PrintKeyValue("Version range", fmt.Sprintf("%d to %d", minVersion, maxVersion))
		PrintKeyValue("Version spread", maxVersion - minVersion + 1)
		
		// Check for gaps
		versionMap := make(map[types.Version]bool)
		for _, v := range versions {
			versionMap[v] = true
		}
		
		gaps := 0
		for v := minVersion; v <= maxVersion; v++ {
			if !versionMap[v] {
				gaps++
			}
		}
		
		if gaps == 0 {
			PrintSuccess("No gaps in version sequence")
		} else {
			PrintInfo("Gaps in version sequence: %d (expected with concurrent execution)", gaps)
		}
	}
	
	// Verify data integrity
	PrintSection("Data Integrity Verification")
	
	latestVersion := jmt.GetLatestVersion()
	PrintKeyValue("Final version", latestVersion)
	
	// Check a sample of keys from each goroutine
	PrintInfo("Verifying sample keys from each goroutine:")
	
	allCorrect := true
	for i := 0; i < numGoroutines; i++ {
		// Check first, middle, and last key for each goroutine
		checkIndices := []int{0, opsPerGoroutine/2, opsPerGoroutine-1}
		
		goroutineCorrect := true
		for _, j := range checkIndices {
			key := types.KeyHash([]byte(fmt.Sprintf("g%d-key-%d", i, j)))
			expectedValue := fmt.Sprintf("goroutine-%d-value-%d", i, j)
			
			// Try to find in any version (since we don't know which version each goroutine got)
			found := false
			for v := initialVersion + 1; v <= latestVersion; v++ {
				value, err := jmt.GetAtVersion(v, key)
				if err == nil && string(value) == expectedValue {
					found = true
					break
				}
			}
			
			if !found {
				goroutineCorrect = false
				allCorrect = false
			}
		}
		
		if goroutineCorrect {
			PrintSuccess("  Goroutine %d: All sample keys verified ✓", i)
		} else {
			PrintError("  Goroutine %d: Some keys missing ✗", i)
		}
	}
	
	if allCorrect {
		PrintSuccess("\nAll concurrent operations completed successfully!")
	}
	
	// Show concurrency behavior
	PrintSection("Concurrency Behavior")
	PrintInfo("• Each batch transaction is thread-safe")
	PrintInfo("• Multiple goroutines can build batches simultaneously")
	PrintInfo("• Batch execution is serialized (one at a time)")
	PrintInfo("• Version numbers are assigned atomically")
	PrintInfo("• All operations within a batch are atomic")
	
	// Performance comparison
	PrintSection("Performance Analysis")
	
	totalOps := numGoroutines * opsPerGoroutine
	opsPerSecond := float64(totalOps) / totalDuration.Seconds()
	
	PrintKeyValue("Total operations", totalOps)
	PrintKeyValue("Total time", totalDuration)
	PrintKeyValue("Operations/second", fmt.Sprintf("%.0f", opsPerSecond))
	PrintKeyValue("Average time/batch", totalDuration / time.Duration(numGoroutines))
}

// To run this example:
// cd examples && go run example_batch_concurrent.go examples_utils.go
// Or: cd examples && ./run_example.sh batch_concurrent