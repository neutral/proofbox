package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// Maximum allowed file size for imports (100MB)
	maxImportFileSize = 100 * 1024 * 1024

	// Maximum allowed export size (1GB)
	maxExportSize = 1024 * 1024 * 1024

	// Maximum batch operations
	maxBatchOperations = 10000
)

// validatePath ensures the path is safe and doesn't allow directory traversal
func validatePath(path string) error {
	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid path: directory traversal detected")
	}

	// Ensure absolute paths don't escape to system directories
	if filepath.IsAbs(cleanPath) {
		// Check for sensitive system directories
		sensitivePrefix := []string{
			"/etc",
			"/sys",
			"/proc",
			"/dev",
			"/boot",
			"/root",
			"/usr/bin",
			"/usr/sbin",
			"/bin",
			"/sbin",
		}

		for _, prefix := range sensitivePrefix {
			if strings.HasPrefix(cleanPath, prefix) {
				return fmt.Errorf("invalid path: access to system directory denied")
			}
		}
	}

	return nil
}

// validateFileSize checks that a file is not too large
func validateFileSize(path string, maxSize int64) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > maxSize {
		return fmt.Errorf("file too large: %d bytes (max %d bytes)", info.Size(), maxSize)
	}

	return nil
}

// sanitizeKey removes or masks sensitive information from keys for logging
func sanitizeKey(key string, hexKey bool) string {
	if hexKey && len(key) > 16 {
		// For hex keys, show first 8 and last 8 characters
		return key[:8] + "..." + key[len(key)-8:]
	}

	if len(key) > 32 {
		// For long keys, truncate
		return key[:32] + "..."
	}

	return key
}

// sanitizeValue masks sensitive values for logging
func sanitizeValue(value string) string {
	if len(value) > 64 {
		return value[:64] + "..."
	}
	return value
}

// validateKeySize ensures keys are the correct size when hex-encoded
func validateKeySize(keyBytes []byte) error {
	if len(keyBytes) != 32 {
		return fmt.Errorf("key must be exactly 32 bytes, got %d", len(keyBytes))
	}
	return nil
}

// validateBatchSize ensures batch operations don't exceed limits
func validateBatchSize(operations int) error {
	if operations > maxBatchOperations {
		return fmt.Errorf("too many batch operations: %d (max %d)", operations, maxBatchOperations)
	}
	return nil
}
