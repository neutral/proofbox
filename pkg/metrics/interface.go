package metrics

// JMTMetrics defines the interface for Jellyfish Merkle Tree metrics collection
type JMTMetrics interface {
	// Core JMT Operations
	RecordCommit(duration float64, batchSize int)
	RecordOperation(opType string) // "insert", "update", "delete"
	RecordLookup(duration float64, found bool)

	// Proof Operations
	RecordProofGeneration(duration float64, proofType string, proofSize int) // proofType: "inclusion", "exclusion"
	RecordProofValidationFailure()

	// Database Operations
	RecordHashOperation()
	RecordDBRead()
	RecordDBWrite()

	// Error Tracking
	RecordError(errorType string) // "storage", "validation", "encoding", etc.

	// Tree State Updates
	UpdateTreeHeight(height int)
	UpdateTreeNodeCount(count int)
	UpdateVersionCount(count int)
	UpdateDatabaseSize(bytes int64)
}

// NoOpMetrics provides a no-operation implementation for when metrics are disabled
type NoOpMetrics struct{}

func (n NoOpMetrics) RecordCommit(duration float64, batchSize int)                       {}
func (n NoOpMetrics) RecordOperation(opType string)                                     {}
func (n NoOpMetrics) RecordLookup(duration float64, found bool)                         {}
func (n NoOpMetrics) RecordProofGeneration(duration float64, proofType string, proofSize int) {}
func (n NoOpMetrics) RecordProofValidationFailure()                                     {}
func (n NoOpMetrics) RecordHashOperation()                                              {}
func (n NoOpMetrics) RecordDBRead()                                                     {}
func (n NoOpMetrics) RecordDBWrite()                                                    {}
func (n NoOpMetrics) RecordError(errorType string)                                      {}
func (n NoOpMetrics) UpdateTreeHeight(height int)                                       {}
func (n NoOpMetrics) UpdateTreeNodeCount(count int)                                     {}
func (n NoOpMetrics) UpdateVersionCount(count int)                                      {}
func (n NoOpMetrics) UpdateDatabaseSize(bytes int64)                                    {}

// Ensure NoOpMetrics implements JMTMetrics interface
var _ JMTMetrics = NoOpMetrics{}