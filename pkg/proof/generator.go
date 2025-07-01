package proof

import (
	"fmt"
	"time"

	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
)

// Generator creates Merkle proofs
type Generator struct {
	reader tree.TreeReaderInterface
}

// ValueLoader is an interface for loading actual values
type ValueLoader interface {
	LoadValue(hash types.Hash) ([]byte, error)
}

// NewGenerator creates a new proof generator
func NewGenerator(reader tree.TreeReaderInterface) *Generator {
	return &Generator{reader: reader}
}

// Generate creates a proof for the given key
func Generate(reader tree.TreeReaderInterface, key types.Key) (*Proof, error) {
	gen := NewGenerator(reader)
	return gen.Generate(key)
}

// Generate creates a proof for the given key
func (g *Generator) Generate(key types.Key) (*Proof, error) {
	start := time.Now()
	metrics := g.reader.Metrics()
	
	// Validate key
	if err := types.ValidateKey(key); err != nil {
		metrics.RecordError("validation")
		return nil, err
	}

	rootHash := g.reader.RootHash()
	version := g.reader.Version()
	
	proof := &Proof{
		Key:      key,
		RootHash: rootHash,
		Version:  version,
		Siblings: make([]SiblingData, 0),
	}
	
	// Handle empty tree
	if rootHash == types.EmptyHash() {
		proof.Type = ProofTypeExclusionEmpty
		
		// Record metrics for empty tree proof
		duration := time.Since(start).Seconds()
		proofSize := proof.EstimateSize()
		metrics.RecordProofGeneration(duration, "exclusion", proofSize)
		
		return proof, nil
	}
	
	// Traverse tree collecting siblings
	nibblePath := key.ToNibblePath()
	nodeKey := types.RootNodeKey(version)
	
	// Track the path we take for neighbor proofs
	pathTaken := make([]types.Nibble, 0, types.MaxTreeDepth)
	
	for depth := 0; depth < types.MaxTreeDepth; depth++ {
		node, err := g.reader.GetNode(nodeKey)
		if err != nil {
			metrics.RecordError("storage")
			return nil, fmt.Errorf("failed to load node at depth %d: %w", depth, err)
		}
		if node == nil {
			// This shouldn't happen with a non-empty root
			return nil, fmt.Errorf("unexpected nil node at depth %d", depth)
		}
		
		switch n := node.(type) {
		case types.LeafNodeInterface:
			if n.Key() == key {
				// Inclusion proof
				proof.Type = ProofTypeInclusion
				// Load the actual value using the value hash
				value, err := g.reader.LoadValue(n.ValueHash())
				if err != nil {
					metrics.RecordError("storage")
					return nil, fmt.Errorf("failed to load value: %w", err)
				}
				proof.Value = value
				
				// Record metrics for inclusion proof
				duration := time.Since(start).Seconds()
				proofSize := proof.EstimateSize()
				metrics.RecordProofGeneration(duration, "inclusion", proofSize)
				
				return proof, nil
			} else {
				// Neighbor exclusion proof
				proof.Type = ProofTypeExclusionNeighbor
				proof.NeighborLeaf = &NeighborLeafData{
					Key:          n.Key(),
					ValueHash:    n.ValueHash(),
					Path:         pathTaken,
					DivergeDepth: depth,
				}
				
				// Record metrics for exclusion proof
				duration := time.Since(start).Seconds()
				proofSize := proof.EstimateSize()
				metrics.RecordProofGeneration(duration, "exclusion", proofSize)
				
				return proof, nil
			}
			
		case types.InternalNodeInterface:
			nibble := nibblePath.Nibbles[depth]
			pathTaken = append(pathTaken, nibble)
			
			// Collect sibling data for this level
			siblingData := SiblingData{
				Depth:    depth,
				Nibble:   nibble,
				Children: make(map[types.Nibble]types.Hash),
			}
			
			// Collect all children hashes (needed for verification)
			for nib := types.Nibble(0); nib <= types.MaxNibbleValue; nib++ {
				if child, exists := n.Child(nib); exists {
					siblingData.Children[nib] = child.Hash
				}
			}
			
			proof.Siblings = append(proof.Siblings, siblingData)
			
			// Check if child exists
			child, exists := n.Child(nibble)
			if !exists {
				// Empty exclusion proof
				proof.Type = ProofTypeExclusionEmpty
				
				// Record metrics for exclusion proof
				duration := time.Since(start).Seconds()
				proofSize := proof.EstimateSize()
				metrics.RecordProofGeneration(duration, "exclusion", proofSize)
				
				return proof, nil
			}
			
			// Continue to child
			nodeKey = types.NodeKey{
				Version: child.Version,
				NibblePath: types.NibblePath{
					Nibbles: nibblePath.Nibbles[:depth+1],
					Length:  uint16(depth + 1),
				},
			}
			
		default:
			return nil, fmt.Errorf("unknown node type: %T", node)
		}
	}
	
	// After processing 64 levels, check if we have a leaf at depth 64
	// This handles the edge case where keys differ only in the last nibble
	node, err := g.reader.GetNode(nodeKey)
	if err != nil {
		metrics.RecordError("storage")
		return nil, fmt.Errorf("failed to load node at depth 64: %w", err)
	}
	
	if node != nil {
		if leaf, ok := node.(types.LeafNodeInterface); ok {
			if leaf.Key() == key {
				// Inclusion proof at maximum depth
				proof.Type = ProofTypeInclusion
				value, err := g.reader.LoadValue(leaf.ValueHash())
				if err != nil {
					metrics.RecordError("storage")
					return nil, fmt.Errorf("failed to load value: %w", err)
				}
				proof.Value = value
				
				// Record metrics for inclusion proof
				duration := time.Since(start).Seconds()
				proofSize := proof.EstimateSize()
				metrics.RecordProofGeneration(duration, "inclusion", proofSize)
				
				return proof, nil
			} else {
				// Neighbor exclusion at maximum depth
				proof.Type = ProofTypeExclusionNeighbor
				proof.NeighborLeaf = &NeighborLeafData{
					Key:          leaf.Key(),
					ValueHash:    leaf.ValueHash(),
					Path:         pathTaken,
					DivergeDepth: types.MaxTreeDepth,
				}
				
				// Record metrics for exclusion proof
				duration := time.Since(start).Seconds()
				proofSize := proof.EstimateSize()
				metrics.RecordProofGeneration(duration, "exclusion", proofSize)
				
				return proof, nil
			}
		}
	}
	
	// If we get here, something is wrong
	metrics.RecordError("tree_structure")
	return nil, fmt.Errorf("proof generation failed: unexpected tree structure")
}

// GenerateBatch generates proofs for multiple keys efficiently
func (g *Generator) GenerateBatch(keys []types.Key) ([]*Proof, error) {
	proofs := make([]*Proof, len(keys))
	
	// TODO: Optimize by sharing common path traversals
	for i, key := range keys {
		proof, err := g.Generate(key)
		if err != nil {
			return nil, fmt.Errorf("failed to generate proof for key %d: %w", i, err)
		}
		proofs[i] = proof
	}
	
	return proofs, nil
}