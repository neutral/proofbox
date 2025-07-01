# CLI Command Reference

This page provides a complete reference for all ProofBox CLI commands, options, and usage patterns.

## Global Options

These options are available for all commands:

| Option | Description | Default |
|--------|-------------|---------|
| `--db PATH` | Database file path | `./pb.db` |
| `--json` | Output in JSON format | false |
| `--help`, `-h` | Show help for command | - |

> **Note**: By default, the CLI outputs data in Go map format (e.g., `map[key:value status:success]`). Use the `--json` flag for structured JSON output that can be processed with tools like `jq`.

## Commands

### `pb init`
Create a new ProofBox database

**Synopsis:**
```bash
pb init --db DATABASE_PATH [OPTIONS]
```

**Options:**
- `--db PATH` (required) - Path for the new database

**Example:**
```bash
pb init --db mydata.db
```

---

### `pb put`
Store a key-value pair

**Synopsis:**
```bash
pb put KEY VALUE --db DATABASE_PATH [OPTIONS]
```

**Options:**
- `--db PATH` (required) - Database path
- `--hex-key` - Interpret key as hexadecimal (must be 64 chars)
- `--file-value PATH` - Read value from file
- `--json` - Output result in JSON format

**Examples:**
```bash
# Simple put
pb put "name" "Alice" --db mydb.db

# Put with hex key
pb put "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" "data" --hex-key --db mydb.db

# Put file content
pb put "document" --file-value doc.txt --db mydb.db
```

---

### `pb get`
Retrieve a value by key

**Synopsis:**
```bash
pb get KEY --db DATABASE_PATH [OPTIONS]
```

**Options:**
- `--db PATH` (required) - Database path
- `--hex-key` - Interpret key as hexadecimal
- `--version N` - Get value at specific version
- `--json` - Output in JSON format

**Examples:**
```bash
# Simple get
pb get "name" --db mydb.db

# Get at specific version
pb get "name" --version 5 --db mydb.db

# Get with JSON output
pb get "name" --db mydb.db --json | jq -r .value
```

---

### `pb delete`
Delete a key-value pair

**Synopsis:**
```bash
pb delete KEY --db DATABASE_PATH [OPTIONS]
```

**Options:**
- `--db PATH` (required) - Database path
- `--hex-key` - Interpret key as hexadecimal
- `--json` - Output result in JSON format

**Example:**
```bash
pb delete "old_key" --db mydb.db
```

---

### `pb prove`
Generate a cryptographic proof for a key

**Synopsis:**
```bash
pb prove KEY --db DATABASE_PATH --output PROOF_FILE [OPTIONS]
```

**Options:**
- `--db PATH` (required) - Database path
- `--output PATH` (required) - Output file for proof
- `--hex-key` - Interpret key as hexadecimal
- `--version N` - Generate proof at specific version

**Examples:**
```bash
# Generate proof for current version
pb prove "account:123" --db mydb.db --output account_proof.json

# Generate proof for specific version
pb prove "account:123" --version 10 --db mydb.db --output account_proof_v10.json
```

---

### `pb verify`
Verify a cryptographic proof

**Synopsis:**
```bash
pb verify PROOF_FILE
```

**Example:**
```bash
pb verify account_proof.json
```

**Output:**
- Success: "Proof is valid" (exit code 0)
- Failure: Error message (exit code 1)

---

### `pb batch`
Execute batch operations atomically from a file

**Synopsis:**
```bash
pb batch OPERATIONS_FILE --db DATABASE_PATH [OPTIONS]
```

**Options:**
- `--db PATH` (required) - Database path
- `--dry-run` - Validate without executing
- `--verbose` - Show detailed results
- `--json` - Output results in JSON format

**Description:**
Executes multiple operations in a single atomic transaction. All operations succeed or all fail together. The entire batch creates only one new version in the database.

**Batch File Format:**
```json
{
  "operations": [
    {"type": "put", "key": "key1", "value": "value1"},
    {"type": "put", "key": "key2", "value": "value2"},
    {"type": "delete", "key": "key3"}
  ]
}
```

**Example:**
```bash
# Execute batch atomically
pb batch operations.json --db mydb.db --verbose

# Validate batch without executing
pb batch operations.json --db mydb.db --dry-run
```

---

### `pb export-keys`
Export values for specific keys

**Synopsis:**
```bash
pb export-keys --db DATABASE_PATH --output OUTPUT_FILE [OPTIONS]
```

**Options:**
- `--db PATH` (required) - Database path
- `--output PATH` (required) - Output file path
- `--keys PATH` - File containing keys (one per line)
- `--format FORMAT` - Output format: json (default) or csv
- `--version N` - Export at specific version

**Examples:**
```bash
# Export from stdin
echo -e "key1\nkey2" | pb export-keys --db mydb.db --output data.json

# Export from file
pb export-keys --keys mykeys.txt --db mydb.db --output data.json

# Export as CSV
pb export-keys --keys mykeys.txt --db mydb.db --format csv --output data.csv
```

---

### `pb import`
Import data atomically into database

**Synopsis:**
```bash
pb import DATA_FILE --db DATABASE_PATH [OPTIONS]
```

**Options:**
- `--db PATH` (required) - Database path
- `--format FORMAT` - Input format: json (default) or csv
- `--validate` - Validate without importing
- `--skip-errors` - Continue processing on validation errors (but still atomic)
- `--json` - Output results in JSON format

**Description:**
Imports key-value pairs from a file in a single atomic transaction. All entries are imported together - if any entry fails validation or import, no changes are made to the database.

**Examples:**
```bash
# Import JSON atomically
pb import data.json --db mydb.db

# Import CSV atomically
pb import data.csv --db mydb.db --format csv

# Validate only (no changes made)
pb import data.json --db mydb.db --validate
```

---

### `pb stats`
Show database statistics

**Synopsis:**
```bash
pb stats --db DATABASE_PATH [OPTIONS]
```

**Options:**
- `--db PATH` (required) - Database path
- `--version N` - Show stats for specific version
- `--json` - Output in JSON format

**Examples:**
```bash
# Human-readable stats
pb stats --db mydb.db

# JSON stats for scripting
pb stats --db mydb.db --json | jq .tree.height
```

---

### `pb root`
Show root hash of the tree

**Synopsis:**
```bash
pb root --db DATABASE_PATH [OPTIONS]
```

**Options:**
- `--db PATH` (required) - Database path
- `--version N` - Show root at specific version
- `--json` - Output in JSON format

**Example:**
```bash
pb root --db mydb.db
```

---

### `pb repl`
Start interactive REPL mode

**Synopsis:**
```bash
pb repl --db DATABASE_PATH [OPTIONS]
```

**Options:**
- `--db PATH` (required) - Database path

**REPL Commands:**
- `put KEY VALUE` - Store key-value pair
- `get KEY` - Retrieve value
- `delete KEY` - Delete key
- `stats` - Show statistics
- `root` - Show root hash
- `help` - Show REPL help
- `exit` or `quit` - Exit REPL

**Example:**
```bash
pb repl --db mydb.db
> put greeting "Hello, World!"
> get greeting
> stats
> exit
```

---

### `pb completion`
Generate shell completion scripts

**Synopsis:**
```bash
pb completion SHELL
```

**Supported Shells:**
- `bash`
- `zsh`
- `fish`
- `powershell`

**Examples:**
```bash
# Enable for current session
source <(pb completion bash)

# Install permanently (Linux)
pb completion bash > /etc/bash_completion.d/pb

# Install permanently (macOS with Homebrew)
pb completion bash > $(brew --prefix)/etc/bash_completion.d/pb
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (see stderr for details) |


## See Also

- [CLI Quick Start Tutorial](../../tutorials/cli-quickstart.md)
- [CLI How-to Guides](../../how-to/cli/index.md)
- [Configuration Reference](../configuration.md)