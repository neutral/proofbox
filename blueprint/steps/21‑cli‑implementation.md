---
id: step.21.cli-implementation
depends_on:
tags: [cli, interface, step]
---

## Objective

Implement the `jmtcli` command-line interface tool with subcommands for tree operations, proof generation/verification, and administrative functions.

## Implements

- **CLI Basic Requirement** (`/blueprint/features/cli/basic/requirement.md`)
- Provides user-friendly interface to JMT operations
- Enables testing and debugging of tree functionality

## Technical Details

### CLI Structure

Create `cmd/jmtcli/main.go`:

```go
package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "github.com/acme/jmt/internal/cli"
)

var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "jmtcli",
        Short: "Jellyfish Merkle Tree CLI",
        Long: `A command-line interface for interacting with Jellyfish Merkle Trees.

This tool provides commands for:
- Creating and managing JMT databases
- Inserting, updating, and deleting key-value pairs
- Generating and verifying Merkle proofs
- Inspecting tree structure and statistics`,
        Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
    }

    // Global flags
    var (
        dbPath   string
        verbose  bool
        format   string
    )

    rootCmd.PersistentFlags().StringVarP(&dbPath, "database", "d", "./jmt.db", "Path to JMT database")
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
    rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "text", "Output format (text, json, hex)")

    // Add subcommands
    rootCmd.AddCommand(
        cli.NewInitCommand(),
        cli.NewPutCommand(),
        cli.NewGetCommand(),
        cli.NewDeleteCommand(),
        cli.NewProveCommand(),
        cli.NewVerifyCommand(),
        cli.NewRootCommand(),
        cli.NewStatsCommand(),
        cli.NewBatchCommand(),
        cli.NewExportCommand(),
        cli.NewImportCommand(),
        cli.NewReplCommand(),
    )

    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### Core Commands Implementation

Create `internal/cli/commands.go`:

```go
package cli

import (
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "os"

    "github.com/spf13/cobra"
    "github.com/acme/jmt/pkg/tree"
    "github.com/acme/jmt/pkg/types"
    "github.com/acme/jmt/pkg/storage/pebble"
)

// Context holds shared state for commands
type Context struct {
    DB      *pebble.DB
    Tree    *tree.Tree
    Verbose bool
    Format  string
}

// NewInitCommand creates the init subcommand
func NewInitCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "init",
        Short: "Initialize a new JMT database",
        RunE: func(cmd *cobra.Command, args []string) error {
            dbPath, _ := cmd.Flags().GetString("database")

            // Check if database already exists
            if _, err := os.Stat(dbPath); err == nil {
                return fmt.Errorf("database already exists at %s", dbPath)
            }

            // Create new database
            db, err := pebble.Open(dbPath, nil)
            if err != nil {
                return fmt.Errorf("failed to create database: %w", err)
            }
            defer db.Close()

            // Initialize empty tree
            tree := tree.New(db)
            if err := tree.Initialize(); err != nil {
                return fmt.Errorf("failed to initialize tree: %w", err)
            }

            fmt.Printf("Initialized empty JMT database at %s\n", dbPath)
            return nil
        },
    }
}

// NewPutCommand creates the put subcommand
func NewPutCommand() *cobra.Command {
    var (
        keyStr   string
        valueStr string
        keyFile  string
        valueFile string
    )

    cmd := &cobra.Command{
        Use:   "put",
        Short: "Insert or update a key-value pair",
        Example: `  jmtcli put --key="user:123" --value="John Doe"
  jmtcli put --key-file=key.bin --value-file=data.json`,
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx, err := newContext(cmd)
            if err != nil {
                return err
            }
            defer ctx.Close()

            // Parse key
            key, err := parseKey(keyStr, keyFile)
            if err != nil {
                return fmt.Errorf("invalid key: %w", err)
            }

            // Parse value
            value, err := parseValue(valueStr, valueFile)
            if err != nil {
                return fmt.Errorf("invalid value: %w", err)
            }

            // Perform put operation
            version, err := ctx.Tree.Put(key, value)
            if err != nil {
                return fmt.Errorf("put failed: %w", err)
            }

            // Output result
            result := map[string]interface{}{
                "version": version,
                "key":     formatKey(key, ctx.Format),
                "size":    len(value),
            }

            return ctx.Output(result)
        },
    }

    cmd.Flags().StringVar(&keyStr, "key", "", "Key string (will be hashed)")
    cmd.Flags().StringVar(&valueStr, "value", "", "Value string")
    cmd.Flags().StringVar(&keyFile, "key-file", "", "Read key from file")
    cmd.Flags().StringVar(&valueFile, "value-file", "", "Read value from file")

    return cmd
}

// NewGetCommand creates the get subcommand
func NewGetCommand() *cobra.Command {
    var (
        keyStr  string
        keyFile string
        version uint64
    )

    cmd := &cobra.Command{
        Use:   "get",
        Short: "Retrieve a value by key",
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx, err := newContext(cmd)
            if err != nil {
                return err
            }
            defer ctx.Close()

            // Parse key
            key, err := parseKey(keyStr, keyFile)
            if err != nil {
                return fmt.Errorf("invalid key: %w", err)
            }

            // Use specified version or latest
            var value []byte
            if version > 0 {
                value, err = ctx.Tree.GetAtVersion(types.Version(version), key)
            } else {
                value, err = ctx.Tree.Get(key)
            }

            if err != nil {
                if errors.Is(err, types.ErrKeyNotFound) {
                    return fmt.Errorf("key not found")
                }
                return fmt.Errorf("get failed: %w", err)
            }

            // Output result
            result := map[string]interface{}{
                "key":   formatKey(key, ctx.Format),
                "value": formatValue(value, ctx.Format),
                "size":  len(value),
            }

            if version > 0 {
                result["version"] = version
            }

            return ctx.Output(result)
        },
    }

    cmd.Flags().StringVar(&keyStr, "key", "", "Key string")
    cmd.Flags().StringVar(&keyFile, "key-file", "", "Read key from file")
    cmd.Flags().Uint64Var(&version, "version", 0, "Specific version to query")

    return cmd
}

// NewProveCommand creates the prove subcommand
func NewProveCommand() *cobra.Command {
    var (
        keyStr  string
        keyFile string
        output  string
    )

    cmd := &cobra.Command{
        Use:   "prove",
        Short: "Generate a Merkle proof for a key",
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx, err := newContext(cmd)
            if err != nil {
                return err
            }
            defer ctx.Close()

            // Parse key
            key, err := parseKey(keyStr, keyFile)
            if err != nil {
                return fmt.Errorf("invalid key: %w", err)
            }

            // Generate proof
            proof, err := ctx.Tree.GenerateProof(key)
            if err != nil {
                return fmt.Errorf("proof generation failed: %w", err)
            }

            // Serialize proof
            proofData, err := json.Marshal(proof)
            if err != nil {
                return fmt.Errorf("failed to serialize proof: %w", err)
            }

            // Output to file or stdout
            if output != "" {
                if err := os.WriteFile(output, proofData, 0644); err != nil {
                    return fmt.Errorf("failed to write proof: %w", err)
                }
                fmt.Printf("Proof written to %s\n", output)
            } else {
                fmt.Println(string(proofData))
            }

            return nil
        },
    }

    cmd.Flags().StringVar(&keyStr, "key", "", "Key string")
    cmd.Flags().StringVar(&keyFile, "key-file", "", "Read key from file")
    cmd.Flags().StringVarP(&output, "output", "o", "", "Output proof to file")

    return cmd
}

// NewVerifyCommand creates the verify subcommand
func NewVerifyCommand() *cobra.Command {
    var proofFile string

    cmd := &cobra.Command{
        Use:   "verify",
        Short: "Verify a Merkle proof",
        Args:  cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            // Read proof from file or stdin
            var proofData []byte
            var err error

            if proofFile != "" {
                proofData, err = os.ReadFile(proofFile)
            } else if len(args) > 0 {
                proofData = []byte(args[0])
            } else {
                proofData, err = io.ReadAll(os.Stdin)
            }

            if err != nil {
                return fmt.Errorf("failed to read proof: %w", err)
            }

            // Parse proof
            var proof types.Proof
            if err := json.Unmarshal(proofData, &proof); err != nil {
                return fmt.Errorf("invalid proof format: %w", err)
            }

            // Verify proof
            verifier := proof.NewVerifier()
            if err := verifier.Verify(&proof); err != nil {
                fmt.Println("❌ Proof verification FAILED:", err)
                return err
            }

            fmt.Println("✅ Proof verification PASSED")
            return nil
        },
    }

    cmd.Flags().StringVarP(&proofFile, "file", "f", "", "Read proof from file")

    return cmd
}

// NewBatchCommand creates the batch subcommand
func NewBatchCommand() *cobra.Command {
    var (
        scriptFile string
        format     string
    )

    cmd := &cobra.Command{
        Use:   "batch",
        Short: "Execute batch operations from file",
        Long: `Execute multiple operations from a file.

File format (JSON):
[
  {"op": "put", "key": "key1", "value": "value1"},
  {"op": "put", "key": "key2", "value": "value2"},
  {"op": "delete", "key": "key3"}
]`,
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx, err := newContext(cmd)
            if err != nil {
                return err
            }
            defer ctx.Close()

            // Read batch file
            data, err := os.ReadFile(scriptFile)
            if err != nil {
                return fmt.Errorf("failed to read batch file: %w", err)
            }

            // Parse operations
            var ops []BatchOperation
            if err := json.Unmarshal(data, &ops); err != nil {
                return fmt.Errorf("invalid batch format: %w", err)
            }

            // Execute batch
            batch := ctx.Tree.NewBatch()
            for i, op := range ops {
                key := types.KeyHash([]byte(op.Key))

                switch op.Op {
                case "put":
                    if err := batch.Put(key, []byte(op.Value)); err != nil {
                        return fmt.Errorf("operation %d failed: %w", i, err)
                    }
                case "delete":
                    if err := batch.Delete(key); err != nil {
                        return fmt.Errorf("operation %d failed: %w", i, err)
                    }
                default:
                    return fmt.Errorf("unknown operation: %s", op.Op)
                }
            }

            // Commit batch
            version, err := batch.Commit()
            if err != nil {
                return fmt.Errorf("batch commit failed: %w", err)
            }

            fmt.Printf("Batch committed successfully at version %d (%d operations)\n", version, len(ops))
            return nil
        },
    }

    cmd.Flags().StringVarP(&scriptFile, "file", "f", "", "Batch operations file")
    cmd.MarkFlagRequired("file")

    return cmd
}

// NewReplCommand creates an interactive REPL
func NewReplCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "repl",
        Short: "Start interactive REPL session",
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx, err := newContext(cmd)
            if err != nil {
                return err
            }
            defer ctx.Close()

            repl := NewRepl(ctx)
            return repl.Run()
        },
    }
}
```

### REPL Implementation

Create `internal/cli/repl.go`:

```go
package cli

import (
    "bufio"
    "fmt"
    "os"
    "strings"

    "github.com/c-bata/go-prompt"
)

// Repl provides an interactive command interface
type Repl struct {
    ctx      *Context
    history  []string
    txActive bool
    batch    *tree.Batch
}

// NewRepl creates a new REPL instance
func NewRepl(ctx *Context) *Repl {
    return &Repl{
        ctx:     ctx,
        history: make([]string, 0),
    }
}

// Run starts the REPL
func (r *Repl) Run() error {
    fmt.Println("JMT Interactive Shell")
    fmt.Println("Type 'help' for commands, 'exit' to quit")
    fmt.Println()

    p := prompt.New(
        r.executor,
        r.completer,
        prompt.OptionPrefix("jmt> "),
        prompt.OptionHistory(r.history),
        prompt.OptionTitle("JMT REPL"),
    )

    p.Run()
    return nil
}

// executor processes commands
func (r *Repl) executor(line string) {
    line = strings.TrimSpace(line)
    if line == "" {
        return
    }

    r.history = append(r.history, line)
    parts := strings.Fields(line)

    if len(parts) == 0 {
        return
    }

    cmd := parts[0]
    args := parts[1:]

    switch cmd {
    case "help":
        r.showHelp()
    case "exit", "quit":
        fmt.Println("Goodbye!")
        os.Exit(0)
    case "put":
        r.cmdPut(args)
    case "get":
        r.cmdGet(args)
    case "delete":
        r.cmdDelete(args)
    case "prove":
        r.cmdProve(args)
    case "root":
        r.cmdRoot(args)
    case "stats":
        r.cmdStats(args)
    case "begin":
        r.cmdBeginTx(args)
    case "commit":
        r.cmdCommitTx(args)
    case "rollback":
        r.cmdRollbackTx(args)
    default:
        fmt.Printf("Unknown command: %s\n", cmd)
    }
}

// completer provides command completion
func (r *Repl) completer(d prompt.Document) []prompt.Suggest {
    s := []prompt.Suggest{
        {Text: "put", Description: "Insert or update a key-value pair"},
        {Text: "get", Description: "Retrieve a value by key"},
        {Text: "delete", Description: "Delete a key"},
        {Text: "prove", Description: "Generate a Merkle proof"},
        {Text: "root", Description: "Show current root hash"},
        {Text: "stats", Description: "Show tree statistics"},
        {Text: "begin", Description: "Start a transaction"},
        {Text: "commit", Description: "Commit current transaction"},
        {Text: "rollback", Description: "Rollback current transaction"},
        {Text: "help", Description: "Show help"},
        {Text: "exit", Description: "Exit the REPL"},
    }
    return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

// Command implementations
func (r *Repl) cmdPut(args []string) {
    if len(args) < 2 {
        fmt.Println("Usage: put <key> <value>")
        return
    }

    key := types.KeyHash([]byte(args[0]))
    value := []byte(strings.Join(args[1:], " "))

    if r.txActive {
        if err := r.batch.Put(key, value); err != nil {
            fmt.Printf("Error: %v\n", err)
            return
        }
        fmt.Println("Added to transaction")
    } else {
        version, err := r.ctx.Tree.Put(key, value)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            return
        }
        fmt.Printf("Stored at version %d\n", version)
    }
}

func (r *Repl) cmdGet(args []string) {
    if len(args) < 1 {
        fmt.Println("Usage: get <key>")
        return
    }

    key := types.KeyHash([]byte(args[0]))
    value, err := r.ctx.Tree.Get(key)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Value: %s\n", string(value))
}

func (r *Repl) cmdBeginTx(args []string) {
    if r.txActive {
        fmt.Println("Transaction already active")
        return
    }

    r.batch = r.ctx.Tree.NewBatch()
    r.txActive = true
    fmt.Println("Transaction started")
}

func (r *Repl) cmdCommitTx(args []string) {
    if !r.txActive {
        fmt.Println("No active transaction")
        return
    }

    version, err := r.batch.Commit()
    if err != nil {
        fmt.Printf("Commit failed: %v\n", err)
        return
    }

    r.txActive = false
    r.batch = nil
    fmt.Printf("Transaction committed at version %d\n", version)
}
```

## Testing Requirements

### CLI Integration Tests

```go
func TestCLICommands(t *testing.T) {
    // Create temporary database
    tmpDir := t.TempDir()
    dbPath := filepath.Join(tmpDir, "test.db")

    // Test init command
    cmd := exec.Command("jmtcli", "init", "-d", dbPath)
    output, err := cmd.CombinedOutput()
    assert.NoError(t, err)
    assert.Contains(t, string(output), "Initialized")

    // Test put command
    cmd = exec.Command("jmtcli", "put", "-d", dbPath, "--key=test", "--value=hello")
    output, err = cmd.CombinedOutput()
    assert.NoError(t, err)

    // Test get command
    cmd = exec.Command("jmtcli", "get", "-d", dbPath, "--key=test")
    output, err = cmd.CombinedOutput()
    assert.NoError(t, err)
    assert.Contains(t, string(output), "hello")
}

func TestBatchOperations(t *testing.T) {
    // Create batch file
    batch := []BatchOperation{
        {Op: "put", Key: "key1", Value: "value1"},
        {Op: "put", Key: "key2", Value: "value2"},
        {Op: "delete", Key: "key1"},
    }

    batchFile := filepath.Join(t.TempDir(), "batch.json")
    data, _ := json.Marshal(batch)
    os.WriteFile(batchFile, data, 0644)

    // Execute batch
    cmd := exec.Command("jmtcli", "batch", "-f", batchFile)
    output, err := cmd.CombinedOutput()
    assert.NoError(t, err)
    assert.Contains(t, string(output), "3 operations")
}
```

## Implementation Steps

1. Create main CLI entry point in `cmd/jmtcli/main.go`
2. Implement core commands (init, put, get, delete)
3. Add proof commands (prove, verify)
4. Implement batch operations
5. Create interactive REPL
6. Add administrative commands (stats, export, import)
7. Write integration tests
8. Add shell completion scripts
9. Create man pages and documentation

## Performance Considerations

- Use buffered I/O for large file operations
- Implement streaming for export/import
- Cache database connections in REPL mode
- Optimize batch operations with single transaction

## Security Notes

- Validate all user inputs
- Sanitize file paths to prevent directory traversal
- Set appropriate file permissions on database
- Add option for encrypted value storage

## Done When ✓

- [ ] Main CLI structure with cobra
- [ ] Core commands: init, put, get, delete
- [ ] Proof commands: prove, verify
- [ ] Root hash and stats commands
- [ ] Batch operations from JSON file
- [ ] Interactive REPL with completion
- [ ] Export/import functionality
- [ ] Comprehensive help text
- [ ] Integration tests for all commands
- [ ] Shell completion scripts (bash, zsh)
- [ ] Performance acceptable for large operations
- [ ] Security considerations addressed
