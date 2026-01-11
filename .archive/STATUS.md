# Project Status: Gomion TUI Immutability Refactor

**Date**: 2026-01-04
**Session**: Architectural decision point - reverting changes
**Plan File**: `/Users/mikeschinkel/.claude/plans/goofy-herding-bubble.md`

## Current State

**User is reverting all immutability changes** to get back to a clean baseline before deciding on architectural direction.

## What Was Being Attempted

Converting the gomion TUI codebase from mutable pointer-based architecture to immutable value-based architecture following BubbleTea's Elm architecture pattern.

### Changes That Were Made (Being Reverted)

1. **Directory** from `*Directory` → `Directory` (value type)
2. **ChangeSet** from pointer receivers → value receivers with immutable methods
3. **File.Metadata** made private (`metadata *FileMetadata`) with accessor methods
4. Attempted to convert **Node[T]** from mutable to immutable
5. Created **FileDispositionModel** for centralized dimension calculations
6. Various other immutability conversions throughout the codebase

## The Fundamental Problem Encountered

### Node[T] Recursive Structure Issue

**Go constraint**: Recursive value types are illegal (infinite size).

```go
// ❌ Won't compile
type Node struct {
    children []Node
}

// ✅ Must use pointers
type Node struct {
    children []*Node
}
```

### The Cloning Explosion Problem

When attempting to make Node[T] immutable with copy-on-write semantics:

1. **Each mutation requires cloning**: With*() methods must clone the node
2. **Recursive mutations are exponential**: Operations like `collapseAll()` on a deep tree:
   - Depth 3: ~10 clones per operation
   - Depth 5: ~100 clones
   - Depth 10: ~1000 clones
3. **Multiple With*() calls compound**: Each With*() clones children, so chaining them multiplies clones
4. **Performance unacceptable**: O(n²) or worse for tree mutations

### Type Compatibility Issues

`FileDispositionNode` and `bubbletree.Node[T]` became incompatible during refactor even though the former is derived from the latter.

## The Architectural Question

**Core dilemma**: How to reconcile immutability goals with recursive tree structures that require pointers and frequent mutations?

### Option 1: Accept Mutation in BubbleTree

**Reality**: BubbleTree (third-party library) was never designed for immutability.

**Evidence**: All BubbleTea components use mutation internally:
- `viewport.Model` - pointer receiver, mutates state
- `table.Model` - pointer receiver, mutates state
- `textinput.Model` - pointer receiver, mutates state

**Proposal**: Keep BubbleTree mutable, contain it within immutable model wrapper:
```go
type FileDispositionTreeModel struct {
    tree *bubbletree.Tree[File]  // Mutable component (like viewport)
    // ... other value fields
}

// Value receiver, returns new model (even though tree mutates internally)
func (m FileDispositionTreeModel) Update(msg tea.Msg) (FileDispositionTreeModel, tea.Cmd)
```

**Elm architecture maintained at MODEL level**, not component level.

### Option 2: Build Immutable Tree from Scratch

Replace BubbleTree entirely with custom immutable tree implementation.

**Challenges**:
- Significant engineering effort
- Need persistent data structures (structural sharing)
- Complex to implement correctly
- May still have performance issues

### Option 3: Hybrid Approach

Some types immutable (File, Directory), some mutable (Node[T], Tree).

**Distinction**:
- **Business data** (File, Directory) → value types, immutable
- **UI scaffolding** (Node, Tree) → pointer types, mutable, contained by immutable models

## Key Files Affected

### Core Types
- `gommod/gomtui/types.go` - File, Directory, FileMetadata, ChangeSet
- `gommod/bubbletree/node.go` - Node[T] generic type
- `gommod/bubbletree/tree.go` - Tree[T] management
- `gommod/gomtui/file_disposition_node.go` - FileDispositionNode wrapper

### Models
- `gommod/gomtui/file_disposition_tree_model.go` - Tree model wrapper
- `gommod/gomtui/files_table_model.go` - Directory table model
- `gommod/gomtui/file_content_model.go` - File content viewer
- `gommod/gomtui/editor_state.go` - Main editor state

### Utilities
- `gommod/gomtui/file_disposition_layout.go` - Layout dimension calculations
- `gommod/gomtui/file_metadata.go` - Metadata loading (has methods that should be on File)
- `gommod/gomtui/change_set.go` - ChangeSet operations

## Completed Work (Still Valuable)

These pieces were successfully completed and are architecturally sound:

1. ✅ **FileDispositionModel** - Centralized dimension calculations (good abstraction)
2. ✅ **ChangeSet immutability** - Successfully converted to value semantics
3. ✅ **Directory table integration** - FilesTableModel displaying directory contents
4. ✅ **File metadata caching** - Metadata loading and git status enrichment
5. ✅ **Layout vertical fill fix** - Table height properly fills pane

## What Needs Decision

**User needs to decide architectural direction**:

1. **Accept BubbleTree mutation** and focus immutability at model level only?
2. **Replace BubbleTree** with custom immutable tree (major undertaking)?
3. **Hybrid approach** - some types immutable, some mutable with clear boundaries?

## Questions for Next Session

1. What's the PRIMARY goal: Perfect immutability or pragmatic BubbleTea architecture?
2. Is BubbleTree mutation acceptable if contained within immutable model?
3. Should we focus on making business data (File) immutable and accept UI scaffolding (Node) as mutable?
4. Is the performance cost of deep cloning acceptable for any approach?

## Recommended First Step After Revert

**Before making ANY code changes**:

1. Decide on architectural philosophy (pure vs pragmatic immutability)
2. Define clear boundaries: What MUST be immutable vs what CAN be mutable
3. Choose one small type to convert as proof-of-concept
4. Validate the approach before converting everything

## Related Documentation

- **Plan file**: `/Users/mikeschinkel/.claude/plans/goofy-herding-bubble.md` - Full architectural analysis
- **BubbleTree library**: `gommod/bubbletree/` - Third-party tree component
- **Go type constraints**: Recursive value types are impossible in Go (infinite size)

## Summary

User attempted comprehensive immutability refactor, hit fundamental Go language constraints with recursive tree structures, encountered exponential cloning costs, and is now reverting to reassess architectural approach. Need to decide: pragmatic vs pure immutability, where to draw the boundaries, and whether BubbleTree mutation is acceptable within contained context.
