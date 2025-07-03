package commands

import (
	"encoding/hex"
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
	"github.com/spf13/cobra"
)

var (
	version uint64
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Retrieve a value by key",
	Long: `Retrieve a value from the Jellyfish Merkle Tree by its key.
You can specify a particular version to retrieve historical values.`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().BoolVar(&hexKey, "hex-key", false, "interpret key as hex-encoded bytes")
	getCmd.Flags().Uint64Var(&version, "version", 0, "version to retrieve (default: latest)")
}

func runGet(cmd *cobra.Command, args []string) error {
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

	// Create tree
	tree, err := openTree(store, keyEncoder)
	if err != nil {
		return err
	}

	// Use latest version if not specified
	if !cmd.Flags().Changed("version") {
		version = uint64(tree.GetLatestVersion())
	}

	// Get value
	value, err := tree.Get(types.Version(version), key)
	if err != nil {
		if err == types.ErrKeyNotFound {
			result := map[string]interface{}{
				"status":  "not_found",
				"key":     keyStr,
				"version": version,
			}
			return outputResult(result)
		}
		return fmt.Errorf("failed to get value: %w", err)
	}

	// Check if value is nil (key not found)
	if value == nil {
		result := map[string]interface{}{
			"status":  "not_found",
			"key":     keyStr,
			"version": version,
		}
		return outputResult(result)
	}

	// Prepare result
	result := map[string]interface{}{
		"status":  "found",
		"key":     keyStr,
		"value":   string(value),
		"version": version,
	}

	if verboseMode {
		result["key_hex"] = hex.EncodeToString(key[:])
		result["value_hex"] = hex.EncodeToString(value)
		result["value_size"] = len(value)
	}

	return outputResult(result)
}
