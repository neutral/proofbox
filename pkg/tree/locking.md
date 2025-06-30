# Locking Hierarchy Documentation

## Lock Order (MUST be acquired in this order to prevent deadlocks):

1. **Tree.writeMu** (Mutex) - Serializes all write operations
2. **Tree.mu** (RWMutex) - Protects tree metadata (rootHashes, latestVer)
3. **VersionManager.mu** (RWMutex) - Protects version state
4. **NodeCache.mu** (RWMutex) - Protects cache state

## Rules:

1. **Never hold multiple locks at the same level**
2. **Always acquire locks in the order above**
3. **Release locks in reverse order**
4. **Use defer for unlock to ensure cleanup**

## Lock Responsibilities:

### Tree.writeMu
- Guards all write operations (Put, Delete, Commit, Abort)
- Ensures single writer at a time
- Must be held for entire write operation

### Tree.mu
- Guards rootHashes map
- Guards latestVer field
- Can be held concurrently by readers

### VersionManager.mu
- Guards version maps and state
- Guards pending operations
- Internal to VersionManager

### NodeCache.mu
- Guards LRU cache state
- Internal to cache operations
- Never held across external calls