// Package commands implements all CLI commands for ProofBox
package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	dbPath      string
	jsonOutput  bool
	hexKey      bool
	verboseMode bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pb",
	Short: "ProofBox CLI - A Jellyfish Merkle Tree implementation",
	Long: `ProofBox (pb) is a command-line interface for interacting with
a Jellyfish Merkle Tree implementation. It provides commands for
tree operations, proof generation/verification, and management.`,
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "./pb.db", "database path")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&verboseMode, "verbose", false, "verbose output")
}

// outputResult prints the result in either JSON or human-readable format
func outputResult(result interface{}) error {
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	// Human-readable output
	switch v := result.(type) {
	case string:
		fmt.Println(v)
	case []byte:
		fmt.Println(string(v))
	default:
		fmt.Printf("%+v\n", v)
	}
	return nil
}

// outputError handles error output
func outputError(err error) {
	if jsonOutput {
		result := map[string]string{
			"error": err.Error(),
		}
		encoder := json.NewEncoder(os.Stderr)
		encoder.SetIndent("", "  ")
		encoder.Encode(result)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}
