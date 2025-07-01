# ProofBox CLI (pb)

The ProofBox CLI (`pb`) is a command-line interface for interacting with the Jellyfish Merkle Tree implementation. This tool allows you to store, retrieve, and verify data using cryptographic proofs.

## Table of Contents
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Basic Usage](#basic-usage)
- [Command Reference](#command-reference)
- [Examples](#examples)
- [Advanced Features](#advanced-features)
- [Troubleshooting](#troubleshooting)

## Installation

### Option 1: Install Pre-built Binary (Easiest)
```bash
# Install from source
go install github.com/neutral/proofbox/cmd/pb@latest

# Verify installation
pb --help
```

### Option 2: Build from Source
```bash
# Clone the repository
git clone https://github.com/neutral/proofbox.git
cd proofbox

# Build the CLI
go build -o pb ./cmd/pb

# Move to your PATH (optional)
sudo mv pb /usr/local/bin/

# Verify installation
pb --help
```

## Quick Start

Here's how to get started in under 2 minutes:

```bash
# 1. Create a new database
pb init --db mydata.db

# 2. Store your first key-value pair
pb put "my-first-key" "Hello, ProofBox!" --db mydata.db

# 3. Retrieve the value
pb get "my-first-key" --db mydata.db

# 4. Generate a cryptographic proof
pb prove "my-first-key" --db mydata.db --output proof.json

# 5. Verify the proof (works without the database!)
pb verify proof.json
```

## Basic Usage

### Creating a Database

Before storing any data, you need to initialize a database:

```bash
# Create a database named 'myapp.db'
pb init --db myapp.db

# Create a database in a specific directory
pb init --db /path/to/data/myapp.db

# Use memory backend (for testing)
pb init --db memory.db --backend memory
```

### Storing Data

Store key-value pairs in your database:

```bash
# Store a simple text value
pb put "user:alice" "Alice Smith" --db myapp.db

# Store JSON data
pb put "config:app" '{"theme": "dark", "lang": "en"}' --db myapp.db

# Store data from a file
echo "This is my document content" > document.txt
pb put "doc:readme" --file-value document.txt --db myapp.db

# Store with a hex-encoded key (must be exactly 64 hex characters)
pb put "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" "hex key value" --hex-key --db myapp.db
```

### Retrieving Data

Get values back from your database:

```bash
# Get a value
pb get "user:alice" --db myapp.db

# Get a value at a specific version
pb get "user:alice" --version 1 --db myapp.db

# Get with JSON output (useful for scripts)
pb get "user:alice" --db myapp.db --json

# Get with a hex-encoded key
pb get "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" --hex-key --db myapp.db
```

### Deleting Data

Remove keys from the database:

```bash
# Delete a key
pb delete "user:alice" --db myapp.db

# Delete with hex-encoded key
pb delete "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" --hex-key --db myapp.db
```

## Command Reference

### Core Commands

| Command | Description | Example |
|---------|-------------|---------|
| `init` | Create a new database | `pb init --db myapp.db` |
| `put` | Store a key-value pair | `pb put "key" "value" --db myapp.db` |
| `get` | Retrieve a value | `pb get "key" --db myapp.db` |
| `delete` | Remove a key | `pb delete "key" --db myapp.db` |
| `root` | Show the root hash | `pb root --db myapp.db` |
| `stats` | Display database statistics | `pb stats --db myapp.db` |

### Proof Commands

| Command | Description | Example |
|---------|-------------|---------|
| `prove` | Generate a Merkle proof | `pb prove "key" --db myapp.db --output proof.json` |
| `verify` | Verify a Merkle proof | `pb verify proof.json` |

### Batch Operations

| Command | Description | Example |
|---------|-------------|---------|
| `batch` | Execute multiple operations | `pb batch operations.json --db myapp.db` |
| `export-keys` | Export specific keys | `pb export-keys --keys keys.txt --db myapp.db --output data.json` |
| `import` | Import data from file | `pb import data.json --db myapp.db` |

### Interactive Mode

| Command | Description | Example |
|---------|-------------|---------|
| `repl` | Start interactive session | `pb repl --db myapp.db` |

## Examples

### Example 1: User Management System

```bash
# Initialize database
pb init --db users.db

# Add users
pb put "user:001" "John Doe" --db users.db
pb put "user:002" "Jane Smith" --db users.db
pb put "user:003" "Bob Johnson" --db users.db

# Add user metadata
pb put "meta:user:001" '{"role": "admin", "created": "2024-01-15"}' --db users.db
pb put "meta:user:002" '{"role": "user", "created": "2024-01-16"}' --db users.db

# Check database stats
pb stats --db users.db --json | jq .

# Export all user data
echo -e "user:001\nuser:002\nuser:003\nmeta:user:001\nmeta:user:002" > user_keys.txt
pb export-keys --keys user_keys.txt --db users.db --output users_backup.json
```

### Example 2: Configuration Management

```bash
# Initialize database
pb init --db config.db

# Store application configurations
pb put "config:database" '{"host": "localhost", "port": 5432}' --db config.db
pb put "config:cache" '{"ttl": 3600, "size": "100MB"}' --db config.db
pb put "config:api" '{"version": "v1", "rateLimit": 100}' --db config.db

# Generate proof for critical config
pb prove "config:database" --db config.db --output db_config_proof.json

# Verify the proof (can be done on another machine!)
pb verify db_config_proof.json
```

### Example 3: Batch Operations

Create a file `batch_ops.json`:
```json
{
  "operations": [
    {"type": "put", "key": "product:001", "value": "Laptop"},
    {"type": "put", "key": "product:002", "value": "Mouse"},
    {"type": "put", "key": "product:003", "value": "Keyboard"},
    {"type": "put", "key": "price:001", "value": "999.99"},
    {"type": "put", "key": "price:002", "value": "29.99"},
    {"type": "put", "key": "price:003", "value": "79.99"}
  ]
}
```

Execute the batch:
```bash
# Dry run first
pb batch batch_ops.json --db products.db --dry-run

# Execute the batch
pb batch batch_ops.json --db products.db

# View results in JSON
pb batch batch_ops.json --db products.db --json --verbose
```

### Example 4: Data Migration

```bash
# Export data from source database
echo -e "user:alice\nuser:bob\nuser:charlie" | pb export-keys --db source.db --output users.json

# Or export as CSV
echo -e "user:alice\nuser:bob\nuser:charlie" | pb export-keys --db source.db --format csv --output users.csv

# Import into new database
pb init --db target.db
pb import users.json --db target.db

# Verify the import
pb stats --db target.db --json
```

## Advanced Features

### Interactive REPL Mode

The REPL provides an interactive environment with command completion:

```bash
# Start REPL
pb repl --db myapp.db

# In REPL, you can run commands without the 'pb' prefix:
> put key1 value1
> put key2 value2
> get key1
> stats
> help
> exit
```

### Shell Completions

Enable tab completion for better productivity:

```bash
# For bash
pb completion bash > ~/.pb_completion
echo "source ~/.pb_completion" >> ~/.bashrc
source ~/.bashrc

# For zsh
pb completion zsh > ~/.pb_completion
echo "source ~/.pb_completion" >> ~/.zshrc
source ~/.zshrc

# Now you can use TAB to complete commands:
pb <TAB>           # Shows all commands
pb get --<TAB>     # Shows all flags for get command
```

### JSON Output Mode

Use JSON output for scripting and automation:

```bash
# Get value as JSON
pb get "key" --db myapp.db --json | jq -r .value

# Check if key exists
if pb get "key" --db myapp.db --json | jq -r .status | grep -q "found"; then
    echo "Key exists"
fi

# Get database size in MB
pb stats --db myapp.db --json | jq -r .storage.size.database_mb
```

### Working with Versions

ProofBox maintains a version history:

```bash
# See current version
pb stats --db myapp.db --json | jq .tree.latest

# Get value at specific version
pb get "key" --version 5 --db myapp.db

# Generate proof for specific version
pb prove "key" --version 5 --db myapp.db --output proof_v5.json
```

## Security Features

ProofBox includes several security features:

1. **Path Validation**: Prevents directory traversal attacks
2. **File Size Limits**: 100MB limit for import files
3. **Sanitized Logging**: Sensitive data is masked in verbose output
4. **Batch Size Limits**: Maximum 10,000 operations per batch

## Performance Tips

1. **Use Batch Operations** for bulk imports:
   ```bash
   # Good: Single batch operation
   pb batch large_import.json --db myapp.db
   
   # Bad: Many individual puts
   for i in {1..1000}; do pb put "key$i" "value$i" --db myapp.db; done
   ```

2. **Use JSON Output** for scripting to avoid parsing text:
   ```bash
   # Good: Parse JSON
   value=$(pb get "key" --db myapp.db --json | jq -r .value)
   
   # Bad: Parse text output
   value=$(pb get "key" --db myapp.db | grep "value:" | cut -d' ' -f2)
   ```

3. **Monitor Performance** with stats:
   ```bash
   pb stats --db myapp.db --json | jq '.storage.latency_us'
   ```

## Troubleshooting

### Common Issues

1. **"database not found"**
   ```bash
   # Make sure to initialize first
   pb init --db myapp.db
   ```

2. **"key must be 32 bytes"**
   ```bash
   # Hex keys must be exactly 64 characters (32 bytes)
   # Pad with zeros if needed
   pb put "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" "value" --hex-key --db myapp.db
   ```

3. **"file too large"**
   ```bash
   # Split large files before importing
   split -l 1000 large_file.json part_
   for file in part_*; do pb import "$file" --db myapp.db; done
   ```

4. **Permission Denied**
   ```bash
   # Check file permissions
   ls -la myapp.db
   chmod 644 myapp.db  # If needed
   ```

### Getting Help

```bash
# General help
pb --help

# Command-specific help
pb put --help
pb prove --help

# View all available commands
pb help
```

## Exit Codes

- `0`: Success
- `1`: Error (see error message for details)

## Environment Variables

- `PB_DB`: Default database path (optional)
  ```bash
  export PB_DB="/path/to/default.db"
  pb put "key" "value"  # Uses $PB_DB
  ```

## Contributing

For bug reports and feature requests, visit: https://github.com/neutral/proofbox/issues