package commands

import (
	"encoding/hex"
	"os"
	"testing"

	"github.com/neutral/proofbox/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunPut(t *testing.T) {
	// Setup test database
	dbPath := setupTestDB(t)
	defer cleanupTestDB(dbPath)

	// Initialize database
	rootCmd.SetArgs([]string{"init", "--db", dbPath})
	err := rootCmd.Execute()
	require.NoError(t, err)

	// Reset command for next use
	rootCmd.SetArgs([]string{})

	tests := []struct {
		name      string
		args      []string
		setupFile func() string
		wantErr   bool
		validate  func(t *testing.T)
	}{
		{
			name:    "basic put",
			args:    []string{"put", "testkey", "testvalue", "--db", dbPath, "--json"},
			wantErr: false,
			validate: func(t *testing.T) {
				// Verify the key was stored
				rootCmd.SetArgs([]string{"get", "testkey", "--db", dbPath, "--json"})
				err := rootCmd.Execute()
				assert.NoError(t, err)
			},
		},
		{
			name: "put with hex key",
			args: []string{"put",
				"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				"hexvalue",
				"--hex-key", "--db", dbPath, "--json"},
			wantErr: false,
		},
		{
			name:    "put with invalid hex key",
			args:    []string{"put", "notahexkey", "value", "--hex-key", "--db", dbPath},
			wantErr: true,
		},
		{
			name:    "put with wrong size hex key",
			args:    []string{"put", "0123456789abcdef", "value", "--hex-key", "--db", dbPath},
			wantErr: true,
		},
		{
			name: "put from file",
			setupFile: func() string {
				tmpFile := createTempFile(t, "value.txt", "file content")
				return tmpFile
			},
			args:    []string{"put", "filekey", "--file-value", "", "--db", dbPath, "--json"},
			wantErr: false,
		},
		{
			name:    "put missing value",
			args:    []string{"put", "onlykey", "--db", dbPath},
			wantErr: true,
		},
		{
			name:    "put with malicious file path",
			args:    []string{"put", "key", "--file-value", "../../../etc/passwd", "--db", dbPath},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args

			// Setup file if needed
			if tt.setupFile != nil {
				filePath := tt.setupFile()
				// Replace empty file-value flag with actual path
				for i, arg := range args {
					if arg == "" && i > 0 && args[i-1] == "--file-value" {
						args[i] = filePath
					}
				}
			}

			// Reset flags
			hexKey = false
			fileValue = ""
			verboseMode = false
			jsonOutput = false

			rootCmd.SetArgs(args)
			err := rootCmd.Execute()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validate != nil {
				tt.validate(t)
			}
		})
	}
}

func TestPutKeyEncoding(t *testing.T) {
	// Test that keys are properly encoded
	testKey := "alice"
	expectedHash := types.KeyHash([]byte(testKey))

	// Verify the hash is 32 bytes
	assert.Equal(t, 32, len(expectedHash))

	// Test hex encoding
	hexStr := hex.EncodeToString(expectedHash[:])
	assert.Equal(t, 64, len(hexStr))

	// Test decoding
	decoded, err := hex.DecodeString(hexStr)
	assert.NoError(t, err)
	assert.Equal(t, expectedHash[:], decoded)
}

func TestPutValueSanitization(t *testing.T) {
	// Setup test database
	dbPath := setupTestDB(t)
	defer cleanupTestDB(dbPath)

	// Initialize database
	rootCmd.SetArgs([]string{"init", "--db", dbPath})
	err := rootCmd.Execute()
	require.NoError(t, err)

	// Test verbose mode with long value
	longValue := string(make([]byte, 100))
	for i := range longValue {
		longValue = longValue[:i] + "a" + longValue[i+1:]
	}

	// Reset flags
	hexKey = false
	fileValue = ""
	verboseMode = true
	jsonOutput = true

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"put", "longkey", longValue, "--db", dbPath, "--verbose", "--json"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout

	// Read output
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check that value_preview is sanitized
	assert.Contains(t, output, "value_preview")
	assert.Contains(t, output, "...")
}
