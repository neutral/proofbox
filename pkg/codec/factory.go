package codec

import (
	"fmt"
	"sync"

	"github.com/neutral/proofbox/pkg/types"
)

// NodeFactory is a function that creates a node from decoded data
type NodeFactory func(nodeType types.NodeType, data []byte, version types.Version) (types.Node, error)

var (
	factoryMu sync.RWMutex
	factory   NodeFactory
)

// RegisterNodeFactory registers the factory function for creating nodes
func RegisterNodeFactory(f NodeFactory) {
	factoryMu.Lock()
	defer factoryMu.Unlock()
	factory = f
}

// DecodeNode decodes a node using the registered factory
func DecodeNode(data []byte, version types.Version) (types.Node, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("empty node data")
	}

	factoryMu.RLock()
	f := factory
	factoryMu.RUnlock()

	if f == nil {
		return nil, fmt.Errorf("no node factory registered")
	}

	nodeType := types.NodeType(data[0])
	nodeData := data[1:]

	return f(nodeType, nodeData, version)
}