package commands

import (
	"encoding/hex"
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
	"github.com/spf13/cobra"
)

// rootHashCmd represents the root command
var rootHashCmd = &cobra.Command{
	Use:   "root",
	Short: "Get the root hash of the tree",
	Long: `Get the root hash of the Jellyfish Merkle Tree at a specific version.
If no version is specified, returns the root hash of the latest version.`,
	RunE: runRootHash,
}

func init() {
	rootCmd.AddCommand(rootHashCmd)
	rootHashCmd.Flags().Uint64Var(&version, "version", 0, "version to get root hash for (default: latest)")
}

func runRootHash(cmd *cobra.Command, args []string) error {
	// Open database
	store, keyEncoder, err := openDatabase(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	// Create tree
	tree, err := openTree(store, keyEncoder)
	if err != nil {
		return err
	}

	// Use latest version if not specified
	if !cmd.Flags().Changed("version") {
		version = uint64(tree.GetLatestVersion())
	}

	// Get root hash
	rootHash, err := tree.GetRootHash(types.Version(version))
	if err != nil {
		return fmt.Errorf("failed to get root hash: %w", err)
	}

	result := map[string]interface{}{
		"version":   version,
		"root_hash": hex.EncodeToString(rootHash[:]),
	}

	if verboseMode {
		// Add tree statistics
		height, nodeCount, _ := tree.GetStats()
		result["stats"] = map[string]interface{}{
			"height":     height,
			"node_count": nodeCount,
		}
	}

	return outputResult(result)
}
