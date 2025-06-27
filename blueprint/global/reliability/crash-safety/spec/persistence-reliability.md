# Persistence and Reliability Specification

## Overview
JMT's versioned, append-only design provides strong reliability and crash-safety guarantees essential for blockchain systems.

## Crash Safety Properties

### Append-Only Design
- Never modifies existing nodes
- New versions only add nodes
- Partial writes don't corrupt existing state
- Natural write-ahead logging via version ordering

### Atomic Version Commits
- All nodes for version written together
- Database transaction ensures atomicity
- Either version fully exists or doesn't
- No partial state visible

### Recovery Properties
After crash:
1. Find highest complete version in storage
2. Root at that version represents consistent state
3. Incomplete higher versions ignored
4. Resume from last complete version

## Persistence Guarantees

### Durability
- Nodes written to persistent storage
- Leverages database durability (fsync, journaling)
- Version number tracks committed state
- Root hash authenticates entire state

### Consistency
- Each version internally consistent
- Tree structure invariants maintained
- No dangling references between versions
- Cryptographic verification possible

## Operational Reliability

### Backup Strategy
Version-based structure enables:
- Incremental backups (only new versions)
- Point-in-time recovery (any version)
- Consistent snapshots without stopping writes
- Parallel backup of version ranges

### Replication Support
- Read replicas can lag by versions
- Version number provides sync point
- Catch-up via replaying versions
- Verification via root hash comparison

### Error Detection
- Cryptographic hashes detect corruption
- Version sequence detects missing data
- Tree structure violations caught during traversal
- Background verification possible

## Implementation Requirements

### Storage Layer
Must provide:
- Atomic batch writes
- Ordered key iteration
- Point lookups by key
- Durability guarantees

### Write Pattern
1. Accumulate all changes for version
2. Compute all new nodes
3. Write batch atomically
4. Update version pointer
5. Confirm durability

### Read Pattern
- Always read from complete version
- Never read partially written version
- Version isolation prevents inconsistency
- Verify hashes during traversal (optional)

## Failure Scenarios Handled
- Power loss during write: Incomplete version ignored
- Disk corruption: Detected by hash mismatch
- Partial write: Atomic batch prevents
- Network partition: Version number coordinates