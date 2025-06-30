package tree

import (
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
)

// Put inserts or updates a key-value pair, creating a new version
func (t *Tree) Put(key types.Key, value []byte) (types.Version, error) {
	// Validate inputs first, before creating a version
	if err := t.validateKey(key); err != nil {
		return 0, fmt.Errorf("invalid key: %w", err)
	}
	if err := t.validateValue(value); err != nil {
		return 0, fmt.Errorf("invalid value: %w", err)
	}
	
	// Begin new version
	version, err := t.BeginVersion()
	if err != nil {
		return 0, fmt.Errorf("failed to begin version: %w", err)
	}
	
	// Put in the new version
	if err := t.PutVersioned(version, key, value); err != nil {
		// Abort on error
		t.AbortVersion(version)
		return 0, fmt.Errorf("failed to put: %w", err)
	}
	
	// Commit the version
	if err := t.CommitVersion(version); err != nil {
		// Abort on error
		t.AbortVersion(version)
		return 0, fmt.Errorf("failed to commit: %w", err)
	}
	
	return version, nil
}
