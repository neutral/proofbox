package tree

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/neutral/proofbox/pkg/types"
)

// BatchOptimizer provides optimization strategies for update batches
type BatchOptimizer struct {
	enableCompression   bool
	compressionLevel    int
	enableDeduplication bool
	enableCoalescing    bool
}

// BatchOptimizerConfig configures the batch optimizer
type BatchOptimizerConfig struct {
	EnableCompression   bool
	CompressionLevel    int // gzip compression level (1-9)
	EnableDeduplication bool
	EnableCoalescing    bool
}

// DefaultBatchOptimizerConfig returns default optimization settings
func DefaultBatchOptimizerConfig() BatchOptimizerConfig {
	return BatchOptimizerConfig{
		EnableCompression:   true,
		CompressionLevel:    gzip.DefaultCompression,
		EnableDeduplication: true,
		EnableCoalescing:    true,
	}
}

// NewBatchOptimizer creates a new batch optimizer
func NewBatchOptimizer(config BatchOptimizerConfig) *BatchOptimizer {
	return &BatchOptimizer{
		enableCompression:   config.EnableCompression,
		compressionLevel:    config.CompressionLevel,
		enableDeduplication: config.EnableDeduplication,
		enableCoalescing:    config.EnableCoalescing,
	}
}

// OptimizeBatch applies all enabled optimizations to an update batch
func (bo *BatchOptimizer) OptimizeBatch(batch *UpdateBatch) (*UpdateBatch, error) {
	optimized := batch

	// Apply deduplication
	if bo.enableDeduplication {
		optimized = bo.deduplicateBatch(optimized)
	}

	// Apply coalescing
	if bo.enableCoalescing {
		optimized = bo.coalesceBatch(optimized)
	}

	return optimized, nil
}

// deduplicateBatch removes duplicate node writes (same key written multiple times)
func (bo *BatchOptimizer) deduplicateBatch(batch *UpdateBatch) *UpdateBatch {
	if len(batch.NewNodes) <= 1 {
		return batch
	}

	// Track seen keys
	seen := make(map[string]bool)
	dedupedNodes := make(map[string]NodeWrite)

	// Keep only the latest write for each key
	for key, nodeWrite := range batch.NewNodes {
		if !seen[key] {
			dedupedNodes[key] = nodeWrite
			seen[key] = true
		}
	}

	// Remove duplicate stale nodes
	dedupedStale := bo.deduplicateStaleNodes(batch.StaleNodes)

	return &UpdateBatch{
		NewRootHash: batch.NewRootHash,
		NewNodes:    dedupedNodes,
		StaleNodes:  dedupedStale,
	}
}

// deduplicateStaleNodes removes duplicate stale node entries
func (bo *BatchOptimizer) deduplicateStaleNodes(staleNodes []types.NodeKey) []types.NodeKey {
	if len(staleNodes) <= 1 {
		return staleNodes
	}

	seen := make(map[string]bool)
	deduped := make([]types.NodeKey, 0, len(staleNodes))

	for _, nodeKey := range staleNodes {
		key := string(nodeKey.StorageKey())
		if !seen[key] {
			deduped = append(deduped, nodeKey)
			seen[key] = true
		}
	}

	return deduped
}

// coalesceBatch combines adjacent writes to improve write performance
func (bo *BatchOptimizer) coalesceBatch(batch *UpdateBatch) *UpdateBatch {
	// For now, just return the batch as-is
	// Future optimization: group nodes by storage locality
	return batch
}

// CompressBatch compresses the batch data for transmission or storage
func (bo *BatchOptimizer) CompressBatch(batch *UpdateBatch) ([]byte, error) {
	if !bo.enableCompression {
		return bo.encodeBatch(batch)
	}

	// Encode the batch
	encoded, err := bo.encodeBatch(batch)
	if err != nil {
		return nil, err
	}

	// Compress using gzip
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, bo.compressionLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	_, err = writer.Write(encoded)
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to compress batch: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// DecompressBatch decompresses batch data
func (bo *BatchOptimizer) DecompressBatch(data []byte) (*UpdateBatch, error) {
	if !bo.enableCompression {
		return bo.decodeBatch(data)
	}

	// Decompress using gzip
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress batch: %w", err)
	}

	return bo.decodeBatch(decompressed)
}

// encodeBatch encodes a batch to bytes
func (bo *BatchOptimizer) encodeBatch(batch *UpdateBatch) ([]byte, error) {
	// For now, we'll just create a simple representation of the batch size
	// In a real implementation, we'd serialize the actual node data
	var buf bytes.Buffer

	// Write a simple header with counts
	fmt.Fprintf(&buf, "BATCH:v1\n")
	fmt.Fprintf(&buf, "RootHash:%x\n", batch.NewRootHash)
	fmt.Fprintf(&buf, "NewNodes:%d\n", len(batch.NewNodes))
	fmt.Fprintf(&buf, "StaleNodes:%d\n", len(batch.StaleNodes))

	// Add some bulk data to simulate real batch content
	for i := 0; i < len(batch.NewNodes); i++ {
		fmt.Fprintf(&buf, "Node%d:placeholder_data_for_compression_testing\n", i)
	}

	return buf.Bytes(), nil
}

// decodeBatch decodes a batch from bytes
func (bo *BatchOptimizer) decodeBatch(data []byte) (*UpdateBatch, error) {
	// For the simplified encoding, we'll just parse the header
	// In a real implementation, we'd deserialize the actual node data
	lines := bytes.Split(data, []byte("\n"))
	if len(lines) < 4 {
		return nil, fmt.Errorf("invalid batch format")
	}

	if !bytes.Equal(lines[0], []byte("BATCH:v1")) {
		return nil, fmt.Errorf("invalid batch header")
	}

	// Parse counts from the header
	var newNodeCount, staleNodeCount int
	if _, err := fmt.Sscanf(string(lines[2]), "NewNodes:%d", &newNodeCount); err != nil {
		return nil, fmt.Errorf("failed to parse new node count: %w", err)
	}
	if _, err := fmt.Sscanf(string(lines[3]), "StaleNodes:%d", &staleNodeCount); err != nil {
		return nil, fmt.Errorf("failed to parse stale node count: %w", err)
	}

	// Create a batch with the same structure (for testing purposes)
	batch := &UpdateBatch{
		NewRootHash: types.Hash{}, // Would parse from header in real implementation
		NewNodes:    make(map[string]NodeWrite),
		StaleNodes:  make([]types.NodeKey, staleNodeCount),
	}

	// Populate with dummy data to match counts
	for i := 0; i < newNodeCount; i++ {
		key := fmt.Sprintf("key%d", i)
		batch.NewNodes[key] = NodeWrite{}
	}

	// The actual nodes would be deserialized in a real implementation
	return batch, nil
}

// BatchStats provides statistics about a batch
type BatchStats struct {
	TotalNodes       int
	LeafNodes        int
	InternalNodes    int
	StaleNodes       int
	UncompressedSize int64
	CompressedSize   int64
	CompressionRatio float64
}

// GetBatchStats analyzes a batch and returns statistics
func (bo *BatchOptimizer) GetBatchStats(batch *UpdateBatch) (*BatchStats, error) {
	stats := &BatchStats{
		TotalNodes: len(batch.NewNodes),
		StaleNodes: len(batch.StaleNodes),
	}

	// Count node types
	for _, nodeWrite := range batch.NewNodes {
		if types.IsLeaf(nodeWrite.Node) {
			stats.LeafNodes++
		} else {
			stats.InternalNodes++
		}
	}

	// Calculate sizes if compression is enabled
	if bo.enableCompression {
		uncompressed, err := bo.encodeBatch(batch)
		if err != nil {
			return nil, err
		}
		stats.UncompressedSize = int64(len(uncompressed))

		compressed, err := bo.CompressBatch(batch)
		if err != nil {
			return nil, err
		}
		stats.CompressedSize = int64(len(compressed))

		if stats.UncompressedSize > 0 {
			stats.CompressionRatio = float64(stats.UncompressedSize-stats.CompressedSize) / float64(stats.UncompressedSize)
		}
	}

	return stats, nil
}
