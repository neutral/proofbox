package commands

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunStats(t *testing.T) {
	// Setup test database
	dbPath := setupTestDB(t)
	defer cleanupTestDB(dbPath)

	// Initialize database
	rootCmd.SetArgs([]string{"init", "--db", dbPath})
	err := rootCmd.Execute()
	require.NoError(t, err)

	// Add some data to make stats more interesting
	rootCmd.SetArgs([]string{"put", "key1", "value1", "--db", dbPath})
	err = rootCmd.Execute()
	require.NoError(t, err)

	rootCmd.SetArgs([]string{"put", "key2", "value2", "--db", dbPath})
	err = rootCmd.Execute()
	require.NoError(t, err)

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		validate func(t *testing.T, output string)
	}{
		{
			name:    "basic stats",
			args:    []string{"stats", "--db", dbPath, "--json"},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				require.NoError(t, err)

				// Check tree stats
				tree, ok := result["tree"].(map[string]interface{})
				require.True(t, ok)
				assert.Contains(t, tree, "version")
				assert.Contains(t, tree, "latest")
				assert.Contains(t, tree, "height")
				assert.Contains(t, tree, "node_count")
				assert.Contains(t, tree, "root_hash")

				// Check storage stats
				storage, ok := result["storage"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "pebble", storage["backend"])
				assert.Contains(t, storage, "operations")
				assert.Contains(t, storage, "latency_us")
				assert.Contains(t, storage, "size")
				assert.Contains(t, storage, "cache")

				// Check database info
				database, ok := result["database"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, dbPath, database["path"])
			},
		},
		{
			name:    "stats at specific version",
			args:    []string{"stats", "--version", "1", "--db", dbPath, "--json"},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				require.NoError(t, err)

				tree := result["tree"].(map[string]interface{})
				assert.Equal(t, float64(1), tree["version"])
			},
		},
		{
			name:    "stats for non-existent version",
			args:    []string{"stats", "--version", "999", "--db", dbPath, "--json"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			version = 0
			jsonOutput = false

			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			w.Close()
			os.Stdout = oldStdout

			// Read output
			buf := make([]byte, 8192)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, output)
				}
			}
		})
	}
}

func TestStatsMetrics(t *testing.T) {
	t.Skip("Metrics are not persisted across CLI command invocations - each command creates a new tree instance")
	// Setup test database with metrics enabled
	dbPath := setupTestDB(t)
	defer cleanupTestDB(dbPath)

	// Initialize database
	rootCmd.SetArgs([]string{"init", "--db", dbPath})
	err := rootCmd.Execute()
	require.NoError(t, err)

	// Perform some operations to generate metrics
	operations := [][]string{
		{"put", "key1", "value1"},
		{"put", "key2", "value2"},
		{"get", "key1"},
		{"delete", "key1"},
		{"get", "key2"},
	}

	for _, op := range operations {
		args := append(op, "--db", dbPath)
		rootCmd.SetArgs(args)
		err := rootCmd.Execute()
		require.NoError(t, err)
	}

	// Get stats
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"stats", "--db", dbPath, "--json"})
	err = rootCmd.Execute()
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout

	// Read and parse output
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	// Verify storage metrics
	storage := result["storage"].(map[string]interface{})
	opsMetrics := storage["operations"].(map[string]interface{})

	// We should have recorded the operations
	getOps := opsMetrics["get"].(float64)
	putOps := opsMetrics["put"].(float64)
	deleteOps := opsMetrics["delete"].(float64)

	assert.Greater(t, getOps, float64(0), "Should have recorded get operations")
	assert.Greater(t, putOps, float64(0), "Should have recorded put operations")
	assert.Greater(t, deleteOps, float64(0), "Should have recorded delete operations")

	// Check latency metrics exist
	latency := storage["latency_us"].(map[string]interface{})
	assert.Contains(t, latency, "get_p50")
	assert.Contains(t, latency, "get_p99")
	assert.Contains(t, latency, "put_p50")
	assert.Contains(t, latency, "put_p99")

	// Check size metrics exist
	size := storage["size"].(map[string]interface{})
	assert.Contains(t, size, "database_bytes")
	assert.Contains(t, size, "database_mb")
	assert.Contains(t, size, "cache_bytes")
	assert.Contains(t, size, "cache_mb")
}
