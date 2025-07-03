package storage

import (
	"time"

	"github.com/neutral/proofbox/pkg/types"
)

// PruningPolicy determines which versions should be pruned from storage.
type PruningPolicy interface {
	// ShouldPrune returns true if the version should be removed.
	ShouldPrune(version types.Version, info VersionInfo) bool
}

// VersionInfo contains metadata about a stored version.
type VersionInfo struct {
	Version   types.Version
	Timestamp time.Time
	Size      uint64 // Approximate size in bytes
}

// KeepLastNVersions keeps only the most recent N versions.
type KeepLastNVersions struct {
	N              int
	currentVersion types.Version
}

// NewKeepLastNVersions creates a policy that keeps the last N versions.
func NewKeepLastNVersions(n int, currentVersion types.Version) PruningPolicy {
	return &KeepLastNVersions{
		N:              n,
		currentVersion: currentVersion,
	}
}

// ShouldPrune returns true if the version is old enough to prune.
func (p *KeepLastNVersions) ShouldPrune(version types.Version, info VersionInfo) bool {
	// Never prune the current version
	if version == p.currentVersion {
		return false
	}

	// Keep if within N versions of current
	return p.currentVersion-version > types.Version(p.N)
}

// KeepVersionsNewerThan keeps versions created within the specified duration.
type KeepVersionsNewerThan struct {
	Age time.Duration
	Now time.Time
}

// NewKeepVersionsNewerThan creates a policy based on version age.
func NewKeepVersionsNewerThan(age time.Duration) PruningPolicy {
	return &KeepVersionsNewerThan{
		Age: age,
		Now: time.Now(),
	}
}

// ShouldPrune returns true if the version is older than the age limit.
func (p *KeepVersionsNewerThan) ShouldPrune(version types.Version, info VersionInfo) bool {
	return p.Now.Sub(info.Timestamp) > p.Age
}

// KeepMilestoneVersions keeps every Nth version as a milestone.
type KeepMilestoneVersions struct {
	Interval       int
	currentVersion types.Version
}

// NewKeepMilestoneVersions creates a policy that keeps milestone versions.
func NewKeepMilestoneVersions(interval int, currentVersion types.Version) PruningPolicy {
	return &KeepMilestoneVersions{
		Interval:       interval,
		currentVersion: currentVersion,
	}
}

// ShouldPrune returns true if the version is not a milestone.
func (p *KeepMilestoneVersions) ShouldPrune(version types.Version, info VersionInfo) bool {
	// Never prune the current version
	if version == p.currentVersion {
		return false
	}

	// Keep if it's a milestone version
	return uint64(version)%uint64(p.Interval) != 0
}

// CompositePruningPolicy combines multiple policies with AND/OR logic.
type CompositePruningPolicy struct {
	Policies []PruningPolicy
	All      bool // true = AND, false = OR
}

// NewCompositePruningPolicy creates a policy that combines multiple policies.
func NewCompositePruningPolicy(all bool, policies ...PruningPolicy) PruningPolicy {
	return &CompositePruningPolicy{
		Policies: policies,
		All:      all,
	}
}

// ShouldPrune applies all policies and combines results.
func (p *CompositePruningPolicy) ShouldPrune(version types.Version, info VersionInfo) bool {
	if len(p.Policies) == 0 {
		return false
	}

	for _, policy := range p.Policies {
		shouldPrune := policy.ShouldPrune(version, info)

		if p.All && !shouldPrune {
			// AND logic: if any policy says keep, we keep
			return false
		}
		if !p.All && shouldPrune {
			// OR logic: if any policy says prune, we prune
			return true
		}
	}

	// AND logic: all said prune
	// OR logic: none said prune
	return p.All
}
