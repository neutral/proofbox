package tree

import (
	"fmt"

	"github.com/neutral/proofbox/pkg/types"
)

// Delete removes a key from the tree, creating a new version
func (t *Tree) Delete(key types.Key) (types.Version, error) {
	// Begin new version
	version, err := t.BeginVersion()
	if err != nil {
		return 0, fmt.Errorf("failed to begin version: %w", err)
	}

	// Delete in the new version
	if err := t.DeleteVersioned(version, key); err != nil {
		// Abort on error
		t.AbortVersion(version)
		return 0, fmt.Errorf("failed to delete: %w", err)
	}

	// Commit the version
	if err := t.CommitVersion(version); err != nil {
		// Abort on error
		t.AbortVersion(version)
		return 0, fmt.Errorf("failed to commit: %w", err)
	}

	return version, nil
}
