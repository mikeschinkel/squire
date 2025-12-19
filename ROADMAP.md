# Squire Roadmap

This document tracks planned features and enhancements for Squire.

## Status Key

- 🔴 **Not Started** - Feature planned but not yet begun
- 🟡 **In Progress** - Feature currently being implemented
- 🟢 **Completed** - Feature implemented and merged

---

## Planned Features

### External Module Dependencies (`--all` flag for `requires-tree`)

**Status:** 🔴 Not Started
**Priority:** Medium
**Related Command:** `squire requires-tree`

**Description:**

Add support for the `--all` flag to the `requires-tree` command to include external (non-Squire-managed) modules in the dependency tree visualization.

**Current Behavior:**

The tree currently only shows Squire-managed modules (those discovered via `DiscoverModules` from `.squire/config.json`). External dependencies are not included in the visualization.

**Desired Behavior:**

When `--all` is specified:
- Include external Go modules that are required by Squire-managed modules
- Show the complete dependency tree, not just internal modules
- Use `go list -m -json` or similar tooling to discover external module metadata
- Handle cases where external module information is unavailable gracefully

**Implementation Notes:**

1. Extend `retinue.ModuleSet` to support external modules
2. Add discovery logic to query external dependencies via Go toolchain
3. Consider depth limits to avoid overly deep trees
4. Handle network/proxy issues when resolving external modules
5. Optionally cache external module metadata for performance

**Acceptance Criteria:**

- `squire requires-tree . --all` includes both internal and external modules
- External modules are visually distinguishable (or documented as such)
- Command handles missing/unreachable external modules without crashing
- Performance is acceptable for typical Go projects (hundreds of dependencies)

**Related Documentation:**

- PRD: `docs/squire-cli-tree-command-prd.md` (Section 6.2)
- Phase 2 PRD: `docs/squire-phase-2-prd.md`

---

## Completed Features

### Module Discovery & Dependency Ordering

**Status:** 🟢 Completed
**Completed:** Phase 2

- Module discovery from `.squire/config.json`
- Dependency-safe ordering via topological sort
- Module classification (lib/cmd/test)
- Versioned vs non-versioned heuristics

### Basic Dependency Tree Visualization

**Status:** 🟢 Completed
**Completed:** 2025-12-11

- ASCII tree rendering of internal module dependencies
- `--show-dirs` flag for directory-based labels
- `--show-all` flag for module path + location
- Markdown embedding with `--embed`, `--before`, `--after` flags
- Flag validation for mutual exclusivity

---

## Future Considerations

These are ideas for future exploration, not yet committed to the roadmap:

- **Release Planning & Tagging** - Automated versioning and git tag management
- **Changelog Generation** - Automatic changelog creation from commit history
- **Multi-Repo Orchestration** - Commands that operate across multiple related repositories
- **ClearPath Linter** - Custom linting rules for ClearPath coding style
- **GitHub Workflow Management** - Ensure standard workflows across all repos
- **GoReleaser Integration** - Scaffolding and automation for compiled binaries
- **TUI Mode** - Interactive terminal UI for complex operations
