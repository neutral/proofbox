---
id: step.19.cli-implementation
depends_on:
tags: [cli, interface, step]
---

## Objective

Implement the `jmtcli` command-line interface tool with subcommands for tree operations, proof generation/verification, and administrative functions.

## Implements

- **CLI Basic Requirement** (`/blueprint/features/cli/basic/requirement.md`)
- Provides user-friendly interface to JMT operations
- Enables testing and debugging of tree functionality

## Implementation Steps

1. Create main CLI entry point in `cmd/jmtcli/main.go`
2. Implement core commands (init, put, get, delete)
3. Add proof commands (prove, verify)
4. Implement batch operations
5. Create interactive REPL
6. Add administrative commands (stats, export, import)
7. Write integration tests
8. Add shell completion scripts
9. Create man pages and documentation

## Performance Considerations

- Use buffered I/O for large file operations
- Implement streaming for export/import
- Cache database connections in REPL mode
- Optimize batch operations with single transaction

## Security Notes

- Validate all user inputs
- Sanitize file paths to prevent directory traversal
- Set appropriate file permissions on database
- Add option for encrypted value storage

## Done When ✓

- [ ] Main CLI structure with cobra
- [ ] Core commands: init, put, get, delete
- [ ] Proof commands: prove, verify
- [ ] Root hash and stats commands
- [ ] Batch operations from JSON file
- [ ] Interactive REPL with completion
- [ ] Export/import functionality
- [ ] Comprehensive help text
- [ ] Integration tests for all commands
- [ ] Shell completion scripts (bash, zsh)
- [ ] Performance acceptable for large operations
- [ ] Security considerations addressed
