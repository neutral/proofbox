package tree

import (
	"sync/atomic"
)

// TreeStats holds atomic tree statistics
type TreeStats struct {
	height    atomic.Int64
	nodeCount atomic.Int64
	version   atomic.Int64
}

// NewTreeStats creates a new TreeStats instance
func NewTreeStats() *TreeStats {
	return &TreeStats{}
}

// UpdateStats atomically updates all tree statistics
func (ts *TreeStats) UpdateStats(height, nodeCount, version int64) {
	ts.height.Store(height)
	ts.nodeCount.Store(nodeCount) 
	ts.version.Store(version)
}

// GetHeight returns the current tree height
func (ts *TreeStats) GetHeight() int64 {
	return ts.height.Load()
}

// GetNodeCount returns the current node count
func (ts *TreeStats) GetNodeCount() int64 {
	return ts.nodeCount.Load()
}

// GetVersion returns the current version
func (ts *TreeStats) GetVersion() int64 {
	return ts.version.Load()
}

// UpdateHeight atomically updates the tree height
func (ts *TreeStats) UpdateHeight(height int64) {
	ts.height.Store(height)
}

// UpdateNodeCount atomically updates the node count
func (ts *TreeStats) UpdateNodeCount(count int64) {
	ts.nodeCount.Store(count)
}

// UpdateVersion atomically updates the version
func (ts *TreeStats) UpdateVersion(version int64) {
	ts.version.Store(version)
}