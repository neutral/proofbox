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

	"github.com/chzyer/readline"
	"github.com/neutral/proofbox/pkg/proof"
	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
	"github.com/spf13/cobra"
)

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Start an interactive REPL session",
	Long: `Start an interactive Read-Eval-Print Loop (REPL) session with the ProofBox database.
The REPL provides command completion and maintains the database connection for efficient operations.`,
	RunE: runREPL,
}

func init() {
	rootCmd.AddCommand(replCmd)
}

type replContext struct {
	tree    *tree.Tree
	version types.Version
}

func runREPL(cmd *cobra.Command, args []string) error {
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

	ctx := &replContext{
		tree:    tree,
		version: tree.GetLatestVersion(),
	}

	// Configure readline
	config := &readline.Config{
		Prompt:          fmt.Sprintf("pb:%s> ", dbPath),
		HistoryFile:     os.ExpandEnv("$HOME/.pb_history"),
		AutoComplete:    createCompleter(),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	}

	rl, err := readline.NewEx(config)
	if err != nil {
		return err
	}
	defer rl.Close()

	fmt.Println("ProofBox Interactive REPL")
	fmt.Printf("Database: %s\n", dbPath)
	fmt.Printf("Version: %d\n", ctx.version)
	fmt.Println("Type 'help' for available commands, 'exit' to quit")
	fmt.Println()

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			continue
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if err := executeReplCommand(ctx, line); err != nil {
			fmt.Printf("Error: %v\n", err)
		}

		// Update prompt with current version
		ctx.version = ctx.tree.GetLatestVersion()
		rl.SetPrompt(fmt.Sprintf("pb:%s@v%d> ", dbPath, ctx.version))
	}

	fmt.Println("\nGoodbye!")
	return nil
}

func createCompleter() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("put",
			readline.PcItem("<key>",
				readline.PcItem("<value>"),
			),
		),
		readline.PcItem("get",
			readline.PcItem("<key>"),
		),
		readline.PcItem("delete",
			readline.PcItem("<key>"),
		),
		readline.PcItem("root"),
		readline.PcItem("prove",
			readline.PcItem("<key>"),
		),
		readline.PcItem("stats"),
		readline.PcItem("version",
			readline.PcItem("latest"),
			readline.PcItem("<number>"),
		),
		readline.PcItem("batch",
			readline.PcItem("<file>"),
		),
		readline.PcItem("export",
			readline.PcItem("<file>"),
		),
		readline.PcItem("import",
			readline.PcItem("<file>"),
		),
		readline.PcItem("help"),
		readline.PcItem("clear"),
		readline.PcItem("exit"),
		readline.PcItem("quit"),
	)
}

func filterInput(r rune) (rune, bool) {
	switch r {
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

func executeReplCommand(ctx *replContext, line string) error {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "put":
		if len(parts) < 3 {
			return fmt.Errorf("usage: put <key> <value>")
		}
		return replPut(ctx, parts[1], strings.Join(parts[2:], " "))

	case "get":
		if len(parts) < 2 {
			return fmt.Errorf("usage: get <key>")
		}
		return replGet(ctx, parts[1])

	case "delete":
		if len(parts) < 2 {
			return fmt.Errorf("usage: delete <key>")
		}
		return replDelete(ctx, parts[1])

	case "root":
		return replRoot(ctx)

	case "prove":
		if len(parts) < 2 {
			return fmt.Errorf("usage: prove <key>")
		}
		return replProve(ctx, parts[1])

	case "stats":
		return replStats(ctx)

	case "version":
		switch {
		case len(parts) == 1:
			fmt.Printf("Current version: %d\n", ctx.version)
		case parts[1] == "latest":
			ctx.version = ctx.tree.GetLatestVersion()
			fmt.Printf("Switched to latest version: %d\n", ctx.version)
		default:
			var v uint64
			if _, err := fmt.Sscanf(parts[1], "%d", &v); err != nil {
				return fmt.Errorf("invalid version number: %s", parts[1])
			}
			ctx.version = types.Version(v)
			fmt.Printf("Switched to version: %d\n", ctx.version)
		}
		return nil

	case "batch":
		if len(parts) < 2 {
			return fmt.Errorf("usage: batch <file>")
		}
		return replBatch(ctx, parts[1])

	case "export":
		if len(parts) < 2 {
			return fmt.Errorf("usage: export <file>")
		}
		return replExportKeys(ctx, parts[1])

	case "import":
		if len(parts) < 2 {
			return fmt.Errorf("usage: import <file>")
		}
		return replImport(ctx, parts[1])

	case "help":
		printReplHelp()
		return nil

	case "clear":
		fmt.Print("\033[H\033[2J")
		return nil

	case "exit", "quit":
		os.Exit(0)

	default:
		return fmt.Errorf("unknown command: %s (type 'help' for available commands)", cmd)
	}

	return nil
}

func replPut(ctx *replContext, keyStr, value string) error {
	key := types.KeyHash([]byte(keyStr))
	newVersion, err := ctx.tree.Put(key, []byte(value))
	if err != nil {
		return err
	}
	fmt.Printf("Put '%s' => '%s' at version %d\n", keyStr, value, newVersion)
	return nil
}

func replGet(ctx *replContext, keyStr string) error {
	key := types.KeyHash([]byte(keyStr))
	value, err := ctx.tree.Get(ctx.version, key)
	if err != nil {
		return err
	}
	if value == nil {
		fmt.Printf("Key '%s' not found at version %d\n", keyStr, ctx.version)
	} else {
		fmt.Printf("'%s' => '%s'\n", keyStr, string(value))
	}
	return nil
}

func replDelete(ctx *replContext, keyStr string) error {
	key := types.KeyHash([]byte(keyStr))
	newVersion, err := ctx.tree.Delete(key)
	if err != nil {
		return err
	}
	fmt.Printf("Deleted '%s' at version %d\n", keyStr, newVersion)
	return nil
}

func replRoot(ctx *replContext) error {
	rootHash, err := ctx.tree.GetRootHash(ctx.version)
	if err != nil {
		return err
	}
	fmt.Printf("Root hash at version %d: %x\n", ctx.version, rootHash)
	return nil
}

func replProve(ctx *replContext, keyStr string) error {
	key := types.KeyHash([]byte(keyStr))

	// Create reader for the specified version
	reader, err := ctx.tree.Reader(ctx.version)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Close()

	// Generate proof
	generator := proof.NewGenerator(reader)
	merkleProof, err := generator.Generate(key)
	if err != nil {
		return fmt.Errorf("failed to generate proof: %w", err)
	}

	fmt.Printf("Proof for key '%s' at version %d:\n", keyStr, ctx.version)
	fmt.Printf("  Type: %s\n", proofTypeToString(merkleProof.Type))
	if merkleProof.Type == proof.ProofTypeInclusion {
		fmt.Printf("  Value: %s\n", string(merkleProof.Value))
	}
	fmt.Printf("  Siblings: %d\n", len(merkleProof.Siblings))
	fmt.Printf("  Root hash: %x\n", merkleProof.RootHash)
	return nil
}

func replStats(ctx *replContext) error {
	latestVersion := ctx.tree.GetLatestVersion()
	rootHash, err := ctx.tree.GetRootHash(latestVersion)
	if err != nil {
		return err
	}

	// Get tree statistics
	height, nodeCount, _ := ctx.tree.GetStats()

	fmt.Printf("Database Statistics:\n")
	fmt.Printf("  Latest version: %d\n", latestVersion)
	fmt.Printf("  Current version: %d\n", ctx.version)
	fmt.Printf("  Root hash: %x\n", rootHash[:8])
	fmt.Printf("\nTree Statistics:\n")
	fmt.Printf("  Height: %d\n", height)
	fmt.Printf("  Node count: %d\n", nodeCount)

	// Note: Storage metrics not available in REPL context without access to storage
	fmt.Printf("\n(Run 'pb stats' for detailed storage metrics)\n")
	return nil
}

func replBatch(ctx *replContext, filename string) error {
	// Validate file path
	if err := validatePath(filename); err != nil {
		return fmt.Errorf("invalid batch file path: %w", err)
	}

	// Check file size
	if err := validateFileSize(filename, maxImportFileSize); err != nil {
		return fmt.Errorf("batch file too large: %w", err)
	}

	// Read batch file
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open batch file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	succeeded := 0
	failed := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if err := executeReplCommand(ctx, line); err != nil {
			fmt.Printf("Line %d failed: %v\n", lineNum, err)
			failed++
		} else {
			succeeded++
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading batch file: %w", err)
	}

	fmt.Printf("\nBatch complete: %d succeeded, %d failed\n", succeeded, failed)
	return nil
}

func replExportKeys(ctx *replContext, filename string) error {
	fmt.Println("Enter keys to export (one per line, empty line to finish):")
	var keys []string
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}
		keys = append(keys, line)
	}

	if len(keys) == 0 {
		return fmt.Errorf("no keys provided")
	}

	// Create export data
	var entries []ExportEntry
	rootHash, err := ctx.tree.GetRootHash(ctx.version)
	if err != nil {
		return fmt.Errorf("failed to get root hash: %w", err)
	}

	// Export each key
	for _, keyStr := range keys {
		key := types.KeyHash([]byte(keyStr))
		value, err := ctx.tree.Get(ctx.version, key)
		if err != nil {
			fmt.Printf("Warning: key '%s' not found\n", keyStr)
			continue
		}
		if value != nil {
			entries = append(entries, ExportEntry{
				Key:   keyStr,
				Value: string(value),
			})
		}
	}

	if len(entries) == 0 {
		return fmt.Errorf("no valid keys found to export")
	}

	// Write to file
	exportData := ExportData{
		Version:    uint64(ctx.version),
		ExportTime: time.Now().UTC().Format(time.RFC3339),
		RootHash:   hex.EncodeToString(rootHash[:]),
		EntryCount: len(entries),
		Entries:    entries,
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(exportData); err != nil {
		return fmt.Errorf("failed to write export data: %w", err)
	}

	fmt.Printf("Exported %d keys to %s\n", len(entries), filename)
	return nil
}

func replImport(ctx *replContext, filename string) error {
	// Open import file
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open import file: %w", err)
	}
	defer file.Close()

	// Determine format by extension
	var entries []ExportEntry
	if strings.HasSuffix(filename, ".csv") {
		// CSV format
		csvReader := csv.NewReader(file)

		// Read header
		header, err := csvReader.Read()
		if err != nil {
			return fmt.Errorf("failed to read CSV header: %w", err)
		}
		if len(header) < 2 || header[0] != "key" || header[1] != "value" {
			return fmt.Errorf("invalid CSV header: expected 'key,value'")
		}

		// Read records
		for {
			record, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read CSV record: %w", err)
			}
			if len(record) >= 2 {
				entries = append(entries, ExportEntry{
					Key:   record[0],
					Value: record[1],
				})
			}
		}
	} else {
		// JSON format
		var exportData ExportData
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&exportData); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		entries = exportData.Entries
	}

	if len(entries) == 0 {
		return fmt.Errorf("no entries found in import file")
	}

	// Import entries
	imported := 0
	failed := 0
	for _, entry := range entries {
		key := types.KeyHash([]byte(entry.Key))
		value := []byte(entry.Value)

		if entry.ValueHex != "" {
			var err error
			value, err = hex.DecodeString(entry.ValueHex)
			if err != nil {
				fmt.Printf("Warning: failed to decode hex value for key '%s': %v\n", entry.Key, err)
				failed++
				continue
			}
		}

		_, err := ctx.tree.Put(key, value)
		if err != nil {
			fmt.Printf("Warning: failed to import key '%s': %v\n", entry.Key, err)
			failed++
		} else {
			imported++
		}
	}

	fmt.Printf("Import complete: %d succeeded, %d failed\n", imported, failed)
	if imported > 0 {
		fmt.Printf("New version: %d\n", ctx.tree.GetLatestVersion())
	}
	return nil
}

func printReplHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  put <key> <value>  - Insert or update a key-value pair")
	fmt.Println("  get <key>          - Retrieve value for a key")
	fmt.Println("  delete <key>       - Delete a key")
	fmt.Println("  root               - Show root hash")
	fmt.Println("  prove <key>        - Generate proof for a key")
	fmt.Println("  stats              - Show database statistics")
	fmt.Println("  version [<num>]    - Show or set current version")
	fmt.Println("  batch <file>       - Execute commands from file")
	fmt.Println("  export <file>      - Export keys to file (interactive)")
	fmt.Println("  import <file>      - Import data from file")
	fmt.Println("  help               - Show this help message")
	fmt.Println("  clear              - Clear the screen")
	fmt.Println("  exit/quit          - Exit the REPL")
}
