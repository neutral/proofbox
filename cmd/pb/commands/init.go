package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/storage/pebble"
	"github.com/neutral/proofbox/pkg/tree"
	"github.com/spf13/cobra"
)

var (
	backend string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new ProofBox database",
	Long: `Initialize a new ProofBox database at the specified path.
This creates the necessary directory structure and initializes
an empty Jellyfish Merkle Tree.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&backend, "backend", "pebble", "storage backend (pebble or memory)")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if database already exists
	if _, err := os.Stat(dbPath); err == nil {
		return fmt.Errorf("database already exists at %s", dbPath)
	}

	// Create database directory
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create storage backend
	var store storage.Storage
	var err error

	switch backend {
	case "pebble":
		store, err = pebble.NewStorage(dbPath, nil)
		if err != nil {
			return fmt.Errorf("failed to create pebble storage: %w", err)
		}

		// Set appropriate directory permissions (owner read/write/execute only)
		// This helps protect the database from unauthorized access
		// Note: directories need execute permission to be accessible
		if err := os.Chmod(dbPath, 0700); err != nil {
			// Log warning but don't fail - some filesystems don't support chmod
			if verboseMode {
				fmt.Fprintf(os.Stderr, "Warning: could not set database permissions: %v\n", err)
			}
		}
	case "memory":
		// For memory backend, we still need to create a marker file
		if err := os.WriteFile(dbPath, []byte("memory"), 0644); err != nil {
			return fmt.Errorf("failed to create marker file: %w", err)
		}
		return outputResult(map[string]string{
			"status":  "success",
			"backend": "memory",
			"path":    dbPath,
			"message": "Memory backend initialized (data will not persist)",
		})
	default:
		return fmt.Errorf("unsupported backend: %s", backend)
	}
	defer store.Close()

	// Create tree to initialize the database
	keyEncoder := storage.NewDefaultKeyEncoder()
	config := tree.DefaultTreeConfig()
	config.MetricsEnabled = false

	_, err = tree.NewTree(store, keyEncoder, config)
	if err != nil {
		return fmt.Errorf("failed to initialize tree: %w", err)
	}

	result := map[string]string{
		"status":  "success",
		"backend": backend,
		"path":    dbPath,
		"message": "Database initialized successfully",
	}

	return outputResult(result)
}
