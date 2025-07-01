# Local Usage Guide

This guide explains how to use ProofBox in its current form - as a local CLI tool and Go library.

## Goal

Set up and use ProofBox for local development and testing with the Jellyfish Merkle Tree.

## What ProofBox Currently Offers

ProofBox is a local tool that provides:
- CLI interface for merkle tree operations
- Go library for embedding in applications
- Local PebbleDB storage
- Cryptographic proof generation

## Installation Options

### Option 1: Install CLI with Go

```bash
# Requires Go 1.21+
go install github.com/neutral/proofbox/cmd/pb@latest

# Verify installation
pb --help
```

### Option 2: Build from Source

```bash
# Clone repository
git clone https://github.com/neutral/proofbox.git
cd proofbox

# Build
go build -o pb ./cmd/pb

# Move to PATH (optional)
sudo mv pb /usr/local/bin/
```

### Option 3: Use as Go Library

In your Go project:

```bash
go get github.com/neutral/proofbox
```

## Local CLI Usage

### Initialize Database

```bash
# Create a new database file
pb init --db myapp.db

# Or use default location
pb init  # Creates ./pb.db
```

### Basic Operations

```bash
# Store data
pb put "user:123" "John Doe" --db myapp.db
pb put "config:version" "1.0.0" --db myapp.db

# Retrieve data
pb get "user:123" --db myapp.db

# Delete data
pb delete "config:old" --db myapp.db

# View statistics
pb stats --db myapp.db
```

### Batch Operations

Create a batch file `operations.json`:

```json
{
  "operations": [
    {"type": "put", "key": "item:1", "value": "First item"},
    {"type": "put", "key": "item:2", "value": "Second item"},
    {"type": "delete", "key": "item:old"}
  ]
}
```

Execute:
```bash
pb batch operations.json --db myapp.db
```

### Generate and Verify Proofs

```bash
# Generate proof
pb prove "user:123" --db myapp.db --output proof.json

# Verify proof (works without database!)
pb verify proof.json
```

### Interactive REPL

```bash
# Start interactive session
pb repl --db myapp.db

# In REPL:
pb> put test:key "test value"
pb> get test:key
pb> stats
pb> exit
```

## Using as a Go Library

### Basic Example

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/neutral/proofbox/pkg/storage"
    "github.com/neutral/proofbox/pkg/storage/pebble"
    "github.com/neutral/proofbox/pkg/tree"
    "github.com/neutral/proofbox/pkg/types"
)

func main() {
    // Open database
    store, err := pebble.NewStorage("./myapp.db", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Create tree
    keyEncoder := storage.NewDefaultKeyEncoder()
    config := tree.DefaultTreeConfig()
    jmt, err := tree.NewTree(store, keyEncoder, config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Store data
    key := types.KeyHash([]byte("hello"))
    value := []byte("world")
    version, err := jmt.Put(key, value)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Stored at version %d\n", version)
    
    // Retrieve data
    retrieved, err := jmt.Get(key)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Retrieved: %s\n", string(retrieved))
}
```

### Embedding in Web Application

```go
package main

import (
    "encoding/json"
    "log"
    "net/http"
    
    "github.com/neutral/proofbox/pkg/storage/pebble"
    "github.com/neutral/proofbox/pkg/tree"
    "github.com/neutral/proofbox/pkg/types"
)

type Server struct {
    tree *tree.Tree
}

func (s *Server) handlePut(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Key   string `json:"key"`
        Value string `json:"value"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    key := types.KeyHash([]byte(req.Key))
    version, err := s.tree.Put(key, []byte(req.Value))
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "version": version,
    })
}

func main() {
    // Initialize ProofBox
    store, _ := pebble.NewStorage("./webapp.db", nil)
    defer store.Close()
    
    keyEncoder := storage.NewDefaultKeyEncoder()
    jmt, _ := tree.NewTree(store, keyEncoder, tree.DefaultTreeConfig())
    
    server := &Server{tree: jmt}
    
    // Set up routes
    http.HandleFunc("/put", server.handlePut)
    // Add more handlers...
    
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Data Management

### Backup Your Data

Since ProofBox uses local files, backup is straightforward:

```bash
# Export specific keys
echo -e "important:1\nimportant:2" | pb export-keys --db myapp.db --output backup.json

# Or backup the entire database file
cp myapp.db myapp.db.backup

# For time-based backups
cp myapp.db "myapp-$(date +%Y%m%d-%H%M%S).db"
```

### Import Data

```bash
# From JSON
pb import data.json --db myapp.db

# From CSV
pb import data.csv --db myapp.db --format csv
```

## Development Workflow

### 1. Local Development

```bash
# Create test database
pb init --db test.db

# Run your tests
go test ./...

# Clean up
rm test.db
```

### 2. Integration Testing

```go
func setupTestDB(t *testing.T) *tree.Tree {
    // Use in-memory storage for tests
    store := memory.NewStorage()
    keyEncoder := storage.NewDefaultKeyEncoder()
    jmt, err := tree.NewTree(store, keyEncoder, tree.DefaultTreeConfig())
    require.NoError(t, err)
    return jmt
}
```

### 3. Performance Testing

```bash
# Use the benchmark suite
cd proofbox
go test -bench=. ./bench/...

# Or create test data
pb batch large-dataset.json --db perf-test.db
pb stats --db perf-test.db
```

## Best Practices

### 1. Database Location
- Keep databases in a dedicated directory
- Use descriptive names
- Regular backups

```bash
mkdir -p ~/proofbox-data
pb init --db ~/proofbox-data/production.db
```

### 2. Key Naming
- Use prefixes for organization: `user:123`, `config:setting`
- Keep keys under 32 bytes when possible
- Use consistent naming schemes

### 3. Error Handling
Always check for errors, especially:
- Database initialization
- Put/Get operations
- Proof generation

### 4. Concurrent Access
- ProofBox handles concurrent reads well
- Serialize writes from multiple processes
- Use batch operations for bulk updates

## Limitations

Current ProofBox limitations:
1. **Local only** - No network API
2. **Single database** - Each file is independent
3. **No replication** - Manual backup needed
4. **File-based** - Not suitable for distributed systems

## When to Use ProofBox

✅ **Good for:**
- Development and testing
- Embedded databases in applications
- Proof-of-concept implementations
- Learning about Merkle trees

❌ **Not suitable for:**
- Production web services (use as embedded library)
- Distributed systems
- High-availability requirements
- Multi-node deployments

## Next Steps

- Read the [CLI Command Reference](../../reference/cli/commands.md)
- Explore [Library API Documentation](../../reference/api/index.md)
- Learn about [Proof Generation](../library/proof-generation.md)
- Understand [Batch Operations](../library/batch-processing.md)

## Getting Help

- Check `pb --help` for command options
- Read error messages carefully
- Review the [FAQ](../../faq/index.md)
- Ask in [GitHub Discussions](https://github.com/neutral/proofbox/discussions)