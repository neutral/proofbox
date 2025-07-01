# REPL Mode

This guide shows how to use ProofBox's interactive Read-Eval-Print Loop (REPL) for exploratory database operations.

## Goal

Use the interactive REPL session for quick database exploration, testing operations, and debugging.

## Prerequisites

- ProofBox CLI installed
- Database initialized and accessible
- Basic familiarity with ProofBox commands

## Starting REPL Mode

### Basic Launch

```bash
# Start REPL with default database
$ pb repl
ProofBox Interactive REPL
Database: ./pb.db
Version: 10
Type 'help' for available commands, 'exit' to quit

pb:./pb.db@v10> 

# Start REPL with specific database
$ pb repl --db mydata.db
ProofBox Interactive REPL
Database: mydata.db
Version: 5
Type 'help' for available commands, 'exit' to quit

pb:mydata.db@v5> 
```

### REPL Environment

The REPL provides:
- Command history (up/down arrows)
- Auto-completion (Tab key)
- Multi-line input support
- Session state persistence

## REPL Commands

### Basic Operations

```bash
# Put a value
pb> put user:100 "Alice Smith"
Put 'user:100' => 'Alice Smith' at version 11

# Get a value
pb> get user:100
'user:100' => 'Alice Smith'

# Get non-existent key
pb> get user:999
Key 'user:999' not found at version 11

# Delete a value
pb> delete user:100
Deleted 'user:100' at version 12
```

### Database Information

```bash
# Show current stats
pb> stats
Database: ./pb.db
Latest Version: 12
Root Hash: a8f3b2c4...
Node Count: 127
Tree Height: 4

# Get root hash
pb> root
Root hash at version 12: a8f3b2c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2
```

### Proof Operations

```bash
# Generate proof for existing key
pb> prove user:200
Proof generated for key 'user:200'
Type: inclusion
Value: Bob Jones
Siblings: 4
Root Hash: a8f3b2c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2

# Generate proof for non-existent key
pb> prove user:999
Proof generated for key 'user:999'
Type: non-inclusion
Siblings: 4
Root Hash: a8f3b2c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2
```

### Working with Versions

```bash
# Check current version
pb> version
Current version: 10

# Switch to specific version
pb> version 5
Switched to version: 5

# Prompt updates to show version
pb:./pb.db@v5> get user:100
'user:100' => 'Alice Smith'

# Switch back to latest
pb> version
Current version: 5
```

## Advanced REPL Features

### Multi-line Input

```bash
# Multi-line values must be on single line in REPL
pb> put config:app '{"name":"MyApp","version":"1.0","features":["auth","api"]}'
Put 'config:app' => '{"name":"MyApp","version":"1.0","features":["auth","api"]}' at version 13
```

### Batch Operations in REPL

```bash
# Execute batch from file
pb> batch operations.json
Batch execution complete
Total: 3 operations
Successful: 3
Failed: 0
Duration: 15ms
```

### Environment Commands

```bash
# Clear screen
pb> clear

# Show help
pb> help
Available commands:
  put <key> <value>  - Insert or update a key-value pair
  get <key>          - Retrieve value for a key
  delete <key>       - Delete a key
  root               - Show root hash
  prove <key>        - Generate proof for a key
  stats              - Show database statistics
  version [<num>]    - Show or set current version
  batch <file>       - Execute commands from file
  export <file>      - Export keys to file (interactive)
  import <file>      - Import data from file
  help               - Show this help
  clear              - Clear screen
  exit               - Exit REPL

# Exit REPL (or quit)
pb> exit
```

## Practical Usage Patterns

### Data Exploration

```bash
# Quick data checks
pb> get user:latest
pb> get config:version
pb> stats

# Test before implementing
pb> put test:key "test value"
Put 'test:key' => 'test value' at version 15
pb> get test:key
'test:key' => 'test value'
pb> delete test:key
Deleted 'test:key' at version 16
```

### Debugging Sessions

```bash
# Investigate issues
pb> get problematic:key
Key 'problematic:key' not found at version 16

pb> prove problematic:key
Proof generated for key 'problematic:key'
Type: non-inclusion
Siblings: 5

# Check different versions
pb> version 50
Switched to version: 50
pb> get problematic:key
'problematic:key' => 'old value'
```

### Performance Testing

```bash
# Performance testing (timing not built-in)
pb> put perf:test "x"
Put 'perf:test' => 'x' at version 17

# Manual timing by observation
pb> get perf:test
'perf:test' => 'x'

pb> prove perf:test
Proof generated for key 'perf:test'
Type: inclusion
```

## REPL Configuration

### Startup Options

```bash
# Verbose mode
$ pb repl --verbose
ProofBox Interactive REPL
Database: ./pb.db
Version: 10
Type 'help' for available commands, 'exit' to quit

pb:./pb.db@v10> 

# Note: Custom prompt not supported, prompt shows db path and version 
```

### History Management

```bash
# REPL saves command history
# Saved in ~/.pb_history

# Navigate history
↑ (up arrow)    - Previous command
↓ (down arrow)  - Next command
Ctrl+R          - Search history (if supported by readline)
```

## Tips and Tricks

### Efficient Workflows

1. **Test First**: Use REPL to test operations before scripting
2. **Explore Data**: Quickly check values without writing scripts
3. **Debug Issues**: Investigate problems interactively
4. **Learn Commands**: Experiment with options safely

### Common Patterns

```bash
# Check if key exists
pb> get mykey
pb> prove mykey

# Update and verify
pb> put mykey "new value"
Put 'mykey' => 'new value' at version 18
pb> get mykey
'mykey' => 'new value'

# Clean up tests (no wildcard support)
pb> delete test:1
pb> delete test:2
```

### REPL Scripts

Create REPL scripts for repeated tasks:

```bash
# Create batch file instead (REPL doesn't support input redirection)
# batch.json
{
  "operations": [
    {"type": "put", "key": "user:400", "value": "Script User"},
    {"type": "put", "key": "config:script", "value": "enabled"}
  ]
}

# Run in REPL
pb> batch batch.json
```

## Error Handling

### Common Errors

```bash
# Invalid command
pb> ptu user:100 value
Error: unknown command: ptu (type 'help' for available commands)

# Missing arguments
pb> put user:100
Error: usage: put <key> <value>

# Missing key
pb> get
Error: usage: get <key>
```

### Recovery

```bash
# Interrupt current line
pb> [Ctrl+C]
# Returns to prompt

# Exit REPL
pb> [Ctrl+D]
# Or type 'exit' or 'quit'
```

## Advanced Usage

### Import/Export in REPL

```bash
# Export keys (interactive)
pb> export backup.json
Enter keys to export (one per line, empty line to finish):
user:100
user:200

Exported 2 keys to backup.json

# Import data
pb> import data.json
Import complete: 10 succeeded, 0 failed
New version: 25
```

### File Operations

```bash
# Execute batch file
pb> batch commands.json
Batch execution complete

# Import/Export as shown above
# Note: Session save/load not supported
```

## Integration with Other Tools

### Piping to REPL

```bash
# REPL is interactive only, use batch files instead
# Create batch.json with operations
$ pb batch batch.json

# Or use individual commands
$ pb get user:100
$ pb stats
```

### Output Processing

```bash
# Use individual commands for scripting
$ pb get user:100 --json | jq '.value'

# Or batch operations
$ pb batch operations.json --json | jq .
```

## Best Practices

1. **Use for Exploration**: REPL is ideal for discovering data
2. **Not for Production**: Use scripts/programs for production operations  
3. **Test Safely**: Create test keys to experiment
4. **Keep History**: Useful commands can be converted to scripts
5. **Document Findings**: Note useful patterns discovered

## Troubleshooting

### REPL Won't Start

```bash
# Check database exists
$ ls ./pb.db

# Initialize if needed
$ pb init
$ pb repl
```

### Commands Not Working

```bash
# Check all commands with help
pb> help
[shows all available commands]

# Verify database connection
pb> stats
[shows database statistics]
```

### Performance Issues

```bash
# REPL operations should be fast
# If slow, check:
# - Database size
# - System resources
# - Network (if remote)
```

## Next Steps

- Use REPL findings to write [Batch Operations](batch-operations.md)
- Export discovered data with [Import/Export](import-export.md)
- Implement findings in your application

## Quick Reference

```bash
# Starting REPL
pb repl                  # Default database
pb repl --db <path>      # Specific database

# Basic commands
put <key> <value>        # Store value
get <key>                # Retrieve value  
delete <key>             # Remove key
prove <key>              # Generate proof
stats                    # Database info
root                     # Current root hash
version [<num>]          # Show/set version
batch <file>             # Execute batch file
export <file>            # Export keys
import <file>            # Import data
help                     # Show commands
clear                    # Clear screen
exit/quit                # Quit REPL

# Navigation
↑/↓                      # History
Tab                      # Auto-complete
Ctrl+C                   # Interrupt
Ctrl+D                   # Exit
```