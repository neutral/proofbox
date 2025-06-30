# Node Cache

## Purpose

Provides thread-safe caching of tree nodes to reduce storage I/O and improve read performance.

## Design

Built on top of hashicorp/golang-lru for proven LRU eviction:
- Fixed maximum size prevents memory exhaustion
- Thread-safe operations using RWMutex
- Atomic counters for lock-free statistics

## Performance

- Get operations use read lock (concurrent reads)
- Put/Remove use write lock (brief contention)
- Hit/miss tracking with atomic operations
- Zero allocations for cache key encoding

## Cache Key Encoding

Uses the same encoding as storage keys for efficiency:
- No additional allocations
- Direct string conversion
- Unique keys across versions and paths

## Statistics

Tracks:
- Hit count and rate
- Miss count
- Current size

Useful for monitoring cache effectiveness and tuning size.

## Future Improvements

- Bloom filter for existence checks
- Separate caches for internal vs leaf nodes
- Adaptive sizing based on hit rate
- Write-through vs write-back strategies