package tree

import (
	"fmt"
	"sync"
	"sync/atomic"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/neutral/proofbox/pkg/types"
)

// NodeCache provides thread-safe caching of nodes
type NodeCache struct {
	mu      sync.RWMutex
	cache   *lru.Cache[string, types.Node]
	hits    atomic.Uint64
	misses  atomic.Uint64
	maxSize int
}

// NewNodeCache creates a new LRU cache for nodes
func NewNodeCache(maxSize int) (*NodeCache, error) {
	if maxSize <= 0 {
		return nil, fmt.Errorf("cache size must be positive, got %d", maxSize)
	}

	cache, err := lru.New[string, types.Node](maxSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %w", err)
	}

	return &NodeCache{
		cache:   cache,
		maxSize: maxSize,
	}, nil
}

// Get retrieves a node from cache
func (c *NodeCache) Get(key types.NodeKey) (types.Node, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cacheKey := encodeCacheKey(key)

	if val, ok := c.cache.Get(cacheKey); ok {
		c.hits.Add(1)
		return val, true
	}

	c.misses.Add(1)
	return nil, false
}

// Put adds a node to cache
func (c *NodeCache) Put(key types.NodeKey, node types.Node) {
	if node == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	cacheKey := encodeCacheKey(key)
	c.cache.Add(cacheKey, node)
}

// Remove evicts a node from cache
func (c *NodeCache) Remove(key types.NodeKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cacheKey := encodeCacheKey(key)
	c.cache.Remove(cacheKey)
}

// Clear removes all entries from cache
func (c *NodeCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.Purge()
}

// Stats returns cache statistics
func (c *NodeCache) Stats() (hits, misses uint64, size int) {
	c.mu.RLock()
	size = c.cache.Len()
	c.mu.RUnlock()

	hits = c.hits.Load()
	misses = c.misses.Load()
	return
}

// HitRate returns the cache hit rate as a percentage
func (c *NodeCache) HitRate() float64 {
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) * 100 / float64(total)
}

// encodeCacheKey creates a string key for the cache
func encodeCacheKey(key types.NodeKey) string {
	// Use efficient encoding: version + nibble path
	// This avoids allocations compared to fmt.Sprintf
	storageKey := makeNodeKey(key)
	return string(storageKey)
}