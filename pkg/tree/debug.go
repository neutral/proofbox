package tree

import (
	"fmt"
	"strings"
	"github.com/neutral/proofbox/pkg/types"
)

// DebugPrintTree prints the tree structure for debugging
func (t *Tree) DebugPrintTree(version types.Version) error {
	rootHash, err := t.GetRootHash(version)
	if err != nil {
		return err
	}
	
	if rootHash == types.EmptyHash() {
		fmt.Println("Empty tree")
		return nil
	}
	
	snapshot := t.db.NewSnapshot()
	defer snapshot.Close()
	
	reader := &TreeReader{
		tree:     t,
		snapshot: snapshot,
		version:  version,
		rootHash: rootHash,
	}
	
	root, err := reader.loadNode(types.RootNodeKey(version))
	if err != nil {
		return err
	}
	
	return debugPrintNode(reader, root, version, types.NibblePath{}, 0)
}

func debugPrintNode(reader *TreeReader, node types.Node, version types.Version, path types.NibblePath, indent int) error {
	prefix := strings.Repeat("  ", indent)
	
	switch n := node.(type) {
	case *LeafNode:
		fmt.Printf("%sLeaf at %v: key=%x\n", prefix, path, n.Key())
		
	case *InternalNode:
		fmt.Printf("%sInternal at %v: %d children\n", prefix, path, n.NumChildren())
		for nibble := types.Nibble(0); nibble <= 15; nibble++ {
			if child, exists := n.Child(nibble); exists {
				childPath := path.Append(nibble)
				childKey := types.NodeKey{
					Version:    child.Version,
					NibblePath: childPath,
				}
				
				childNode, err := reader.loadNode(childKey)
				if err != nil {
					fmt.Printf("%s  Child %X: ERROR: %v\n", prefix, nibble, err)
					continue
				}
				
				fmt.Printf("%s  Child %X:\n", prefix, nibble)
				if err := debugPrintNode(reader, childNode, child.Version, childPath, indent+2); err != nil {
					return err
				}
			}
		}
	}
	
	return nil
}