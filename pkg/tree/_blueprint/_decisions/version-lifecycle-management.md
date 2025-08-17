# ADR-013: Version Lifecycle Management

## Status

Accepted

## Context

The Jellyfish Merkle Tree needs to support multiple versions for state management. This requires careful management of version states to prevent inconsistencies and resource leaks.

## Decision

We implement explicit version lifecycle management with three states:

- **Pending**: Version is being constructed, modifications allowed
- **Committed**: Version is finalized, read-only
- **Aborted**: Version was cancelled, not accessible

Versions must be explicitly begun, and then either committed or aborted. No implicit version creation occurs.

## Consequences

### Positive

- Clear state transitions prevent invalid operations
- Resource cleanup is explicit (abort releases resources)
- Multiple pending versions can exist concurrently
- Failed operations don't leave partial state

### Negative

- More verbose API (must call BeginVersion)
- Must remember to commit or abort
- Slightly more complex than auto-versioning

## Implementation Notes

- VersionManager tracks all version states
- Only committed versions can be parent versions
- Aborted versions are retained for diagnostics but not accessible for reads
