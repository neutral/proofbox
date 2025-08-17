# Development Rules and Conventions

## Core Development Rules

1. **Consult Blueprint First**

   - Always check `_blueprint/` folder for requirements when implementing features
   - Blueprint is the authoritative source of truth
   - Write blueprint first before implementing the source code
   - Always verify implemented source code against blueprint requirements and specifications
   - Capture blueprint specifications and log decisions in the global `pkg/_blueprint` folder

2. **Documentation Requirements**

   - Source files require both code comments and `.desc.md` description files
   - Keep documentation concise and focused on non-obvious aspects
   - Description files (.desc.md) provide the "why" behind the code, complementing the "what" in the source files and making the codebase more maintainable and understandable for future developers. Tests do not need to be documented with description files.

3. **fmt and lint at every step**

   - Always run fmt after generating code
   - Always run lint before completing the implementation step

## Code Conventions

### Go Standards

- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Keep functions focused and testable
- Handle errors explicitly

### Testing Strategy

- Write unit tests for all core components
- Include benchmark tests for performance-critical code
- Integration tests for PebbleDB interactions
- Proof verification tests against reference implementations

### Git Workflow

- Clear, descriptive commit messages
- Reference blueprint step IDs in commits when applicable
- Keep commits focused on single concerns

## Blueprint File Conventions

All blueprint files use YAML front-matter:

```yaml
id: unique-identifier
status: draft|active|deprecated
depends_on: [list, of, dependencies]
tags: [relevant, labels]
```

## Performance Considerations

- Profile before optimizing
- Document performance-critical sections
- Maintain benchmarks for regression detection
- Target specific performance goals from NFRs
