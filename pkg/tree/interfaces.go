package tree

import (
	"github.com/neutral/proofbox/pkg/metrics"
	"github.com/neutral/proofbox/pkg/types"
)

// TreeReaderInterface provides read-only access to tree nodes for proof generation
type TreeReaderInterface interface {
	// GetNode retrieves a node by its key
	GetNode(key types.NodeKey) (types.Node, error)

	// LoadValue loads a value by its hash
	LoadValue(hash types.Hash) ([]byte, error)

	// RootHash returns the root hash of the tree at this version
	RootHash() types.Hash

	// Version returns the version being read
	Version() types.Version

	// Close releases any resources held by the reader
	Close() error

	// Metrics returns the metrics collector for this reader
	Metrics() metrics.JMTMetrics
}

// Ensure TreeReader implements TreeReaderInterface
var _ TreeReaderInterface = (*TreeReader)(nil)
