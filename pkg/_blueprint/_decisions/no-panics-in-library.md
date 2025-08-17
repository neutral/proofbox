---
id: adr.no-panics
status: accepted
date: 2024-01-15
---

# ADR: No Panics in Library Code

## Status

Accepted

## Context

When encountering errors or invalid states, library code can either:
1. Panic and crash the program
2. Return errors for the caller to handle

This decision affects API usability, reliability, and debugging.

## Decision

Never use panic in library code. All errors must be returned as error values for the caller to handle.

## Consequences

### Positive
- **Reliability**: Library never crashes the host application
- **Error Handling**: Callers can decide how to handle errors
- **Testing**: Easier to test error conditions
- **Production Safety**: No unexpected crashes in production
- **Debugging**: Clear error messages with context

### Negative
- **API Verbosity**: More error returns to check
- **Error Propagation**: Must propagate errors up call stack
- **No Fail-Fast**: Invalid states might persist longer

### Neutral
- Standard Go practice for libraries
- Aligns with Go proverbs: "Don't panic"

## Implementation Details

Instead of panic:
```go
// BAD
if nibble > 15 {
    panic(fmt.Sprintf("invalid nibble: %d", nibble))
}

// GOOD
if nibble > 15 {
    return fmt.Errorf("invalid nibble: %d", nibble)
}
```

Error wrapping for context:
```go
if err := validateKey(key); err != nil {
    return nil, fmt.Errorf("invalid key for leaf node: %w", err)
}
```

## Alternatives Considered

1. **Panic on Programming Errors**
   - Panic for "impossible" states (nil receiver, etc.)
   - Rejected: Even programming errors should be recoverable

2. **Panic with Recover**
   - Provide recovery mechanism
   - Rejected: Complex, non-idiomatic in Go

3. **Debug vs Release Behavior**
   - Panic in debug, error in release
   - Rejected: Inconsistent behavior

4. **Assertion Library**
   - Use assert package that can be disabled
   - Rejected: Hidden control flow

## Guidelines

DO:
- Return descriptive errors
- Include context in error messages
- Use error wrapping with %w
- Document which errors functions can return

DON'T:
- Use panic for any error condition
- Ignore errors that "can't happen"
- Return generic errors without context

## References

- Effective Go: "Panic and Recover"
- Go Code Review Comments
- Fixed in blueprint during step 06 review