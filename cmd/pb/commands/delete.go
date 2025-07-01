package commands

import (
	"encoding/hex"
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a key from the tree",
	Long: `Delete a key and its associated value from the Jellyfish Merkle Tree.
This creates a new version of the tree without the specified key.`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().BoolVar(&hexKey, "hex-key", false, "interpret key as hex-encoded bytes")
}

func runDelete(cmd *cobra.Command, args []string) error {
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

	// Delete key
	version, err := tree.Delete(key)
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
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
		result["key_hex"] = hex.EncodeToString(key[:])
	}

	return outputResult(result)
}