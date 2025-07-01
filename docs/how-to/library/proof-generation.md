# Proof Generation with ProofBox Library

This guide demonstrates how to generate and verify cryptographic proofs using the ProofBox Go library.

## Goal

Learn to generate Merkle proofs for key existence (or non-existence) and understand how to verify data integrity using these proofs.

## Prerequisites

- ProofBox library integrated in your project
- Understanding of basic ProofBox operations
- Basic knowledge of Merkle trees and cryptographic proofs

## Understanding Proofs in ProofBox

### What is a Merkle Proof?

A Merkle proof provides cryptographic evidence that:
- A key-value pair exists in the tree (inclusion proof)
- A key does not exist in the tree (non-inclusion proof)

The proof consists of:
- Sibling hashes along the path from leaf to root
- Enough information to reconstruct the root hash

### Why Use Proofs?

1. **Verification**: Prove data integrity without full tree access
2. **Light Clients**: Verify data with minimal storage
3. **Audit Trail**: Cryptographic evidence of state at a version
4. **Trust Minimization**: Verify without trusting the data provider

## Basic Proof Generation

### Setting Up

```go
import (
    "fmt"
    "log"
    
    "github.com/neutral/proofbox/pkg/proof"
    "github.com/neutral/proofbox/pkg/storage"
    "github.com/neutral/proofbox/pkg/storage/pebble"
    "github.com/neutral/proofbox/pkg/tree"
    "github.com/neutral/proofbox/pkg/types"
)

// Initialize tree as usual
store, err := pebble.NewStorage("./proofs.db", nil)
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

### Generating a Proof

```go
// Add some data first
key := types.KeyHash([]byte("user:alice"))
value := []byte("Alice Smith")
version, err := jmt.Put(key, value)
if err != nil {
    log.Fatal(err)
}

// Create a reader for the specific version
reader, err := jmt.Reader(version)
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

// Create proof generator
generator := proof.NewGenerator(reader)

// Generate proof for the key
merkleProof, err := generator.Generate(key)
if err != nil {
    log.Fatal(err)
}

// Check proof type
if merkleProof.IsInclusion() {
    fmt.Println("Inclusion proof generated")
    fmt.Printf("Value in proof: %s\n", string(merkleProof.Value))
} else {
    fmt.Println("Non-inclusion proof generated")
}
```

### Proof Structure

```go
// Examine proof details
fmt.Printf("Proof for key: %x\n", merkleProof.Key)
fmt.Printf("Root hash: %x\n", merkleProof.RootHash)
fmt.Printf("Number of siblings: %d\n", len(merkleProof.Siblings))

// Iterate through siblings
for i, sibling := range merkleProof.Siblings {
    fmt.Printf("Sibling at depth %d:\n", i)
    for nibble, hash := range sibling.Children {
        fmt.Printf("  Child %x: %x\n", nibble, hash)
    }
}
```

## Advanced Proof Operations

### Batch Proof Generation

```go
// Generate proofs for multiple keys efficiently
func generateBatchProofs(jmt *tree.Tree, version types.Version, keys []string) (map[string]*proof.Proof, error) {
    // Create reader once for all proofs
    reader, err := jmt.Reader(version)
    if err != nil {
        return nil, fmt.Errorf("failed to create reader: %w", err)
    }
    defer reader.Close()
    
    generator := proof.NewGenerator(reader)
    proofs := make(map[string]*proof.Proof)
    
    for _, keyStr := range keys {
        key := types.KeyHash([]byte(keyStr))
        
        merkleProof, err := generator.Generate(key)
        if err != nil {
            log.Printf("Failed to generate proof for %s: %v", keyStr, err)
            continue
        }
        
        proofs[keyStr] = merkleProof
    }
    
    return proofs, nil
}
```

### Proof Serialization

```go
// Serialize proof for transmission or storage
func serializeProof(p *proof.Proof) ([]byte, error) {
    // Encode proof to bytes
    encoded, err := p.Encode()
    if err != nil {
        return nil, fmt.Errorf("failed to encode proof: %w", err)
    }
    
    return encoded, nil
}

// Deserialize proof
func deserializeProof(data []byte) (*proof.Proof, error) {
    p := &proof.Proof{}
    
    err := p.Decode(data)
    if err != nil {
        return nil, fmt.Errorf("failed to decode proof: %w", err)
    }
    
    return p, nil
}
```

### Proof Verification

```go
// Verify a proof independently
func verifyProof(p *proof.Proof, expectedRootHash types.Hash) bool {
    // Create verifier
    verifier := proof.NewVerifier()
    
    // Verify the proof
    valid, err := verifier.Verify(p, expectedRootHash)
    if err != nil {
        log.Printf("Verification error: %v", err)
        return false
    }
    
    return valid
}

// Complete verification example
func verifyDataIntegrity(jmt *tree.Tree, keyStr string, expectedValue []byte) error {
    key := types.KeyHash([]byte(keyStr))
    version := jmt.GetLatestVersion()
    
    // Get root hash for version
    rootHash, err := jmt.GetRootHash(version)
    if err != nil {
        return fmt.Errorf("failed to get root hash: %w", err)
    }
    
    // Generate proof
    reader, err := jmt.Reader(version)
    if err != nil {
        return fmt.Errorf("failed to create reader: %w", err)
    }
    defer reader.Close()
    
    generator := proof.NewGenerator(reader)
    merkleProof, err := generator.Generate(key)
    if err != nil {
        return fmt.Errorf("failed to generate proof: %w", err)
    }
    
    // Verify proof
    verifier := proof.NewVerifier()
    valid, err := verifier.Verify(merkleProof, rootHash)
    if err != nil {
        return fmt.Errorf("verification failed: %w", err)
    }
    
    if !valid {
        return fmt.Errorf("proof is invalid")
    }
    
    // Check value matches
    if merkleProof.IsInclusion() {
        if string(merkleProof.Value) != string(expectedValue) {
            return fmt.Errorf("value mismatch")
        }
    } else {
        return fmt.Errorf("key not found (non-inclusion proof)")
    }
    
    return nil
}
```

## Proof Use Cases

### Light Client Verification

```go
// Light client that only stores root hashes
type LightClient struct {
    rootHashes map[types.Version]types.Hash
}

func (lc *LightClient) VerifyValue(
    version types.Version, 
    key string, 
    value []byte, 
    merkleProof *proof.Proof,
) error {
    // Get trusted root hash
    rootHash, exists := lc.rootHashes[version]
    if !exists {
        return fmt.Errorf("unknown version %d", version)
    }
    
    // Verify proof against trusted root
    verifier := proof.NewVerifier()
    valid, err := verifier.Verify(merkleProof, rootHash)
    if err != nil {
        return fmt.Errorf("verification error: %w", err)
    }
    
    if !valid {
        return fmt.Errorf("invalid proof")
    }
    
    // Check value
    if !merkleProof.IsInclusion() {
        return fmt.Errorf("key does not exist")
    }
    
    if string(merkleProof.Value) != string(value) {
        return fmt.Errorf("value mismatch")
    }
    
    return nil
}
```

### Audit Trail

```go
// Create audit record with proof
type AuditRecord struct {
    Timestamp time.Time
    Key       string
    Value     []byte
    Version   types.Version
    RootHash  types.Hash
    Proof     []byte // Serialized proof
}

func createAuditRecord(jmt *tree.Tree, keyStr string) (*AuditRecord, error) {
    key := types.KeyHash([]byte(keyStr))
    version := jmt.GetLatestVersion()
    
    // Get current value
    value, err := jmt.Get(version, key)
    if err != nil {
        return nil, err
    }
    
    // Get root hash
    rootHash, err := jmt.GetRootHash(version)
    if err != nil {
        return nil, err
    }
    
    // Generate proof
    reader, err := jmt.Reader(version)
    if err != nil {
        return nil, err
    }
    defer reader.Close()
    
    generator := proof.NewGenerator(reader)
    merkleProof, err := generator.Generate(key)
    if err != nil {
        return nil, err
    }
    
    // Serialize proof
    codec := proof.NewProofCodec()
    proofData, err := codec.Encode(merkleProof)
    if err != nil {
        return nil, err
    }
    
    return &AuditRecord{
        Timestamp: time.Now(),
        Key:       keyStr,
        Value:     value,
        Version:   version,
        RootHash:  rootHash,
        Proof:     proofData,
    }, nil
}
```

### Cross-System Verification

```go
// Export proof for external verification
type ProofExport struct {
    Key      string `json:"key"`
    Value    string `json:"value,omitempty"`
    Version  uint64 `json:"version"`
    RootHash string `json:"root_hash"`
    Proof    string `json:"proof"` // Base64 encoded
    Type     string `json:"type"`  // "inclusion" or "non_inclusion"
}

func exportProof(jmt *tree.Tree, keyStr string) (*ProofExport, error) {
    key := types.KeyHash([]byte(keyStr))
    version := jmt.GetLatestVersion()
    
    // Generate proof
    reader, err := jmt.Reader(version)
    if err != nil {
        return nil, err
    }
    defer reader.Close()
    
    generator := proof.NewGenerator(reader)
    merkleProof, err := generator.Generate(key)
    if err != nil {
        return nil, err
    }
    
    // Get root hash
    rootHash, err := jmt.GetRootHash(version)
    if err != nil {
        return nil, err
    }
    
    // Serialize proof
    codec := proof.NewProofCodec()
    proofData, err := codec.Encode(merkleProof)
    if err != nil {
        return nil, err
    }
    
    export := &ProofExport{
        Key:      keyStr,
        Version:  uint64(version),
        RootHash: fmt.Sprintf("%x", rootHash),
        Proof:    base64.StdEncoding.EncodeToString(proofData),
    }
    
    if merkleProof.IsInclusion() {
        export.Type = "inclusion"
        export.Value = string(merkleProof.Value)
    } else {
        export.Type = "non_inclusion"
    }
    
    return export, nil
}
```

## Performance Considerations

### Proof Size

```go
// Analyze proof size
func analyzeProofSize(p *proof.Proof) {
    encoded, err := p.Encode()
    if err != nil {
        log.Printf("Failed to encode: %v", err)
        return
    }
    
    fmt.Printf("Proof size: %d bytes\n", len(encoded))
    fmt.Printf("Number of siblings: %d\n", len(p.Siblings))
    
    // Estimate components
    baseSize := 32 + 32 + 1 // key + root hash + flags
    siblingSize := 0
    for _, sibling := range p.Siblings {
        // Each sibling: depth (1) + count (1) + (nibble (1) + hash (32)) * count
        siblingSize += 2 + len(sibling.Children) * 33
    }
    
    fmt.Printf("Base overhead: %d bytes\n", baseSize)
    fmt.Printf("Sibling data: %d bytes\n", siblingSize)
    fmt.Printf("Value size: %d bytes\n", len(p.Value))
}
```

### Caching Proofs

```go
// Cache frequently requested proofs
type ProofCache struct {
    cache map[string]*proof.Proof
    mu    sync.RWMutex
    ttl   time.Duration
}

func NewProofCache(ttl time.Duration) *ProofCache {
    return &ProofCache{
        cache: make(map[string]*proof.Proof),
        ttl:   ttl,
    }
}

func (pc *ProofCache) Get(version types.Version, key types.Key) (*proof.Proof, bool) {
    pc.mu.RLock()
    defer pc.mu.RUnlock()
    
    cacheKey := fmt.Sprintf("%d:%x", version, key)
    p, exists := pc.cache[cacheKey]
    return p, exists
}

func (pc *ProofCache) Set(version types.Version, key types.Key, p *proof.Proof) {
    pc.mu.Lock()
    defer pc.mu.Unlock()
    
    cacheKey := fmt.Sprintf("%d:%x", version, key)
    pc.cache[cacheKey] = p
    
    // Simple TTL implementation
    go func() {
        time.Sleep(pc.ttl)
        pc.mu.Lock()
        delete(pc.cache, cacheKey)
        pc.mu.Unlock()
    }()
}
```

## Error Handling

### Common Proof Errors

```go
// Handle proof generation errors
func handleProofError(err error) {
    if err == nil {
        return
    }
    
    switch {
    case errors.Is(err, types.ErrKeyNotFound):
        fmt.Println("Key not found - non-inclusion proof generated")
    case errors.Is(err, types.ErrInvalidVersion):
        fmt.Println("Invalid version specified")
    case errors.Is(err, types.ErrStorageFailure):
        fmt.Println("Storage read error")
    default:
        fmt.Printf("Unexpected error: %v\n", err)
    }
}
```

## Complete Example

Here's a comprehensive example demonstrating proof generation and verification:

```go
package main

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    "log"
    
    "github.com/neutral/proofbox/pkg/proof"
    "github.com/neutral/proofbox/pkg/storage"
    "github.com/neutral/proofbox/pkg/storage/pebble"
    "github.com/neutral/proofbox/pkg/tree"
    "github.com/neutral/proofbox/pkg/types"
)

func main() {
    // Initialize storage and tree
    store, err := pebble.NewStorage("./proof_demo.db", nil)
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
    
    // Add test data
    testData := map[string]string{
        "user:alice":   "Alice Smith",
        "user:bob":     "Bob Jones",
        "config:theme": "dark",
    }
    
    for k, v := range testData {
        key := types.KeyHash([]byte(k))
        _, err := jmt.Put(key, []byte(v))
        if err != nil {
            log.Printf("Failed to put %s: %v", k, err)
        }
    }
    
    // Get current version and root hash
    version := jmt.GetLatestVersion()
    rootHash, err := jmt.GetRootHash(version)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Tree version: %d\n", version)
    fmt.Printf("Root hash: %x\n\n", rootHash)
    
    // Generate and verify proofs
    testKeys := []string{"user:alice", "user:charlie", "config:theme"}
    
    for _, keyStr := range testKeys {
        fmt.Printf("=== Proof for key: %s ===\n", keyStr)
        
        // Generate proof
        key := types.KeyHash([]byte(keyStr))
        reader, err := jmt.Reader(version)
        if err != nil {
            log.Fatal(err)
        }
        
        generator := proof.NewGenerator(reader)
        merkleProof, err := generator.Generate(key)
        reader.Close()
        
        if err != nil {
            log.Printf("Failed to generate proof: %v", err)
            continue
        }
        
        // Display proof info
        switch merkleProof.Type {
        case proof.ProofTypeInclusion:
            fmt.Printf("Type: Inclusion\n")
            fmt.Printf("Value: %s\n", string(merkleProof.Value))
        case proof.ProofTypeExclusionEmpty:
            fmt.Printf("Type: Non-inclusion (empty)\n")
        case proof.ProofTypeExclusionNeighbor:
            fmt.Printf("Type: Non-inclusion (neighbor)\n")
        }
        
        // Serialize proof
        codec := proof.NewProofCodec()
        proofData, err := codec.Encode(merkleProof)
        if err != nil {
            log.Printf("Failed to encode: %v", err)
            continue
        }
        
        fmt.Printf("Proof size: %d bytes\n", len(proofData))
        fmt.Printf("Encoded: %s...\n", base64.StdEncoding.EncodeToString(proofData)[:50])
        
        // Verify proof
        verifier := proof.NewVerifier()
        err = verifier.Verify(merkleProof)
        if err != nil {
            log.Printf("Verification error: %v", err)
            fmt.Printf("Verification: failed\n\n")
            continue
        }
        
        fmt.Printf("Verification: passed\n\n")
    }
    
    // Export proof for external use
    exportData, err := exportProofForKey(jmt, "user:alice")
    if err != nil {
        log.Fatal(err)
    }
    
    jsonData, err := json.MarshalIndent(exportData, "", "  ")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Exported proof:\n%s\n", string(jsonData))
}

func exportProofForKey(jmt *tree.Tree, keyStr string) (map[string]interface{}, error) {
    key := types.KeyHash([]byte(keyStr))
    version := jmt.GetLatestVersion()
    
    reader, err := jmt.Reader(version)
    if err != nil {
        return nil, err
    }
    defer reader.Close()
    
    generator := proof.NewGenerator(reader)
    merkleProof, err := generator.Generate(key)
    if err != nil {
        return nil, err
    }
    
    rootHash, err := jmt.GetRootHash(version)
    if err != nil {
        return nil, err
    }
    
    codec := proof.NewProofCodec()
    proofData, err := codec.Encode(merkleProof)
    if err != nil {
        return nil, err
    }
    
    export := map[string]interface{}{
        "key":       keyStr,
        "version":   version,
        "root_hash": fmt.Sprintf("%x", rootHash),
        "proof":     base64.StdEncoding.EncodeToString(proofData),
    }
    
    switch merkleProof.Type {
    case proof.ProofTypeInclusion:
        export["type"] = "inclusion"
        export["value"] = string(merkleProof.Value)
    case proof.ProofTypeExclusionEmpty:
        export["type"] = "non_inclusion_empty" 
    case proof.ProofTypeExclusionNeighbor:
        export["type"] = "non_inclusion_neighbor"
    }
    
    return export, nil
}
```

## Best Practices

### Proof Generation

1. **Reuse Readers**: Create one reader for multiple proofs at same version
2. **Cache Proofs**: Cache frequently requested proofs
3. **Batch Generation**: Generate multiple proofs efficiently
4. **Error Handling**: Always handle both inclusion and non-inclusion cases

### Proof Verification

1. **Verify Root Hash**: Always verify against trusted root hash
2. **Check Proof Type**: Handle inclusion vs non-inclusion differently
3. **Validate Values**: For inclusion proofs, verify value matches
4. **Time Bounds**: Consider proof age for time-sensitive data

### Security Considerations

1. **Root Hash Trust**: The root hash must come from a trusted source
2. **Proof Freshness**: Old proofs may not reflect current state
3. **Value Integrity**: Inclusion proofs include the value
4. **Non-inclusion**: Absence proofs are as important as presence

## Troubleshooting

### Large Proof Sizes

```go
// For deep trees, proofs can be large
// Monitor and optimize:
fmt.Printf("Tree depth estimate: %d\n", len(proof.Siblings))
fmt.Printf("Average siblings per level: %.1f\n", 
    float64(totalSiblings) / float64(len(proof.Siblings)))
```

### Verification Failures

- Ensure root hash matches the proof's version
- Check key encoding is consistent
- Verify proof hasn't been corrupted
- Confirm version exists in tree

## Next Steps

- Learn about [Version Management](version-management.md) for historical proofs
- Explore batch proof generation for efficiency
- Read about proof optimization in the reference docs

## Quick Reference

```go
// Generate proof
reader, _ := jmt.Reader(version)
generator := proof.NewGenerator(reader)
merkleProof, _ := generator.Generate(key)
reader.Close()

// Check proof type
if merkleProof.IsInclusion() {
    // Key exists, value available
} else {
    // Key doesn't exist
}

// Verify proof
verifier := proof.NewVerifier()
valid, _ := verifier.Verify(merkleProof, rootHash)

// Serialize/Deserialize
encoded, _ := merkleProof.Encode()
decoded := &proof.Proof{}
decoded.Decode(encoded)
```