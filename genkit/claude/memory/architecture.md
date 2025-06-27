# Architecture and Structure

## Project Overview
ProofBox is a Go implementation of the Jellyfish Merkle Tree (JMT) - a verifiable, versioned key-value store originally developed for blockchain use.

## Blueprint-Driven Development

The `/blueprint/` directory is the authoritative source for all requirements:

```
blueprint/
├── global/         # Vision, business goals, and NFRs
├── features/       # Functional requirements by capability
├── steps/          # 25 detailed implementation steps
├── decisions/      # Architecture Decision Records (ADRs)
└── archive/        # Deprecated content
```

Active development tracked via `.active.md` suffix on step files.

## Technical Specifications

### Core Data Structure
- **Type**: Sparse, addressable radix-16 Merkle tree
- **Node Types**: 
  - Internal Nodes (16-way branching)
  - Leaf Nodes only
- **Key Space**: 256-bit keys with 256-bit root hash
- **Branching Factor**: 16 (hexary tree)

### Storage Design
- **Backend**: PebbleDB (LSM-tree based)
- **Versioning**: Append-only storage with version-based node keys
- **Optimization**: Designed for Log-Structured Merge trees

### Performance Targets
- Write throughput: <10 MB/s sustained on commodity SSDs
- Proof size: Optimized for conciseness
- Memory usage: Efficient for large state sizes

## Implementation Progress

Currently at Step 01 of 25 (Project Scaffold). Key milestones:

1. **Foundation** (Steps 1-5)
   - Project setup, core data structures
   - Hasher implementation, key/node types

2. **Core Operations** (Steps 6-12)
   - Tree operations (insert, update, delete)
   - Proof generation and verification
   - Batch operations

3. **Persistence** (Steps 13-18)
   - PebbleDB integration
   - Versioning and transaction support
   - Recovery mechanisms

4. **Production** (Steps 19-25)
   - Performance optimization
   - Monitoring and metrics
   - CI/CD setup

## Key Design Decisions

1. **Radix-16 Tree**: Balance between proof size and tree depth
2. **PebbleDB**: Modern LSM implementation optimized for SSDs
3. **Append-only Storage**: Enables historic version queries
4. **Rust-inspired API**: Type-safe, explicit error handling