package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenDatabase(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() string
		wantErr     bool
		errContains string
	}{
		{
			name: "non-existent database",
			setup: func() string {
				return "/non/existent/path"
			},
			wantErr:     true,
			errContains: "database not found",
		},
		{
			name: "memory backend marker",
			setup: func() string {
				tmpFile, err := os.CreateTemp("", "memory_db")
				require.NoError(t, err)
				_, err = tmpFile.WriteString("memory")
				require.NoError(t, err)
				tmpFile.Close()
				t.Cleanup(func() { os.Remove(tmpFile.Name()) })
				return tmpFile.Name()
			},
			wantErr: false,
		},
		{
			name: "pebble database",
			setup: func() string {
				tmpDir := t.TempDir()
				dbPath := filepath.Join(tmpDir, "test.db")
				
				// Initialize a minimal pebble database
				rootCmd.SetArgs([]string{"init", "--db", dbPath})
				err := rootCmd.Execute()
				require.NoError(t, err)
				
				return dbPath
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := tt.setup()
			
			store, keyEncoder, err := openDatabase(dbPath)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, store)
				assert.Nil(t, keyEncoder)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, store)
				assert.NotNil(t, keyEncoder)
				if store != nil {
					store.Close()
				}
			}
		})
	}
}

func TestOpenTree(t *testing.T) {
	// Create a memory storage for testing
	tmpFile, err := os.CreateTemp("", "memory_db")
	require.NoError(t, err)
	_, err = tmpFile.WriteString("memory")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	store, keyEncoder, err := openDatabase(tmpFile.Name())
	require.NoError(t, err)
	defer store.Close()

	// Test opening tree
	tree, err := openTree(store, keyEncoder)
	assert.NoError(t, err)
	assert.NotNil(t, tree)

	// Verify tree is functional
	latestVersion := tree.GetLatestVersion()
	assert.Equal(t, uint64(0), uint64(latestVersion))
}

func TestOutputResult(t *testing.T) {
	// Save original values
	origJSON := jsonOutput
	origStdout := os.Stdout

	tests := []struct {
		name       string
		jsonOutput bool
		result     interface{}
		wantJSON   bool
	}{
		{
			name:       "JSON output mode",
			jsonOutput: true,
			result:     map[string]string{"status": "success"},
			wantJSON:   true,
		},
		{
			name:       "Human-readable output mode",
			jsonOutput: false,
			result:     map[string]string{"status": "success"},
			wantJSON:   false,
		},
		{
			name:       "Complex nested structure",
			jsonOutput: true,
			result: map[string]interface{}{
				"tree": map[string]interface{}{
					"version": 1,
					"height":  5,
				},
			},
			wantJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set JSON output mode
			jsonOutput = tt.jsonOutput

			// Capture output
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := outputResult(tt.result)
			assert.NoError(t, err)

			w.Close()
			os.Stdout = origStdout

			// Read captured output
			buf := make([]byte, 1024)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			if tt.wantJSON {
				// Should be valid JSON
				assert.Contains(t, output, "{")
				assert.Contains(t, output, "}")
			}
		})
	}

	// Restore original values
	jsonOutput = origJSON
	os.Stdout = origStdout
}