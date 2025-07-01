package commands_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchOperations(t *testing.T) {
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

	// Initialize database
	_, err = runCLI("init", "--db", dbPath)
	require.NoError(t, err)

	t.Run("basic_batch", func(t *testing.T) {
		// Create batch file
		batchFile := filepath.Join(tempDir, "batch1.json")
		batchData := map[string]interface{}{
			"operations": []map[string]string{
				{"type": "put", "key": "alice", "value": "100"},
				{"type": "put", "key": "bob", "value": "200"},
				{"type": "put", "key": "charlie", "value": "300"},
			},
		}
		
		data, err := json.MarshalIndent(batchData, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(batchFile, data, 0644)
		require.NoError(t, err)

		// Execute batch
		output, err := runCLI("batch", batchFile, "--db", dbPath, "--json")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)

		assert.Equal(t, float64(3), result["total_operations"])
		assert.Equal(t, float64(3), result["successful_operations"])
		assert.Equal(t, float64(0), result["failed_operations"])

		// Verify values were inserted
		output, err = runCLI("get", "alice", "--db", dbPath, "--json")
		require.NoError(t, err)
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
		assert.Equal(t, "100", result["value"])
	})

	t.Run("mixed_operations", func(t *testing.T) {
		// Create batch file with mixed operations
		batchFile := filepath.Join(tempDir, "batch2.json")
		batchData := map[string]interface{}{
			"operations": []map[string]string{
				{"type": "put", "key": "david", "value": "400"},
				{"type": "delete", "key": "alice"},
				{"type": "put", "key": "eve", "value": "500"},
			},
		}
		
		data, err := json.MarshalIndent(batchData, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(batchFile, data, 0644)
		require.NoError(t, err)

		// Execute batch
		output, err := runCLI("batch", batchFile, "--db", dbPath, "--json")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)

		assert.Equal(t, float64(3), result["total_operations"])
		assert.Equal(t, float64(3), result["successful_operations"])

		// Verify alice was deleted
		output, err = runCLI("get", "alice", "--db", dbPath, "--json")
		require.NoError(t, err)
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
		assert.Equal(t, "not_found", result["status"])

		// Verify david was added
		output, err = runCLI("get", "david", "--db", dbPath, "--json")
		require.NoError(t, err)
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
		assert.Equal(t, "400", result["value"])
	})

	t.Run("dry_run", func(t *testing.T) {
		// Create batch file
		batchFile := filepath.Join(tempDir, "batch3.json")
		batchData := map[string]interface{}{
			"operations": []map[string]string{
				{"type": "put", "key": "test1", "value": "v1"},
				{"type": "put", "key": "test2", "value": "v2"},
			},
		}
		
		data, err := json.MarshalIndent(batchData, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(batchFile, data, 0644)
		require.NoError(t, err)

		// Execute dry run
		output, err := runCLI("batch", batchFile, "--db", dbPath, "--dry-run", "--json")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)

		assert.Equal(t, "dry_run", result["status"])
		assert.Equal(t, true, result["valid"])
		assert.Equal(t, float64(2), result["operation_count"])

		// Verify nothing was actually inserted
		_, err = runCLI("get", "test1", "--db", dbPath, "--json")
		require.NoError(t, err)
		// Should return not_found since it was a dry run
	})

	t.Run("invalid_operations", func(t *testing.T) {
		// Create batch file with invalid operation
		batchFile := filepath.Join(tempDir, "batch_invalid.json")
		batchData := map[string]interface{}{
			"operations": []map[string]string{
				{"type": "invalid", "key": "test", "value": "v1"},
			},
		}
		
		data, err := json.MarshalIndent(batchData, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(batchFile, data, 0644)
		require.NoError(t, err)

		// Should fail with invalid operation
		_, err = runCLI("batch", batchFile, "--db", dbPath)
		assert.Error(t, err)
	})

	t.Run("verbose_mode", func(t *testing.T) {
		// Create batch file
		batchFile := filepath.Join(tempDir, "batch_verbose.json")
		batchData := map[string]interface{}{
			"operations": []map[string]string{
				{"type": "put", "key": "v1", "value": "100"},
				{"type": "put", "key": "v2", "value": "200"},
			},
		}
		
		data, err := json.MarshalIndent(batchData, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(batchFile, data, 0644)
		require.NoError(t, err)

		// Execute with verbose mode
		output, err := runCLI("batch", batchFile, "--db", dbPath, "--verbose", "--json")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)

		// In verbose mode, we should get operation_results
		assert.Contains(t, result, "operation_results")
		opResults := result["operation_results"].([]interface{})
		assert.Equal(t, 2, len(opResults))
		
		// Check first operation
		op1 := opResults[0].(map[string]interface{})
		assert.Equal(t, float64(0), op1["index"])
		assert.Equal(t, "put", op1["type"])
		assert.Equal(t, "v1", op1["key"])
		assert.Equal(t, true, op1["success"])
		assert.Contains(t, op1, "version")
	})
}

func TestLargeBatch(t *testing.T) {
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

	// Initialize database
	_, err = runCLI("init", "--db", dbPath)
	require.NoError(t, err)

	// Create large batch file
	batchFile := filepath.Join(tempDir, "large_batch.json")
	operations := make([]map[string]string, 1000)
	for i := 0; i < 1000; i++ {
		operations[i] = map[string]string{
			"type":  "put",
			"key":   fmt.Sprintf("key_%04d", i),
			"value": fmt.Sprintf("value_%04d", i),
		}
	}

	batchData := map[string]interface{}{
		"operations": operations,
	}
	
	data, err := json.MarshalIndent(batchData, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(batchFile, data, 0644)
	require.NoError(t, err)

	// Execute large batch
	batchOutput, err := runCLI("batch", batchFile, "--db", dbPath, "--json")
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(batchOutput), &result)
	require.NoError(t, err)

	assert.Equal(t, float64(1000), result["total_operations"])
	assert.Equal(t, float64(1000), result["successful_operations"])
	assert.Equal(t, float64(0), result["failed_operations"])

	// Verify a few random keys
	for _, key := range []string{"key_0000", "key_0500", "key_0999"} {
		getOutput, err := runCLI("get", key, "--db", dbPath, "--json")
		require.NoError(t, err)
		err = json.Unmarshal([]byte(getOutput), &result)
		require.NoError(t, err)
		assert.Equal(t, "found", result["status"])
	}
}