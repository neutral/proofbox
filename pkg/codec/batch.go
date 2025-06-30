package codec

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/neutral/proofbox/pkg/types"
)

// BatchEncoder efficiently encodes multiple nodes
type BatchEncoder struct {
	buffer bytes.Buffer
	codec  NodeCodec
}

// NewBatchEncoder creates a new batch encoder
func NewBatchEncoder() *BatchEncoder {
	return &BatchEncoder{
		codec: NodeCodec{},
	}
}

// Add encodes and adds a node to the batch
func (b *BatchEncoder) Add(key types.NodeKey, node types.Node) error {
	// Encode node key
	keyData := types.EncodeNodeKey(key)
	if err := binary.Write(&b.buffer, binary.BigEndian, uint32(len(keyData))); err != nil {
		return err
	}
	b.buffer.Write(keyData)

	// Encode node
	nodeData, err := b.codec.EncodeNode(node)
	if err != nil {
		return err
	}
	if err := binary.Write(&b.buffer, binary.BigEndian, uint32(len(nodeData))); err != nil {
		return err
	}
	b.buffer.Write(nodeData)

	return nil
}

// Bytes returns the encoded batch data
func (b *BatchEncoder) Bytes() []byte {
	return b.buffer.Bytes()
}

// Len returns the current size of the batch
func (b *BatchEncoder) Len() int {
	return b.buffer.Len()
}

// Reset clears the buffer for reuse
func (b *BatchEncoder) Reset() {
	b.buffer.Reset()
}

// BatchDecoder decodes nodes from a batch
type BatchDecoder struct {
	data []byte
	pos  int
}

// NewBatchDecoder creates a new batch decoder
func NewBatchDecoder(data []byte) *BatchDecoder {
	return &BatchDecoder{
		data: data,
		pos:  0,
	}
}

// Next decodes the next node key and node from the batch
func (d *BatchDecoder) Next() (types.NodeKey, types.Node, error) {
	if d.pos >= len(d.data) {
		return types.NodeKey{}, nil, nil // EOF
	}

	// Read key length
	if d.pos+4 > len(d.data) {
		return types.NodeKey{}, nil, errors.New("truncated key length")
	}
	keyLen := binary.BigEndian.Uint32(d.data[d.pos : d.pos+4])
	d.pos += 4

	// Read key data
	if d.pos+int(keyLen) > len(d.data) {
		return types.NodeKey{}, nil, errors.New("truncated key data")
	}
	keyData := d.data[d.pos : d.pos+int(keyLen)]
	d.pos += int(keyLen)

	// Decode key
	key, err := types.DecodeNodeKey(keyData)
	if err != nil {
		return types.NodeKey{}, nil, err
	}

	// Read node length
	if d.pos+4 > len(d.data) {
		return types.NodeKey{}, nil, errors.New("truncated node length")
	}
	nodeLen := binary.BigEndian.Uint32(d.data[d.pos : d.pos+4])
	d.pos += 4

	// Read node data
	if d.pos+int(nodeLen) > len(d.data) {
		return types.NodeKey{}, nil, errors.New("truncated node data")
	}
	nodeData := d.data[d.pos : d.pos+int(nodeLen)]
	d.pos += int(nodeLen)

	// Decode node
	node, err := DecodeNode(nodeData, key.Version)
	if err != nil {
		return types.NodeKey{}, nil, err
	}

	return key, node, nil
}

// HasMore returns true if there are more nodes to decode
func (d *BatchDecoder) HasMore() bool {
	return d.pos < len(d.data)
}
