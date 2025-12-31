# ADR 001: Use "Requires" Terminology

## Status

Accepted

## Context

Gomion manages dependencies between Go modules across multiple repositories. We needed to establish consistent terminology throughout the codebase, configuration files, CLI commands, and documentation for referring to these relationships.

Three primary options were considered:
1. **"dependencies"** or **"deps"** - Common in many ecosystems
2. **"requires"** - Aligned with Go's `go.mod` syntax
3. **"imports"** - Focused on code-level imports

### Why This Matters

Inconsistent terminology creates confusion:
- Users must remember different terms in different contexts
- Developers may use different terms in code vs. documentation
- Configuration schemas become harder to understand
- CLI commands lose intuitive discoverability

### Go Ecosystem Alignment

Go's module system uses `require` directives in `go.mod` files:
```go
module github.com/example/mymodule

require (
    github.com/mikeschinkel/go-dt v0.3.3
    github.com/mikeschinkel/go-cliutil v0.3.0
)
```

This established precedent in the Go ecosystem makes "requires" the most natural choice.

## Decision

**Use "requires" terminology consistently across all Gomion code, configuration, and documentation.**

This applies to:
- JSON configuration field names: `requires`
- Go type names: `RepoRequirement`, `Requires`
- CLI command namespaces: `gomion requires tree`
- Documentation and user-facing messages
- Internal code comments

## Consequences

### Positive

1. **Alignment with Go**: Matches Go's native `require` directive syntax
2. **Clarity**: Distinct from code-level "imports" and generic "dependencies"
3. **Consistency**: Single term used everywhere reduces cognitive load
4. **Discoverability**: Users familiar with Go will find this terminology intuitive

### Negative

1. **Migration**: Existing code using "dependencies" or "deps" must be updated
2. **Length**: "requires" is longer than "deps" (though more explicit)

### Implementation Requirements

All occurrences must use "requires":

**Configuration Files:**
```json
{
  "modules": { ... },
  "requires": [
    {"path": "~/Projects/go-pkgs/go-dt"}
  ]
}
```

**Go Types:**
```go
type RepoRequirement struct {
    Path string `json:"path"`
}

type Module struct {
    Requires []string
}
```

**CLI Commands:**
```bash
gomion requires tree
gomion requires list
gomion update  # updates requires field
```

**NOT:**
```bash
gomion deps tree     # ✗ Wrong
gomion dependencies  # ✗ Wrong
```

### Notes

- This decision was made during Phase 2 implementation of multi-repo module tree visualization
- The term "requires" specifically refers to **repository-level dependencies** that Gomion manages
- This is distinct from Go module dependencies (which may include third-party packages Gomion doesn't manage)
- "Requires" in Gomion context means: "repositories containing modules that this repository's modules depend on"

## References

- Go Modules Reference: https://go.dev/ref/mod
- Plan: `/Users/mikeschinkel/.claude/plans/curious-toasting-rabin.md`
- Implementation: Phase 2 of multi-repo module tree visualization feature
