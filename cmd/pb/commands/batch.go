package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/neutral/proofbox/pkg/types"
	"github.com/spf13/cobra"
)

var (
	dryRun bool
)

// BatchOperation represents a single operation in a batch
type BatchOperation struct {
	Type  string `json:"type"`  // "put" or "delete"
	Key   string `json:"key"`   // Key as string
	Value string `json:"value"` // Value for put operations
}

// BatchFile represents the structure of a batch JSON file
type BatchFile struct {
	Operations []BatchOperation `json:"operations"`
}

// BatchResult represents the result of a batch execution
type BatchResult struct {
	TotalOperations   int                      `json:"total_operations"`
	SuccessfulOps     int                      `json:"successful_operations"`
	FailedOps         int                      `json:"failed_operations"`
	StartVersion      types.Version            `json:"start_version"`
	EndVersion        types.Version            `json:"end_version"`
	Duration          string                   `json:"duration"`
	Errors            []string                 `json:"errors,omitempty"`
	OperationResults  []OperationResult        `json:"operation_results,omitempty"`
}

// OperationResult represents the result of a single operation
type OperationResult struct {
	Index   int    `json:"index"`
	Type    string `json:"type"`
	Key     string `json:"key"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Version uint64 `json:"version,omitempty"`
}

// batchCmd represents the batch command
var batchCmd = &cobra.Command{
	Use:   "batch <operations.json>",
	Short: "Execute batch operations atomically from a JSON file",
	Long: `Execute multiple operations atomically from a JSON file.
All operations are executed in a single transaction - either all succeed
or all fail. The file should contain an array of operations, each specifying
a type (put or delete), key, and value (for put operations).

Example JSON format:
{
  "operations": [
    {"type": "put", "key": "key1", "value": "value1"},
    {"type": "put", "key": "key2", "value": "value2"},
    {"type": "delete", "key": "key3"}
  ]
}`,
	Args: cobra.ExactArgs(1),
	RunE: runBatch,
}

func init() {
	rootCmd.AddCommand(batchCmd)
	batchCmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate operations without executing")
}

func runBatch(cmd *cobra.Command, args []string) error {
	// Read batch file
	batchFilePath := args[0]
	
	// Validate file path
	if err := validatePath(batchFilePath); err != nil {
		return fmt.Errorf("invalid batch file path: %w", err)
	}

	// Check file size
	if err := validateFileSize(batchFilePath, maxImportFileSize); err != nil {
		return err
	}
	
	file, err := os.Open(batchFilePath)
	if err != nil {
		return fmt.Errorf("failed to open batch file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read batch file: %w", err)
	}

	// Parse batch file
	var batch BatchFile
	if err := json.Unmarshal(data, &batch); err != nil {
		return fmt.Errorf("failed to parse batch file: %w", err)
	}

	if len(batch.Operations) == 0 {
		return fmt.Errorf("no operations found in batch file")
	}

	// Validate batch size
	if err := validateBatchSize(len(batch.Operations)); err != nil {
		return err
	}

	// Validate operations
	for i, op := range batch.Operations {
		if err := validateOperation(op); err != nil {
			return fmt.Errorf("invalid operation at index %d: %w", i, err)
		}
	}

	if dryRun {
		result := map[string]interface{}{
			"status":           "dry_run",
			"valid":            true,
			"operation_count":  len(batch.Operations),
			"operations":       batch.Operations,
		}
		return outputResult(result)
	}

	// Open database
	store, keyEncoder, err := openDatabase(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	// Create tree
	tree, err := openTree(store, keyEncoder)
	if err != nil {
		return err
	}

	// Execute batch atomically
	startTime := time.Now()
	startVersion := tree.GetLatestVersion()
	
	result := &BatchResult{
		TotalOperations: len(batch.Operations),
		StartVersion:    startVersion,
		Errors:          make([]string, 0),
	}

	if verboseMode {
		result.OperationResults = make([]OperationResult, 0, len(batch.Operations))
	}

	// Create batch transaction for atomic execution
	batchTx := tree.NewBatchTransaction()

	// Add all operations to the batch
	for i, op := range batch.Operations {
		key := types.KeyHash([]byte(op.Key))
		
		var err error
		switch op.Type {
		case "put":
			err = batchTx.BatchPut(key, []byte(op.Value))
		case "delete":
			err = batchTx.BatchDelete(key)
		}

		if err != nil {
			// Validation error - fail early
			result.FailedOps = len(batch.Operations)
			result.Errors = append(result.Errors, fmt.Sprintf("op %d: validation failed: %v", i, err))
			result.Duration = time.Since(startTime).String()
			return outputResult(result)
		}
	}

	// Execute the batch atomically
	version, err := batchTx.Execute()
	if err != nil {
		// All operations failed
		result.FailedOps = len(batch.Operations)
		result.Errors = append(result.Errors, fmt.Sprintf("batch execution failed: %v", err))
		
		if verboseMode {
			// Mark all operations as failed
			for i, op := range batch.Operations {
				result.OperationResults = append(result.OperationResults, OperationResult{
					Index:   i,
					Type:    op.Type,
					Key:     op.Key,
					Success: false,
					Error:   "batch execution failed",
				})
			}
		}
	} else {
		// All operations succeeded
		result.SuccessfulOps = len(batch.Operations)
		result.EndVersion = version
		
		if verboseMode {
			// Mark all operations as successful with the same version
			for i, op := range batch.Operations {
				result.OperationResults = append(result.OperationResults, OperationResult{
					Index:   i,
					Type:    op.Type,
					Key:     op.Key,
					Success: true,
					Version: uint64(version),
				})
			}
		}
	}

	// Set final results
	result.Duration = time.Since(startTime).String()

	// Output results
	return outputResult(result)
}

func validateOperation(op BatchOperation) error {
	switch op.Type {
	case "put":
		if op.Key == "" {
			return fmt.Errorf("missing key for put operation")
		}
		if op.Value == "" {
			return fmt.Errorf("missing value for put operation")
		}
	case "delete":
		if op.Key == "" {
			return fmt.Errorf("missing key for delete operation")
		}
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
	return nil
}