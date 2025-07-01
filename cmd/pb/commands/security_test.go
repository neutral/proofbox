package commands

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid relative path",
			path:    "data/export.json",
			wantErr: false,
		},
		{
			name:    "valid absolute path",
			path:    "/home/user/data/export.json",
			wantErr: false,
		},
		{
			name:    "directory traversal with ..",
			path:    "../../../etc/passwd",
			wantErr: true,
			errMsg:  "directory traversal detected",
		},
		{
			name:    "directory traversal in middle",
			path:    "data/../../../etc/passwd",
			wantErr: true,
			errMsg:  "directory traversal detected",
		},
		{
			name:    "system directory /etc",
			path:    "/etc/passwd",
			wantErr: true,
			errMsg:  "access to system directory denied",
		},
		{
			name:    "system directory /root",
			path:    "/root/.ssh/id_rsa",
			wantErr: true,
			errMsg:  "access to system directory denied",
		},
		{
			name:    "system directory /sys",
			path:    "/sys/kernel/config",
			wantErr: true,
			errMsg:  "access to system directory denied",
		},
		{
			name:    "valid tmp path",
			path:    "/tmp/export.json",
			wantErr: false,
		},
		{
			name:    "valid home path",
			path:    "/home/user/export.json",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFileSize(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test_file_")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write some data
	testData := []byte("test data")
	_, err = tmpFile.Write(testData)
	require.NoError(t, err)
	tmpFile.Close()

	tests := []struct {
		name    string
		path    string
		maxSize int64
		wantErr bool
	}{
		{
			name:    "file within size limit",
			path:    tmpFile.Name(),
			maxSize: 1024,
			wantErr: false,
		},
		{
			name:    "file exceeds size limit",
			path:    tmpFile.Name(),
			maxSize: 5,
			wantErr: true,
		},
		{
			name:    "non-existent file",
			path:    "/non/existent/file",
			maxSize: 1024,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileSize(tt.path, tt.maxSize)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		hexKey   bool
		expected string
	}{
		{
			name:     "short key",
			key:      "alice",
			hexKey:   false,
			expected: "alice",
		},
		{
			name:     "long key truncated",
			key:      "this_is_a_very_long_key_that_should_be_truncated_for_display",
			hexKey:   false,
			expected: "this_is_a_very_long_key_that_sho...",
		},
		{
			name:     "hex key shortened",
			key:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			hexKey:   true,
			expected: "01234567...89abcdef",
		},
		{
			name:     "short hex key unchanged",
			key:      "0123456789abcdef",
			hexKey:   true,
			expected: "0123456789abcdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeKey(tt.key, tt.hexKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "short value",
			value:    "test value",
			expected: "test value",
		},
		{
			name:     "long value truncated",
			value:    "this is a very long value that exceeds the maximum display length and should be truncated",
			expected: "this is a very long value that exceeds the maximum display lengt...",
		},
		{
			name:     "exactly 64 chars",
			value:    "0123456789012345678901234567890123456789012345678901234567890123",
			expected: "0123456789012345678901234567890123456789012345678901234567890123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeValue(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateKeySize(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
		wantErr bool
	}{
		{
			name:    "correct size 32 bytes",
			keySize: 32,
			wantErr: false,
		},
		{
			name:    "too small",
			keySize: 16,
			wantErr: true,
		},
		{
			name:    "too large",
			keySize: 64,
			wantErr: true,
		},
		{
			name:    "empty",
			keySize: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			err := validateKeySize(key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "32 bytes")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBatchSize(t *testing.T) {
	tests := []struct {
		name       string
		operations int
		wantErr    bool
	}{
		{
			name:       "within limit",
			operations: 100,
			wantErr:    false,
		},
		{
			name:       "at limit",
			operations: maxBatchOperations,
			wantErr:    false,
		},
		{
			name:       "exceeds limit",
			operations: maxBatchOperations + 1,
			wantErr:    true,
		},
		{
			name:       "zero operations",
			operations: 0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBatchSize(tt.operations)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "too many batch operations")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecurityConstants(t *testing.T) {
	// Ensure constants have reasonable values
	assert.Greater(t, maxImportFileSize, int64(1024*1024), "Import file size should be at least 1MB")
	assert.Less(t, maxImportFileSize, int64(1024*1024*1024), "Import file size should be less than 1GB")
	
	assert.Greater(t, maxBatchOperations, 100, "Should allow at least 100 batch operations")
	assert.Less(t, maxBatchOperations, 1000000, "Batch operations limit should be reasonable")
}