# Validation Requirements Specification

## Overview
This specification defines comprehensive validation requirements for the Jellyfish Merkle Tree implementation to ensure data integrity, security, and proper error handling.

## Input Validation

### Key Validation
```go
// ValidateKey ensures a key is valid
func ValidateKey(key Key) error {
    // Keys are fixed size, check for zero key
    if key == (Key{}) {
        return ErrEmptyKey
    }
    return nil
}
```

### Nibble Validation
```go
// ValidateNibble ensures nibble is in valid range [0-15]
func ValidateNibble(n Nibble) error {
    if n > MaxNibbleValue {
        return fmt.Errorf("invalid nibble %d: exceeds max value %d", n, MaxNibbleValue)
    }
    return nil
}
```

### Version Validation
```go
// ValidateVersion ensures version is valid and prevents overflow
func ValidateVersion(v Version) error {
    if v > MaxVersion {
        return fmt.Errorf("invalid version %d: exceeds max version %d", v, MaxVersion)
    }
    return nil
}

// ValidateVersionIncrement checks version increases by exactly 1
func ValidateVersionIncrement(oldVersion, newVersion Version) error {
    if oldVersion == MaxVersion {
        return fmt.Errorf("version overflow: cannot increment beyond %d", MaxVersion)
    }
    if newVersion != oldVersion + 1 {
        return fmt.Errorf("invalid version increment: expected %d, got %d", oldVersion+1, newVersion)
    }
    return nil
}
```

### Value Size Validation
```go
// ValidateValueSize ensures value doesn't exceed maximum
func ValidateValueSize(value []byte) error {
    if len(value) > MaxValueSize {
        return fmt.Errorf("value too large: %d bytes exceeds max %d", len(value), MaxValueSize)
    }
    return nil
}
```

## Tree Structure Validation

### Depth Validation
```go
// ValidateTreeDepth ensures we don't exceed maximum tree depth
func ValidateTreeDepth(depth int) error {
    if depth < 0 {
        return fmt.Errorf("invalid tree depth: negative value %d", depth)
    }
    if depth > MaxTreeDepth {
        return fmt.Errorf("tree depth %d exceeds maximum %d", depth, MaxTreeDepth)
    }
    return nil
}
```

### Node Children Validation
```go
// ValidateInternalNode ensures internal node constraints
func ValidateInternalNode(node *InternalNode) error {
    if node == nil {
        return fmt.Errorf("internal node is nil")
    }
    
    numChildren := node.NumChildren()
    if numChildren == 0 {
        return fmt.Errorf("internal node has no children")
    }
    if numChildren > 16 {
        return fmt.Errorf("internal node has too many children: %d > 16", numChildren)
    }
    
    // Validate each child reference
    children := node.Children()
    for nibble, child := range children {
        if err := ValidateNibble(nibble); err != nil {
            return fmt.Errorf("invalid child nibble: %w", err)
        }
        if child.Hash == (Hash{}) {
            return fmt.Errorf("child at nibble %d has empty hash", nibble)
        }
        if err := ValidateVersion(child.Version); err != nil {
            return fmt.Errorf("child at nibble %d: %w", nibble, err)
        }
    }
    
    return nil
}
```

### Path Validation
```go
// ValidateNibblePath ensures path is valid for tree operations
func ValidateNibblePath(path []Nibble, currentDepth int) error {
    if len(path) == 0 {
        return fmt.Errorf("empty nibble path")
    }
    
    totalDepth := currentDepth + len(path)
    if err := ValidateTreeDepth(totalDepth); err != nil {
        return fmt.Errorf("path would exceed max depth: %w", err)
    }
    
    for i, nibble := range path {
        if err := ValidateNibble(nibble); err != nil {
            return fmt.Errorf("invalid nibble at position %d: %w", i, err)
        }
    }
    
    return nil
}
```

## Operation Validation

### Batch Operation Validation
```go
// ValidateBatchSize ensures batch doesn't exceed limits
func ValidateBatchSize(operations int) error {
    if operations <= 0 {
        return fmt.Errorf("batch must contain at least one operation")
    }
    if operations > MaxBatchSize {
        return fmt.Errorf("batch size %d exceeds maximum %d", operations, MaxBatchSize)
    }
    return nil
}

// ValidateBatchOperations checks for conflicts within a batch
func ValidateBatchOperations(ops []BatchOperation) error {
    if err := ValidateBatchSize(len(ops)); err != nil {
        return err
    }
    
    // Check for duplicate keys
    seen := make(map[Key]int)
    for i, op := range ops {
        if err := ValidateKey(op.Key); err != nil {
            return fmt.Errorf("operation %d: %w", i, err)
        }
        
        if prevIndex, exists := seen[op.Key]; exists {
            return fmt.Errorf("duplicate key in batch: operations %d and %d both modify key %x", 
                prevIndex, i, op.Key)
        }
        seen[op.Key] = i
        
        // Validate operation-specific constraints
        switch op.Type {
        case OpTypeInsert, OpTypeUpdate:
            if err := ValidateValueSize(op.Value); err != nil {
                return fmt.Errorf("operation %d: %w", i, err)
            }
        case OpTypeDelete:
            // Delete operations don't need value
            if len(op.Value) > 0 {
                return fmt.Errorf("operation %d: delete operation should not have value", i)
            }
        default:
            return fmt.Errorf("operation %d: unknown operation type %v", i, op.Type)
        }
    }
    
    return nil
}
```

## Storage Validation

### Storage Key Validation
```go
// ValidateStorageKey ensures storage keys are well-formed
func ValidateStorageKey(key []byte) error {
    if len(key) == 0 {
        return fmt.Errorf("empty storage key")
    }
    
    // Check prefix
    switch key[0] {
    case 'n': // Node key
        if len(key) < 10 { // prefix(1) + version(8) + path(1+)
            return fmt.Errorf("node key too short: %d bytes", len(key))
        }
    case 'r': // Root key  
        if len(key) != 9 { // prefix(1) + version(8)
            return fmt.Errorf("root key wrong size: expected 9, got %d", len(key))
        }
    case 'v': // Value key
        if len(key) != 41 { // prefix(1) + version(8) + hash(32)
            return fmt.Errorf("value key wrong size: expected 41, got %d", len(key))
        }
    default:
        return fmt.Errorf("unknown storage key prefix: %c", key[0])
    }
    
    return nil
}
```

## Proof Validation

### Merkle Proof Validation
```go
// ValidateProof ensures a Merkle proof is well-formed
func ValidateProof(proof *MerkleProof) error {
    if proof == nil {
        return fmt.Errorf("proof is nil")
    }
    
    // Validate key
    if err := ValidateKey(proof.Key); err != nil {
        return fmt.Errorf("invalid proof key: %w", err)
    }
    
    // Validate siblings
    if len(proof.Siblings) > MaxTreeDepth {
        return fmt.Errorf("too many siblings in proof: %d > %d", 
            len(proof.Siblings), MaxTreeDepth)
    }
    
    for i, sibling := range proof.Siblings {
        if sibling.Hash == (Hash{}) && sibling.Hash != EmptyTreeHash {
            return fmt.Errorf("invalid sibling hash at position %d", i)
        }
    }
    
    // Validate value for inclusion proofs
    if proof.Type == ProofTypeInclusion {
        if len(proof.Value) == 0 {
            return fmt.Errorf("inclusion proof missing value")
        }
        if err := ValidateValueSize(proof.Value); err != nil {
            return fmt.Errorf("invalid proof value: %w", err)
        }
    }
    
    return nil
}
```

## Concurrency Validation

### Transaction Validation
```go
// ValidateTransaction ensures transaction state is consistent
func ValidateTransaction(tx *Transaction) error {
    if tx == nil {
        return fmt.Errorf("transaction is nil")
    }
    
    // Check version consistency
    if tx.readVersion > tx.writeVersion {
        return fmt.Errorf("read version %d > write version %d", 
            tx.readVersion, tx.writeVersion)
    }
    
    // Validate all pending operations
    if err := ValidateBatchOperations(tx.pendingOps); err != nil {
        return fmt.Errorf("invalid pending operations: %w", err)
    }
    
    return nil
}
```

## Error Handling Requirements

### Error Types
All validation functions must return descriptive errors that include:
1. What validation failed
2. The invalid value(s)
3. The expected constraints

### Error Wrapping
Use error wrapping to maintain context:
```go
if err := ValidateKey(key); err != nil {
    return fmt.Errorf("failed to validate key in operation %d: %w", opIndex, err)
}
```

### No Panics
Validation functions must NEVER panic. All error conditions must be returned as errors.

## Performance Considerations

1. **Early Validation**: Validate inputs as early as possible to avoid wasted computation
2. **Batch Validation**: When validating multiple items, collect all errors rather than failing on first
3. **Cached Validation**: For expensive validations, consider caching results for immutable data

## Security Requirements

1. **Bounds Checking**: All array/slice access must be bounds-checked
2. **Integer Overflow**: Check for overflow before arithmetic operations
3. **Resource Limits**: Enforce maximum sizes to prevent DoS attacks
4. **Input Sanitization**: Never trust external input without validation

## Testing Requirements

Each validation function must have tests covering:
1. Valid inputs (positive cases)
2. Each validation failure condition
3. Boundary conditions (min/max values)
4. Nil/empty inputs
5. Concurrent access (where applicable)