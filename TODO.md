# TODO - Post-Rename Session Continuation

**IMPORTANT**: This file tracks work-in-progress from pre-rename session. After renaming `~/Projects/gomion` to `~/Projects/gomion`, read this file to continue where we left off.

## Context: Where We Are

Working on **gomion** (formerly gomion/gomcfg) - a "Swiss Army Knife" CLI for Go development with integrated TUI staging editor.

### What We Just Completed (Phase 1-2)

✅ **Phase 1**: Foundation & Data Structures
- TUI entry point with cached worktree
- ChangeSet types with per-ChangeSet Git index files
- Error handling with doterr pattern
- Path management with dt package
- Git operations in gitutils (GetChangedFiles, GetWorkingDiff)

✅ **Phase 2**: Takes Generation & Loading
- gomioncfg/gompkg two-layer architecture (scalar JSON ↔ validated runtime types)
- AI prompt template (text/template + //go:embed)
- GeneratePlanTakes() with real askai.Agent integration
- CreateDefaultTake() fallback
- Caching with graceful degradation

### Critical Discovery During Implementation

**WORKFLOW FLAW IDENTIFIED**: Current plan has fundamental issues:

1. **Wrong scope**: Getting ALL repo files, need MODULE-scoped files
2. **No file filtering**: No way to exclude files developer doesn't want to commit
3. **Missing human-in-the-loop**: Need File Selection View BEFORE generating takes

## Major Decisions Made

### Architecture Decision: Filter Functions (Option #3)

**Problem**: Need to filter changed files to module scope, but gitutils shouldn't know about Go modules.

**Solution**:
- gitutils accepts a **filter func** or **iterator** (from dt.WalkDir)
- Module-specific logic lives in **gompkg** (formerly gomcfgpkg/gomionsvc)
- Keeps gitutils language-agnostic and reusable
- Future-proof for other languages

### Workflow Correction

**OLD (wrong)**:
```
Get all files → Generate takes → Select files
```

**NEW (correct)**:
```
1. Get changed files (module-scoped)
2. File Selection View - mark what NOT to include
3. Generate takes ONLY on INCLUDE files
4. Takes View
5. Hunk Assignment View
```

**Key insight**: Developer FIRST decides what NOT to include, THEN decides on commits.

### DO-NOT-INCLUDE as Special ChangeSet

**Decision**: Treat DO-NOT-INCLUDE as a special ChangeSet instead of separate concept.

**Benefits**:
- UI doesn't need special cases
- Uses same data structures
- Just ignore this ChangeSet when generating takes

**File Dispositions**:
- `INCLUDE` - Will be committed (generate takes for these)
- `DO-NOT-INCLUDE` - Special ChangeSet, skip for now
- `GITIGNORE` - Add to .gitignore
- `GITEXCLUDE` - Add to .git/info/exclude

## Project Rename: gomion/gomcfg → gomion

**Name**: gomion (Go + Minion from Despicable Me)
**Mascot**: assets/gomion-mascot.jpg

### Package Renames

```
OLD                NEW
---                ---
gomion         →   gomion
gompkg      →   gommod
gomionsvc      →   gompkg
gomioncfg      →   gomcfg
gomiontui      →   gomtui

gomioncfg      →   gomcfg (merge)
gomioncfgpkg   →   gompkg (merge)
gomioncfgtui   →   gomtui (merge)
gomionmod      →   gommod (merge)
gomioncfg      →   gomion (merge)
```

**Rationale**: Started with separate gomcfg TUI, but scope grew to encompass gomion. Unify as one project.

## Immediate Next Steps (NEW SESSION)

### 1. Verify Rename Completed ✓
- [ ] Confirm working directory is `/Users/mikeschinkel/Projects/gomion`
- [ ] Verify package names updated (gompkg → gommod, etc.)
- [ ] Check imports updated throughout codebase
- [ ] Build succeeds: `cd cmd/gomion && make build`

### 2. Unify Plan Files
- [ ] Read `cmd/gomcfg/GRU_PLAN.md` (if exists)
- [ ] Read `PLAN.md` (main plan)
- [ ] Merge GRU_PLAN.md content into PLAN.md
- [ ] Merge GRU_DONE.md content into DONE.md
- [ ] Delete GRU_PLAN.md and GRU_DONE.md after merge

### 3. Fix issues
- [ ] ParsePlanTakes() had two versions, need to resolve to one
- [ ] Parse functions in `gommod/gompkg/parse.go` should be moved to their associated type type, e.g. `ParsePlanTakes()` -> `plan_takes.go`, etc. 

### 3. Update PLAN.md with Revised Architecture

**New Phase Breakdown**:

**Phase 1**: Foundation (DONE - see DONE.md)

**Phase 2**: File Selection View (NEW - implement first)
- Two/three-pane layout:
  - Left: Tree view (github.com/Digital-Shane/treeview)
  - Right: File content display
  - Top/indicator: File disposition (INCLUDE/DO-NOT-INCLUDE/GITIGNORE/GITEXCLUDE)
- Module-scoped file list (default) with toggle to full-repo
- Auto-detect module from go.mod location
- Filter changed files to module
- Mark files with dispositions
- Persist to `.git/info/commit-files.json` (later - in-memory for now)
- Can switch back to this view anytime

**Phase 3**: Takes Generation & Loading (partial DONE)
- Generate takes ONLY on INCLUDE files
- Exclude DO-NOT-INCLUDE ChangeSet when calling AI
- Rest already implemented

**Phase 4**: Takes Exploration View (was Phase 3)
- Browse Takes and ChangeSets
- Select which Take to use
- (existing plan from old Phase 3)

**Phase 5**: Hunk Assignment View (was Phase 4)
- Assign hunks to ChangeSets
- (existing plan from old Phase 4)

### 4. Implement gitutils Filter Support

Add to `gitutils/working.go`:

```go
// FileFilter is a function that returns true if file should be included
type FileFilter func(dt.RelFilepath) bool

// GetChangedFilesFiltered returns changed files matching filter
func (r *Repo) GetChangedFilesFiltered(
    ctx context.Context,
    filter FileFilter,
) (files []dt.RelFilepath, err error)
```

### 5. Implement Module-Scoped Filtering

Add to `gompkg` (or `gomcfg`):

```go
// CreateModuleFileFilter returns a filter for files within module directory
func CreateModuleFileFilter(moduleDir dt.DirPath) gitutils.FileFilter {
    return func(file dt.RelFilepath) bool {
        // Return true if file is within moduleDir
    }
}

// AutoDetectModule finds go.mod and returns module directory
func AutoDetectModule(startDir dt.DirPath) (moduleDir dt.DirPath, err error)
```

### 6. Update File Disposition Types

Add to `gomtui/types.go`:

```go
type FileDisposition int
const (
    Include FileDisposition = iota
    DoNotInclude  // Special ChangeSet - skip for takes
    GitIgnore     // Add to .gitignore
    GitExclude    // Add to .git/info/exclude
)

type FileWithDisposition struct {
    Path        dt.RelFilepath
    Disposition FileDisposition
    Content     string  // For display in right pane
    // ... other fields
}
```

### 7. Start File Selection View UI

Create `gomtui/file_selection_view.go`:
- Integrate github.com/Digital-Shane/treeview
- Two-pane layout with BubbleTea
- File content display
- Disposition toggling with keyboard shortcuts
- Module/repo toggle

## Implementation Order

1. ✅ Verify rename
2. ✅ Unify plan files
3. ✅ Update PLAN.md
4. 🔨 Implement gitutils filter support
5. 🔨 Implement module detection and filtering
6. 🔨 Update file disposition types
7. 🔨 Build File Selection View UI
8. 🔨 Test File Selection View end-to-end
9. ⏳ Then move to Takes View (old Phase 3)

## Key Files to Reference

**Documentation**:
- `PLAN.md` - Overall plan (needs update)
- `DONE.md` - Completed work (Phase 1-2)
- `cmd/gomcfg/GRU_PLAN.md` - May have file selection details (merge this)

**Architecture Examples**:
- `gommod/gomcfg/plan_takes.go` - Scalar types for JSON
- `gommod/gompkg/plan_takes.go` - Validated runtime types
- `gommod/gompkg/plan_takes_generator.go` - AI generation
- `gommod/gitutils/working.go` - Git operations
- `gommod/askai/agent.go` - AI agent (already integrated)

**Current TUI**:
- `cmd/gomion/gommod/gomtui/types.go` - Core data structures
- `cmd/gomion/gommod/gomtui/run.go` - Entry point
- `cmd/gomion/gommod/gomtui/git_index.go` - Per-ChangeSet index

## Critical Constraints (Don't Forget!)

1. **All git operations in gitutils** - Never call git commands directly
2. **doterr error handling** - Each package has own doterr.go, use NewErr() with sentinels
3. **dt package for paths** - No .String() casting, use dt methods directly
4. **ClearPath style** - Single return, goto end, no else
5. **Scalar types in gomcfg** - NO time.Time, dt.DirPath in config types (JSON only)
6. **Parse functions** - gomcfg → gompkg conversion with validation
7. **GOEXPERIMENT=jsonv2** - Required for build

## Questions to Resolve (if needed)

1. Should module filter live in `gompkg` or `gomcfg`? (Suggest gompkg - business logic)
2. File disposition persistence format? (Suggest `.git/info/commit-files.json`)
3. Tree view library alternatives if Digital-Shane/treeview doesn't fit?

## Success Criteria for File Selection View

- [ ] Auto-detects module from go.mod
- [ ] Shows changed files scoped to module (default)
- [ ] Can toggle to show full repo files
- [ ] Tree view displays file hierarchy
- [ ] Right pane shows selected file content
- [ ] Can mark files as INCLUDE/DO-NOT-INCLUDE/GITIGNORE/GITEXCLUDE
- [ ] DO-NOT-INCLUDE treated as special ChangeSet
- [ ] Can navigate with keyboard
- [ ] Can proceed to Takes View (only INCLUDE files)
- [ ] Can return to File Selection View later

## Build & Test Commands

```bash
cd ~/Projects/gomion

# Build
cd cmd/gomion && make build

# Run
./bin/gomion

# Test (when working on specific package)
cd gommod/gompkg && GOEXPERIMENT=jsonv2 go test -v ./...
```

---

## Session Restart Instructions

When starting new session:

1. Read this TODO.md file
2. Verify all renames completed successfully
3. Unify GRU_PLAN.md with PLAN.md
4. Ask: "Should I start with updating PLAN.md or proceed directly to implementation?"
5. Follow "Immediate Next Steps" section above
