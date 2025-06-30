---
id: step.24.fuzzing
depends_on:
  - step.23.monitoring
tags: [fuzzing, testing, security, step]
---

## Objective

Implement comprehensive fuzzing to catch edge cases, version isolation bugs, and security vulnerabilities that traditional testing might miss.

## Motivation

The version isolation bug discovered in step 14 highlighted a critical gap in our testing:
- Traditional tests often follow "happy paths" and expected usage patterns
- Edge cases involving shared internal structures across versions were missed
- Negative test cases (verifying what shouldn't exist) were underrepresented

Fuzzing addresses these gaps by:
- Generating unexpected input combinations
- Testing invariants across random operations
- Finding edge cases humans don't anticipate
- Verifying both positive and negative properties

## Technical Details

### Fuzzing Architecture

**Property-Based Testing:**
- Define invariants that must hold for all operations
- Generate random sequences of operations
- Verify invariants after each operation
- Shrink failing cases to minimal reproduction

**Coverage-Guided Fuzzing:**
- Use go-fuzz for coverage-guided fuzzing
- Target specific code paths (tree traversal, version boundaries)
- Mutate inputs to maximize code coverage
- Persist interesting test cases

**Differential Fuzzing:**
- Compare JMT behavior against reference implementation
- Verify proof generation/verification consistency
- Test version isolation properties
- Ensure structural sharing correctness

### Key Invariants to Test

**Version Isolation:**
```go
// Keys added in version N should not be visible in versions < N
func InvariantVersionIsolation(ops []Operation) bool {
    tree := NewTree()
    keyVersionMap := make(map[Key]Version)
    
    for _, op := range ops {
        switch op.Type {
        case OpPut:
            v, _ := tree.Put(op.Key, op.Value)
            keyVersionMap[op.Key] = v
        case OpGet:
            value, _ := tree.GetAtVersion(op.Version, op.Key)
            if firstVersion, exists := keyVersionMap[op.Key]; exists {
                if op.Version < firstVersion && value != nil {
                    return false // Violation: key visible before creation
                }
            }
        }
    }
    return true
}
```

**Structural Sharing:**
```go
// Unchanged subtrees should share nodes across versions
func InvariantStructuralSharing(ops []Operation) bool {
    // Track node addresses before and after operations
    // Verify unchanged paths use same node instances
}
```

**Proof Consistency:**
```go
// Proofs generated for a key should verify against the correct root hash
func InvariantProofConsistency(ops []Operation) bool {
    // Generate proof at each version
    // Verify proof validates correctly
    // Ensure proof fails against other versions
}
```

### Fuzzing Targets

**Core Operations:**
- Put/Delete with random keys and values
- Version creation, commit, and abort
- Concurrent operations
- Batch operations with varying sizes

**Edge Cases:**
- Empty trees and single-node trees
- Maximum depth trees (keys with common prefixes)
- Version number overflow scenarios
- Large value sizes
- Garbage collection during operations

**Key Patterns:**
- Keys sharing common prefixes (test internal node sharing)
- Keys at opposite ends of the tree
- Sequential keys
- Random distribution
- Adversarial patterns (all 0s, all 1s)

### Implementation Strategy

**Phase 1: Property-Based Testing**
```go
// Use gopter or similar property testing framework
type TreeProperties struct {
    tree *Tree
}

func (p *TreeProperties) VersionIsolation(cmds []Command) bool {
    // Execute commands and verify version isolation
}

func (p *TreeProperties) StructuralSharing(cmds []Command) bool {
    // Verify nodes are shared when appropriate
}

func (p *TreeProperties) CrashConsistency(cmds []Command) bool {
    // Simulate crashes and verify recovery
}
```

**Phase 2: Coverage-Guided Fuzzing**
```go
// Fuzz function for go-fuzz
func Fuzz(data []byte) int {
    ops, err := parseOperations(data)
    if err != nil {
        return 0
    }
    
    tree := NewTree()
    for _, op := range ops {
        if err := executeOp(tree, op); err != nil {
            if isExpectedError(err) {
                return 0
            }
            panic(err) // Unexpected error
        }
    }
    
    // Verify invariants
    if !checkInvariants(tree) {
        panic("invariant violation")
    }
    
    return 1
}
```

**Phase 3: Differential Testing**
```go
// Compare against reference implementation or different configurations
func DifferentialFuzz(data []byte) {
    ops := parseOperations(data)
    
    tree1 := NewTree(ConfigA)
    tree2 := NewTree(ConfigB)
    
    for _, op := range ops {
        result1 := executeOp(tree1, op)
        result2 := executeOp(tree2, op)
        
        if !compareResults(result1, result2) {
            panic("differential test failure")
        }
    }
}
```

### Continuous Fuzzing

**Integration with CI:**
- Run short fuzzing sessions on each PR
- Nightly extended fuzzing runs
- Store corpus of interesting test cases
- Automatically minimize failing cases

**Metrics and Monitoring:**
- Code coverage achieved
- Number of unique paths explored
- Invariant violations found
- Performance regression detection

### Security Fuzzing

**Attack Scenarios:**
- Malicious proof construction
- DoS through deep tree construction
- Memory exhaustion attacks
- Version manipulation attempts

**Input Validation:**
- Fuzz all public API entry points
- Test boundary conditions
- Verify error handling paths
- Check for panics or crashes

## Testing Requirements

### Fuzzer Development
- Implement property-based testing suite
- Create go-fuzz harnesses for core operations
- Build differential testing framework
- Develop invariant checkers

### Corpus Building
- Generate seed inputs covering basic operations
- Include edge cases from previous bugs
- Add adversarial patterns
- Maintain regression corpus

### Performance Fuzzing
- Memory usage under random operations
- CPU usage patterns
- Storage growth characteristics
- Cache effectiveness

## Done When ✓

- [ ] Property-based tests for all invariants
- [ ] go-fuzz integration with >80% code coverage
- [ ] Differential fuzzing against reference
- [ ] Continuous fuzzing in CI pipeline
- [ ] Security-focused fuzzing harnesses
- [ ] Performance regression detection
- [ ] Corpus of interesting test cases maintained
- [ ] Documentation of fuzzing strategy and findings
- [ ] Automated bug minimization workflow
- [ ] Version isolation thoroughly fuzzed