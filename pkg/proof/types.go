package proof

import (
	"github.com/neutral/proofbox/pkg/types"
)

// ProofType identifies the type of proof
type ProofType uint8

const (
	// ProofTypeInclusion proves a key exists with a specific value
	ProofTypeInclusion ProofType = iota
	// ProofTypeExclusionEmpty proves a key doesn't exist (empty child)
	ProofTypeExclusionEmpty
	// ProofTypeExclusionNeighbor proves a key doesn't exist (neighbor leaf found)
	ProofTypeExclusionNeighbor
)

// Proof represents a Merkle proof for a key
type Proof struct {
	Type         ProofType          // Type of proof
	Key          types.Key          // Key being proven
	Value        []byte             // For inclusion proofs
	NeighborLeaf *NeighborLeafData  // For neighbor exclusion proofs
	Siblings     []SiblingData      // Sibling hashes along the path
	RootHash     types.Hash         // Expected root hash
	Version      types.Version      // Tree version
}

// SiblingData contains sibling information at each level
type SiblingData struct {
	Depth    int                    // Depth in the tree (0 = root)
	Nibble   types.Nibble           // Which nibble this is for
	Hash     types.Hash             // Sibling hash
	Children map[types.Nibble]types.Hash // All children at this level for verification
}

// NeighborLeafData contains neighbor leaf information
type NeighborLeafData struct {
	Key          types.Key      // Neighbor's key
	ValueHash    types.Hash     // Neighbor's value hash
	Path         []types.Nibble // Common path to neighbor
	DivergeDepth int            // Depth where keys diverge
}

// IsValid performs basic validation
func (p *Proof) IsValid() error {
	// Validate key
	if err := types.ValidateKey(p.Key); err != nil {
		return err
	}
	
	// Check proof type specific requirements
	switch p.Type {
	case ProofTypeInclusion:
		if p.Value == nil {
			return types.ErrInvalidProof
		}
		if p.NeighborLeaf != nil {
			return types.ErrInvalidProof
		}
	case ProofTypeExclusionEmpty:
		if p.Value != nil || p.NeighborLeaf != nil {
			return types.ErrInvalidProof
		}
	case ProofTypeExclusionNeighbor:
		if p.NeighborLeaf == nil {
			return types.ErrInvalidProof
		}
		if p.Value != nil {
			return types.ErrInvalidProof
		}
		// Validate neighbor data
		if err := types.ValidateKey(p.NeighborLeaf.Key); err != nil {
			return err
		}
		if p.NeighborLeaf.DivergeDepth < 0 || p.NeighborLeaf.DivergeDepth > types.MaxTreeDepth {
			return types.ErrInvalidProof
		}
	default:
		return types.ErrInvalidProof
	}
	
	// Validate siblings
	for i, sib := range p.Siblings {
		if sib.Depth < 0 || sib.Depth > types.MaxTreeDepth {
			return types.ErrInvalidProof
		}
		if sib.Nibble > types.MaxNibbleValue {
			return types.ErrInvalidProof
		}
		// Ensure siblings are in order
		if i > 0 && sib.Depth <= p.Siblings[i-1].Depth {
			return types.ErrInvalidProof
		}
	}
	
	return nil
}

// EstimateSize estimates the serialized size of the proof in bytes
func (p *Proof) EstimateSize() int {
	size := 0
	
	// Fixed size fields
	size += 1   // ProofType
	size += 32  // Key (256-bit key)
	size += 32  // RootHash 
	size += 8   // Version
	
	// Value (for inclusion proofs)
	if p.Value != nil {
		size += 4 + len(p.Value) // Length prefix + data
	}
	
	// Neighbor data (for exclusion proofs)
	if p.NeighborLeaf != nil {
		size += 32               // Key
		size += 32               // ValueHash
		size += 4                // DivergeDepth
		size += 4 + len(p.NeighborLeaf.Path) // Path length + data
	}
	
	// Siblings data
	size += 4 // Number of siblings
	for _, sib := range p.Siblings {
		size += 4               // Depth
		size += 1               // Nibble
		size += 32              // Hash
		size += 4               // Number of children
		size += len(sib.Children) * (1 + 32) // Each child: nibble + hash
	}
	
	return size
}

// ProofBatch represents multiple proofs that can be optimized together
type ProofBatch struct {
	Proofs []*Proof
	// TODO: Add shared path optimization data
}

// MaxProofSize is the maximum allowed size for a serialized proof (10KB)
const MaxProofSize = 10 * 1024