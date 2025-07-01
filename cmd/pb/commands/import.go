package commands

import (
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/neutral/proofbox/pkg/types"
	"github.com/spf13/cobra"
)

var (
	importFormat string
	skipErrors   bool
	validateOnly bool
)

// ImportResult represents the result of an import operation
type ImportResult struct {
	TotalEntries      int           `json:"total_entries"`
	SuccessfulImports int           `json:"successful_imports"`
	FailedImports     int           `json:"failed_imports"`
	StartVersion      types.Version `json:"start_version"`
	EndVersion        types.Version `json:"end_version"`
	Duration          string        `json:"duration"`
	Errors            []string      `json:"errors,omitempty"`
}

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import tree data atomically from a file",
	Long: `Import key-value pairs atomically from a JSON or CSV file.
All entries are imported in a single transaction - either all succeed
or all fail. The file format should match the export format.`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().StringVar(&importFormat, "format", "json", "import format (json or csv)")
	importCmd.Flags().BoolVar(&skipErrors, "skip-errors", false, "continue on import errors")
	importCmd.Flags().BoolVar(&validateOnly, "validate", false, "validate import file without importing")
}

func runImport(cmd *cobra.Command, args []string) error {
	importFile := args[0]

	// Validate file path
	if err := validatePath(importFile); err != nil {
		return fmt.Errorf("invalid import file path: %w", err)
	}

	// Check file size
	if err := validateFileSize(importFile, maxImportFileSize); err != nil {
		return err
	}

	// Open import file
	file, err := os.Open(importFile)
	if err != nil {
		return fmt.Errorf("failed to open import file: %w", err)
	}
	defer file.Close()

	// Parse entries based on format
	var entries []ExportEntry

	switch importFormat {
	case "json":
		var exportData ExportData
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&exportData); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		entries = exportData.Entries

		if verboseMode {
			fmt.Printf("Import file info:\n")
			fmt.Printf("  Original version: %d\n", exportData.Version)
			fmt.Printf("  Export time: %s\n", exportData.ExportTime)
			fmt.Printf("  Root hash: %s\n", exportData.RootHash)
			fmt.Printf("  Entry count: %d\n", exportData.EntryCount)
		}

	case "csv":
		csvReader := csv.NewReader(file)
		
		// Read header
		header, err := csvReader.Read()
		if err != nil {
			return fmt.Errorf("failed to read CSV header: %w", err)
		}

		// Validate header
		if len(header) < 2 || header[0] != "key" || header[1] != "value" {
			return fmt.Errorf("invalid CSV header: expected 'key,value'")
		}

		// Read all records
		for {
			record, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read CSV record: %w", err)
			}

			if len(record) < 2 {
				return fmt.Errorf("invalid CSV record: expected at least 2 fields")
			}

			entries = append(entries, ExportEntry{
				Key:   record[0],
				Value: record[1],
			})
		}

	default:
		return fmt.Errorf("unsupported format: %s", importFormat)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no entries found in import file")
	}

	// Validate batch size
	if err := validateBatchSize(len(entries)); err != nil {
		return err
	}

	// Validate mode
	if validateOnly {
		validCount := 0
		invalidCount := 0
		
		for i, entry := range entries {
			if err := validateImportEntry(entry); err != nil {
				invalidCount++
				if verboseMode {
					fmt.Printf("Entry %d invalid: %v\n", i, err)
				}
			} else {
				validCount++
			}
		}

		result := map[string]interface{}{
			"status":         "validation_complete",
			"total_entries":  len(entries),
			"valid_entries":  validCount,
			"invalid_entries": invalidCount,
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

	// Import entries atomically using batch transaction
	startTime := time.Now()
	startVersion := tree.GetLatestVersion()
	
	result := &ImportResult{
		TotalEntries: len(entries),
		StartVersion: startVersion,
		Errors:       make([]string, 0),
	}

	// Create batch transaction for atomic import
	batch := tree.NewBatchTransaction()

	// Add all entries to the batch
	validEntries := 0
	for i, entry := range entries {
		key, value, err := parseImportEntry(entry)
		if err != nil {
			result.FailedImports++
			errorMsg := fmt.Sprintf("entry %d: validation failed: %v", i, err)
			result.Errors = append(result.Errors, errorMsg)
			
			if !skipErrors {
				return fmt.Errorf("import failed at entry %d: %w", i, err)
			}
			continue
		}

		// Add to batch
		if err := batch.BatchPut(key, value); err != nil {
			result.FailedImports++
			errorMsg := fmt.Sprintf("entry %d: failed to add to batch: %v", i, err)
			result.Errors = append(result.Errors, errorMsg)
			
			if !skipErrors {
				return fmt.Errorf("import failed at entry %d: %w", i, err)
			}
			continue
		}
		
		validEntries++

		// Progress reporting
		if (i+1)%1000 == 0 && verboseMode {
			fmt.Fprintf(os.Stderr, "Prepared %d/%d entries...\n", i+1, len(entries))
		}
	}

	// Execute the batch atomically
	if batch.Size() > 0 {
		version, err := batch.Execute()
		if err != nil {
			result.FailedImports = len(entries)
			result.Errors = append(result.Errors, fmt.Sprintf("batch execution failed: %v", err))
			return fmt.Errorf("import batch execution failed: %w", err)
		}
		result.SuccessfulImports = validEntries
		result.EndVersion = version
		
		if verboseMode {
			fmt.Fprintf(os.Stderr, "Successfully imported %d entries\n", batch.Size())
		}
	} else {
		result.EndVersion = startVersion
	}

	result.Duration = time.Since(startTime).String()

	// Output results
	return outputResult(result)
}

func validateImportEntry(entry ExportEntry) error {
	// Validate key
	if entry.Key == "" {
		return fmt.Errorf("empty key")
	}

	// If key is hex encoded, validate it
	if len(entry.Key) == 64 {
		if _, err := hex.DecodeString(entry.Key); err != nil {
			return fmt.Errorf("invalid hex key: %w", err)
		}
	}

	// Value can be empty (for some use cases)
	// but we'll validate hex encoding if provided
	if entry.ValueHex != "" {
		if _, err := hex.DecodeString(entry.ValueHex); err != nil {
			return fmt.Errorf("invalid hex value: %w", err)
		}
	}

	return nil
}

func parseImportEntry(entry ExportEntry) (types.Key, []byte, error) {
	var key types.Key
	var value []byte

	// Validate key is not empty
	if entry.Key == "" {
		return key, nil, fmt.Errorf("empty key")
	}

	// Parse key
	if len(entry.Key) == 64 {
		// Assume hex-encoded key
		keyBytes, err := hex.DecodeString(entry.Key)
		if err != nil {
			return key, nil, fmt.Errorf("failed to decode hex key: %w", err)
		}
		if len(keyBytes) != 32 {
			return key, nil, fmt.Errorf("key must be 32 bytes")
		}
		copy(key[:], keyBytes)
	} else {
		// Hash the key
		key = types.KeyHash([]byte(entry.Key))
	}

	// Parse value
	if entry.ValueHex != "" {
		var err error
		value, err = hex.DecodeString(entry.ValueHex)
		if err != nil {
			return key, nil, fmt.Errorf("failed to decode hex value: %w", err)
		}
	} else {
		value = []byte(entry.Value)
	}

	return key, value, nil
}