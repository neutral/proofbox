# ProofBox CLI Quick Start Guide

This tutorial will get you up and running with the ProofBox CLI in just 5 minutes. You'll learn the essential commands through hands-on examples.

## Prerequisites

- Command-line terminal access
- Go installed (for installation via `go install`) or ability to build from source

## Step 1: Install ProofBox CLI

Choose one of these installation methods:

```bash
# If you have Go installed:
go install github.com/neutral/proofbox/cmd/pb@latest

# Or build from source:
git clone https://github.com/neutral/proofbox.git
cd proofbox
go build -o pb ./cmd/pb
```

Verify installation:
```bash
pb --help
```

## Step 2: Your First Database

Let's create a database and store some data:

```bash
# Create a new database
pb init --db test.db

# Store some data
pb put "hello" "world" --db test.db
pb put "name" "ProofBox User" --db test.db
pb put "counter" "42" --db test.db

# Retrieve data
pb get "hello" --db test.db
pb get "name" --db test.db
```

> 📝 **Note**: The CLI outputs data in Go map format by default. For prettier output, use `--json` with `jq`:
> ```bash
> pb get "hello" --db test.db --json | jq .
> ```


## Step 3: Working with Proofs

ProofBox's key feature is generating cryptographic proofs for your data:

```bash
# Generate a cryptographic proof for your data
pb prove "hello" --db test.db --output hello_proof.json

# View the proof
cat hello_proof.json | jq .

# Verify the proof (this works even without the database!)
pb verify hello_proof.json
```

The proof demonstrates that the key "hello" has the value "world" at a specific version of the database.

## Step 4: Batch Operations

For bulk operations, create a file called `data.json`:

```json
{
  "operations": [
    {"type": "put", "key": "user:1", "value": "Alice"},
    {"type": "put", "key": "user:2", "value": "Bob"},
    {"type": "put", "key": "user:3", "value": "Charlie"},
    {"type": "put", "key": "score:1", "value": "100"},
    {"type": "put", "key": "score:2", "value": "95"},
    {"type": "put", "key": "score:3", "value": "87"}
  ]
}
```

Run the batch:
```bash
pb batch data.json --db test.db
```

## Step 5: Export and Import

Move data between databases:

```bash
# Export specific keys
echo -e "user:1\nuser:2\nuser:3" | pb export-keys --db test.db --output users.json

# View exported data
cat users.json | jq .

# Import into a new database
pb init --db test2.db
pb import users.json --db test2.db

# Verify the import worked
pb get "user:1" --db test2.db
```

## Interactive Mode (REPL)

For exploration and debugging, use the interactive mode:

```bash
# Start interactive session
pb repl --db test.db

# Now you can type commands directly:
pb> put key1 value1
pb> put key2 value2
pb> get key1
pb> delete key2
pb> stats
pb> help
pb> exit
```

## Quick Reference

| Task | Command |
|------|---------|
| Create database | `pb init --db mydb.db` |
| Store data | `pb put "key" "value" --db mydb.db` |
| Retrieve data | `pb get "key" --db mydb.db` |
| Delete data | `pb delete "key" --db mydb.db` |
| Database stats | `pb stats --db mydb.db` |
| Generate proof | `pb prove "key" --db mydb.db --output proof.json` |
| Verify proof | `pb verify proof.json` |
| Batch import | `pb batch operations.json --db mydb.db` |
| Interactive mode | `pb repl --db mydb.db` |

## Pro Tips

1. **Use JSON output for scripting**:
   ```bash
   value=$(pb get "key" --db mydb.db --json | jq -r .value)
   ```

2. **Check if key exists**:
   ```bash
   pb get "key" --db mydb.db --json | jq -r .status
   # Returns "found" or "not_found"
   ```

3. **Enable tab completion**:
   ```bash
   # For bash:
   source <(pb completion bash)
   
   # To persist across sessions on Linux:
   pb completion bash > /etc/bash_completion.d/pb
   
   # Or on macOS:
   pb completion bash > $(brew --prefix)/etc/bash_completion.d/pb
   ```

## What You've Learned

In this quick start, you've learned how to:
- ✅ Install and set up ProofBox CLI
- ✅ Create databases and store key-value data
- ✅ Generate and verify cryptographic proofs
- ✅ Perform batch operations
- ✅ Export and import data
- ✅ Use interactive REPL mode

## Next Steps

- Learn to [use ProofBox as a library](library-quickstart.md) in your Go applications
- Explore [advanced CLI usage](../how-to/cli/basic-usage.md) in the how-to guides
- Understand [how proofs work](../explanation/jellyfish-merkle-tree.md) in the explanation section

## Need Help?

- Command help: `pb <command> --help`
- Full CLI reference: [CLI Commands](../reference/cli/commands.md)
- Report issues: [GitHub Issues](https://github.com/neutral/proofbox/issues)