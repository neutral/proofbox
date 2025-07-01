# Batch Processing with ProofBox Library

This guide demonstrates how to efficiently perform multiple operations in batches using the ProofBox Go library.

## Goal

Learn to use ProofBox's batch API for high-performance bulk operations, optimizing throughput and minimizing version overhead.

## Prerequisites

- ProofBox library integrated in your project
- Understanding of basic ProofBox operations
- Familiarity with Go concurrency (for parallel batches)

## Understanding Batch Operations

### Benefits of Batching

1. **Performance**: Reduced overhead per operation
2. **Atomicity**: All operations succeed or all fail (true atomicity)
3. **Efficiency**: Optimized tree updates and storage writes
4. **Consistency**: Related updates happen together in a single version

### When to Use Batches

- Bulk data imports
- Multiple related updates
- High-throughput applications
- Periodic data synchronization

## Basic Batch Operations

### Setting Up

```go
import (
    "fmt"
    "log"
    
    "github.com/neutral/proofbox/pkg/storage"
    "github.com/neutral/proofbox/pkg/storage/pebble"
    "github.com/neutral/proofbox/pkg/tree"
    "github.com/neutral/proofbox/pkg/types"
)

// Initialize as usual
store, err := pebble.NewStorage("./batch.db", nil)
if err != nil {
    log.Fatal(err)
}
defer store.Close()

keyEncoder := storage.NewDefaultKeyEncoder()
config := tree.DefaultTreeConfig()
jmt, err := tree.NewTree(store, keyEncoder, config)
if err != nil {
    log.Fatal(err)
}
```

### Creating a Batch

```go
// Create a new batch transaction
batch := tree.NewBatchTransaction(jmt)

// Add operations to the batch
key1 := types.KeyHash([]byte("user:1001"))
err := batch.BatchPut(key1, []byte("Alice"))
if err != nil {
    log.Fatal(err)
}

key2 := types.KeyHash([]byte("user:1002"))
err = batch.BatchPut(key2, []byte("Bob"))
if err != nil {
    log.Fatal(err)
}

key3 := types.KeyHash([]byte("user:old"))
err = batch.BatchDelete(key3)
if err != nil {
    log.Fatal(err)
}

// Execute the batch atomically
newVersion, err := batch.Execute()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Batch committed at version %d\n", newVersion)
fmt.Printf("Batch size: %d operations\n", batch.Size())
```

### Batch Results

```go
// Execute returns the new version
newVersion, err := batch.Execute()
if err != nil {
    log.Fatal(err)
}

// Get additional information
fmt.Printf("New version: %d\n", newVersion)
fmt.Printf("Operations committed: %d\n", batch.Size())

// Get root hash for the new version
rootHash, err := jmt.GetRootHash(newVersion)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("New root hash: %x\n", rootHash)
```

## Advanced Batch Patterns

### Conditional Batching

```go
// Batch with size limits
batch := tree.NewBatchTransaction(jmt)
maxBatchSize := 1000

for _, item := range items {
    key := types.KeyHash([]byte(item.Key))
    err := batch.BatchPut(key, item.Value)
    if err != nil {
        log.Printf("Failed to add to batch: %v", err)
        continue
    }
    
    if batch.Size() >= maxBatchSize {
        version, err := batch.Execute()
        if err != nil {
            log.Printf("Batch failed: %v", err)
        } else {
            log.Printf("Committed batch at version %d", version)
        }
        batch = tree.NewBatchTransaction(jmt) // Start new batch
    }
}

// Commit remaining items
if batch.Size() > 0 {
    version, err := batch.Execute()
    if err != nil {
        log.Printf("Final batch failed: %v", err)
    } else {
        log.Printf("Committed final batch at version %d", version)
    }
}
```

### Batch Builder Pattern

```go
type BatchBuilder struct {
    batch *tree.BatchTransaction
}

func NewBatchBuilder(jmt *tree.Tree) *BatchBuilder {
    return &BatchBuilder{
        batch: tree.NewBatchTransaction(jmt),
    }
}

func (b *BatchBuilder) AddUser(id, name string) *BatchBuilder {
    key := types.KeyHash([]byte(fmt.Sprintf("user:%s", id)))
    if err := b.batch.BatchPut(key, []byte(name)); err != nil {
        log.Printf("Failed to add user %s: %v", id, err)
    }
    return b
}

func (b *BatchBuilder) RemoveUser(id string) *BatchBuilder {
    key := types.KeyHash([]byte(fmt.Sprintf("user:%s", id)))
    if err := b.batch.BatchDelete(key); err != nil {
        log.Printf("Failed to remove user %s: %v", id, err)
    }
    return b
}

func (b *BatchBuilder) Commit() (types.Version, error) {
    return b.batch.Execute()
}

// Usage
builder := NewBatchBuilder(jmt)
version, err := builder.
    AddUser("1001", "Alice").
    AddUser("1002", "Bob").
    RemoveUser("999").
    Commit()
```

### Parallel Batch Preparation

```go
import "sync"

// Prepare batches in parallel
func processConcurrently(jmt *tree.Tree, datasets [][]DataItem) error {
    var wg sync.WaitGroup
    batches := make([]*tree.BatchTransaction, len(datasets))
    
    // Prepare batches concurrently
    for i, dataset := range datasets {
        wg.Add(1)
        go func(idx int, data []DataItem) {
            defer wg.Done()
            
            batch := tree.NewBatchTransaction(jmt)
            for _, item := range data {
                key := types.KeyHash([]byte(item.Key))
                if err := batch.BatchPut(key, item.Value); err != nil {
                    log.Printf("Failed to add key %s: %v", item.Key, err)
                }
            }
            batches[idx] = batch
        }(i, dataset)
    }
    
    wg.Wait()
    
    // Commit batches sequentially
    for i, batch := range batches {
        if batch != nil && batch.Size() > 0 {
            version, err := batch.Execute()
            if err != nil {
                return fmt.Errorf("batch %d commit failed: %w", i, err)
            }
            log.Printf("Batch %d committed at version %d", i, version)
        }
    }
    
    return nil
}
```

## Batch Operations with Complex Data

### JSON Data in Batches

```go
type Product struct {
    ID       string  `json:"id"`
    Name     string  `json:"name"`
    Price    float64 `json:"price"`
    InStock  bool    `json:"in_stock"`
}

func batchImportProducts(jmt *tree.Tree, products []Product) error {
    batch := tree.NewBatchTransaction(jmt)
    
    for _, product := range products {
        // Serialize to JSON
        data, err := json.Marshal(product)
        if err != nil {
            return fmt.Errorf("marshal error for %s: %w", product.ID, err)
        }
        
        // Add to batch
        key := types.KeyHash([]byte(fmt.Sprintf("product:%s", product.ID)))
        if err := batch.BatchPut(key, data); err != nil {
            return fmt.Errorf("failed to add product %s: %w", product.ID, err)
        }
    }
    
    // Commit batch
    version, err := batch.Execute()
    if err != nil {
        return fmt.Errorf("batch commit failed: %w", err)
    }
    
    log.Printf("Imported %d products at version %d", len(products), version)
    return nil
}
```

### Mixed Operations Batch

```go
func updateInventory(jmt *tree.Tree, updates map[string]int, removals []string) error {
    batch := tree.NewBatchTransaction(jmt)
    
    // Add updates
    for productID, quantity := range updates {
        key := types.KeyHash([]byte(fmt.Sprintf("inventory:%s", productID)))
        value := []byte(fmt.Sprintf("%d", quantity))
        if err := batch.BatchPut(key, value); err != nil {
            return fmt.Errorf("failed to update %s: %w", productID, err)
        }
    }
    
    // Add removals
    for _, productID := range removals {
        key := types.KeyHash([]byte(fmt.Sprintf("inventory:%s", productID)))
        if err := batch.BatchDelete(key); err != nil {
            return fmt.Errorf("failed to delete %s: %w", productID, err)
        }
    }
    
    // Commit with validation
    if batch.Size() == 0 {
        return fmt.Errorf("empty batch")
    }
    
    version, err := batch.Execute()
    if err != nil {
        return err
    }
    
    log.Printf("Inventory updated: %d operations at version %d",
        batch.Size(), version)
    return nil
}
```

## Performance Optimization

### Batch Size Optimization

```go
// Benchmark different batch sizes
func findOptimalBatchSize(jmt *tree.Tree, testData []KVPair) {
    batchSizes := []int{10, 100, 1000, 5000, 10000}
    
    for _, size := range batchSizes {
        start := time.Now()
        
        for i := 0; i < len(testData); i += size {
            batch := jmt.NewBatchTransaction()
            
            end := i + size
            if end > len(testData) {
                end = len(testData)
            }
            
            for j := i; j < end; j++ {
                key := types.KeyHash([]byte(testData[j].Key))
                err := batch.BatchPut(key, testData[j].Value)
                if err != nil {
                    log.Printf("Failed to add key: %v", err)
                    continue
                }
            }
            
            _, err := batch.Execute()
            if err != nil {
                log.Printf("Error with batch size %d: %v", size, err)
                break
            }
        }
        
        duration := time.Since(start)
        ops := len(testData)
        throughput := float64(ops) / duration.Seconds()
        
        fmt.Printf("Batch size %d: %v total, %.0f ops/sec\n", 
            size, duration, throughput)
    }
}
```

### Memory-Efficient Batching

```go
// Stream data into batches without loading all into memory
func streamBatchImport(jmt *tree.Tree, reader io.Reader) error {
    scanner := bufio.NewScanner(reader)
    batch := tree.NewBatchTransaction(jmt)
    maxBatchSize := 1000
    
    for scanner.Scan() {
        line := scanner.Text()
        
        // Parse line (assuming CSV format)
        parts := strings.Split(line, ",")
        if len(parts) != 2 {
            continue
        }
        
        key := types.KeyHash([]byte(parts[0]))
        value := []byte(parts[1])
        
        if err := batch.BatchPut(key, value); err != nil {
            log.Printf("Failed to add key %s: %v", parts[0], err)
            continue
        }
        
        if batch.Size() >= maxBatchSize {
            version, err := batch.Execute()
            if err != nil {
                return err
            }
            log.Printf("Committed batch at version %d", version)
            
            batch = tree.NewBatchTransaction(jmt)
        }
    }
    
    // Commit final batch
    if batch.Size() > 0 {
        version, err := batch.Execute()
        if err != nil {
            return err
        }
        log.Printf("Committed final batch at version %d", version)
    }
    
    return scanner.Err()
}
```

## Error Handling

### Batch Validation

```go
// Validate batch before committing
func validateAndCommit(batch *tree.BatchTransaction) error {
    // Check batch size
    if batch.Size() == 0 {
        return fmt.Errorf("cannot commit empty batch")
    }
    
    if batch.Size() > 10000 {
        return fmt.Errorf("batch too large: %d operations", batch.Size())
    }
    
    // Commit with timeout
    done := make(chan struct {
        version types.Version
        err     error
    }, 1)
    
    go func() {
        version, err := batch.Execute()
        done <- struct {
            version types.Version
            err     error
        }{version, err}
    }()
    
    select {
    case result := <-done:
        if result.err != nil {
            return result.err
        }
        log.Printf("Batch committed at version %d", result.version)
        return nil
    case <-time.After(30 * time.Second):
        return fmt.Errorf("batch commit timeout")
    }
}
```

### Recovery from Failed Batches

```go
// Retry logic for batch operations
func commitBatchWithRetry(batch *tree.BatchTransaction, maxRetries int) (types.Version, error) {
    var lastErr error
    
    for i := 0; i < maxRetries; i++ {
        version, err := batch.Execute()
        if err == nil {
            return version, nil
        }
        
        lastErr = err
        log.Printf("Batch commit attempt %d failed: %v", i+1, err)
        
        // Note: After a failed commit, the batch state may be invalid
        // In production, you might need to recreate the batch
        
        // Exponential backoff
        time.Sleep(time.Duration(1<<i) * time.Second)
    }
    
    return 0, fmt.Errorf("batch commit failed after %d retries: %w", maxRetries, lastErr)
}
```

## Complete Example

Here's a comprehensive example showing batch import from a CSV file:

```go
package main

import (
    "bufio"
    "encoding/csv"
    "fmt"
    "log"
    "os"
    "time"
    
    "github.com/neutral/proofbox/pkg/storage"
    "github.com/neutral/proofbox/pkg/storage/pebble"
    "github.com/neutral/proofbox/pkg/tree"
    "github.com/neutral/proofbox/pkg/types"
)

func importCSV(filename string, jmt *tree.Tree) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()
    
    reader := csv.NewReader(file)
    reader.FieldsPerRecord = 2
    
    batch := tree.NewBatchTransaction(jmt)
    totalCount := 0
    start := time.Now()
    
    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Printf("CSV read error: %v", err)
            continue
        }
        
        key := types.KeyHash([]byte(record[0]))
        value := []byte(record[1])
        
        if err := batch.BatchPut(key, value); err != nil {
            log.Printf("Failed to add key %s: %v", record[0], err)
            continue
        }
        
        // Commit batch when it reaches optimal size
        if batch.Size() >= 1000 {
            version, err := batch.Execute()
            if err != nil {
                return fmt.Errorf("batch commit failed: %w", err)
            }
            
            batchSize := batch.Size()
            totalCount += batchSize
            log.Printf("Committed batch: %d operations at version %d", 
                batchSize, version)
            
            batch = tree.NewBatchTransaction(jmt)
        }
    }
    
    // Commit final batch
    if batch.Size() > 0 {
        version, err := batch.Execute()
        if err != nil {
            return fmt.Errorf("final batch commit failed: %w", err)
        }
        
        batchSize := batch.Size()
        totalCount += batchSize
        log.Printf("Committed final batch: %d operations at version %d", 
            batchSize, version)
    }
    
    duration := time.Since(start)
    throughput := float64(totalCount) / duration.Seconds()
    
    log.Printf("Import complete: %d records in %v (%.0f ops/sec)", 
        totalCount, duration, throughput)
    
    return nil
}

func main() {
    // Initialize storage and tree
    store, err := pebble.NewStorage("./import.db", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    keyEncoder := storage.NewDefaultKeyEncoder()
    config := tree.DefaultTreeConfig()
    jmt, err := tree.NewTree(store, keyEncoder, config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Import data
    if err := importCSV("data.csv", jmt); err != nil {
        log.Fatal(err)
    }
    
    // Verify import
    rootHash, err := jmt.GetRootHash(jmt.GetLatestVersion())
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Final root hash: %x\n", rootHash)
}
```

## Best Practices

### Batch Design

1. **Optimal Size**: 100-5000 operations per batch
2. **Memory Usage**: Monitor batch memory consumption
3. **Related Updates**: Group related operations together
4. **Error Handling**: Plan for partial failures

### Performance Tips

1. **Prepare Data**: Transform data before batching
2. **Parallel Preparation**: Prepare batches concurrently
3. **Sequential Commits**: Commit batches in order
4. **Monitor Metrics**: Track throughput and latency

### Common Pitfalls

1. **Oversized Batches**: Can cause memory issues
2. **Too Many Small Batches**: Reduces efficiency
3. **Missing Error Handling**: Can lose data
4. **Concurrent Executes**: Tree doesn't support parallel batch executions

## Troubleshooting

### Out of Memory

```go
// Monitor batch memory usage
func estimateBatchMemory(batch *tree.BatchTransaction) int {
    // Rough estimate: key (32) + average value size + overhead
    // For BatchTransaction, each operation stores key + value
    avgValueSize := 100 // Adjust based on your data
    overhead := 64      // Map entries, pointers, etc.
    
    return batch.Size() * (32 + avgValueSize + overhead)
}
```

### Slow Performance

- Check batch size (too small or too large)
- Verify storage performance
- Consider value sizes
- Monitor system resources

## Next Steps

- Explore [Proof Generation](proof-generation.md) for batch verification
- Learn about [Version Management](version-management.md) for batch rollback
- Read about performance tuning in the reference docs

## Quick Reference

```go
// Create batch transaction
batch := tree.NewBatchTransaction(jmt)

// Add operations
err := batch.BatchPut(key, value)
err = batch.BatchDelete(key)

// Execute batch atomically
version, err := batch.Execute()

// Check batch info
batch.Size()         // Number of operations

// Get root hash after commit
rootHash, _ := jmt.GetRootHash(version)
```