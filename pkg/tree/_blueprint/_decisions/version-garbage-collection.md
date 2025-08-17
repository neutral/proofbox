# ADR-015: Version Garbage Collection

## Status
Accepted

## Context
Without cleanup, version metadata accumulates indefinitely, causing unbounded memory growth. We need a configurable way to remove old versions while preserving required history.

## Decision
Implement pluggable retention policies:
1. **Count-based**: Keep last N versions
2. **Time-based**: Keep versions newer than duration
3. **None**: Keep all versions (no GC)

Default to keeping last 100 versions. GC only removes version metadata, not tree nodes (storage pruning is a separate concern).

## Consequences
### Positive
- Prevents unbounded memory growth
- Configurable for different use cases
- Simple policies that are easy to understand
- Can be extended with custom policies

### Negative
- Storage still grows (only metadata is collected)
- Must be called explicitly (not automatic)
- May remove versions still referenced by readers
- Full storage pruning requires more complex implementation

## Implementation Notes
- Never remove the current committed version
- Time-based policy still respects minimum count
- Removed versions return error on access attempts
- Future work: reference-counted storage pruning