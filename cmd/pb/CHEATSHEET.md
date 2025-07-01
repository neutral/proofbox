# ProofBox CLI Cheat Sheet

## Essential Commands

### Database Operations
```bash
pb init --db mydb.db                    # Create new database
pb stats --db mydb.db                   # Show database statistics
pb stats --db mydb.db --json | jq .     # Stats in JSON format
```

### Basic CRUD Operations
```bash
pb put "key" "value" --db mydb.db       # Store key-value pair
pb get "key" --db mydb.db               # Retrieve value
pb delete "key" --db mydb.db            # Delete key
pb root --db mydb.db                    # Show root hash
```

### Proof Operations
```bash
pb prove "key" --db mydb.db --output proof.json    # Generate proof
pb verify proof.json                                # Verify proof
pb prove "key" --version 5 --db mydb.db            # Proof at version
```

### Batch Operations
```bash
pb batch ops.json --db mydb.db          # Execute batch operations
pb batch ops.json --db mydb.db --dry-run # Test without executing
pb batch ops.json --db mydb.db --verbose # Show detailed results
```

### Import/Export
```bash
# Export specific keys
echo -e "key1\nkey2" | pb export-keys --db mydb.db --output data.json
pb export-keys --keys keys.txt --db mydb.db --output data.json
pb export-keys --keys keys.txt --db mydb.db --format csv --output data.csv

# Import data
pb import data.json --db mydb.db
pb import data.csv --db mydb.db --format csv
pb import data.json --db mydb.db --validate  # Validate only
```

### Interactive REPL
```bash
pb repl --db mydb.db                    # Start interactive mode
# Inside REPL:
> put key value
> get key
> stats
> exit
```

## Advanced Usage

### Working with Hex Keys
```bash
# Use 64-character hex key (32 bytes)
pb put "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" "value" --hex-key --db mydb.db
pb get "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" --hex-key --db mydb.db
```

### File Operations
```bash
pb put "doc" --file-value document.txt --db mydb.db    # Store file content
pb get "doc" --db mydb.db > retrieved.txt              # Save to file
```

### Version Control
```bash
pb get "key" --version 10 --db mydb.db  # Get at specific version
pb prove "key" --version 10 --db mydb.db # Prove at version
pb stats --version 10 --db mydb.db      # Stats at version
```

### JSON Output (for Scripting)
```bash
pb get "key" --db mydb.db --json                      # JSON output
pb get "key" --db mydb.db --json | jq -r .value      # Extract value
pb stats --db mydb.db --json | jq .tree.latest       # Get latest version
pb stats --db mydb.db --json | jq .storage.size.database_mb  # DB size
```

## Batch File Format

### JSON Format
```json
{
  "operations": [
    {"type": "put", "key": "key1", "value": "value1"},
    {"type": "put", "key": "key2", "value": "value2"},
    {"type": "delete", "key": "key3"}
  ]
}
```

### Export Format
```json
{
  "version": 10,
  "export_time": "2024-01-15T10:30:00Z",
  "root_hash": "abcdef...",
  "entry_count": 2,
  "entries": [
    {"key": "key1", "value": "value1"},
    {"key": "key2", "value": "value2"}
  ]
}
```

### CSV Format
```csv
key,value
key1,value1
key2,value2
```

## Shell Completion

```bash
# Enable for current session
source <(pb completion bash)    # Bash
source <(pb completion zsh)      # Zsh

# Install permanently
pb completion bash > /etc/bash_completion.d/pb     # Linux
pb completion bash > $(brew --prefix)/etc/bash_completion.d/pb  # macOS
```

## Environment Variables

```bash
export PB_DB="/path/to/default.db"      # Default database path
```

## Common Patterns

### Check if Key Exists
```bash
if pb get "key" --db mydb.db --json | jq -e '.status == "found"' > /dev/null; then
    echo "Key exists"
else
    echo "Key not found"
fi
```

### Backup Database
```bash
# List all keys (you need to maintain a key list)
cat all_keys.txt | pb export-keys --db mydb.db --output backup.json
```

### Migrate Between Databases
```bash
# Export from source
echo -e "key1\nkey2\nkey3" | pb export-keys --db source.db --output data.json

# Import to target
pb init --db target.db
pb import data.json --db target.db
```

### Batch Delete Keys
```bash
# Create batch file
cat > delete_batch.json << EOF
{
  "operations": [
    {"type": "delete", "key": "temp:1"},
    {"type": "delete", "key": "temp:2"},
    {"type": "delete", "key": "temp:3"}
  ]
}
EOF

pb batch delete_batch.json --db mydb.db
```

## Performance Tips

- Use batch operations for bulk imports (up to 10,000 ops)
- Use JSON output mode when scripting
- Monitor performance with `pb stats --json | jq .storage.latency_us`
- Files larger than 100MB must be split before importing

## Exit Codes
- `0`: Success
- `1`: Error

## Quick Help
```bash
pb --help                 # General help
pb <command> --help       # Command help
pb help                   # List all commands
```