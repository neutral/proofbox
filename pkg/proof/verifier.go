package proof

import (
	"bytes"
	"fmt"

	"github.com/neutral/proofbox/pkg/crypto"
	"github.com/neutral/proofbox/pkg/types"
)

// Verifier verifies Merkle proofs
type Verifier struct {
	hasher crypto.Hasher
}

// NewVerifier creates a new proof verifier
func NewVerifier() *Verifier {
	return &Verifier{
		hasher: crypto.DefaultHasher,
	}
}

// Verify checks if a proof is valid
func Verify(proof *Proof) error {
	v := NewVerifier()
	return v.Verify(proof)
}

// Verify checks if a proof is valid
func (v *Verifier) Verify(proof *Proof) error {
	if err := proof.IsValid(); err != nil {
		return err
	}
	
	// Compute root hash based on proof type
	var computedRoot types.Hash
	var err error
	
	switch proof.Type {
	case ProofTypeInclusion:
		computedRoot, err = v.verifyInclusion(proof)
	case ProofTypeExclusionEmpty:
		computedRoot, err = v.verifyExclusionEmpty(proof)
	case ProofTypeExclusionNeighbor:
		computedRoot, err = v.verifyExclusionNeighbor(proof)
	default:
		return types.ErrInvalidProof
	}
	
	if err != nil {
		return err
	}
	
	// Compare with expected root
	if !bytes.Equal(computedRoot[:], proof.RootHash[:]) {
		return types.ErrHashMismatch
	}
	
	return nil
}

// verifyInclusion verifies an inclusion proof
func (v *Verifier) verifyInclusion(proof *Proof) (types.Hash, error) {
	// Start with leaf hash
	valueHash := v.hasher.Hash(proof.Value)
	
	// Compute leaf node hash according to JMT spec
	// Format: NodeType || Key || ValueHash
	leafData := make([]byte, 0, 1+32+32)
	leafData = append(leafData, byte(types.NodeTypeLeaf))
	leafData = append(leafData, proof.Key[:]...)
	leafData = append(leafData, valueHash[:]...)
	leafHash := v.hasher.Hash(leafData)
	
	// Reconstruct root from leaf
	return v.reconstructRoot(proof.Key, leafHash, proof.Siblings, proof.Type)
}

// verifyExclusionEmpty verifies an empty exclusion proof
func (v *Verifier) verifyExclusionEmpty(proof *Proof) (types.Hash, error) {
	// For empty exclusion, we don't have a leaf at the end
	// We reconstruct from the point where the path leads to empty
	
	// The reconstruction will handle the empty child appropriately
	return v.reconstructRoot(proof.Key, types.Hash{}, proof.Siblings, proof.Type)
}

// verifyExclusionNeighbor verifies a neighbor exclusion proof
func (v *Verifier) verifyExclusionNeighbor(proof *Proof) (types.Hash, error) {
	neighbor := proof.NeighborLeaf
	
	// Verify the neighbor is valid
	if neighbor.DivergeDepth > types.MaxTreeDepth {
		return types.Hash{}, types.ErrInvalidProof
	}
	
	// Verify keys actually diverge at claimed depth
	keyNibbles := proof.Key.ToNibblePath().Nibbles
	neighborNibbles := neighbor.Key.ToNibblePath().Nibbles
	
	// Check common path
	for i := 0; i < neighbor.DivergeDepth && i < len(neighbor.Path); i++ {
		if keyNibbles[i] != neighborNibbles[i] {
			return types.Hash{}, fmt.Errorf("keys diverge before claimed depth")
		}
		if keyNibbles[i] != neighbor.Path[i] {
			return types.Hash{}, fmt.Errorf("path mismatch at depth %d", i)
		}
	}
	
	// Verify they actually diverge at the claimed depth
	if neighbor.DivergeDepth < types.MaxTreeDepth {
		if keyNibbles[neighbor.DivergeDepth] == neighborNibbles[neighbor.DivergeDepth] {
			return types.Hash{}, fmt.Errorf("keys don't diverge at claimed depth")
		}
	}
	
	// Compute neighbor leaf hash
	leafData := make([]byte, 0, 1+32+32)
	leafData = append(leafData, byte(types.NodeTypeLeaf))
	leafData = append(leafData, neighbor.Key[:]...)
	leafData = append(leafData, neighbor.ValueHash[:]...)
	neighborLeafHash := v.hasher.Hash(leafData)
	
	// Reconstruct root using neighbor's position
	return v.reconstructRoot(neighbor.Key, neighborLeafHash, proof.Siblings, proof.Type)
}

// reconstructRoot rebuilds the root hash from a leaf or empty position
func (v *Verifier) reconstructRoot(key types.Key, startHash types.Hash, siblings []SiblingData, proofType ProofType) (types.Hash, error) {
	nibblePath := key.ToNibblePath().Nibbles
	currentHash := startHash
	
	// For empty exclusion, we don't have a leaf at the bottom
	// The proof shows that following the key path leads to an empty child
	isEmptyExclusion := proofType == ProofTypeExclusionEmpty
	
	// Process siblings from deepest to root
	for i := len(siblings) - 1; i >= 0; i-- {
		sibling := siblings[i]
		
		// Build internal node data
		// Format: NodeType || child0 || child1 || ... || child15
		// where each child is either (nibble || hash) for existing children or empty_hash
		nodeData := make([]byte, 0, 1+16*33) // 1 type + 16*(1 nibble + 32 hash)
		nodeData = append(nodeData, byte(types.NodeTypeInternal))
		
		targetNibble := nibblePath[sibling.Depth]
		
		// Add all 16 children in order
		for nibble := types.Nibble(0); nibble <= types.MaxNibbleValue; nibble++ {
			if nibble == targetNibble {
				// This is our path
				if isEmptyExclusion && i == len(siblings)-1 {
					// For empty exclusion, this child doesn't exist - just empty hash
					nodeData = append(nodeData, crypto.EmptyTreeHash[:]...)
				} else {
					// Normal case - add nibble and hash
					nodeData = append(nodeData, byte(nibble))
					nodeData = append(nodeData, currentHash[:]...)
				}
			} else if childHash, exists := sibling.Children[nibble]; exists {
				// Non-empty sibling - add nibble and hash
				nodeData = append(nodeData, byte(nibble))
				nodeData = append(nodeData, childHash[:]...)
			} else {
				// Empty child - just the empty hash, no nibble
				nodeData = append(nodeData, crypto.EmptyTreeHash[:]...)
			}
		}
		
		currentHash = v.hasher.Hash(nodeData)
	}
	
	return currentHash, nil
}

// VerifyBatch verifies multiple proofs efficiently
func (v *Verifier) VerifyBatch(proofs []*Proof) error {
	// TODO: Optimize by sharing computation for common paths
	for i, proof := range proofs {
		if err := v.Verify(proof); err != nil {
			return fmt.Errorf("proof %d verification failed: %w", i, err)
		}
	}
	return nil
}