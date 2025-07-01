# Basic Operations with ProofBox Library

This guide shows how to perform basic CRUD (Create, Read, Update, Delete) operations using the ProofBox Go library.

## Goal

Learn to integrate ProofBox into your Go application for basic key-value storage with cryptographic verification.

## Prerequisites

- Go 1.21 or later installed
- Basic familiarity with Go programming
- Understanding of ProofBox concepts

## Installation

Add ProofBox to your project:

```bash
go get github.com/neutral/proofbox
```

## Setting Up

### Import Required Packages

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
```

### Initialize Storage and Tree

```go
func main() {
    // Create storage backend
    store, err := pebble.NewStorage("./myapp.db", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Create key encoder
    keyEncoder := storage.NewDefaultKeyEncoder()
    
    // Configure tree
    config := tree.DefaultTreeConfig()
    config.MetricsEnabled = false // Disable metrics for simplicity
    
    // Create tree instance
    jmt, err := tree.NewTree(store, keyEncoder, config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Your operations here
}
```

## Basic Operations

### Put Operation (Insert/Update)

Store or update a key-value pair:

```go
// Convert string key to hash
key := types.KeyHash([]byte("user:123"))

// Store value
value := []byte("John Doe")
version, err := jmt.Put(key, value)
if err != nil {
    log.Printf("Failed to put: %v", err)
    return
}

fmt.Printf("Stored at version %d\n", version)
```

### Get Operation (Read)

Retrieve a value by key:

```go
// Get from latest version
key := types.KeyHash([]byte("user:123"))
value, err := jmt.Get(jmt.GetLatestVersion(), key)
if err != nil {
    log.Printf("Failed to get: %v", err)
    return
}

if value == nil {
    fmt.Println("Key not found")
} else {
    fmt.Printf("Value: %s\n", string(value))
}
```

### Get at Specific Version

```go
// Get from specific version
version := types.Version(5)
value, err := jmt.Get(version, key)
if err != nil {
    log.Printf("Failed to get at version %d: %v", version, err)
    return
}
```

### Delete Operation

Remove a key-value pair:

```go
key := types.KeyHash([]byte("user:123"))
version, err := jmt.Delete(key)
if err != nil {
    log.Printf("Failed to delete: %v", err)
    return
}

fmt.Printf("Deleted at version %d\n", version)
```

### Check Existence

Verify if a key exists:

```go
key := types.KeyHash([]byte("user:123"))
value, err := jmt.Get(jmt.GetLatestVersion(), key)
exists := err == nil && value != nil

if exists {
    fmt.Println("Key exists")
} else {
    fmt.Println("Key does not exist")
}
```

## Working with Complex Data

### Storing JSON

```go
import "encoding/json"

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Encode to JSON
user := User{
    ID:    "123",
    Name:  "John Doe",
    Email: "john@example.com",
}

data, err := json.Marshal(user)
if err != nil {
    log.Fatal(err)
}

// Store JSON data
key := types.KeyHash([]byte("user:123"))
version, err := jmt.Put(key, data)
if err != nil {
    log.Fatal(err)
}
```

### Retrieving JSON

```go
// Get JSON data
key := types.KeyHash([]byte("user:123"))
data, err := jmt.Get(jmt.GetLatestVersion(), key)
if err != nil || data == nil {
    log.Fatal("Failed to retrieve user")
}

// Decode JSON
var user User
err = json.Unmarshal(data, &user)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("User: %+v\n", user)
```

## Tree Information

### Get Root Hash

```go
version := jmt.GetLatestVersion()
rootHash, err := jmt.GetRootHash(version)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Root hash at version %d: %x\n", version, rootHash)
```

### Get Latest Version

```go
latestVersion := jmt.GetLatestVersion()
fmt.Printf("Latest version: %d\n", latestVersion)
```

## Error Handling

### Common Errors

```go
// Handle specific errors
value, err := jmt.Get(version, key)
if err != nil {
    switch {
    case types.IsNotFoundError(err):
        fmt.Println("Key not found")
    case types.IsVersionError(err):
        fmt.Println("Invalid version")
    default:
        log.Printf("Unexpected error: %v", err)
    }
}
```

### Graceful Degradation

```go
// Retry logic for transient errors
var value []byte
maxRetries := 3

for i := 0; i < maxRetries; i++ {
    value, err = jmt.Get(jmt.GetLatestVersion(), key)
    if err == nil {
        break
    }
    
    if i < maxRetries-1 {
        time.Sleep(time.Millisecond * 100)
    }
}

if err != nil {
    log.Printf("Failed after %d retries: %v", maxRetries, err)
}
```

## Complete Example

Here's a complete working example:

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    
    "github.com/neutral/proofbox/pkg/storage"
    "github.com/neutral/proofbox/pkg/storage/pebble"
    "github.com/neutral/proofbox/pkg/tree"
    "github.com/neutral/proofbox/pkg/types"
)

type Product struct {
    ID    string  `json:"id"`
    Name  string  `json:"name"`
    Price float64 `json:"price"`
}

func main() {
    // Initialize storage
    store, err := pebble.NewStorage("./products.db", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Create key encoder
    keyEncoder := storage.NewDefaultKeyEncoder()
    
    // Configure tree
    config := tree.DefaultTreeConfig()
    config.MetricsEnabled = false
    
    // Create tree
    jmt, err := tree.NewTree(store, keyEncoder, config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create products
    products := []Product{
        {ID: "001", Name: "Laptop", Price: 999.99},
        {ID: "002", Name: "Mouse", Price: 29.99},
        {ID: "003", Name: "Keyboard", Price: 79.99},
    }
    
    // Store products
    for _, product := range products {
        data, err := json.Marshal(product)
        if err != nil {
            log.Printf("Failed to marshal %s: %v", product.ID, err)
            continue
        }
        
        key := types.KeyHash([]byte(fmt.Sprintf("product:%s", product.ID)))
        version, err := jmt.Put(key, data)
        if err != nil {
            log.Printf("Failed to store %s: %v", product.ID, err)
            continue
        }
        
        fmt.Printf("Stored %s at version %d\n", product.Name, version)
    }
    
    // Retrieve and display a product
    key := types.KeyHash([]byte("product:001"))
    data, err := jmt.Get(jmt.GetLatestVersion(), key)
    if err != nil {
        log.Fatal(err)
    }
    
    var laptop Product
    err = json.Unmarshal(data, &laptop)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("\nRetrieved: %+v\n", laptop)
    
    // Get root hash
    rootHash, err := jmt.GetRootHash(jmt.GetLatestVersion())
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("\nFinal root hash: %x\n", rootHash)
}
```

## Best Practices

### Key Design

1. **Use Hierarchical Keys**: Organize with prefixes like `user:123`, `config:app`
2. **Hash Long Keys**: For keys > 32 bytes, use `types.KeyHash()`
3. **Avoid Special Characters**: Stick to alphanumeric and common separators

### Value Management

1. **Size Limits**: Keep values under 1MB
2. **Compression**: Consider compressing large JSON objects
3. **Versioning**: Design your schema with version evolution in mind

### Performance Tips

1. **Reuse Tree Instance**: Don't create new tree instances for each operation
2. **Batch When Possible**: Use batch operations for multiple updates
3. **Close Resources**: Always defer Close() on storage

### Error Handling

1. **Check All Errors**: ProofBox operations can fail
2. **Handle Not Found**: Distinguish between errors and missing keys
3. **Log Failures**: Track errors for debugging

## Common Patterns

### Cache-Aside Pattern

```go
// Check cache first
if cached, ok := cache.Get(key); ok {
    return cached
}

// Fallback to ProofBox
value, err := jmt.Get(jmt.GetLatestVersion(), key)
if err == nil && value != nil {
    cache.Set(key, value)
}
return value
```

### Conditional Updates

```go
// Read-modify-write pattern
key := types.KeyHash([]byte("counter"))
value, err := jmt.Get(jmt.GetLatestVersion(), key)

counter := 0
if err == nil && value != nil {
    counter = int(value[0])
}

counter++
newValue := []byte{byte(counter)}
version, err := jmt.Put(key, newValue)
```

## Troubleshooting

### Storage Initialization Fails

```go
// Check error details
store, err := pebble.NewStorage("./data/myapp.db", nil)
if err != nil {
    // Ensure directory exists
    os.MkdirAll("./data", 0755)
    
    // Retry
    store, err = pebble.NewStorage("./data/myapp.db", nil)
}
```

### Memory Usage

```go
// Periodically compact storage
if err := store.Compact(); err != nil {
    log.Printf("Compact failed: %v", err)
}
```

## Next Steps

- Learn about [Batch Processing](batch-processing.md) for efficient bulk operations
- Explore [Proof Generation](proof-generation.md) for cryptographic verification
- Understand [Version Management](version-management.md) for time-travel queries

## Quick Reference

```go
// Setup
store, _ := pebble.NewStorage("./db", nil)
keyEncoder := storage.NewDefaultKeyEncoder()
config := tree.DefaultTreeConfig()
jmt, _ := tree.NewTree(store, keyEncoder, config)

// Operations
version, _ := jmt.Put(key, value)    // Insert/Update
value, _ := jmt.Get(version, key)    // Read
version, _ := jmt.Delete(key)        // Delete

// Info
version := jmt.GetLatestVersion()     // Current version
hash, _ := jmt.GetRootHash(version)   // Root hash

// Cleanup
store.Close()
```