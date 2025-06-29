package tree

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/neutral/proofbox/pkg/types"
)

// DefaultReadErrorHandler implements basic recovery
type DefaultReadErrorHandler struct {
	logger *log.Logger
}

// NewDefaultReadErrorHandler creates a new read error handler
func NewDefaultReadErrorHandler(logger *log.Logger) *DefaultReadErrorHandler {
	if logger == nil {
		logger = log.New(log.Writer(), "[tree] ", log.LstdFlags)
	}
	return &DefaultReadErrorHandler{logger: logger}
}

// HandleCorruptedNode handles corrupted node detection
func (h *DefaultReadErrorHandler) HandleCorruptedNode(key types.NodeKey, err error) error {
	h.logger.Printf("ERROR: Corrupted node detected at version=%d path=%x: %v",
		key.Version, key.NibblePath.Nibbles, err)

	// Return wrapped error with context
	return fmt.Errorf("corrupted node at %v: %w", key, err)
}

// HandleMissingNode handles missing node detection
func (h *DefaultReadErrorHandler) HandleMissingNode(key types.NodeKey) error {
	// Missing nodes might be due to pruning
	h.logger.Printf("WARN: Missing node at version=%d path=%x",
		key.Version, key.NibblePath.Nibbles)
	return fmt.Errorf("node not found at %v", key)
}

// DefaultWriteErrorHandler implements exponential backoff
type DefaultWriteErrorHandler struct {
	maxRetries int
	baseDelay  time.Duration
	logger     *log.Logger
}

// NewDefaultWriteErrorHandler creates a new write error handler
func NewDefaultWriteErrorHandler(maxRetries int, baseDelay time.Duration, logger *log.Logger) *DefaultWriteErrorHandler {
	if logger == nil {
		logger = log.New(log.Writer(), "[tree] ", log.LstdFlags)
	}
	return &DefaultWriteErrorHandler{
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
		logger:     logger,
	}
}

// HandleWriteFailure handles write operation failures
func (h *DefaultWriteErrorHandler) HandleWriteFailure(err error) error {
	h.logger.Printf("ERROR: Write failure: %v", err)
	return fmt.Errorf("write operation failed: %w", err)
}

// ShouldRetry determines if an error is retryable
func (h *DefaultWriteErrorHandler) ShouldRetry(err error) bool {
	// Retry on temporary errors
	// In practice, would check for specific error types
	return false // Conservative default
}

// RetryDelay calculates delay for retry attempt
func (h *DefaultWriteErrorHandler) RetryDelay(attempt int) time.Duration {
	// Exponential backoff with jitter
	delay := h.baseDelay * time.Duration(1<<uint(attempt))
	// Add up to 25% jitter
	jitter := time.Duration(rand.Float64() * float64(delay) * 0.25)
	return delay + jitter
}
