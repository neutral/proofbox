# ProofBox Go Library Quick Start

This tutorial will get you started using ProofBox as a Go library in your applications.

## Prerequisites

- Go 1.21 or later installed
- Basic familiarity with Go programming
- Understanding of ProofBox concepts from [Getting Started](getting-started.md)

## Installation

Add ProofBox to your Go module:

```bash
go get github.com/neutral/proofbox
```

## Your First ProofBox Application

Create a new file `main.go`:

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
    // Step 1: Initialize storage backend
    store, err := pebble.NewStorage("./myapp.db", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Step 2: Create a key encoder
    keyEncoder := storage.NewDefaultKeyEncoder()
    
    // Step 3: Initialize the Jellyfish Merkle Tree
    config := tree.DefaultTreeConfig()
    jmt, err := tree.NewTree(store, keyEncoder, config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Step 4: Store some data
    key := types.KeyHash([]byte("hello"))
    value := []byte("world")
    
    version, err := jmt.Put(key, value)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Stored key at version %d\n", version)
    
    // Step 5: Retrieve the data
    retrievedValue, err := jmt.Get(key)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Retrieved value: %s\n", string(retrievedValue))
    
    // Step 6: Get the root hash
    rootHash, err := jmt.GetRootHash(version)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Root hash: %x\n", rootHash)
}
```

Run the application:

```bash
go run main.go
```

Output:
```
Stored key at version 1
Retrieved value: world
Root hash: a8f3b2c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2
```

## Core Operations

### Put and Get

```go
// Store key-value pairs
key1 := types.KeyHash([]byte("user:123"))
value1 := []byte(`{"name": "Alice", "age": 30}`)
version1, err := jmt.Put(key1, value1)

key2 := types.KeyHash([]byte("user:456"))
value2 := []byte(`{"name": "Bob", "age": 25}`)
version2, err := jmt.Put(key2, value2)

// Retrieve values
data, err := jmt.Get(key1)
if err != nil {
    if err == types.ErrKeyNotFound {
        fmt.Println("Key not found")
    } else {
        log.Fatal(err)
    }
}
```

### Delete

```go
// Delete a key
key := types.KeyHash([]byte("user:old"))
version, err := jmt.Delete(key)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Deleted key at version %d\n", version)
```

### Working with Versions

```go
// Get current version
currentVersion := jmt.GetLatestVersion()
fmt.Printf("Current version: %d\n", currentVersion)

// Read from specific version
value, err := jmt.GetAtVersion(5, key)
if err != nil {
    log.Fatal(err)
}

// Check if version exists
if jmt.HasVersion(5) {
    fmt.Println("Version 5 exists")
}
```

## Batch Operations

For better performance with multiple operations:

```go
// Create a batch transaction
batch := jmt.NewBatchTransaction()

// Add multiple operations
users := []struct {
    ID   string
    Name string
}{
    {"1001", "Alice"},
    {"1002", "Bob"},
    {"1003", "Charlie"},
}

for _, user := range users {
    key := types.KeyHash([]byte(fmt.Sprintf("user:%s", user.ID)))
    value := []byte(user.Name)
    
    err := batch.BatchPut(key, value)
    if err != nil {
        log.Printf("Failed to add user %s: %v", user.ID, err)
    }
}

// Delete old entries
err = batch.BatchDelete(types.KeyHash([]byte("user:old")))

// Execute all operations atomically
version, err := batch.Execute()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Batch committed at version %d with %d operations\n", 
    version, batch.Size())
```

## Generating Proofs

ProofBox can generate cryptographic proofs for your data:

```go
import (
    "github.com/neutral/proofbox/pkg/proof"
)

// Create a tree reader
reader, err := jmt.Reader(jmt.GetLatestVersion())
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

// Generate proof
generator := proof.NewGenerator(reader)
key := types.KeyHash([]byte("user:123"))

merkleProof, err := generator.Generate(key)
if err != nil {
    log.Fatal(err)
}

// The proof contains:
// - Key and value (for inclusion proofs)
// - Sibling hashes needed to reconstruct the root
// - Root hash for verification
fmt.Printf("Proof type: %s\n", merkleProof.Type)
fmt.Printf("Value: %s\n", string(merkleProof.Value))
fmt.Printf("Root hash: %x\n", merkleProof.RootHash)
```

## Verifying Proofs

```go
// Create a verifier
verifier := proof.NewVerifier()

// Verify the proof
err = verifier.Verify(merkleProof)
if err != nil {
    fmt.Printf("Proof verification failed: %v\n", err)
} else {
    fmt.Println("Proof verified successfully!")
}
```

## Error Handling

ProofBox uses explicit error handling:

```go
// Common errors to handle
value, err := jmt.Get(key)
if err != nil {
    switch err {
    case types.ErrKeyNotFound:
        // Key doesn't exist
        fmt.Println("Key not found")
    case types.ErrVersionNotFound:
        // Version doesn't exist
        fmt.Println("Version not found")
    default:
        // Other errors
        log.Fatal(err)
    }
}
```

## Configuration Options

Customize tree behavior:

```go
config := tree.TreeConfig{
    CacheSize:           20000,  // Number of nodes to cache
    MaxBatchSize:        5000,   // Maximum operations per batch
    MetricsEnabled:      true,   // Enable Prometheus metrics
    UseParallelBatching: true,   // Use parallel processing
}

jmt, err := tree.NewTree(store, keyEncoder, config)
```

## Complete Example: Key-Value Store

Here's a complete example implementing a simple key-value store:

```go
package main

import (
    "bufio"
    "fmt"
    "log"
    "os"
    "strings"
    
    "github.com/neutral/proofbox/pkg/storage"
    "github.com/neutral/proofbox/pkg/storage/pebble"
    "github.com/neutral/proofbox/pkg/tree"
    "github.com/neutral/proofbox/pkg/types"
)

type KVStore struct {
    tree *tree.Tree
}

func NewKVStore(dbPath string) (*KVStore, error) {
    store, err := pebble.NewStorage(dbPath, nil)
    if err != nil {
        return nil, err
    }
    
    keyEncoder := storage.NewDefaultKeyEncoder()
    config := tree.DefaultTreeConfig()
    
    jmt, err := tree.NewTree(store, keyEncoder, config)
    if err != nil {
        store.Close()
        return nil, err
    }
    
    return &KVStore{tree: jmt}, nil
}

func (kv *KVStore) Put(key, value string) error {
    k := types.KeyHash([]byte(key))
    v := []byte(value)
    
    _, err := kv.tree.Put(k, v)
    return err
}

func (kv *KVStore) Get(key string) (string, error) {
    k := types.KeyHash([]byte(key))
    
    value, err := kv.tree.Get(k)
    if err != nil {
        return "", err
    }
    
    return string(value), nil
}

func (kv *KVStore) Delete(key string) error {
    k := types.KeyHash([]byte(key))
    
    _, err := kv.tree.Delete(k)
    return err
}

func main() {
    // Initialize store
    store, err := NewKVStore("./demo.db")
    if err != nil {
        log.Fatal(err)
    }
    
    // Interactive prompt
    scanner := bufio.NewScanner(os.Stdin)
    fmt.Println("ProofBox KV Store (type 'help' for commands)")
    
    for {
        fmt.Print("> ")
        if !scanner.Scan() {
            break
        }
        
        line := scanner.Text()
        parts := strings.Fields(line)
        
        if len(parts) == 0 {
            continue
        }
        
        switch parts[0] {
        case "put":
            if len(parts) < 3 {
                fmt.Println("Usage: put <key> <value>")
                continue
            }
            key := parts[1]
            value := strings.Join(parts[2:], " ")
            
            if err := store.Put(key, value); err != nil {
                fmt.Printf("Error: %v\n", err)
            } else {
                fmt.Printf("OK\n")
            }
            
        case "get":
            if len(parts) < 2 {
                fmt.Println("Usage: get <key>")
                continue
            }
            
            value, err := store.Get(parts[1])
            if err != nil {
                if err == types.ErrKeyNotFound {
                    fmt.Println("Key not found")
                } else {
                    fmt.Printf("Error: %v\n", err)
                }
            } else {
                fmt.Printf("%s\n", value)
            }
            
        case "delete":
            if len(parts) < 2 {
                fmt.Println("Usage: delete <key>")
                continue
            }
            
            if err := store.Delete(parts[1]); err != nil {
                fmt.Printf("Error: %v\n", err)
            } else {
                fmt.Printf("OK\n")
            }
            
        case "help":
            fmt.Println("Commands:")
            fmt.Println("  put <key> <value> - Store a key-value pair")
            fmt.Println("  get <key>         - Retrieve a value")
            fmt.Println("  delete <key>      - Delete a key")
            fmt.Println("  quit              - Exit")
            
        case "quit":
            return
            
        default:
            fmt.Printf("Unknown command: %s\n", parts[0])
        }
    }
}
```

## Best Practices

1. **Always close resources**: Use `defer` to ensure cleanup
2. **Handle errors explicitly**: Check all error returns
3. **Use batch operations**: For multiple updates, use batches
4. **Configure appropriately**: Tune cache size and batch limits
5. **Version awareness**: Track versions for audit trails

## What You've Learned

In this quick start, you've learned how to:
- ✅ Set up ProofBox as a Go library
- ✅ Initialize storage and tree
- ✅ Perform basic CRUD operations
- ✅ Use batch operations for efficiency
- ✅ Generate and verify cryptographic proofs
- ✅ Handle errors properly
- ✅ Build a simple application

## Next Steps

- Explore [Basic Operations](../how-to/library/basic-operations.md) in depth
- Learn about [Batch Processing](../how-to/library/batch-processing.md) optimization
- Understand [Proof Generation](../how-to/library/proof-generation.md) details
- Read about [Version Management](../how-to/library/version-management.md)

## Need Help?

- API Documentation: See the [Go package documentation](https://pkg.go.dev/github.com/neutral/proofbox)
- Examples: Check the `examples/` directory in the repository
- Issues: [GitHub Issues](https://github.com/neutral/proofbox/issues)