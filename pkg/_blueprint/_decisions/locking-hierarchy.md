# ADR: Locking Hierarchy

## Status
Accepted

## Context
The tree implementation has multiple locks (Tree.writeMu, Tree.mu, VersionManager.mu, NodeCache.mu). Without a defined order, deadlocks are possible when different code paths acquire locks in different sequences.

## Decision
Establish a strict locking hierarchy that must be followed:
1. Tree.writeMu (serializes writes)
2. Tree.mu (protects tree metadata)
3. VersionManager.mu (protects version state)
4. NodeCache.mu (protects cache)

Locks must be acquired in this order and released in reverse order.

## Consequences
### Positive
- Prevents deadlocks by construction
- Clear rules for developers
- Easy to verify in code review
- Documented in locking.md

### Negative
- Must be careful about lock ordering
- May require restructuring code to follow hierarchy
- Additional cognitive load for developers

## Implementation Notes
- Use defer for unlock to ensure cleanup
- Never hold multiple locks at same level
- Document the hierarchy prominently
- Consider lock-free alternatives for hot paths