package proof

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"

	"github.com/neutral/proofbox/pkg/crypto"
	"github.com/neutral/proofbox/pkg/types"
)

// CompressedProof represents a space-optimized proof
type CompressedProof struct {
	Type           ProofType
	KeyHash        types.Hash // Store hash instead of full key
	CompressedData []byte     // Gzip compressed proof data
}

// CompressProof reduces proof size for network transmission
func CompressProof(proof *Proof) (*CompressedProof, error) {
	// Validate proof first
	if err := proof.IsValid(); err != nil {
		return nil, err
	}

	// Serialize proof data
	var buf bytes.Buffer

	// Write key
	buf.Write(proof.Key[:])

	// Write proof type specific data
	switch proof.Type {
	case ProofTypeInclusion:
		if err := binary.Write(&buf, binary.BigEndian, uint32(len(proof.Value))); err != nil {
			return nil, err
		}
		buf.Write(proof.Value)

	case ProofTypeExclusionNeighbor:
		buf.Write(proof.NeighborLeaf.Key[:])
		buf.Write(proof.NeighborLeaf.ValueHash[:])
		if err := binary.Write(&buf, binary.BigEndian, uint16(proof.NeighborLeaf.DivergeDepth)); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.BigEndian, uint16(len(proof.NeighborLeaf.Path))); err != nil {
			return nil, err
		}
		for _, nibble := range proof.NeighborLeaf.Path {
			buf.WriteByte(byte(nibble))
		}
	}

	// Write siblings
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(proof.Siblings))); err != nil {
		return nil, err
	}

	for _, sib := range proof.Siblings {
		if err := binary.Write(&buf, binary.BigEndian, uint16(sib.Depth)); err != nil {
			return nil, err
		}
		buf.WriteByte(byte(sib.Nibble))

		// Write children count and data
		childCount := uint16(len(sib.Children))
		if err := binary.Write(&buf, binary.BigEndian, childCount); err != nil {
			return nil, err
		}

		// Write children in order for deterministic encoding
		for nibble := types.Nibble(0); nibble <= types.MaxNibbleValue; nibble++ {
			if hash, exists := sib.Children[nibble]; exists {
				buf.WriteByte(byte(nibble))
				buf.Write(hash[:])
			}
		}
	}

	// Write version and root hash
	if err := binary.Write(&buf, binary.BigEndian, proof.Version); err != nil {
		return nil, err
	}
	buf.Write(proof.RootHash[:])

	// Compress
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	if _, err := gz.Write(buf.Bytes()); err != nil {
		gz.Close()
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}

	// Check size limit
	if compressed.Len() > MaxProofSize {
		return nil, fmt.Errorf("compressed proof exceeds maximum size: %d > %d", compressed.Len(), MaxProofSize)
	}

	return &CompressedProof{
		Type:           proof.Type,
		KeyHash:        crypto.DefaultHasher.Hash(proof.Key[:]),
		CompressedData: compressed.Bytes(),
	}, nil
}

// DecompressProof reconstructs a proof from compressed form
func DecompressProof(compressed *CompressedProof) (*Proof, error) {
	// Decompress data
	gz, err := gzip.NewReader(bytes.NewReader(compressed.CompressedData))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(gz); err != nil {
		return nil, err
	}

	data := buf.Bytes()
	offset := 0

	// Read key
	if len(data) < offset+32 {
		return nil, fmt.Errorf("insufficient data for key")
	}
	var key types.Key
	copy(key[:], data[offset:offset+32])
	offset += 32

	// Verify key hash matches
	if crypto.DefaultHasher.Hash(key[:]) != compressed.KeyHash {
		return nil, fmt.Errorf("key hash mismatch")
	}

	proof := &Proof{
		Type:     compressed.Type,
		Key:      key,
		Siblings: make([]SiblingData, 0),
	}

	// Read type-specific data
	switch compressed.Type {
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
	}

	// Read siblings
	if len(data) < offset+2 {
		return nil, fmt.Errorf("insufficient data for sibling count")
	}
	siblingCount := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2

	for i := 0; i < int(siblingCount); i++ {
		if len(data) < offset+2+1+2 {
			return nil, fmt.Errorf("insufficient data for sibling %d", i)
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

	// Read version and root hash
	if len(data) < offset+8+32 {
		return nil, fmt.Errorf("insufficient data for version and root hash")
	}

	proof.Version = types.Version(binary.BigEndian.Uint64(data[offset : offset+8]))
	offset += 8

	copy(proof.RootHash[:], data[offset:offset+32])

	// Validate the reconstructed proof
	if err := proof.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid decompressed proof: %w", err)
	}

	return proof, nil
}

// BatchProofOptimization optimizes proofs that share common paths
type BatchProofOptimization struct {
	proofs []*Proof
}

// OptimizeBatch reduces redundancy in batch proofs
func OptimizeBatch(proofs []*Proof) *BatchProofOptimization {
	// TODO: Implement path sharing optimization
	// - Identify common prefixes
	// - Share sibling data
	// - Compress redundant information

	return &BatchProofOptimization{proofs: proofs}
}
