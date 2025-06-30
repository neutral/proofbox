package tree

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/neutral/proofbox/pkg/types"
)

// VersionStatus represents the state of a version
type VersionStatus int

const (
	// VersionStatusPending indicates version is being constructed
	VersionStatusPending VersionStatus = iota
	// VersionStatusCommitted indicates version is finalized and immutable
	VersionStatusCommitted
	// VersionStatusAborted indicates version was cancelled
	VersionStatusAborted
)

// VersionInfo stores metadata for a version
type VersionInfo struct {
	Version       types.Version
	RootHash      types.Hash
	ParentVersion types.Version
	CreatedAt     time.Time
	NodeCount     int64
	Status        VersionStatus
}

// PendingVersion tracks in-progress version creation
type PendingVersion struct {
	version    types.Version
	parent     types.Version
	updater    *TreeUpdater
	startTime  time.Time
	operations int
}

// GetUpdater returns the tree updater for this pending version
func (pv *PendingVersion) GetUpdater() *TreeUpdater {
	return pv.updater
}

// VersionRetentionPolicy defines how versions are retained
type VersionRetentionPolicy int

const (
	// RetentionPolicyCount keeps last N versions
	RetentionPolicyCount VersionRetentionPolicy = iota
	// RetentionPolicyTime keeps versions newer than duration
	RetentionPolicyTime
	// RetentionPolicyNone keeps all versions (no GC)
	RetentionPolicyNone
)

// VersionManager handles multi-version state management
type VersionManager struct {
	mu               sync.RWMutex
	versions         map[types.Version]*VersionInfo
	latestVersion    types.Version
	committedVersion types.Version
	pendingWrites    map[types.Version]*PendingVersion

	// Garbage collection settings
	retentionPolicy   VersionRetentionPolicy
	minVersionsToKeep int           // Minimum versions to retain
	maxAge            time.Duration // For time-based retention
}

// NewVersionManager creates a new version manager
func NewVersionManager() *VersionManager {
	return NewVersionManagerWithRetention(RetentionPolicyCount, 100, 0)
}

// NewVersionManagerWithRetention creates a version manager with retention policy
func NewVersionManagerWithRetention(policy VersionRetentionPolicy, minVersions int, maxAge time.Duration) *VersionManager {
	vm := &VersionManager{
		versions:          make(map[types.Version]*VersionInfo),
		pendingWrites:     make(map[types.Version]*PendingVersion),
		retentionPolicy:   policy,
		minVersionsToKeep: minVersions,
		maxAge:            maxAge,
	}

	// Initialize with version 0
	vm.versions[types.InitialVersion] = &VersionInfo{
		Version:       types.InitialVersion,
		RootHash:      types.EmptyHash(),
		ParentVersion: types.InitialVersion, // Self-referential for initial
		CreatedAt:     time.Now(),
		Status:        VersionStatusCommitted,
	}
	vm.committedVersion = types.InitialVersion

	return vm
}

// Begin starts a new version based on parent
func (vm *VersionManager) Begin(parentVersion types.Version) (types.Version, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Validate parent version exists
	parent, exists := vm.versions[parentVersion]
	if !exists {
		return 0, fmt.Errorf("parent version %d not found", parentVersion)
	}

	if parent.Status != VersionStatusCommitted {
		return 0, fmt.Errorf("parent version %d not committed", parentVersion)
	}

	// Check for version overflow
	if vm.latestVersion == types.MaxVersion {
		return 0, fmt.Errorf("version overflow: cannot create version after %d", vm.latestVersion)
	}

	// Allocate new version number
	newVersion := vm.latestVersion + 1

	// Create pending version
	pending := &PendingVersion{
		version:   newVersion,
		parent:    parentVersion,
		startTime: time.Now(),
	}

	// Note: TreeUpdater will be initialized when tree operations begin

	vm.pendingWrites[newVersion] = pending
	vm.latestVersion = newVersion

	return newVersion, nil
}

// Commit finalizes a pending version
func (vm *VersionManager) Commit(version types.Version, rootHash types.Hash, nodeCount int64) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	pending, exists := vm.pendingWrites[version]
	if !exists {
		return fmt.Errorf("no pending version %d", version)
	}

	// Create version info
	info := &VersionInfo{
		Version:       version,
		RootHash:      rootHash,
		ParentVersion: pending.parent,
		CreatedAt:     time.Now(),
		NodeCount:     nodeCount,
		Status:        VersionStatusCommitted,
	}

	vm.versions[version] = info
	delete(vm.pendingWrites, version)

	// Update committed version if this is latest
	if version > vm.committedVersion {
		vm.committedVersion = version
	}

	return nil
}

// Abort cancels a pending version
func (vm *VersionManager) Abort(version types.Version) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	pending, exists := vm.pendingWrites[version]
	if !exists {
		return fmt.Errorf("no pending version %d", version)
	}

	delete(vm.pendingWrites, version)

	// Mark as aborted
	vm.versions[version] = &VersionInfo{
		Version:       version,
		ParentVersion: pending.parent,
		Status:        VersionStatusAborted,
		CreatedAt:     pending.startTime,
	}

	return nil
}

// GetVersion retrieves version metadata
func (vm *VersionManager) GetVersion(version types.Version) (*VersionInfo, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	info, exists := vm.versions[version]
	if !exists {
		return nil, fmt.Errorf("version %d not found", version)
	}

	return info, nil
}

// GetLatestCommitted returns the latest committed version
func (vm *VersionManager) GetLatestCommitted() types.Version {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.committedVersion
}

// GetPending retrieves a pending version
func (vm *VersionManager) GetPending(version types.Version) *PendingVersion {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.pendingWrites[version]
}

// SetPendingUpdater associates an updater with a pending version
func (vm *VersionManager) SetPendingUpdater(version types.Version, updater *TreeUpdater) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	pending, exists := vm.pendingWrites[version]
	if !exists {
		return fmt.Errorf("no pending version %d", version)
	}

	pending.updater = updater
	return nil
}

// GetAllVersions returns all version info sorted by version number
func (vm *VersionManager) GetAllVersions() []*VersionInfo {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	result := make([]*VersionInfo, 0, len(vm.versions))
	for _, info := range vm.versions {
		result = append(result, info)
	}

	// Sort by version number
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Version > result[j].Version {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// CollectGarbage removes old versions based on retention policy
func (vm *VersionManager) CollectGarbage() ([]types.Version, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.retentionPolicy == RetentionPolicyNone {
		return nil, nil
	}

	// Get all committed versions
	var committedVersions []types.Version
	for v, info := range vm.versions {
		if info.Status == VersionStatusCommitted && v != types.InitialVersion {
			committedVersions = append(committedVersions, v)
		}
	}

	// Sort versions
	sort.Slice(committedVersions, func(i, j int) bool {
		return committedVersions[i] < committedVersions[j]
	})

	// Determine which versions to remove
	var toRemove []types.Version
	now := time.Now()

	switch vm.retentionPolicy {
	case RetentionPolicyCount:
		// Keep only the last N versions
		if len(committedVersions) > vm.minVersionsToKeep {
			toRemove = committedVersions[:len(committedVersions)-vm.minVersionsToKeep]
		}

	case RetentionPolicyTime:
		// Remove versions older than maxAge
		for _, v := range committedVersions {
			info := vm.versions[v]
			if now.Sub(info.CreatedAt) > vm.maxAge {
				toRemove = append(toRemove, v)
			}
		}

		// But always keep minimum number
		if len(committedVersions)-len(toRemove) < vm.minVersionsToKeep {
			toRemove = toRemove[:len(committedVersions)-vm.minVersionsToKeep]
		}
	}

	// Never remove the current committed version
	filtered := make([]types.Version, 0, len(toRemove))
	for _, v := range toRemove {
		if v != vm.committedVersion {
			filtered = append(filtered, v)
		}
	}
	toRemove = filtered

	// Remove the versions
	for _, v := range toRemove {
		delete(vm.versions, v)
	}

	return toRemove, nil
}

// SetRetentionPolicy updates the retention policy
func (vm *VersionManager) SetRetentionPolicy(policy VersionRetentionPolicy, minVersions int, maxAge time.Duration) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	vm.retentionPolicy = policy
	vm.minVersionsToKeep = minVersions
	vm.maxAge = maxAge
}

// RemoveVersion removes version metadata (used by pruner)
func (vm *VersionManager) RemoveVersion(version types.Version) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	_, exists := vm.versions[version]
	if !exists {
		return fmt.Errorf("version %d not found", version)
	}

	// Don't allow removing the committed version
	if version == vm.committedVersion {
		return fmt.Errorf("cannot remove committed version %d", version)
	}

	// Don't allow removing if it's a parent of existing versions
	for _, v := range vm.versions {
		if v.ParentVersion == version && v.Version != version {
			return fmt.Errorf("version %d is parent of version %d", version, v.Version)
		}
	}

	delete(vm.versions, version)
	return nil
}

// IncrementOperations increments the operation count for a pending version
func (vm *VersionManager) IncrementOperations(version types.Version) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	pending, exists := vm.pendingWrites[version]
	if !exists {
		return fmt.Errorf("no pending version %d", version)
	}

	pending.operations++
	return nil
}
