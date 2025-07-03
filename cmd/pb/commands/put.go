package commands

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/neutral/proofbox/pkg/types"
	"github.com/spf13/cobra"
)

var (
	fileValue string
)

// putCmd represents the put command
var putCmd = &cobra.Command{
	Use:   "put <key> <value>",
	Short: "Insert or update a key-value pair",
	Long: `Insert or update a key-value pair in the Jellyfish Merkle Tree.
The key can be provided as a string or hex-encoded bytes.
The value can be provided as a string, hex-encoded bytes, or from a file.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runPut,
}

func init() {
	rootCmd.AddCommand(putCmd)
	putCmd.Flags().BoolVar(&hexKey, "hex-key", false, "interpret key as hex-encoded bytes")
	putCmd.Flags().StringVar(&fileValue, "file-value", "", "read value from file")
}

func runPut(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("key is required")
	}

	// Open database
	store, keyEncoder, err := openDatabase(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	// Parse key
	keyStr := args[0]
	var key types.Key
	if hexKey {
		keyBytes, err := hex.DecodeString(keyStr)
		if err != nil {
			return fmt.Errorf("invalid hex key: %w", err)
		}
		if len(keyBytes) != 32 {
			return fmt.Errorf("key must be 32 bytes, got %d", len(keyBytes))
		}
		copy(key[:], keyBytes)
	} else {
		key = types.KeyHash([]byte(keyStr))
	}

	// Get value
	var value []byte
	switch {
	case fileValue != "":
		// Validate file path
		if err := validatePath(fileValue); err != nil {
			return fmt.Errorf("invalid value file path: %w", err)
		}

		// Check file size
		if err := validateFileSize(fileValue, maxImportFileSize); err != nil {
			return fmt.Errorf("value file too large: %w", err)
		}

		// Read from file
		file, err := os.Open(fileValue)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		value, err = io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	case len(args) > 1:
		// Value from command line
		value = []byte(args[1])
	default:
		return fmt.Errorf("value is required (provide as argument or use --file-value)")
	}

	// Create tree and put value
	tree, err := openTree(store, keyEncoder)
	if err != nil {
		return err
	}

	version, err := tree.Put(key, value)
	if err != nil {
		return fmt.Errorf("failed to put value: %w", err)
	}

	// Get root hash
	rootHash, err := tree.GetRootHash(version)
	if err != nil {
		return fmt.Errorf("failed to get root hash: %w", err)
	}

	result := map[string]interface{}{
		"status":    "success",
		"key":       keyStr,
		"version":   version,
		"root_hash": hex.EncodeToString(rootHash[:]),
	}

	if verboseMode {
		// Sanitize key for display
		sanitizedKeyHex := sanitizeKey(hex.EncodeToString(key[:]), true)
		result["key_hex"] = sanitizedKeyHex
		result["value_size"] = len(value)

		// Only show sanitized value preview if small enough
		if len(value) <= 64 {
			result["value_preview"] = string(value)
		} else {
			result["value_preview"] = sanitizeValue(string(value))
		}
	}

	return outputResult(result)
}
