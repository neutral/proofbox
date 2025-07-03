package commands

import (
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
	"github.com/spf13/cobra"
)

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Display tree statistics",
	Long: `Display statistics about the Jellyfish Merkle Tree including
height, node count, version information, and storage metrics.`,
	RunE: runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().Uint64Var(&version, "version", 0, "version to get stats for (default: latest)")
}

func runStats(cmd *cobra.Command, args []string) error {
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

	// Get tree statistics
	height, nodeCount, _ := tree.GetStats()

	// Get root hash for the version
	rootHash, err := tree.GetRootHash(types.Version(version))
	if err != nil {
		return fmt.Errorf("failed to get root hash: %w", err)
	}

	// Get storage metrics
	metrics := store.Metrics()

	// Build result
	result := map[string]interface{}{
		"tree": map[string]interface{}{
			"version":        version,
			"latest":         tree.GetLatestVersion(),
			"height":         height,
			"node_count":     nodeCount,
			"root_hash":      fmt.Sprintf("%x", rootHash[:8]),
			"root_hash_full": fmt.Sprintf("%x", rootHash[:]),
		},
	}

	// Add storage metrics
	storageInfo := map[string]interface{}{
		"backend": "pebble",
		"operations": map[string]interface{}{
			"get":     metrics.GetOperations(),
			"put":     metrics.PutOperations(),
			"delete":  metrics.DeleteOperations(),
			"batches": metrics.BatchCommits(),
		},
		"latency_us": map[string]interface{}{
			"get_p50": metrics.GetLatencyP50() / 1000, // Convert ns to μs
			"get_p99": metrics.GetLatencyP99() / 1000,
			"put_p50": metrics.PutLatencyP50() / 1000,
			"put_p99": metrics.PutLatencyP99() / 1000,
		},
		"size": map[string]interface{}{
			"database_bytes":  metrics.DatabaseSize(),
			"database_mb":     float64(metrics.DatabaseSize()) / (1024 * 1024),
			"live_data_bytes": metrics.LiveDataSize(),
			"cache_bytes":     metrics.CacheSize(),
			"cache_mb":        float64(metrics.CacheSize()) / (1024 * 1024),
		},
		"cache": map[string]interface{}{
			"hit_rate": fmt.Sprintf("%.2f%%", metrics.CacheHitRate()*100),
		},
	}

	result["storage"] = storageInfo

	// Add database info
	result["database"] = map[string]interface{}{
		"path": dbPath,
	}

	return outputResult(result)
}
