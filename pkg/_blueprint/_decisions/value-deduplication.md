---
id: decision.012.value-deduplication
tags: [decision, storage, deduplication]
---

# ADR: Value Deduplication Strategy

## Status

Accepted

## Context

In a versioned Merkle tree, the same value may appear across multiple versions. We need to decide whether to:
1. Store each value separately per version
2. Deduplicate identical values across versions

## Decision

Use content-addressed storage without version in the key to enable cross-version deduplication.

**Key format**: `'v' || hash(32 bytes)`

## Rationale

### Benefits of Deduplication

1. **Storage Efficiency**: Common values (e.g., zero balances, default configs) stored only once
2. **Network Efficiency**: When syncing, identical values transferred only once  
3. **Cache Efficiency**: One cache entry serves all versions
4. **Simplicity**: No version management for values

### Addressing Concerns

1. **Garbage Collection**: Can use reference counting or generation-based GC
2. **Hash Collisions**: SHA-256 provides 2^256 space, collisions are cryptographically improbable
3. **Version Isolation**: Tree structure provides version isolation, values don't need it

## Implementation

```go
// Store value (any version)
valueKey := []byte{'v'} + sha256(value)
db.Set(valueKey, value)

// Retrieve value (any version)
value := db.Get(valueKey)
verify(sha256(value) == expectedHash)
```

## Consequences

### Positive

- Significant storage savings for repeated values
- Better cache utilization
- Simpler value management

### Negative

- Cannot delete values when specific version is deleted (need GC strategy)
- All versions affected if a value is corrupted (mitigated by hash verification)

### Neutral

- Standard practice in content-addressed systems (Git, IPFS, etc.)
- Compatible with future sharding strategies

## Alternatives Considered

1. **Version in key**: `'v' || version || hash`
   - Rejected: No deduplication benefits
   - Would increase storage linearly with versions

2. **Hybrid approach**: Deduplicate only large values
   - Rejected: Complexity without clear benefit
   - All values benefit from deduplication

## References

- Git object storage model
- IPFS content addressing
- Bitcoin UTXO set management