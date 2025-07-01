package commands_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIBasicOperations(t *testing.T) {
	// Build the CLI
	buildCmd := exec.Command("go", "build", "-o", "pb_test", "..")
	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build CLI: %s", output)
	defer os.Remove("pb_test")

	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Helper to run CLI commands
	runCLI := func(args ...string) (string, error) {
		cmd := exec.Command("./pb_test", args...)
		output, err := cmd.Output()
		return string(output), err
	}

	// Test init
	t.Run("init", func(t *testing.T) {
		output, err := runCLI("init", "--db", dbPath, "--json")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
		assert.Equal(t, "success", result["status"])
		assert.Equal(t, "pebble", result["backend"])
	})

	// Test put
	t.Run("put", func(t *testing.T) {
		output, err := runCLI("put", "alice", "100", "--db", dbPath, "--json")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
		assert.Equal(t, "success", result["status"])
		assert.Equal(t, "alice", result["key"])
		assert.Equal(t, float64(1), result["version"])
	})

	// Test get
	t.Run("get", func(t *testing.T) {
		output, err := runCLI("get", "alice", "--db", dbPath, "--json")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
		assert.Equal(t, "found", result["status"])
		assert.Equal(t, "alice", result["key"])
		assert.Equal(t, "100", result["value"])
	})

	// Test get non-existent key
	t.Run("get_not_found", func(t *testing.T) {
		output, err := runCLI("get", "bob", "--db", dbPath, "--json")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
		assert.Equal(t, "not_found", result["status"])
		assert.Equal(t, "bob", result["key"])
	})

	// Test root hash
	t.Run("root", func(t *testing.T) {
		output, err := runCLI("root", "--db", dbPath, "--json")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
		assert.NotEmpty(t, result["root_hash"])
		assert.Equal(t, float64(1), result["version"])
	})

	// Test prove and verify
	t.Run("prove_verify", func(t *testing.T) {
		proofFile := filepath.Join(tempDir, "alice.proof")

		// Generate proof
		output, err := runCLI("prove", "alice", "--db", dbPath, "--output", proofFile)
		require.NoError(t, err)
		assert.Contains(t, output, "success")

		// Verify proof
		output, err = runCLI("verify", proofFile, "--json")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
		assert.Equal(t, "valid", result["status"])
		assert.Equal(t, "inclusion", result["type"])
		assert.Equal(t, "100", result["value"])
	})

	// Test delete
	t.Run("delete", func(t *testing.T) {
		output, err := runCLI("delete", "alice", "--db", dbPath, "--json")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
		assert.Equal(t, "success", result["status"])
		assert.Equal(t, float64(2), result["version"])

		// Verify key is deleted
		output, err = runCLI("get", "alice", "--db", dbPath, "--json")
		require.NoError(t, err)

		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
		assert.Equal(t, "not_found", result["status"])
	})

	// Test stats
	t.Run("stats", func(t *testing.T) {
		output, err := runCLI("stats", "--db", dbPath, "--json")
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
		
		tree := result["tree"].(map[string]interface{})
		assert.Equal(t, float64(2), tree["latest"])
		assert.NotEmpty(t, tree["root_hash"])
	})
}

func TestCLIErrorHandling(t *testing.T) {
	// Build the CLI
	buildCmd := exec.Command("go", "build", "-o", "pb_test", "..")
	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build CLI: %s", output)
	defer os.Remove("pb_test")

	// Helper to run CLI commands
	runCLI := func(args ...string) (string, error) {
		cmd := exec.Command("./pb_test", args...)
		output, err := cmd.Output()
		return string(output), err
	}

	// Test operations on non-existent database
	t.Run("no_database", func(t *testing.T) {
		_, err := runCLI("get", "test", "--db", "nonexistent.db")
		assert.Error(t, err)
	})

	// Test invalid hex key
	t.Run("invalid_hex_key", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test.db")
		
		// Initialize database
		_, err := runCLI("init", "--db", dbPath)
		require.NoError(t, err)

		// Try to use invalid hex key
		_, err = runCLI("put", "invalidhex", "value", "--db", dbPath, "--hex-key")
		assert.Error(t, err)
	})
}

func TestCLIExitCodes(t *testing.T) {
	// Build the CLI
	buildCmd := exec.Command("go", "build", "-o", "pb_test", "..")
	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build CLI: %s", output)
	defer os.Remove("pb_test")

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Success case
	t.Run("success_exit_0", func(t *testing.T) {
		cmd := exec.Command("./pb_test", "init", "--db", dbPath)
		err := cmd.Run()
		assert.NoError(t, err, "Expected exit code 0")
	})

	// Error case
	t.Run("error_exit_non_0", func(t *testing.T) {
		cmd := exec.Command("./pb_test", "get", "test", "--db", "nonexistent.db")
		err := cmd.Run()
		assert.Error(t, err, "Expected non-zero exit code")
		
		exitError, ok := err.(*exec.ExitError)
		assert.True(t, ok, "Expected ExitError")
		assert.NotEqual(t, 0, exitError.ExitCode(), "Expected non-zero exit code")
	})
}

func TestCLIProofForNonExistentKey(t *testing.T) {
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
		output, err := cmd.Output()
		return string(output), err
	}

	// Initialize and add some data
	_, err = runCLI("init", "--db", dbPath)
	require.NoError(t, err)

	_, err = runCLI("put", "alice", "100", "--db", dbPath)
	require.NoError(t, err)

	// Generate proof for non-existent key
	proofFile := filepath.Join(tempDir, "bob.proof")
	_, err = runCLI("prove", "bob", "--db", dbPath, "--output", proofFile)
	require.NoError(t, err)

	// Verify the exclusion proof
	verifyOutput, err := runCLI("verify", proofFile, "--json")
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(verifyOutput), &result)
	require.NoError(t, err)
	assert.Equal(t, "valid", result["status"])
	assert.Contains(t, []string{"exclusion_empty", "exclusion_neighbor"}, result["type"])
}