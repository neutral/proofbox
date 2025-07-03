package commands

import (
	"bufio"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/neutral/proofbox/pkg/types"
	"github.com/spf13/cobra"
)

var (
	keysFile     string
	keysFormat   string
	exportFormat string
	exportOutput string
	includeHex   bool
)

// ExportEntry represents a single key-value pair for export
type ExportEntry struct {
	Key      string `json:"key"`
	KeyHex   string `json:"key_hex,omitempty"`
	Value    string `json:"value"`
	ValueHex string `json:"value_hex,omitempty"`
}

// ExportData represents the complete export data structure
type ExportData struct {
	Version    uint64        `json:"version"`
	ExportTime string        `json:"export_time"`
	RootHash   string        `json:"root_hash"`
	EntryCount int           `json:"entry_count"`
	Entries    []ExportEntry `json:"entries"`
}

// exportKeysCmd represents the export-keys command
var exportKeysCmd = &cobra.Command{
	Use:   "export-keys",
	Short: "Export specific keys to a file",
	Long: `Export values for a list of specific keys.
Keys can be provided via stdin or a file, one per line.
Supports JSON and CSV output formats.

Example:
  echo -e "alice\nbob\ncharlie" | pb export-keys --db mydb.db --format json
  pb export-keys --keys keys.txt --db mydb.db --output data.json`,
	RunE: runExportKeys,
}

func init() {
	rootCmd.AddCommand(exportKeysCmd)
	exportKeysCmd.Flags().StringVar(&keysFile, "keys", "", "file containing keys to export (default: stdin)")
	exportKeysCmd.Flags().StringVar(&exportFormat, "format", "json", "export format (json or csv)")
	exportKeysCmd.Flags().StringVar(&exportOutput, "output", "", "output file (default: stdout)")
	exportKeysCmd.Flags().BoolVar(&includeHex, "hex", false, "include hex encoding of keys and values")
	exportKeysCmd.Flags().Uint64Var(&version, "version", 0, "version to export (default: latest)")
	exportKeysCmd.Flags().StringVar(&keysFormat, "keys-format", "text", "format of input keys (text or hex)")
}

func runExportKeys(cmd *cobra.Command, args []string) error {
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

	// Read keys from input
	var keyReader io.Reader
	if keysFile == "" {
		keyReader = os.Stdin
	} else {
		// Validate keys file path
		if err := validatePath(keysFile); err != nil {
			return fmt.Errorf("invalid keys file path: %w", err)
		}

		file, err := os.Open(keysFile)
		if err != nil {
			return fmt.Errorf("failed to open keys file: %w", err)
		}
		defer file.Close()
		keyReader = file
	}

	// Parse keys
	scanner := bufio.NewScanner(keyReader)
	var keys []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			keys = append(keys, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read keys: %w", err)
	}

	if len(keys) == 0 {
		return fmt.Errorf("no keys provided")
	}

	// Export entries
	entries := []ExportEntry{}
	notFound := 0
	startTime := time.Now()

	for _, keyStr := range keys {
		var key types.Key

		// Parse key based on format
		if keysFormat == "hex" {
			keyBytes, err := hex.DecodeString(keyStr)
			if err != nil {
				return fmt.Errorf("invalid hex key %s: %w", keyStr, err)
			}
			if len(keyBytes) != 32 {
				return fmt.Errorf("key must be 32 bytes, got %d", len(keyBytes))
			}
			copy(key[:], keyBytes)
		} else {
			key = types.KeyHash([]byte(keyStr))
		}

		// Get value
		value, err := tree.Get(types.Version(version), key)
		if err != nil {
			if verboseMode {
				// Sanitize key for logging
				sanitizedKey := sanitizeKey(keyStr, keysFormat == "hex")
				fmt.Fprintf(os.Stderr, "Key not found: %s\n", sanitizedKey)
			}
			notFound++
			continue
		}

		if value == nil {
			notFound++
			continue
		}

		entry := ExportEntry{
			Key:   keyStr,
			Value: string(value),
		}

		if includeHex {
			entry.KeyHex = hex.EncodeToString(key[:])
			entry.ValueHex = hex.EncodeToString(value)
		}

		entries = append(entries, entry)
	}

	// Prepare output writer
	var writer io.Writer
	if exportOutput == "" || exportOutput == "-" {
		writer = os.Stdout
	} else {
		// Validate output path
		if err := validatePath(exportOutput); err != nil {
			return fmt.Errorf("invalid export file path: %w", err)
		}

		file, err := os.Create(exportOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		writer = file
	}

	// Export based on format
	switch exportFormat {
	case "json":
		exportData := ExportData{
			Version:    version,
			ExportTime: time.Now().UTC().Format(time.RFC3339),
			RootHash:   hex.EncodeToString(rootHash[:]),
			EntryCount: len(entries),
			Entries:    entries,
		}

		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(exportData); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}

	case "csv":
		csvWriter := csv.NewWriter(writer)
		defer csvWriter.Flush()

		// Write header
		header := []string{"key", "value"}
		if includeHex {
			header = append(header, "key_hex", "value_hex")
		}
		if err := csvWriter.Write(header); err != nil {
			return fmt.Errorf("failed to write CSV header: %w", err)
		}

		// Write entries
		for _, entry := range entries {
			record := []string{entry.Key, entry.Value}
			if includeHex {
				record = append(record, entry.KeyHex, entry.ValueHex)
			}
			if err := csvWriter.Write(record); err != nil {
				return fmt.Errorf("failed to write CSV record: %w", err)
			}
		}

	default:
		return fmt.Errorf("unsupported format: %s", exportFormat)
	}

	// Report completion
	if !jsonOutput && exportOutput != "" && exportOutput != "-" {
		duration := time.Since(startTime)
		fmt.Printf("Exported %d entries (of %d requested) to %s in %v\n",
			len(entries), len(keys), exportOutput, duration)
		if notFound > 0 {
			fmt.Printf("Keys not found: %d\n", notFound)
		}
	}

	if jsonOutput && exportOutput != "" && exportOutput != "-" {
		result := map[string]interface{}{
			"status":      "success",
			"requested":   len(keys),
			"exported":    len(entries),
			"not_found":   notFound,
			"version":     version,
			"output_file": exportOutput,
			"duration":    time.Since(startTime).String(),
		}
		return outputResult(result)
	}

	return nil
}
