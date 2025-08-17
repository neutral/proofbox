---
id: fuzzing-process
title: Fuzzing Process and Workflow
status: active
created: 2024-01-03
updated: 2024-01-03
tags: [specs, process, fuzzing, workflow]
---

# Fuzzing Process and Workflow

## Overview

This document defines the process and workflow for fuzzing in ProofBox, from test development through bug discovery and resolution.

## Fuzzing Lifecycle

### 1. Invariant Identification

**Process**:
1. Analyze system requirements
2. Identify critical properties
3. Define measurable invariants
4. Prioritize by risk/impact

**Example Invariants**:
- Version isolation: No future data leakage
- Proof validity: Cryptographic consistency
- Memory safety: No use-after-free
- Concurrency: No data races

**Documentation**:
- Invariant specification document
- Test acceptance criteria
- Validation methodology

### 2. Test Development

**Generator Development**:
```go
// 1. Define operation types
type OpType int
const (
    OpPut OpType = iota
    OpGet
    OpDelete
)

// 2. Create operation generator
func OperationGen() *rapid.Generator[Operation] {
    return rapid.Custom(func(t *rapid.T) Operation {
        // Generate random but valid operations
    })
}

// 3. Create sequence generator
func OperationSliceGen(min, max int) *rapid.Generator[[]Operation] {
    return rapid.SliceOfN(OperationGen(), min, max)
}
```

**Invariant Checker Development**:
```go
// Define checker interface
type InvariantChecker interface {
    Check(tree *tree.Tree, ops []Operation) error
}

// Implement specific invariant
func CheckVersionIsolation(tree *tree.Tree, ops []Operation) error {
    tracker := NewVersionTracker()
    // Apply operations and validate invariant
}
```

**Test Integration**:
```go
func TestInvariant(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        ops := OperationSliceGen(1, 100).Draw(t, "ops")
        tree := NewTestTree()
        err := CheckInvariant(tree, ops)
        require.NoError(t, err)
    })
}
```

### 3. Test Execution

**Local Development**:
```bash
# Quick validation (20 iterations)
go test -run TestInvariant -rapid.checks=20

# Standard testing (100 iterations)
go test -run TestInvariant

# Comprehensive testing (1000 iterations)
go test -run TestInvariant -rapid.checks=1000

# With race detection
go test -race -run TestInvariant
```

**CI/CD Pipeline**:

**PR Validation**:
- Triggered on: Pull request creation/update
- Tests run: Quick fuzzing (20 iterations)
- Duration: ~30 seconds
- Required: Must pass for merge

**Merge Validation**:
- Triggered on: Merge to main
- Tests run: Standard fuzzing (100 iterations)
- Duration: ~5 minutes
- Actions: Update coverage reports

**Nightly Testing**:
- Triggered on: Daily schedule (2 AM UTC)
- Tests run: Comprehensive (1000 iterations)
- Duration: ~1 hour
- Actions: Performance baselines, reports

### 4. Failure Handling

**Automatic Minimization**:
1. Rapid detects test failure
2. Shrinking algorithm activates
3. Systematically reduces input
4. Finds minimal reproduction
5. Saves to testdata/rapid/

**Failure Analysis Workflow**:

```bash
# 1. Reproduce failure
go test -run TestInvariant -rapid.failfile=testdata/rapid/failure.fail

# 2. Enable verbose logging
go test -v -run TestInvariant -rapid.failfile=testdata/rapid/failure.fail

# 3. Add debug instrumentation
// In test code
if debug {
    log.Printf("Operation %d: %+v", i, op)
    log.Printf("State: %+v", getCurrentState())
}

# 4. Run with debugger
dlv test -- -test.run TestInvariant -rapid.failfile=testdata/rapid/failure.fail
```

**Root Cause Analysis**:
1. Understand the minimal failing case
2. Trace through operation sequence
3. Identify invariant violation point
4. Locate buggy code
5. Develop fix

### 5. Bug Resolution

**Fix Development**:
1. Create failing test from minimized case
2. Implement fix
3. Verify test passes
4. Run full test suite
5. Check performance impact

**Regression Testing**:
```go
func TestRegression_Issue42(t *testing.T) {
    // Specific sequence that triggered bug
    ops := []Operation{
        {Type: OpPut, Key: key1, Value: val1},
        {Type: OpDelete, Key: key1},
        {Type: OpGet, Key: key1, Version: 0},
    }
    
    tree := NewTestTree()
    err := CheckInvariant(tree, ops)
    require.NoError(t, err)
}
```

### 6. Continuous Improvement

**Metrics Collection**:
- Bugs found per week
- Test execution time
- Code coverage
- Invariant violations

**Process Refinement**:
- Weekly bug triage
- Monthly process review
- Quarterly goal setting
- Annual methodology assessment

## Fuzzing Infrastructure

### Test Organization

```
pkg/fuzz/
├── generators/          # Input generation
│   └── operations.go    # Operation generators
├── invariants/          # Invariant checkers
│   ├── version_isolation.go
│   ├── proof_consistency.go
│   └── structural_sharing.go
├── properties/          # Property-based tests
│   ├── concurrent_test.go
│   ├── proof_consistency_test.go
│   └── version_isolation_test.go
├── performance/         # Performance tests
│   └── performance_test.go
├── corpus/             # Corpus builder
│   └── builder.go
└── test_helpers.go     # Shared utilities
```

### Corpus Management

**Corpus Builder Usage**:
```bash
# Generate comprehensive corpus
go run pkg/fuzz/corpus/cmd/build_corpus.go -output corpus/

# Categories generated:
# - edge_cases/
# - version_isolation/
# - performance/
# - security/
```

**Corpus Maintenance**:
- Gitignored to prevent repo bloat
- Shared via artifact storage
- Updated with new edge cases
- Pruned of redundant cases

### Performance Monitoring

**Benchmarking**:
```bash
# Run performance tests
go test -bench=. ./pkg/fuzz/performance/

# With memory profiling
go test -bench=. -benchmem ./pkg/fuzz/performance/

# Generate profiles
go test -bench=. -cpuprofile=cpu.prof ./pkg/fuzz/performance/
```

**Key Metrics**:
- Operations per second
- Memory allocations
- Latency percentiles
- Cache hit rates

## Best Practices

### Test Development

1. **Start Simple**: Basic invariant, add complexity
2. **Think Adversarially**: What could break this?
3. **Use Generators**: Don't hand-craft test cases
4. **Check Boundaries**: Empty, single, maximum
5. **Document Intent**: Why this invariant matters

### Debugging

1. **Use Minimization**: Work with smallest case
2. **Add Logging**: Trace operation flow
3. **Visualize State**: Print tree structure
4. **Binary Search**: Isolate failing operation
5. **Check Assumptions**: Verify prerequisites

### Performance

1. **Profile First**: Don't optimize blindly
2. **Batch Operations**: Reduce overhead
3. **Reuse Resources**: Cache expensive setup
4. **Parallel Tests**: Utilize all cores
5. **Fail Fast**: Stop on first error

## Integration with Development

### Pre-commit Hooks

```bash
#!/bin/bash
# .git/hooks/pre-commit

# Run quick fuzzing
go test -run Fuzz -rapid.checks=10 ./pkg/fuzz/...
```

### IDE Integration

**VS Code tasks.json**:
```json
{
    "label": "Fuzz Test",
    "type": "shell",
    "command": "go test -v -run ${selectedText} -rapid.checks=50",
    "problemMatcher": ["$go"]
}
```

### Review Process

**Fuzzing Checklist**:
- [ ] New features have fuzz tests
- [ ] Invariants are documented
- [ ] Tests run in <1 minute
- [ ] No flaky tests
- [ ] Regression tests added

## Incident Response

### Fuzzing Bug Discovery

**Severity Levels**:
- **Critical**: Data corruption, security
- **High**: Incorrect behavior, crashes
- **Medium**: Performance degradation
- **Low**: Minor inconsistencies

**Response Times**:
- Critical: Fix within 24 hours
- High: Fix within 1 week
- Medium: Fix within 1 sprint
- Low: Fix in next release

**Communication**:
1. File issue with reproduction
2. Assign to domain expert
3. Update status daily
4. Notify when resolved
5. Post-mortem if critical

## Future Enhancements

1. **Distributed Fuzzing**: Scale across machines
2. **Guided Fuzzing**: Use coverage feedback
3. **Grammar-Based**: Domain-specific languages
4. **Stateful Fuzzing**: Model protocol states
5. **AI-Assisted**: Learn from patterns