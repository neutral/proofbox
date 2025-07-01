# ProofBox CLI Quick Start Guide

## 5-Minute Tutorial

Copy and paste these commands to get started with ProofBox:

### Step 1: Install ProofBox CLI

```bash
# If you have Go installed:
go install github.com/neutral/proofbox/cmd/pb@latest

# Or build from source:
git clone https://github.com/neutral/proofbox.git
cd proofbox
go build -o pb ./cmd/pb
```

### Step 2: Your First Database

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

### Step 3: Working with Proofs

```bash
# Generate a cryptographic proof for your data
pb prove "hello" --db test.db --output hello_proof.json

# View the proof
cat hello_proof.json | jq .

# Verify the proof (this works even without the database!)
pb verify hello_proof.json
```

### Step 4: Batch Operations

Create a file called `data.json`:
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

### Step 5: Export and Import

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

## Common Use Cases

### 1. Key-Value Store
```bash
# Store application settings
pb put "app:theme" "dark" --db app.db
pb put "app:language" "en" --db app.db
pb put "app:version" "1.0.0" --db app.db

# Retrieve settings
pb get "app:theme" --db app.db --json | jq -r .value
```

### 2. User Data Management
```bash
# Store user data
pb put "user:alice@example.com" '{"name": "Alice", "role": "admin"}' --db users.db
pb put "user:bob@example.com" '{"name": "Bob", "role": "user"}' --db users.db

# Generate proof of user existence
pb prove "user:alice@example.com" --db users.db --output alice_proof.json
```

### 3. Configuration Backup
```bash
# Export all config keys
echo -e "app:theme\napp:language\napp:version" > config_keys.txt
pb export-keys --keys config_keys.txt --db app.db --output config_backup.json

# Restore from backup
pb import config_backup.json --db app_restored.db
```

### 4. Interactive Mode (REPL)
```bash
# Start interactive session
pb repl --db test.db

# Now you can type commands directly:
> put key1 value1
> put key2 value2
> get key1
> delete key2
> stats
> help
> exit
```

## Useful Commands Reference

| What you want to do | Command |
|-------------------|---------|
| Create new database | `pb init --db mydb.db` |
| Store data | `pb put "key" "value" --db mydb.db` |
| Retrieve data | `pb get "key" --db mydb.db` |
| Delete data | `pb delete "key" --db mydb.db` |
| See database stats | `pb stats --db mydb.db` |
| Generate proof | `pb prove "key" --db mydb.db --output proof.json` |
| Verify proof | `pb verify proof.json` |
| Batch import | `pb batch operations.json --db mydb.db` |
| Export keys | `echo "key1" \| pb export-keys --db mydb.db --output data.json` |
| Import data | `pb import data.json --db mydb.db` |
| Interactive mode | `pb repl --db mydb.db` |
| Get help | `pb --help` or `pb <command> --help` |

## Pro Tips

1. **Use JSON output for scripts**:
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
   pb completion bash > ~/.pb_completion
   source ~/.pb_completion
   ```

4. **Set default database**:
   ```bash
   export PB_DB="/path/to/default.db"
   pb put "key" "value"  # No need for --db flag
   ```

## Need More Help?

- Full documentation: See [README.md](README.md)
- Command help: `pb <command> --help`
- Report issues: https://github.com/neutral/proofbox/issues