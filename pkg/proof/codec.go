package proof

import (
	"encoding/binary"
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
)

// ProofCodec handles proof serialization
type ProofCodec struct{}

// NewProofCodec creates a new proof codec
func NewProofCodec() *ProofCodec {
	return &ProofCodec{}
}

// Encode serializes a proof for storage or network transmission
func (c *ProofCodec) Encode(proof *Proof) ([]byte, error) {
	if err := proof.IsValid(); err != nil {
		return nil, fmt.Errorf("cannot encode invalid proof: %w", err)
	}

	// Estimate size to avoid reallocations
	estimatedSize := 1 + 32 + 8 + 32 // Type + Key + Version + RootHash
	buf := make([]byte, 0, estimatedSize)
	
	// Write proof type
	buf = append(buf, byte(proof.Type))
	
	// Write key
	buf = append(buf, proof.Key[:]...)
	
	// Write version
	versionBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(versionBytes, uint64(proof.Version))
	buf = append(buf, versionBytes...)
	
	// Write root hash
	buf = append(buf, proof.RootHash[:]...)
	
	// Write type-specific data
	switch proof.Type {
	case ProofTypeInclusion:
		// Value length and data
		valueLenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(valueLenBytes, uint32(len(proof.Value)))
		buf = append(buf, valueLenBytes...)
		buf = append(buf, proof.Value...)
		
	case ProofTypeExclusionNeighbor:
		// Neighbor data
		buf = append(buf, proof.NeighborLeaf.Key[:]...)
		buf = append(buf, proof.NeighborLeaf.ValueHash[:]...)
		
		divergeBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(divergeBytes, uint16(proof.NeighborLeaf.DivergeDepth))
		buf = append(buf, divergeBytes...)
		
		pathLenBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(pathLenBytes, uint16(len(proof.NeighborLeaf.Path)))
		buf = append(buf, pathLenBytes...)
		
		for _, nibble := range proof.NeighborLeaf.Path {
			buf = append(buf, byte(nibble))
		}
	}
	
	// Write siblings count
	siblingCountBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(siblingCountBytes, uint16(len(proof.Siblings)))
	buf = append(buf, siblingCountBytes...)
	
	// Write each sibling
	for _, sib := range proof.Siblings {
		depthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(depthBytes, uint16(sib.Depth))
		buf = append(buf, depthBytes...)
		
		buf = append(buf, byte(sib.Nibble))
		
		// Write children count
		childCountBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(childCountBytes, uint16(len(sib.Children)))
		buf = append(buf, childCountBytes...)
		
		// Write children in nibble order for deterministic encoding
		for nibble := types.Nibble(0); nibble <= types.MaxNibbleValue; nibble++ {
			if hash, exists := sib.Children[nibble]; exists {
				buf = append(buf, byte(nibble))
				buf = append(buf, hash[:]...)
			}
		}
	}
	
	// Check size limit
	if len(buf) > MaxProofSize {
		return nil, fmt.Errorf("encoded proof exceeds maximum size: %d > %d", len(buf), MaxProofSize)
	}
	
	return buf, nil
}

// Decode deserializes a proof from bytes
func (c *ProofCodec) Decode(data []byte) (*Proof, error) {
	if len(data) < 1+32+8+32 {
		return nil, fmt.Errorf("insufficient data for proof header")
	}
	
	offset := 0
	
	// Read proof type
	proofType := ProofType(data[offset])
	offset++
	
	// Read key
	var key types.Key
	copy(key[:], data[offset:offset+32])
	offset += 32
	
	// Read version
	version := types.Version(binary.BigEndian.Uint64(data[offset : offset+8]))
	offset += 8
	
	// Read root hash
	var rootHash types.Hash
	copy(rootHash[:], data[offset:offset+32])
	offset += 32
	
	proof := &Proof{
		Type:     proofType,
		Key:      key,
		Version:  version,
		RootHash: rootHash,
		Siblings: make([]SiblingData, 0),
	}
	
	// Read type-specific data
	switch proofType {
	case ProofTypeInclusion:
		if len(data) < offset+4 {
			return nil, fmt.Errorf("insufficient data for value length")
		}
		valueLen := binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
		
		if len(data) < offset+int(valueLen) {
			return nil, fmt.Errorf("insufficient data for value")
		}
		proof.Value = make([]byte, valueLen)
		copy(proof.Value, data[offset:offset+int(valueLen)])
		offset += int(valueLen)
		
	case ProofTypeExclusionNeighbor:
		if len(data) < offset+32+32+2+2 {
			return nil, fmt.Errorf("insufficient data for neighbor")
		}
		
		neighbor := &NeighborLeafData{}
		copy(neighbor.Key[:], data[offset:offset+32])
		offset += 32
		copy(neighbor.ValueHash[:], data[offset:offset+32])
		offset += 32
		
		neighbor.DivergeDepth = int(binary.BigEndian.Uint16(data[offset : offset+2]))
		offset += 2
		
		pathLen := binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2
		
		if len(data) < offset+int(pathLen) {
			return nil, fmt.Errorf("insufficient data for path")
		}
		
		neighbor.Path = make([]types.Nibble, pathLen)
		for i := 0; i < int(pathLen); i++ {
			neighbor.Path[i] = types.Nibble(data[offset])
			offset++
		}
		
		proof.NeighborLeaf = neighbor
		
	case ProofTypeExclusionEmpty:
		// No additional data for empty exclusion
		
	default:
		return nil, fmt.Errorf("unknown proof type: %d", proofType)
	}
	
	// Read siblings
	if len(data) < offset+2 {
		return nil, fmt.Errorf("insufficient data for sibling count")
	}
	siblingCount := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	
	for i := 0; i < int(siblingCount); i++ {
		if len(data) < offset+2+1+2 {
			return nil, fmt.Errorf("insufficient data for sibling %d header", i)
		}
		
		sib := SiblingData{
			Depth:    int(binary.BigEndian.Uint16(data[offset : offset+2])),
			Nibble:   types.Nibble(data[offset+2]),
			Children: make(map[types.Nibble]types.Hash),
		}
		offset += 3
		
		childCount := binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2
		
		for j := 0; j < int(childCount); j++ {
			if len(data) < offset+1+32 {
				return nil, fmt.Errorf("insufficient data for child %d of sibling %d", j, i)
			}
			
			nibble := types.Nibble(data[offset])
			offset++
			
			var hash types.Hash
			copy(hash[:], data[offset:offset+32])
			offset += 32
			
			sib.Children[nibble] = hash
		}
		
		proof.Siblings = append(proof.Siblings, sib)
	}
	
	// Validate the decoded proof
	if err := proof.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid decoded proof: %w", err)
	}
	
	return proof, nil
}