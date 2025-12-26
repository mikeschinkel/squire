# go-tuipoc Integration Plan

## Overview

This document analyzes the relationship between the `go-tuipoc` proof-of-concept and `squire`, identifying what functionality has been migrated and what remains to be integrated.

## Executive Summary

**go-tuipoc** and **squire** serve complementary purposes:

- **go-tuipoc**: Determines the **next semver version** for a **single module** based on API changes
- **squire**: Orchestrates **release workflows** across a **multi-repo dependency graph**

**Status**: Core shared utilities have been migrated (apidiffr, gitutils, modutils), but the version determination logic remains in go-tuipoc and needs to be integrated into squire.

## What go-tuipoc Does

**Repository**: `~/Projects/go-pkgs/go-tuipoc`

**Purpose**: Analyze a single Go module to determine what semver version bump is needed based on API changes since the last release.

### Core Functionality

**Files unique to go-tuipoc** (not yet in squire):

1. **`analyzer.go`** - Main analysis orchestration
   - Coordinates the entire analysis workflow
   - Manages git repo, baseline, and API diff operations
   - Produces a final `Result` with version recommendation

2. **`baseline.go`** - Baseline determination
   - Finds the latest reachable semver tag
   - Handles cases where no baseline exists (first release)
   - Determines what to compare against

3. **`judgment.go`** - Version verdict logic
   - Analyzes API diff report
   - Classifies changes as MAJOR, MINOR, or PATCH
   - Applies semver rules to produce version recommendation

4. **`result.go`** - Result structures
   - Contains analysis output format
   - Includes verdict, baseline, status, signals

5. **`precondition.go`** - Precondition checking
   - Validates module is ready for analysis
   - Checks for required git state

6. **`review/tests.go`** - Test signal analysis
   - Detects added/modified/removed test files
   - Provides informational context for version decisions

7. **`cli.go`** - CLI interface for standalone tool

### Analysis Workflow

```
1. Initialize analyzer for module directory
2. Determine baseline (latest semver tag)
3. Check module status (in-flux dependencies, dirty state)
4. If no baseline exists → recommend v0.1.0
5. If in-flux → withhold verdict
6. Compare current API vs baseline API using apidiffr
7. Collect test change signals
8. Apply semver rules to determine verdict:
   - Breaking API changes → MAJOR bump
   - New exported symbols → MINOR bump
   - Bug fixes only → PATCH bump
9. Output verdict: "You should bump to v1.5.0"
```

### Example Output

```
Module: github.com/example/mymodule
Baseline: v1.4.2
Status: Clean (no in-flux dependencies)

API Changes:
- Breaking: Removed function ProcessData()
- Breaking: Changed signature of HandleRequest()

Verdict: MAJOR version bump required
Recommendation: v2.0.0

Reason: Detected breaking API changes
```

## What squire Does

**Repository**: `~/Projects/squire`

**Purpose**: Orchestrate releases across multiple interdependent repositories by determining which module should be processed next in the dependency graph.

### Core Functionality

**Package: `squirepkg/retinue/`**

1. **`engine.go`** - Multi-repo orchestration
   - Builds dependency graph across multiple repos
   - Determines "leaf-most" module (ready to release)
   - Checks git status and remote tag sync
   - Reports what module to process next

2. **`go_mod_graph.go`** - Dependency graph builder
   - Parses go.mod files across workspace
   - Maps module dependencies
   - Identifies local vs external dependencies

3. **`go_module.go`** - Module analysis
   - Checks if module is "in-flux" (not ready for release)
   - Validates no pseudo-versions in dependencies
   - Validates no replace directives
   - Checks git dirty state (excluding submodules)

4. **`repo.go`** - Repository management
   - Git repository operations
   - Remote tracking
   - Tag management

### Orchestration Workflow

```
1. Scan workspace for all go.mod files
2. Build dependency graph
3. Find leaf-most module (no in-flux dependencies)
4. Check module git status:
   - If missing remote tags → fetch them
   - If dirty → show interactive menu to resolve
   - If ahead of upstream → prompt to push
5. Output: "Process ~/Projects/go-dt/dtx next"
6. User can then run version determination (future: integrated)
7. User tags and releases
8. Repeat for next module in dependency order
```

### Example Output

```
Analyzing dependency graph:

Dependent repo:
- Dir:    ~/Projects/xmlui/cli
- Branch: main
- Remote: origin

Leaf-most dependency found:
- Module: ~/Projects/go-pkgs/go-dt/dtx/go.mod
- Repo:   ~/Projects/go-pkgs/go-dt
- Branch: main
- Remote: origin
- Status: Clean
- Verdict: Ready to process
- Reason:  No in-flux dependencies, clean working tree
```

## Shared Components (Already Migrated)

These packages exist in both go-tuipoc and squire:

### 1. apidiffr (API Diffing)
**Location in squire**: `squirepkg/apidiffr/`

**Purpose**: Compare Go package APIs between two versions and detect breaking changes

**Files**:
- `diff.go` - API comparison logic
- `report.go` - Diff report formatting
- `doterr.go` - Error handling

**Status**: ✅ Fully migrated to squire

### 2. gitutils (Git Operations)
**Location in squire**: `squirepkg/gitutils/`

**Purpose**: Git repository operations (tags, status, remotes, etc.)

**Files**:
- `repo.go` - Repository management
- `upstream_state.go` - Remote tracking state
- `errors.go` - Sentinel errors
- `doterr.go` - Error handling
- `dev_unix.go`, `dev_other.go` - Platform-specific helpers

**Status**: ✅ Fully migrated to squire

### 3. modutils (Module Utilities)
**Location in squire**: `squirepkg/modutils/`

**Purpose**: go.mod file parsing and dependency analysis

**Files**:
- `module.go` - Module loading and parsing
- `deps.go` - Dependency analysis
- `require.go` - Require directive handling
- `replace.go` - Replace directive handling
- `versions.go` - Version parsing and validation
- `path_version.go` - Module path/version utilities

**Status**: ✅ Fully migrated to squire

## What Remains in go-tuipoc

### Core Version Determination Logic

These components are **unique to go-tuipoc** and provide the "what version should this be?" functionality:

1. **Analyzer (`analyzer.go`)**
   - Orchestrates the entire version analysis
   - Integrates baseline, status, API diff, and test review
   - Produces version recommendation

2. **Baseline (`baseline.go`)**
   - Finds latest semver tag to compare against
   - Handles first-release scenarios
   - Manages cached worktree for comparison

3. **Judgment (`judgment.go`)**
   - Applies semver rules to API changes
   - Classifies changes as MAJOR/MINOR/PATCH
   - Produces human-readable verdict

4. **Result (`result.go`)**
   - Structures analysis output
   - Includes verdict, reasoning, baseline info

5. **Test Review (`review/tests.go`)**
   - Detects test file changes
   - Provides informational signals for version decisions

6. **Preconditions (`precondition.go`)**
   - Validates module is ready for analysis

## Integration Strategy

### Phase 1: Copy Core Logic (High Priority)

**Goal**: Bring version determination into squire

**Tasks**:
1. Copy `analyzer.go` → `squirepkg/tuipoc/analyzer.go`
2. Copy `baseline.go` → `squirepkg/tuipoc/baseline.go`
3. Copy `judgment.go` → `squirepkg/tuipoc/judgment.go`
4. Copy `result.go` → `squirepkg/tuipoc/result.go`
5. Copy `precondition.go` → `squirepkg/tuipoc/precondition.go`
6. Copy `review/tests.go` → `squirepkg/tuipoc/review/tests.go`
7. Update imports to use squire's existing apidiffr, gitutils, modutils

**Integration Point**:
- Add `AnalyzeVersion()` function that squire's engine can call
- When engine determines "process this module next", also determine "bump to version X.Y.Z"

### Phase 2: Integrate with squire process Command (High Priority)

**Goal**: Show version recommendation in `squire process` output

**Current Output**:
```
Leaf-most dependency found:
- Module: ~/Projects/go-dt/dtx/go.mod
- Status: Clean
- Verdict: Ready to process
```

**Enhanced Output**:
```
Leaf-most dependency found:
- Module: ~/Projects/go-dt/dtx/go.mod
- Status: Clean
- Verdict: Ready to release
- Current Version: v0.4.0
- Recommended Version: v0.5.0 (MINOR bump)
- Reason: Added 3 new exported functions
```

**Implementation**:
- Modify `retinue.Engine.Run()` to call `tuipoc.AnalyzeVersion()`
- Add version fields to `retinue.EngineResult`
- Display version recommendation in `process_cmd.go`

### Phase 3: Add squire version Command (Medium Priority)

**Goal**: Provide standalone version analysis command

**Command**:
```bash
squire version [module-dir]
```

**Output**:
```
Analyzing module: ~/Projects/go-dt/dtx

Baseline: v0.4.0 (git tag, 2024-12-15)
Current HEAD: abc123f

API Changes Since v0.4.0:
✅ Added: func NewValidator() *Validator
✅ Added: func (v *Validator) Validate() error
✅ Added: type ValidationError struct

Test Changes:
📝 Modified: validator_test.go (+45 lines)
📝 Added: validation_integration_test.go

Status: Clean (no in-flux dependencies)

Verdict: MINOR version bump required
Recommended: v0.5.0

Reason: Added new exported symbols, no breaking changes
```

**Implementation**:
- Create `squirepkg/squirecmds/version_cmd.go`
- Wrap `tuipoc.AnalyzeVersion()` with CLI interface
- Add formatting for human-readable output
- Support `--json` flag for machine-readable output

### Phase 4: Stability Contract Validation (Future)

**Goal**: Implement the stability level validation from go-tuipoc PLAN.md

This is the extensive feature set described in go-tuipoc's PLAN.md for validating Contract: annotations and time-based stability guarantees.

**Status**: Deferred until core version determination is integrated

See go-tuipoc's PLAN.md for detailed specification.

## Key Differences in Purpose

| Aspect | go-tuipoc | squire |
|--------|-----------|--------|
| **Scope** | Single module | Multi-repo workspace |
| **Question** | What version? | Which module next? |
| **Input** | One module directory | Dependency graph |
| **Output** | Version recommendation | Module processing order |
| **Use Case** | "Should I bump to v2.0.0?" | "Which of my 10 modules should I release first?" |
| **Workflow** | Analysis → Decision | Orchestration → Coordination |

Both are needed for a complete release workflow:
1. **squire** determines **which** module to release next
2. **tuipoc** (integrated) determines **what version** to use
3. User tags and releases
4. Repeat

## Dependency Flow

```
go-tuipoc (standalone)                 squire (production)
├── analyzer.go ─────────────────┐     ├── retinue/
│   ├── baseline.go              │     │   ├── engine.go ←──── NEEDS INTEGRATION
│   ├── judgment.go              │     │   ├── go_mod_graph.go
│   └── result.go                │     │   └── go_module.go
├── review/tests.go              │     │
├── precondition.go              │     ├── apidiffr/ ←────── ✅ MIGRATED
├── apidiffr/ ───────────────────┼─────┤
├── gitutils/ ───────────────────┼─────├── gitutils/ ←────── ✅ MIGRATED
└── modutils/ ───────────────────┼─────└── modutils/ ←────── ✅ MIGRATED
                                 │
                                 └─── TO BE INTEGRATED
```

## Testing Strategy

### Phase 1 Testing
- Unit tests for analyzer, baseline, judgment
- Integration tests with real git repos
- Verify version recommendations match go-tuipoc

### Phase 2 Testing
- Test `squire process` shows correct version recommendations
- Test with modules that have no baseline (v0.1.0)
- Test with modules that have breaking changes (MAJOR)
- Test with modules that are in-flux (withhold verdict)

### Phase 3 Testing
- Test `squire version` command standalone
- Test JSON output format
- Test error handling for invalid modules

## Migration Checklist

### Immediate (Phase 1)
- [ ] Create `squirepkg/tuipoc/` package
- [ ] Copy analyzer.go and adapt imports
- [ ] Copy baseline.go and adapt imports
- [ ] Copy judgment.go and adapt imports
- [ ] Copy result.go and adapt imports
- [ ] Copy precondition.go and adapt imports
- [ ] Create `squirepkg/tuipoc/review/` package
- [ ] Copy review/tests.go and adapt imports
- [ ] Write unit tests for migrated code
- [ ] Write integration tests

### Short Term (Phase 2)
- [ ] Add version analysis to retinue.Engine
- [ ] Update retinue.EngineResult with version fields
- [ ] Update process_cmd.go to display version recommendation
- [ ] Test integrated workflow
- [ ] Update documentation

### Medium Term (Phase 3)
- [ ] Create squire version command
- [ ] Implement human-readable output
- [ ] Implement JSON output
- [ ] Add to squire help documentation

### Future (Phase 4)
- [ ] Implement stability contract validation
- [ ] See go-tuipoc PLAN.md for detailed tasks

## go-tuipoc Repository Fate

Once the core version determination logic is integrated into squire, the go-tuipoc repository can be:

**Option 1**: Archive it
- Mark as archived on GitHub
- Reference in squire docs as "proof-of-concept that became squire's version analysis"

**Option 2**: Keep as experimental ground
- Continue using for prototyping new version analysis features
- Migrate proven features to squire when ready

**Option 3**: Delete it
- Core value has been extracted
- No longer needed

**Recommendation**: Archive it with a clear README pointing to squire for the production version.

## References

- go-tuipoc: `~/Projects/go-pkgs/go-tuipoc`
- squire: `~/Projects/squire`
- Semver specification: https://semver.org/
- Go modules documentation: https://go.dev/ref/mod
