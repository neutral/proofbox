# Basic CLI Usage

This guide covers essential ProofBox CLI operations for everyday use.

## Goal

Learn how to perform basic key-value operations using the ProofBox command-line interface.

## Prerequisites

- ProofBox CLI installed (`pb` command available)
- A ProofBox database directory (will be created automatically)

## Basic Commands

### Starting a Database

ProofBox automatically creates a database when you perform your first operation:

```bash
# Set a value (creates database if needed)
$ pb put mykey "Hello, ProofBox"
Key: mykey
Value: Hello, ProofBox
Version: 1
Status: inserted

# Specify a custom database location
$ pb put mykey "Hello" --db /path/to/database
```

### Setting Values

Store key-value pairs in the database:

```bash
# Put a simple value
$ pb put user:123 "John Doe"

# Put a JSON value
$ pb put config:app '{"name":"MyApp","version":"1.0"}'

# Update an existing value
$ pb put user:123 "Jane Doe"
Status: updated  # Note: shows 'updated' instead of 'inserted'
```

### Getting Values

Retrieve values by key:

```bash
# Get a value
$ pb get user:123
map[key:user:123 status:found value:Jane Doe version:2]

# Get a non-existent key
$ pb get user:999
map[key:user:999 reason:key not found status:not_found]

# Get with specific version
$ pb get user:123 --version 1
map[key:user:123 status:found value:John Doe version:1]
```

### Deleting Values

Remove key-value pairs:

```bash
# Delete a key
$ pb delete user:123
Key: user:123
Status: deleted
Version: 3

# Delete non-existent key
$ pb delete user:999
Key: user:999
Status: not_found
```

### Generating Proofs

Create cryptographic proofs for key existence:

```bash
# Generate proof for existing key
$ pb prove user:456
map[encoded:... key:user:456 root_hash:5f3a... siblings:[...] type:inclusion value:Test User version:5]

# Generate proof for non-existent key
$ pb prove user:nonexistent
map[encoded:... key:user:nonexistent root_hash:5f3a... siblings:[...] type:non_inclusion version:5]

# Proof output includes:
# - encoded: Serialized proof data
# - siblings: Array of sibling nodes in the proof path
# - type: inclusion (exists) or non_inclusion (doesn't exist)
```

### Exporting Keys

Export keys from the database:

```bash
# Export specific keys to a file
$ pb export-keys output.json user:123 user:456
map[exported:2 file:output.json status:success]

# Export all keys (requires listing keys first)
# Note: There's no built-in 'list all' command, use export-keys with specific keys
```

### Database Information

Get database statistics and information:

```bash
# Show database stats
$ pb stats
map[database:map[path:./pb.db] storage:map[backend:pebble ...] tree:map[height:0 latest:5 node_count:0 root_hash:5f3a4b... version:5]]

# Show CLI info
$ pb --help | head -n 1
ProofBox (pb) is a command-line interface
```

## Working with Different Data Types

### Text Data

```bash
$ pb put message "Hello World"
$ pb put document "Multi-line\ntext content"
```

### JSON Data

```bash
# Put JSON
$ pb put user:profile '{"name":"Alice","age":30,"email":"alice@example.com"}'

# Pretty-print JSON output
$ pb get user:profile --json | jq .
```

### Binary Data

```bash
# Put from file
$ pb put image:logo < logo.png

# Get to file
$ pb get image:logo > retrieved_logo.png
```

## Common Options

### Database Location

```bash
# Use custom database path
$ pb get mykey --db /custom/path/db

# Set via environment variable (not currently supported)
# export PB_DB=/custom/path/db
```

### Output Formats

```bash
# Default format (Go map)
$ pb get mykey
map[key:mykey status:found value:test version:1]

# JSON format with jq
$ pb get mykey --json | jq .
{
  "key": "mykey",
  "value": "test",
  "status": "found",
  "version": 1
}
```

### Version Control

```bash
# Always work with latest version (default)
$ pb put key value

# Query historical versions
$ pb get key --version 10

# List at historical version (not supported by current CLI)
# $ pb list --version 10
```

## Error Handling

Common errors and their meanings:

```bash
# Database errors
Error: failed to open database: permission denied

# Key not found
$ pb get nonexistent
map[key:nonexistent reason:key not found status:not_found]

# Invalid version
$ pb get key --version 999999
Error: version 999999 not found

# Value too large (>1MB)
$ pb put key < huge_file.dat
Error: value too large: maximum size is 1048576 bytes
```

## Best Practices

1. **Key Naming**: Use hierarchical keys like `user:123:profile` for organization
2. **Database Path**: Keep your database in a dedicated directory
3. **Backups**: Regularly backup your database directory
4. **Versions**: Remember that each write operation creates a new version
5. **Performance**: Use batch operations for multiple updates

## Troubleshooting

### Database Won't Open

```bash
# Check permissions
$ ls -la /path/to/db

# Ensure directory exists
$ mkdir -p /path/to/db
```

### Unexpected Output Format

The CLI uses Go's map format by default. For cleaner output:

```bash
# Use JSON format with jq
$ pb get key --json | jq -r .value
```

### Version Issues

```bash
# Check current version
$ pb stats | grep current_version

# Ensure version exists
$ pb get key --version 1
```

## Next Steps

- Learn about [Batch Operations](batch-operations.md) for efficient bulk updates
- Explore [Import and Export](import-export.md) for data migration
- Try [REPL Mode](repl-mode.md) for interactive sessions

## Quick Reference

```bash
# Basic operations
pb put <key> <value>         # Put a value
pb get <key>                 # Get a value
pb delete <key>              # Delete a value
pb prove <key>               # Generate proof
pb export-keys               # Export keys (no list command)
pb stats                     # Show statistics

# Common options
--db <path>                  # Database location
--version <n>                # Specific version
--json                       # JSON output
--limit <n>                  # Pagination limit
--offset <n>                 # Pagination offset
```