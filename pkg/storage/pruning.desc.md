# Storage Pruning Policies

This module defines policies for automatic cleanup of old versions in versioned storage systems, helping manage disk space while preserving required historical data.

## Purpose

Pruning policies provide configurable strategies for removing old versions:
- Prevent unbounded storage growth in long-running systems
- Balance between history retention and resource usage
- Support different retention strategies for various use cases

## Policy Types

### KeepLastN
Retains only the most recent N versions. This is ideal for systems that need a fixed-size rolling history window, such as maintaining the last 100 versions for rollback capability.

### KeepAfterTime
Removes versions older than a specified duration. Useful for compliance scenarios where data must be retained for a specific time period (e.g., 30 days for audit trails).

### KeepNothing
Aggressive pruning that removes all old versions except the current one. Suitable for systems that don't require historical data and need to minimize storage usage.

### Custom Policy
The PruningPolicy interface allows implementation of custom retention strategies, such as keeping every Nth version or preserving versions that match specific criteria.

## Design Decisions

- **Interface-based Design**: The PruningPolicy interface allows easy extension with custom policies without modifying core storage code
- **Simple Predicates**: Policies are implemented as simple predicates that return true if a version should be pruned
- **Timestamp Support**: Policies can access version timestamps for time-based retention decisions
- **Composability**: Multiple policies can be combined using logical operators for complex retention rules

## Integration

Pruning policies integrate with the storage layer's garbage collection system. The storage implementation periodically evaluates versions against the configured policy and removes those marked for pruning, typically during low-activity periods to minimize performance impact.