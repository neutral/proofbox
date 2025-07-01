package commands_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportImport(t *testing.T) {
	// Build the CLI
	buildCmd := exec.Command("go", "build", "-o", "pb_test", "..")
	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build CLI: %s", output)
	defer os.Remove("pb_test")

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Helper to run CLI commands
	runCLI := func(args ...string) (string, error) {
		cmd := exec.Command("./pb_test", args...)
		// Only capture stdout to avoid pebble log messages
		output, err := cmd.Output()
		if err != nil {
			// If there's an error, get stderr for debugging
			if exitErr, ok := err.(*exec.ExitError); ok {
				return string(output), fmt.Errorf("%w: %s", err, exitErr.Stderr)
			}
		}
		return string(output), err
	}

	// Initialize database and add test data
	_, err = runCLI("init", "--db", dbPath)
	require.NoError(t, err)

	// Add test data
	testData := map[string]string{
		"user:001":    "Alice",
		"user:002":    "Bob",
		"user:003":    "Charlie",
		"config:name": "TestApp",
		"config:ver":  "1.0.0",
	}

	for k, v := range testData {
		_, err = runCLI("put", k, v, "--db", dbPath)
		require.NoError(t, err)
	}

	t.Run("export_import_json", func(t *testing.T) {
		// Create keys file
		keysFile := filepath.Join(tempDir, "keys.txt")
		keysContent := strings.Join([]string{
			"user:001",
			"user:002",
			"user:003",
			"config:name",
			"config:ver",
		}, "\n")
		err := os.WriteFile(keysFile, []byte(keysContent), 0644)
		require.NoError(t, err)

		// Export data
		exportFile := filepath.Join(tempDir, "export.json")
		output, err := runCLI("export-keys", "--keys", keysFile, "--db", dbPath, 
			"--output", exportFile, "--format", "json")
		require.NoError(t, err)

		// Verify export file exists
		_, err = os.Stat(exportFile)
		require.NoError(t, err)

		// Read and verify export content
		exportData, err := os.ReadFile(exportFile)
		require.NoError(t, err)

		var exported map[string]interface{}
		err = json.Unmarshal(exportData, &exported)
		require.NoError(t, err)

		assert.Equal(t, float64(5), exported["entry_count"])
		entries := exported["entries"].([]interface{})
		assert.Len(t, entries, 5)

		// Create new database for import
		importDbPath := filepath.Join(tempDir, "import.db")
		_, err = runCLI("init", "--db", importDbPath)
		require.NoError(t, err)

		// Import data
		output, err = runCLI("import", exportFile, "--db", importDbPath, "--json")
		require.NoError(t, err)

		var importResult map[string]interface{}
		err = json.Unmarshal([]byte(output), &importResult)
		require.NoError(t, err)

		assert.Equal(t, float64(5), importResult["total_entries"])
		assert.Equal(t, float64(5), importResult["successful_imports"])
		assert.Equal(t, float64(0), importResult["failed_imports"])

		// Verify imported data
		for k, expectedValue := range testData {
			output, err = runCLI("get", k, "--db", importDbPath, "--json")
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal([]byte(output), &result)
			require.NoError(t, err)

			assert.Equal(t, "found", result["status"])
			assert.Equal(t, expectedValue, result["value"])
		}
	})

	t.Run("export_import_csv", func(t *testing.T) {
		// Create keys file for specific keys
		keysFile := filepath.Join(tempDir, "keys_csv.txt")
		keysContent := "user:001\nuser:002\nconfig:name"
		err := os.WriteFile(keysFile, []byte(keysContent), 0644)
		require.NoError(t, err)

		// Export as CSV
		csvFile := filepath.Join(tempDir, "export.csv")
		_, err = runCLI("export-keys", "--keys", keysFile, "--db", dbPath,
			"--output", csvFile, "--format", "csv")
		require.NoError(t, err)

		// Verify CSV content
		csvData, err := os.ReadFile(csvFile)
		require.NoError(t, err)
		lines := strings.Split(strings.TrimSpace(string(csvData)), "\n")
		assert.Equal(t, 4, len(lines)) // header + 3 data rows
		assert.Equal(t, "key,value", lines[0])

		// Create new database for CSV import
		csvImportDb := filepath.Join(tempDir, "csv_import.db")
		_, err = runCLI("init", "--db", csvImportDb)
		require.NoError(t, err)

		// Import CSV
		output, err := runCLI("import", csvFile, "--db", csvImportDb, 
			"--format", "csv", "--json")
		require.NoError(t, err)

		var importResult map[string]interface{}
		err = json.Unmarshal([]byte(output), &importResult)
		require.NoError(t, err)

		assert.Equal(t, float64(3), importResult["successful_imports"])
	})

	t.Run("export_with_hex", func(t *testing.T) {
		// Create a single key file
		keysFile := filepath.Join(tempDir, "single_key.txt")
		err := os.WriteFile(keysFile, []byte("user:001"), 0644)
		require.NoError(t, err)

		// Export with hex encoding
		hexExportFile := filepath.Join(tempDir, "hex_export.json")
		_, err = runCLI("export-keys", "--keys", keysFile, "--db", dbPath,
			"--output", hexExportFile, "--hex", "--json")
		require.NoError(t, err)

		// Read and verify hex content
		hexData, err := os.ReadFile(hexExportFile)
		require.NoError(t, err)

		var hexExport map[string]interface{}
		err = json.Unmarshal(hexData, &hexExport)
		require.NoError(t, err)

		entries := hexExport["entries"].([]interface{})
		entry := entries[0].(map[string]interface{})
		assert.Contains(t, entry, "key_hex")
		assert.Contains(t, entry, "value_hex")
	})

	t.Run("import_validation", func(t *testing.T) {
		// Create invalid import file
		invalidFile := filepath.Join(tempDir, "invalid.json")
		invalidData := map[string]interface{}{
			"entries": []map[string]string{
				{"key": "", "value": "empty key"},
				{"key": "valid", "value": "good"},
			},
		}
		data, _ := json.MarshalIndent(invalidData, "", "  ")
		err := os.WriteFile(invalidFile, data, 0644)
		require.NoError(t, err)

		// Try validation only
		output, err := runCLI("import", invalidFile, "--validate", "--json")
		require.NoError(t, err)

		var validationResult map[string]interface{}
		err = json.Unmarshal([]byte(output), &validationResult)
		require.NoError(t, err)

		assert.Equal(t, "validation_complete", validationResult["status"])
		assert.Equal(t, float64(2), validationResult["total_entries"])
		assert.Equal(t, float64(1), validationResult["valid_entries"])
		assert.Equal(t, float64(1), validationResult["invalid_entries"])
	})

	t.Run("import_skip_errors", func(t *testing.T) {
		// Create mixed valid/invalid data
		mixedFile := filepath.Join(tempDir, "mixed.json")
		mixedData := map[string]interface{}{
			"entries": []map[string]string{
				{"key": "good1", "value": "value1"},
				{"key": "", "value": "bad - empty key"},
				{"key": "good2", "value": "value2"},
			},
		}
		data, _ := json.MarshalIndent(mixedData, "", "  ")
		err := os.WriteFile(mixedFile, data, 0644)
		require.NoError(t, err)

		// Create new database
		skipErrorDb := filepath.Join(tempDir, "skip_error.db")
		_, err = runCLI("init", "--db", skipErrorDb)
		require.NoError(t, err)

		// Import with skip-errors
		output, err := runCLI("import", mixedFile, "--db", skipErrorDb,
			"--skip-errors", "--json")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)

		assert.Equal(t, float64(3), result["total_entries"])
		assert.Equal(t, float64(2), result["successful_imports"])
		assert.Equal(t, float64(1), result["failed_imports"])

		// Verify good entries were imported
		output, err = runCLI("get", "good1", "--db", skipErrorDb, "--json")
		require.NoError(t, err)
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
		assert.Equal(t, "found", result["status"])
	})

	t.Run("export_nonexistent_keys", func(t *testing.T) {
		// Create keys file with some non-existent keys
		keysFile := filepath.Join(tempDir, "mixed_keys.txt")
		keysContent := "user:001\nnonexistent1\nuser:002\nnonexistent2"
		err := os.WriteFile(keysFile, []byte(keysContent), 0644)
		require.NoError(t, err)

		// Export should succeed but only export found keys
		output, err := runCLI("export-keys", "--keys", keysFile, "--db", dbPath,
			"--json", "--output", "-")
		require.NoError(t, err)

		var exportData map[string]interface{}
		err = json.Unmarshal([]byte(output), &exportData)
		require.NoError(t, err)

		// Should only have 2 entries (user:001 and user:002)
		assert.Equal(t, float64(2), exportData["entry_count"])
	})
}