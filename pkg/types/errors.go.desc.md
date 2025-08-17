# Error Types Description

## Overview
This file defines all error types used throughout the Jellyfish Merkle Tree implementation. Errors are organized into logical categories to make error handling consistent and predictable.

## Design Principles

### Sentinel Errors
We use sentinel errors (pre-declared error variables) rather than creating new error instances. This approach:
- Enables reliable error checking with `errors.Is()`
- Reduces allocations in hot paths
- Makes error types discoverable through IDE autocomplete
- Ensures consistent error messages

### Error Categories

#### 1. Data Integrity Errors
Critical errors indicating corruption or inconsistency:
- `ErrCorruptedNode`: Node hash doesn't match content - data corruption detected
- `ErrInvalidNodeType`: Unknown node type in storage - likely version mismatch
- `ErrHashMismatch`: Computed vs stored hash mismatch - integrity violation
- `ErrInvalidProof`: Malformed or invalid proof - security concern

#### 2. Operational Errors
Normal "not found" conditions that occur during regular operations:
- `ErrKeyNotFound`: Key doesn't exist in tree - expected for lookups
- `ErrVersionNotFound`: Version doesn't exist - normal for old versions
- `ErrNodeNotFound`: Referenced node missing - could be pruned
- `ErrEmptyTree`: Operation on empty tree - initial state

#### 3. Input Validation Errors
Invalid input from callers that should be caught early:
- `ErrInvalidKey`: Key format/size incorrect
- `ErrEmptyKey`: Empty keys not allowed
- `ErrInvalidVersion`: Version number invalid (e.g., future version)
- `ErrValueTooLarge`: Value exceeds configured limits
- `ErrInvalidNibble`: Nibble value > 15
- `ErrInvalidNodeKey`: Malformed node key structure
- `ErrInvalidNibbleCount`: Path length exceeds tree depth

#### 4. Resource Errors
Resource exhaustion that prevents operation:
- `ErrMaxDepthExceeded`: Tree depth limit hit - prevents infinite loops
- `ErrBatchTooLarge`: Batch exceeds memory/processing limits
- `ErrOutOfMemory`: Memory allocation failed

#### 5. Storage Errors
Underlying storage layer failures:
- `ErrStorageFailure`: Generic storage operation failure
- `ErrTransactionAborted`: Transaction rolled back
- `ErrDatabaseClosed`: Operation on closed database

#### 6. Transaction Errors
Concurrency and transaction state issues:
- `ErrTxConflict`: Concurrent modification conflict
- `ErrTxStateInvalid`: Invalid transaction state transition

## Usage Guidelines

### Error Checking
Always use `errors.Is()` for checking error types:
```go
if errors.Is(err, types.ErrKeyNotFound) {
    // Handle missing key
}
```

### Error Context
Use `WrapError()` to add context while preserving the error type:
```go
if err != nil {
    return types.WrapError(err, "failed to load node at version %d", version)
}
```

### Recovery Decisions
Different error categories warrant different recovery strategies:
- **Data Integrity**: Fail fast, alert operators
- **Operational**: Return gracefully, these are expected
- **Input Validation**: Return immediately with clear error
- **Resource**: May retry with backoff or smaller batch
- **Storage**: Often retryable, check `IsRetryable()`
- **Transaction**: May retry or require application-level handling

## Consistency with Specification
These error types directly implement the error categories defined in `../_blueprint/_specs/error-handling.md`, ensuring consistency between specification and implementation.
