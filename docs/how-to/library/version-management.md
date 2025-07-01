# Version Management with ProofBox Library

This guide demonstrates how to work with ProofBox's versioning system to query historical data, manage versions, and implement time-travel queries.

## Goal

Learn to leverage ProofBox's append-only versioning system for historical queries, rollback capabilities, and multi-version data management.

## Prerequisites

- ProofBox library integrated in your project
- Understanding of basic ProofBox operations
- Familiarity with version control concepts

## Understanding Versions in ProofBox

### Version Concepts

1. **Append-Only**: Each write creates a new version
2. **Immutable History**: Past versions never change
3. **Sequential**: Versions increment monotonically
4. **Persistent**: All versions are retained by default

### Version Lifecycle

- Version 0: Initial empty tree state
- Each Put/Delete increments version
- Batch operations create single version
- Versions are never reused

## Basic Version Operations

### Setting Up

```go
import (
    "fmt"
    "log"
    "time"
    
    "github.com/neutral/proofbox/pkg/storage"
    "github.com/neutral/proofbox/pkg/storage/pebble"
    "github.com/neutral/proofbox/pkg/tree"
    "github.com/neutral/proofbox/pkg/types"
)

// Initialize tree as usual
store, err := pebble.NewStorage("./versions.db", nil)
if err != nil {
    log.Fatal(err)
}
defer store.Close()

keyEncoder := storage.NewDefaultKeyEncoder()
config := tree.DefaultTreeConfig()
jmt, err := tree.NewTree(store, keyEncoder, config)
if err != nil {
    log.Fatal(err)
}
```

### Tracking Versions

```go
// Get current version
currentVersion := jmt.GetLatestVersion()
fmt.Printf("Current version: %d\n", currentVersion)

// Track version changes
key := types.KeyHash([]byte("config:setting"))
value := []byte("value1")

// Each operation returns the new version
version1, err := jmt.Put(key, value)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("After put: version %d\n", version1)

// Update creates new version
value2 := []byte("value2")
version2, err := jmt.Put(key, value2)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("After update: version %d\n", version2)

// Delete also creates new version
version3, err := jmt.Delete(key)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("After delete: version %d\n", version3)
```

### Reading at Specific Versions

```go
// Read value at different versions
key := types.KeyHash([]byte("user:data"))

// Get at specific version
value, err := jmt.Get(version1, key)
if err != nil {
    log.Printf("Error at version %d: %v", version1, err)
} else if value != nil {
    fmt.Printf("Value at version %d: %s\n", version1, string(value))
} else {
    fmt.Printf("Key not found at version %d\n", version1)
}

// Compare across versions
for v := types.Version(0); v <= jmt.GetLatestVersion(); v++ {
    value, err := jmt.Get(v, key)
    if err != nil {
        continue
    }
    if value != nil {
        fmt.Printf("Version %d: %s\n", v, string(value))
    } else {
        fmt.Printf("Version %d: <not found>\n", v)
    }
}
```

## Advanced Version Management

### Version History Tracking

```go
// Track version metadata
type VersionMetadata struct {
    Version   types.Version
    Timestamp time.Time
    Operation string
    KeyCount  int
    RootHash  types.Hash
}

type VersionTracker struct {
    tree     *tree.Tree
    metadata map[types.Version]*VersionMetadata
}

func NewVersionTracker(tree *tree.Tree) *VersionTracker {
    return &VersionTracker{
        tree:     tree,
        metadata: make(map[types.Version]*VersionMetadata),
    }
}

func (vt *VersionTracker) RecordOperation(operation string) error {
    version := vt.tree.GetLatestVersion()
    rootHash, err := vt.tree.GetRootHash(version)
    if err != nil {
        return err
    }
    
    vt.metadata[version] = &VersionMetadata{
        Version:   version,
        Timestamp: time.Now(),
        Operation: operation,
        RootHash:  rootHash,
    }
    
    return nil
}

// Usage
tracker := NewVersionTracker(jmt)

// Track operations
key := types.KeyHash([]byte("data"))
jmt.Put(key, []byte("value"))
tracker.RecordOperation("Added initial data")

jmt.Put(key, []byte("updated"))
tracker.RecordOperation("Updated data value")

// Query history
for v, meta := range tracker.metadata {
    fmt.Printf("Version %d: %s at %s\n", 
        v, meta.Operation, meta.Timestamp.Format(time.RFC3339))
}
```

### Time-Travel Queries

```go
// Query data as it existed at specific time
type TimeTravelDB struct {
    tree        *tree.Tree
    versionTime map[types.Version]time.Time
}

func NewTimeTravelDB(tree *tree.Tree) *TimeTravelDB {
    return &TimeTravelDB{
        tree:        tree,
        versionTime: make(map[types.Version]time.Time),
    }
}

func (tt *TimeTravelDB) Put(key types.Key, value []byte) (types.Version, error) {
    version, err := tt.tree.Put(key, value)
    if err == nil {
        tt.versionTime[version] = time.Now()
    }
    return version, err
}

func (tt *TimeTravelDB) GetAtTime(key types.Key, timestamp time.Time) ([]byte, types.Version, error) {
    // Find the latest version before timestamp
    var targetVersion types.Version
    found := false
    
    for v := tt.tree.GetLatestVersion(); v >= 0; v-- {
        if vTime, exists := tt.versionTime[v]; exists && vTime.Before(timestamp) {
            targetVersion = v
            found = true
            break
        }
    }
    
    if !found {
        return nil, 0, fmt.Errorf("no version found before %v", timestamp)
    }
    
    value, err := tt.tree.Get(targetVersion, key)
    return value, targetVersion, err
}

// Usage
ttdb := NewTimeTravelDB(jmt)

// Add data with timestamps
key := types.KeyHash([]byte("stock:price"))
ttdb.Put(key, []byte("100.50"))
time.Sleep(time.Second)
ttdb.Put(key, []byte("101.25"))
time.Sleep(time.Second)
ttdb.Put(key, []byte("99.75"))

// Query at different times
queryTime := time.Now().Add(-1 * time.Second)
value, version, err := ttdb.GetAtTime(key, queryTime)
if err == nil && value != nil {
    fmt.Printf("Price at %v (v%d): %s\n", queryTime.Format(time.RFC3339), version, string(value))
}
```

### Version Branching (Conceptual)

```go
// Simulate branching by tracking version lineage
type VersionBranch struct {
    tree       *tree.Tree
    branches   map[string]types.Version  // branch name -> head version
    lineage    map[types.Version]string  // version -> branch name
}

func NewVersionBranch(tree *tree.Tree) *VersionBranch {
    return &VersionBranch{
        tree:     tree,
        branches: make(map[string]types.Version),
        lineage:  make(map[types.Version]string),
    }
}

func (vb *VersionBranch) CreateBranch(name string, fromVersion types.Version) error {
    // Validate version exists
    _, err := vb.tree.GetRootHash(fromVersion)
    if err != nil {
        return fmt.Errorf("invalid base version %d: %w", fromVersion, err)
    }
    
    vb.branches[name] = fromVersion
    return nil
}

func (vb *VersionBranch) SwitchBranch(name string) (types.Version, error) {
    version, exists := vb.branches[name]
    if !exists {
        return 0, fmt.Errorf("branch %s not found", name)
    }
    return version, nil
}

// Note: Actual branching would require tree modifications
// This is conceptual - ProofBox doesn't natively support branches
```

## Version Analysis

### Version Comparison

```go
// Compare data between versions
func compareVersions(tree *tree.Tree, key types.Key, v1, v2 types.Version) {
    value1, err1 := tree.Get(v1, key)
    value2, err2 := tree.Get(v2, key)
    
    fmt.Printf("Comparing key %x:\n", key)
    fmt.Printf("  Version %d: ", v1)
    if err1 != nil {
        fmt.Printf("error: %v\n", err1)
    } else if value1 == nil {
        fmt.Printf("<not found>\n")
    } else {
        fmt.Printf("%s\n", string(value1))
    }
    
    fmt.Printf("  Version %d: ", v2)
    if err2 != nil {
        fmt.Printf("error: %v\n", err2)
    } else if value2 == nil {
        fmt.Printf("<not found>\n")
    } else {
        fmt.Printf("%s\n", string(value2))
    }
    
    if err1 == nil && err2 == nil {
        if value1 == nil && value2 == nil {
            fmt.Println("  Status: Both versions missing key")
        } else if value1 == nil {
            fmt.Println("  Status: Key added")
        } else if value2 == nil {
            fmt.Println("  Status: Key deleted")
        } else if string(value1) != string(value2) {
            fmt.Println("  Status: Value changed")
        } else {
            fmt.Println("  Status: No change")
        }
    }
}
```

### Version Range Queries

```go
// Find all changes to a key within version range
func getKeyHistory(tree *tree.Tree, key types.Key, startVersion, endVersion types.Version) []types.Version {
    var changes []types.Version
    var lastValue []byte
    
    for v := startVersion; v <= endVersion; v++ {
        value, err := tree.Get(v, key)
        if err != nil {
            continue
        }
        
        // Check if value changed
        changed := false
        if v == startVersion {
            changed = true
        } else if (value == nil) != (lastValue == nil) {
            changed = true
        } else if value != nil && string(value) != string(lastValue) {
            changed = true
        }
        
        if changed {
            changes = append(changes, v)
            lastValue = value
        }
    }
    
    return changes
}

// Usage
key := types.KeyHash([]byte("config:feature"))
changes := getKeyHistory(jmt, key, 0, jmt.GetLatestVersion())
fmt.Printf("Key changed in versions: %v\n", changes)
```

### Version Statistics

```go
// Analyze version growth and patterns
type VersionStats struct {
    TotalVersions   int
    VersionsPerHour map[time.Time]int
    AverageGrowth   float64
}

func analyzeVersionGrowth(tracker *VersionTracker) *VersionStats {
    stats := &VersionStats{
        VersionsPerHour: make(map[time.Time]int),
    }
    
    // Count versions and group by hour
    for _, meta := range tracker.metadata {
        stats.TotalVersions++
        
        hourKey := meta.Timestamp.Truncate(time.Hour)
        stats.VersionsPerHour[hourKey]++
    }
    
    // Calculate average growth
    if len(stats.VersionsPerHour) > 0 {
        total := 0
        for _, count := range stats.VersionsPerHour {
            total += count
        }
        stats.AverageGrowth = float64(total) / float64(len(stats.VersionsPerHour))
    }
    
    return stats
}
```

## Performance Considerations

### Version Reader Caching

```go
// Cache readers for frequently accessed versions
type ReaderCache struct {
    tree    *tree.Tree
    cache   map[types.Version]tree.TreeReaderInterface
    maxSize int
}

func NewReaderCache(tree *tree.Tree, maxSize int) *ReaderCache {
    return &ReaderCache{
        tree:    tree,
        cache:   make(map[types.Version]tree.TreeReaderInterface),
        maxSize: maxSize,
    }
}

func (rc *ReaderCache) GetReader(version types.Version) (tree.TreeReaderInterface, error) {
    // Check cache
    if reader, exists := rc.cache[version]; exists {
        return reader, nil
    }
    
    // Create new reader
    reader, err := rc.tree.Reader(version)
    if err != nil {
        return nil, err
    }
    
    // Add to cache (simple eviction - remove oldest)
    if len(rc.cache) >= rc.maxSize {
        // Find and remove oldest
        var oldestVersion types.Version
        for v := range rc.cache {
            if oldestVersion == 0 || v < oldestVersion {
                oldestVersion = v
            }
        }
        if old, exists := rc.cache[oldestVersion]; exists {
            old.Close()
            delete(rc.cache, oldestVersion)
        }
    }
    
    rc.cache[version] = reader
    return reader, nil
}

func (rc *ReaderCache) Close() {
    for _, reader := range rc.cache {
        reader.Close()
    }
}
```

### Efficient Multi-Version Queries

```go
// Query multiple keys across versions efficiently
func multiVersionQuery(tree *tree.Tree, keys []string, versions []types.Version) map[string]map[types.Version][]byte {
    results := make(map[string]map[types.Version][]byte)
    
    // Initialize result structure
    for _, key := range keys {
        results[key] = make(map[types.Version][]byte)
    }
    
    // Use reader cache for efficiency
    readerCache := NewReaderCache(tree, 10)
    defer readerCache.Close()
    
    for _, version := range versions {
        reader, err := readerCache.GetReader(version)
        if err != nil {
            log.Printf("Failed to get reader for version %d: %v", version, err)
            continue
        }
        
        for _, keyStr := range keys {
            key := types.KeyHash([]byte(keyStr))
            value, err := tree.Get(version, key)
            if err == nil && value != nil {
                results[keyStr][version] = value
            }
        }
    }
    
    return results
}
```

## Version Management Patterns

### Checkpoint System

```go
// Create named checkpoints at important versions
type CheckpointManager struct {
    tree        *tree.Tree
    checkpoints map[string]types.Version
}

func NewCheckpointManager(tree *tree.Tree) *CheckpointManager {
    return &CheckpointManager{
        tree:        tree,
        checkpoints: make(map[string]types.Version),
    }
}

func (cm *CheckpointManager) CreateCheckpoint(name string) error {
    version := cm.tree.GetLatestVersion()
    
    // Verify version is valid
    _, err := cm.tree.GetRootHash(version)
    if err != nil {
        return fmt.Errorf("invalid version %d: %w", version, err)
    }
    
    cm.checkpoints[name] = version
    log.Printf("Created checkpoint '%s' at version %d", name, version)
    return nil
}

func (cm *CheckpointManager) RestoreCheckpoint(name string) (types.Version, error) {
    version, exists := cm.checkpoints[name]
    if !exists {
        return 0, fmt.Errorf("checkpoint '%s' not found", name)
    }
    
    // Note: ProofBox doesn't modify current version
    // This returns the checkpoint version for querying
    return version, nil
}

// Usage
cm := NewCheckpointManager(jmt)

// Create checkpoints
cm.CreateCheckpoint("before_migration")
// ... perform operations ...
cm.CreateCheckpoint("after_migration")

// Query at checkpoint
checkpointVersion, _ := cm.RestoreCheckpoint("before_migration")
value, _ := jmt.Get(checkpointVersion, key)
```

### Version Retention Policy

```go
// Implement retention policy (conceptual - would need storage support)
type RetentionPolicy struct {
    tree              *tree.Tree
    keepLast          int           // Keep last N versions
    keepDuration      time.Duration // Keep versions newer than duration
    keepCheckpoints   bool          // Always keep checkpoint versions
    checkpointVersions map[types.Version]bool
}

func (rp *RetentionPolicy) ShouldRetain(version types.Version, timestamp time.Time) bool {
    currentVersion := rp.tree.GetLatestVersion()
    
    // Always keep checkpoint versions
    if rp.keepCheckpoints && rp.checkpointVersions[version] {
        return true
    }
    
    // Keep recent versions
    if currentVersion - version < types.Version(rp.keepLast) {
        return true
    }
    
    // Keep by time
    if time.Since(timestamp) < rp.keepDuration {
        return true
    }
    
    return false
}
```

## Complete Example

Here's a comprehensive example demonstrating version management:

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/neutral/proofbox/pkg/storage"
    "github.com/neutral/proofbox/pkg/storage/pebble"
    "github.com/neutral/proofbox/pkg/tree"
    "github.com/neutral/proofbox/pkg/types"
)

type VersionedKV struct {
    tree       *tree.Tree
    history    map[types.Version]VersionInfo
}

type VersionInfo struct {
    Timestamp   time.Time
    Description string
    Changes     int
}

func NewVersionedKV(dbPath string) (*VersionedKV, error) {
    store, err := pebble.NewStorage(dbPath, nil)
    if err != nil {
        return nil, err
    }
    
    keyEncoder := storage.NewDefaultKeyEncoder()
    config := tree.DefaultTreeConfig()
    
    tree, err := tree.NewTree(store, keyEncoder, config)
    if err != nil {
        store.Close()
        return nil, err
    }
    
    return &VersionedKV{
        tree:    tree,
        history: make(map[types.Version]VersionInfo),
    }, nil
}

func (vkv *VersionedKV) Put(key, value string, description string) (types.Version, error) {
    k := types.KeyHash([]byte(key))
    v := []byte(value)
    
    version, err := vkv.tree.Put(k, v)
    if err != nil {
        return 0, err
    }
    
    vkv.history[version] = VersionInfo{
        Timestamp:   time.Now(),
        Description: description,
        Changes:     1,
    }
    
    return version, nil
}

func (vkv *VersionedKV) GetHistory(key string) {
    k := types.KeyHash([]byte(key))
    
    fmt.Printf("History for key '%s':\n", key)
    
    for v := types.Version(0); v <= vkv.tree.GetLatestVersion(); v++ {
        value, err := vkv.tree.Get(v, k)
        if err != nil {
            continue
        }
        
        info, hasInfo := vkv.history[v]
        
        if value != nil {
            fmt.Printf("  v%d: %s", v, string(value))
            if hasInfo {
                fmt.Printf(" (%s at %s)", info.Description, 
                    info.Timestamp.Format("15:04:05"))
            }
            fmt.Println()
        }
    }
}

func (vkv *VersionedKV) Rollback(toVersion types.Version, key string) ([]byte, error) {
    k := types.KeyHash([]byte(key))
    return vkv.tree.Get(toVersion, k)
}

func main() {
    vkv, err := NewVersionedKV("./versioned.db")
    if err != nil {
        log.Fatal(err)
    }
    
    // Simulate configuration changes
    configKey := "app:config"
    
    v1, _ := vkv.Put(configKey, `{"theme":"light","lang":"en"}`, "Initial config")
    fmt.Printf("Created v%d\n", v1)
    time.Sleep(100 * time.Millisecond)
    
    v2, _ := vkv.Put(configKey, `{"theme":"dark","lang":"en"}`, "Changed theme")
    fmt.Printf("Created v%d\n", v2)
    time.Sleep(100 * time.Millisecond)
    
    v3, _ := vkv.Put(configKey, `{"theme":"dark","lang":"es"}`, "Changed language")
    fmt.Printf("Created v%d\n", v3)
    
    // Show history
    fmt.Println("\nConfiguration history:")
    vkv.GetHistory(configKey)
    
    // Rollback to v1
    fmt.Println("\nRolling back to v1:")
    oldConfig, err := vkv.Rollback(v1, configKey)
    if err == nil && oldConfig != nil {
        fmt.Printf("Restored config: %s\n", string(oldConfig))
    }
    
    // Compare versions
    fmt.Println("\nVersion comparison:")
    for v := v1; v <= v3; v++ {
        rootHash, _ := vkv.tree.GetRootHash(v)
        info := vkv.history[v]
        fmt.Printf("v%d: %s (root: %x...)\n", v, info.Description, rootHash[:8])
    }
}
```

## Best Practices

### Version Management

1. **Track Metadata**: Store description and timestamp with versions
2. **Use Checkpoints**: Mark important versions for easy reference
3. **Plan Retention**: Consider storage implications of keeping all versions
4. **Cache Readers**: Reuse readers for repeated queries at same version

### Query Patterns

1. **Batch Version Queries**: Query multiple keys at once
2. **Range Scans**: Identify version ranges with changes
3. **Time-based Access**: Map versions to timestamps for temporal queries
4. **Minimize Readers**: Create readers once for multiple operations

### Performance Tips

1. **Latest Version**: Queries at latest version are fastest
2. **Sequential Access**: Access versions in order when possible
3. **Reader Lifecycle**: Close readers when done
4. **Version Gaps**: Handle missing versions gracefully

## Troubleshooting

### Version Not Found

```go
// Check if version exists
_, err := tree.GetRootHash(version)
if err != nil {
    fmt.Printf("Version %d does not exist\n", version)
}
```

### Performance Degradation

- Too many open readers
- Accessing very old versions
- Large version gaps
- Insufficient caching

### Memory Usage

- Close readers after use
- Limit reader cache size
- Don't store all version metadata in memory

## Next Steps

- Implement custom retention policies
- Build time-series queries on top of versions
- Create version-based analytics
- Explore checkpoint-based workflows

## Quick Reference

```go
// Get current version
version := jmt.GetLatestVersion()

// Read at version
value, err := jmt.Get(version, key)

// Create reader for version
reader, err := jmt.Reader(version)
defer reader.Close()

// Get root hash at version
rootHash, err := jmt.GetRootHash(version)

// Every write creates new version
newVersion, err := jmt.Put(key, value)
newVersion, err := jmt.Delete(key)
```