# Development Rules and Conventions

## Core Development Rules

1. **Consult Blueprint First**
   - Always check `/blueprint/` for requirements before implementing features
   - Blueprint is the authoritative source of truth

2. **No Simultaneous Updates**
   - Never update blueprint and codebase in the same session
   - Complete blueprint changes first, then implement

3. **Documentation Requirements**
   - Source files require both code comments and `.desc.md` description files
   - Keep documentation concise and focused on non-obvious aspects

4. **Active Step Tracking**
   - Mark blueprint steps with `.active.md` suffix when working on them
   - Only one step should be active at a time

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