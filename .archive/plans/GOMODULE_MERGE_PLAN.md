# GoModule Consolidation Plan

## Problem Statement

We have two separate types representing the same concept:
- `gomodutils.Module` - Simple, read-only go.mod parser and analyzer
- `retinue.GoModule` - Rich, graph-aware module with git integration

Both model a Go module (go.mod file), but `retinue.GoModule` is significantly more feature-rich. Following the "one source of truth" principle, we should consolidate these into a single unified type in `gomodutils`.

## Design Decisions

1. ✅ **Naming:** `gomodutils.Module` (descriptive in context)
2. ✅ **Graph rename:** `GoModGraph` → `Graph` (simpler in gomodutils context)
3. ✅ **Graph optional:** Module works without graph for basic parsing/analysis
4. ✅ **Wrapper:** `retinue.ModuleExt` for Gomion-specific release logic
5. ✅ **Location:** Graph and Repo go in gomodutils package
6. ✅ **Priority:** Complete BEFORE PRE-commit work begins

## Analysis Summary

### gomodutils.Module (Current State)

**Strengths:**
- Clean, simple API
- Focused on parsing and analysis
- Helper types: `PathVersion`, `Require`, `Replace`, `ModulePath`
- Package-level `AnalyzeStatus()` function for dependency analysis
- Read-only by design

**Weaknesses:**
- No graph awareness
- No git integration
- No repository association
- No monorepo support
- Must create separate instances for analysis (wasteful)

**API Surface:**
```go
type Module struct {
    Path     ModulePath
    Filepath dt.Filepath
    Requires []Require
    Replaces []Replace
}

func NewModule(goModPath dt.Filepath) Module
func (m *Module) Load() error
func (m *Module) LooksPublishable() bool
func AnalyzeStatus(m Module) Status  // Package-level function
```

### retinue.GoModule (Current State)

**Strengths:**
- Graph-aware via `SetGraph()` and `RequireDirs()`
- Git integration via `IsInFlux()` for release readiness
- Repository association via `Repo()`
- Monorepo support (excludes sibling modules from git status)
- Embeds `modfile.File` for full parsed structure access
- Enforces initialization order via panic guards

**Weaknesses:**
- More complex initialization (`NewGoModule()` → `SetGraph()` → `Load()`)
- Wraps `gomodutils.AnalyzeStatus()` instead of having it built-in
- Has TODO comments about moving AnalyzeStatus to the type itself

**API Surface:**
```go
type GoModule struct {
    ModuleFile ModuleFile
    Graph      *Graph
    repo       *Repo
    *parsedModFile              // Embedded modfile.File
    loaded     bool
}

func NewGoModule(modFile dt.Filepath) *GoModule
func (m *GoModule) SetGraph(graph *Graph) error
func (m *GoModule) Load() error
func (m *GoModule) ModulePath() ModulePath
func (m *GoModule) Dir() dt.DirPath
func (m *GoModule) Key() ModuleKey
func (m *GoModule) Repo() *Repo
func (m *GoModule) Requires() []ModulePath
func (m *GoModule) RequireDirs() []ModuleDir
func (m *GoModule) AnalyzeStatus() (gomodutils.Status, error)
func (m *GoModule) IsInFlux(ctx context.Context) (bool, string, error)
func (m *GoModule) HasReplaceDirectives() bool
```

## Consolidation Strategy

### Goal
Create a unified `gomodutils.Module` that:
1. Has the rich functionality of `retinue.GoModule`
2. Keeps the clean helper types from `gomodutils`
3. Makes graph-awareness **optional** (works standalone or with graph)
4. Moves Gomion-specific logic to `retinue.ModuleExt` wrapper

### Phase 1: Prepare gomodutils Package

#### 1.1 Add New Fields to gomodutils.Module

**File:** `gompkg/gomodutils/module.go`

**Current:**
```go
type Module struct {
    Path     ModulePath
    Filepath dt.Filepath
    Requires []Require
    Replaces []Replace
}
```

**New (merged):**
```go
type Module struct {
    // Core fields (from current goutils)
    Path     ModulePath          // Module path from go.mod
    Filepath dt.Filepath         // Path to go.mod file
    Requires []Require           // Required dependencies
    Replaces []Replace           // Replace directives

    // Graph integration (optional - from gompkg)
    Graph    *Graph              // Optional: dependency graph (nil if standalone)
    repo     *Repo               // Optional: parent repository (nil if standalone)

    // Parsed structure (from gompkg)
    modfile  *modfile.File       // Parsed go.mod structure
    loaded   bool                // Whether Load() has been called
}
```

**Key Design:** Graph and repo are **optional** - Module works standalone or graph-aware.

**Functionality Matrix:**
| Method | Requires Load | Requires Graph | Purpose |
|--------|---------------|----------------|---------|
| `Load()` | - | - | Parse go.mod |
| `Path`, `Filepath` | ✓ | - | Basic metadata |
| `Requires`, `Replaces` | ✓ | - | Dependencies |
| `AnalyzeStatus()` | ✓ | - | Dependency analysis |
| `Dir()`, `Key()` | - | - | Path operations |
| `RequireDirs()` | ✓ | ✓ | Local dependencies |
| `Repo()` | - | ✓ | Repository access |

#### 1.2 Update Constructor

**New signature:**
```go
func NewModule(goModPath dt.Filepath) *Module  // Returns pointer now (was value)
```

**Implementation:**
```go
func NewModule(goModPath dt.Filepath) *Module {
    return &Module{
        Filepath: goModPath,
        loaded:   false,
        Graph:    nil,  // Optional
        repo:     nil,  // Optional
    }
}
```

**Rationale:** Return pointer for consistency with retinue pattern and to support optional graph/repo assignment.

#### 1.3 Add SetGraph() Method

**New method:**
```go
func (m *Module) SetGraph(graph *Graph) error {
    var err error

    m.Graph = graph

    // Look up repo from graph
    m.repo, err = graph.repoForModule(m)
    if err != nil {
        goto end
    }

end:
    return err
}
```

**Behavior:**
- Sets `m.Graph` reference
- Looks up and sets `m.repo` from graph
- **Optional** - can be skipped for standalone usage
- Returns error if module dir not found in graph

#### 1.4 Update Load() Method

**Current behavior:** Populates `Path`, `Requires`, `Replaces` from parsing.

**New behavior:** Also stores `modfile.File` in `m.modfile` for full access.

**Implementation:**
```go
func (m *Module) Load() error {
    var content []byte
    var mf *modfile.File
    var err error

    // Read file
    content, err = m.Filepath.ReadFile()
    if err != nil {
        goto end
    }

    // Parse (syntax-only validation - intentional for local dev)
    mf, err = modfile.ParseLax(m.Filepath.String(), content)
    if err != nil {
        goto end
    }

    // Store parsed structure
    m.modfile = mf

    // Extract module path
    if mf.Module == nil || mf.Module.Mod.Path == "" {
        panic(ErrGoModuleNameNotParsed)
    }
    m.Path = ModulePath(mf.Module.Mod.Path)

    // Extract requires
    m.Requires = make([]Require, len(mf.Require))
    for i, req := range mf.Require {
        m.Requires[i] = NewRequire(
            NewPathVersion(ModulePath(req.Mod.Path), dt.Version(req.Mod.Version)),
            req.Indirect,
        )
    }

    // Extract replaces
    m.Replaces = make([]Replace, len(mf.Replace))
    for i, rep := range mf.Replace {
        m.Replaces[i] = NewReplace(
            NewPathVersion(ModulePath(rep.Old.Path), dt.Version(rep.Old.Version)),
            NewPathVersion(ModulePath(rep.New.Path), dt.Version(rep.New.Version)),
        )
    }

    m.loaded = true

end:
    return err
}
```

#### 1.5 Move AnalyzeStatus to Method

**Current:** Package-level function `func AnalyzeStatus(m Module) Status`

**New:** Method `func (m *Module) AnalyzeStatus() (Status, error)`

**File:** `gompkg/gomodutils/deps.go`

**Rationale:**
- Aligns with TODO comment in retinue code
- More intuitive API
- Module already loaded, no need to pass it
- **Does NOT require graph** - works standalone

**Implementation:**
```go
func (m *Module) AnalyzeStatus() (Status, error) {
    m.chkLoaded("AnalyzeStatus")

    // Build replaces map for efficient lookup
    replaces := make(replacesMap)
    for _, rep := range m.Replaces {
        replaces[rep.Old.String()] = rep
        replaces[rep.Old.PathAt()] = rep
    }

    // Analyze each direct dependency
    var deps []DependencyState
    var inFlux bool

    for _, req := range m.Requires {
        if req.Indirect {
            continue  // Skip indirect dependencies
        }

        state, isInFlux := analyzeRequire(req, replaces)
        deps = append(deps, state)
        if isInFlux {
            inFlux = true
        }
    }

    return Status{Deps: deps, InFlux: inFlux}, nil
}
```

#### 1.6 Add Methods from retinue.GoModule

**New methods to add to gomodutils.Module:**

**File:** `gompkg/gomodutils/module.go`

```go
// Directory access - NO GRAPH REQUIRED
func (m *Module) Dir() dt.DirPath {
    return m.Filepath.Dir()
}

// Unique key for deduplication - NO GRAPH REQUIRED
func (m *Module) Key() ModuleKey {
    return ModuleKey(fmt.Sprintf("%s:%s", m.Path, m.Filepath))
}

// Repository access - REQUIRES GRAPH
func (m *Module) Repo() *Repo {
    m.chkSetGraph("Repo")
    return m.repo
}

// Dependency paths - NO GRAPH REQUIRED
func (m *Module) RequiredModulePaths() []ModulePath {
    m.chkLoaded("RequiredModulePaths")
    paths := make([]ModulePath, len(m.Requires))
    for i, req := range m.Requires {
        paths[i] = req.Path
    }
    return paths
}

// Alias for RequiredModulePaths - NO GRAPH REQUIRED
func (m *Module) Requires() []ModulePath {
    return m.RequiredModulePaths()
}

// Graph-aware local dependency directories - REQUIRES GRAPH
func (m *Module) RequireDirs() []ModuleDir {
    m.chkLoaded("RequireDirs")
    m.chkSetGraph("RequireDirs")

    var dirs []ModuleDir
    for _, req := range m.Requires {
        // Only include dependencies found in graph (local modules)
        if mod, ok := m.Graph.modulesByPath[req.Path]; ok {
            dirs = append(dirs, mod.Dir())
        }
    }
    return dirs
}

// Check for replace directives - NO GRAPH REQUIRED
func (m *Module) HasReplaceDirectives() bool {
    m.chkLoaded("HasReplaceDirectives")
    return len(m.Replaces) > 0
}

// Guard function - ensures Load() was called
func (m *Module) chkLoaded(funcName string) {
    if !m.loaded {
        panic(fmt.Sprintf("ERROR: Must call Module.Load() before Module.%s()", funcName))
    }
}

// Guard function - ensures SetGraph() was called
func (m *Module) chkSetGraph(funcName string) {
    if m.Graph == nil {
        panic(fmt.Sprintf("ERROR: Must call Module.SetGraph() before calling Module.%s()", funcName))
    }
}
```

#### 1.7 Add Helper Types from retinue

**File:** `gompkg/gomodutils/types.go` (new file)

```go
package gomodutils

import "github.com/mikeschinkel/go-dt"

// Type aliases for clarity and compatibility
type ModuleFile = dt.Filepath   // Path to go.mod file
type ModuleKey string            // Unique identifier: "{path}:{file}"
type ModuleDir = dt.DirPath      // Directory containing go.mod
type RepoDir = dt.DirPath        // Git repository root directory
```

### Phase 2: Move Graph and Repo to gomodutils

Currently `GoModGraph` and `Repo` are in retinue package, but they're generic go.mod functionality.

#### 2.1 Create gomodutils/graph.go

**Move from:** `gompkg/retinue/go_mod_graph.go`
**Move to:** `gompkg/gomodutils/graph.go`

**Key changes:**
1. Rename `GoModGraph` → `Graph`
2. Change `*GoModule` → `*Module`
3. Update package from `retinue` to `gomodutils`
4. Update all method receivers and signatures

**Example transformation:**
```go
// Before (gompkg)
type GoModGraph struct {
    modules        []*GoModule
    modulesByPath  map[ModulePath]*GoModule
    // ...
}

func NewGoModuleGraph() *GoModGraph {
    return &GoModGraph{
        modules:       make([]*GoModule, 0),
        modulesByPath: make(map[ModulePath]*GoModule),
        // ...
    }
}

// After (goutils)
type Graph struct {
    modules        []*Module
    modulesByPath  map[ModulePath]*Module
    // ...
}

func NewGraph() *Graph {
    return &Graph{
        modules:       make([]*Module, 0),
        modulesByPath: make(map[ModulePath]*Module),
        // ...
    }
}
```

#### 2.2 Create gomodutils/repo.go

**Move from:** `gompkg/retinue/repo.go`
**Move to:** `gompkg/gomodutils/repo.go`

**Key changes:**
1. Change `*GoModule` → `*Module`
2. Update package from `retinue` to `gomodutils`
3. Update all method receivers and signatures

**Example transformation:**
```go
// Before (gompkg)
type Repo struct {
    Dir     RepoDir
    modules []*GoModule
}

func (r *Repo) Modules() []*GoModule {
    return r.modules
}

// After (goutils)
type Repo struct {
    Dir     RepoDir
    modules []*Module
}

func (r *Repo) Modules() []*Module {
    return r.modules
}
```

### Phase 3: Update retinue Package

#### 3.1 Remove Duplicate Files

**Delete these files:**
- `gompkg/retinue/go_module.go` - Module now in gomodutils
- `gompkg/retinue/go_mod_graph.go` - Graph now in gomodutils
- `gompkg/retinue/repo.go` - Repo now in gomodutils

#### 3.2 Create Gomion-Specific Wrapper

**File:** `gompkg/retinue/module_ext.go` (new file)

```go
package retinue

import (
    "context"
    "fmt"

    "github.com/mikeschinkel/go-dt"
    "github.com/mikeschinkel/gomion/gompkg/gomodutils"
    "github.com/mikeschinkel/gomion/gompkg/gitutils"
)

// ModuleExt extends goutils.Module with Gomion-specific release logic
type ModuleExt struct {
    *gomodutils.Module
}

// NewModuleExt wraps a goutils.Module with Gomion-specific functionality
func NewModuleExt(m *gomodutils.Module) *ModuleExt {
    return &ModuleExt{Module: m}
}

// IsInFlux checks if module is ready for release (Gomion-specific)
// Returns: (inFlux bool, reason string, error)
func (m *ModuleExt) IsInFlux(ctx context.Context) (bool, string, error) {
    var err error
    var status gomodutils.Status
    var repo *gitutils.Repo
    var counts gitutils.StatusCounts

    // Check dependency status
    status, err = m.AnalyzeStatus()
    if err != nil {
        goto end
    }
    if status.InFlux {
        return true, "has in-flux dependencies", nil
    }

    // Check git status
    repo, err = gitutils.Open(m.Dir())
    if err != nil {
        goto end
    }

    excludePaths := m.getSubmodulePathsToExclude()
    counts, err = repo.StatusCountsInPathExcluding(ctx, m.Dir(), excludePaths)
    if err != nil {
        goto end
    }

    if counts.IsDirty() {
        return true, "dirty working tree", nil
    }

    // Check replace directives
    if m.HasReplaceDirectives() {
        return true, "has replace directives", nil
    }

end:
    return false, "", err
}

// getSubmodulePathsToExclude returns paths of sibling modules in same repo
// Used to exclude them from git status checks (monorepo support)
func (m *ModuleExt) getSubmodulePathsToExclude() []dt.PathSegments {
    if m.Graph == nil {
        return nil
    }

    repo := m.Repo()
    if repo == nil {
        return nil
    }

    var paths []dt.PathSegments
    for _, mod := range repo.Modules() {
        if mod.Dir() != m.Dir() {
            // Get relative path from repo root to module dir
            relPath := mod.Dir().RelativeTo(repo.Dir())
            paths = append(paths, relPath.Segments())
        }
    }
    return paths
}
```

**Rationale:**
- `IsInFlux()` requires git integration and is Gomion-specific for release planning
- Keep it in retinue as an extension to avoid coupling gomodutils with gitutils
- ModuleExt is a thin wrapper - delegates most work to embedded Module

#### 3.3 Update retinue Imports and Types

**Update all files in retinue package:**

**Before:**
```go
import "github.com/mikeschinkel/gomion/gommod/gompkg"

m := retinue.NewGoModule(filepath)
m.SetGraph(graph)
m.Load()

graph := retinue.NewGoModuleGraph()
```

**After:**
```go
import "github.com/mikeschinkel/gomion/gommod/goutils"
import "github.com/mikeschinkel/gomion/gommod/gompkg"

m := gomodutils.NewModule(filepath)
m.SetGraph(graph)
m.Load()

graph := gomodutils.NewGraph()

// If using IsInFlux()
ext := retinue.NewModuleExt(m)
inFlux, reason, err := ext.IsInFlux(ctx)
```

**Files to update in retinue:**
- `engine.go` - ReleaseEngine uses Graph and Module
- `verdict.go` - Uses Module for API diff
- Any other files using GoModule, GoModGraph, or Repo

### Phase 4: Update All Other References

#### 4.1 Find All References

**Search commands:**
```bash
# Find GoModule references
grep -r "GoModule" gommod/ --include="*.go"

# Find GoModGraph references
grep -r "GoModGraph" gommod/ --include="*.go"

# Find gompkg imports
grep -r "gompkg/retinue" gommod/ --include="*.go"
```

#### 4.2 Update Type References

**Replace throughout codebase:**
- `retinue.GoModule` → `gomodutils.Module`
- `retinue.GoModGraph` → `gomodutils.Graph`
- `retinue.Repo` → `gomodutils.Repo`
- `retinue.NewGoModule()` → `gomodutils.NewModule()`
- `retinue.NewGoModuleGraph()` → `gomodutils.NewGraph()`

**Add imports:**
```go
import "github.com/mikeschinkel/gomion/gommod/goutils"
```

**Wrap with ModuleExt where needed:**
```go
// If IsInFlux() is called
import "github.com/mikeschinkel/gomion/gommod/gompkg"

ext := retinue.NewModuleExt(module)
inFlux, reason, err := ext.IsInFlux(ctx)
```

### Phase 5: Update Tests

#### 5.1 Move Tests to gomodutils

**Move files:**
- `gompkg/retinue/go_module_test.go` → `gompkg/gomodutils/module_test.go`
- `gompkg/retinue/go_mod_graph_test.go` → `gompkg/gomodutils/graph_test.go`
- `gompkg/retinue/repo_test.go` → `gompkg/gomodutils/repo_test.go`

**Update test code:**
```go
// Before
package retinue

func TestGoModule_Load(t *testing.T) {
    m := NewGoModule(testPath)
    err := m.Load()
    // ...
}

// After
package gomodutils

func TestModule_Load(t *testing.T) {
    m := NewModule(testPath)
    err := m.Load()
    // ...
}
```

#### 5.2 Add Tests for Optional Graph

**New test:** `gompkg/gomodutils/module_test.go`

```go
func TestModule_StandaloneUsage(t *testing.T) {
    // Test that Module works without SetGraph()
    mod := NewModule(testGoModPath)
    err := mod.Load()
    require.NoError(t, err)

    // These should work (no graph required)
    assert.NotEmpty(t, mod.Path)
    assert.NotEmpty(t, mod.Requires)
    assert.NotNil(t, mod.Dir())

    status, err := mod.AnalyzeStatus()
    require.NoError(t, err)
    assert.NotNil(t, status)

    // This should panic (requires graph)
    assert.Panics(t, func() {
        mod.RequireDirs()
    })
}

func TestModule_GraphAwareUsage(t *testing.T) {
    // Test that Module works WITH SetGraph()
    mod := NewModule(testGoModPath)
    graph := NewGraph()

    // Build graph with test modules
    err := graph.Build(testModuleDir)
    require.NoError(t, err)

    err = mod.SetGraph(graph)
    require.NoError(t, err)

    err = mod.Load()
    require.NoError(t, err)

    // Now graph-dependent methods should work
    dirs := mod.RequireDirs()
    assert.NotEmpty(t, dirs)

    repo := mod.Repo()
    assert.NotNil(t, repo)
}
```

#### 5.3 Add Tests for ModuleExt

**New test:** `gompkg/retinue/module_ext_test.go`

```go
func TestModuleExt_IsInFlux(t *testing.T) {
    // Test IsInFlux() functionality
    mod := gomodutils.NewModule(testGoModPath)
    graph := gomodutils.NewGraph()

    err := graph.Build(testModuleDir)
    require.NoError(t, err)

    err = mod.SetGraph(graph)
    require.NoError(t, err)

    err = mod.Load()
    require.NoError(t, err)

    // Wrap with ModuleExt
    ext := NewModuleExt(mod)

    ctx := context.Background()
    inFlux, reason, err := ext.IsInFlux(ctx)
    require.NoError(t, err)

    // Assert based on test fixture state
    // (test data would need clean or dirty repo)
    t.Logf("InFlux: %v, Reason: %s", inFlux, reason)
}
```

## Implementation Sequence

### Step 1: Prepare gomodutils (2-3 hours)
1. ✅ Add new fields to Module struct (Filepath, Graph, repo, modfile, loaded)
2. ✅ Update NewModule() to return pointer
3. ✅ Add SetGraph() method
4. ✅ Update Load() to store modfile.File
5. ✅ Move AnalyzeStatus to method (in deps.go)
6. ✅ Add methods from retinue.GoModule (Dir, Key, Repo, RequireDirs, etc.)
7. ✅ Add guard functions (chkLoaded, chkSetGraph)
8. ✅ Create types.go with type aliases

### Step 2: Move Graph and Repo (1-2 hours)
1. ✅ Copy go_mod_graph.go → gomodutils/graph.go
2. ✅ Rename GoModGraph → Graph
3. ✅ Update *GoModule → *Module references
4. ✅ Copy repo.go → gomodutils/repo.go
5. ✅ Update *GoModule → *Module references
6. ✅ Update package declarations and imports

### Step 3: Create retinue Wrapper (1 hour)
1. ✅ Create retinue/module_ext.go
2. ✅ Implement ModuleExt with embedded *Module
3. ✅ Move IsInFlux() to ModuleExt
4. ✅ Move getSubmodulePathsToExclude() helper
5. ✅ Add NewModuleExt() constructor

### Step 4: Update retinue Package (1-2 hours)
1. ✅ Delete old go_module.go
2. ✅ Delete old go_mod_graph.go
3. ✅ Delete old repo.go
4. ✅ Update engine.go imports and references
5. ✅ Update verdict.go imports and references
6. ✅ Update all other retinue files
7. ✅ Wrap modules with ModuleExt where IsInFlux() is called

### Step 5: Update Tests (2-3 hours)
1. ✅ Move module tests to gomodutils/module_test.go
2. ✅ Move graph tests to gomodutils/graph_test.go
3. ✅ Move repo tests to gomodutils/repo_test.go
4. ✅ Update test package declarations
5. ✅ Update test type references
6. ✅ Add standalone vs graph-aware tests
7. ✅ Add ModuleExt tests
8. ✅ Run tests and fix failures: `go test ./gompkg/gomodutils/...`
9. ✅ Run tests and fix failures: `go test ./gompkg/retinue/...`

### Step 6: Verify and Polish (1 hour)
1. ✅ Run full test suite: `go test ./...`
2. ✅ Build gomion: `go build ./cmd/...`
3. ✅ Check for compilation errors
4. ✅ Search for any remaining GoModule references: `grep -r "GoModule" gompkg/`
5. ✅ Search for any remaining GoModGraph references: `grep -r "GoModGraph" gompkg/`
6. ✅ Manual smoke test of `gomion next` if possible

**Total Estimated Time:** 8-11 hours (about 1-1.5 days)

## Success Criteria

1. ✅ Single unified `gomodutils.Module` type
2. ✅ Works standalone (parse & analyze) OR graph-aware (dependencies & repo)
3. ✅ All retinue functionality preserved in ModuleExt wrapper
4. ✅ Graph renamed from GoModGraph to Graph
5. ✅ All existing tests pass
6. ✅ No duplicate module representations in codebase
7. ✅ Clean separation: generic (gomodutils) vs Gomion-specific (retinue)
8. ✅ Follows "one source of truth" principle
9. ✅ Zero compilation errors
10. ✅ Ready to begin PRE-commit work

## Files Affected

### New Files (Create)
```
gompkg/gomodutils/
  └── types.go                  - Type aliases (ModuleKey, ModuleFile, etc.)

gompkg/retinue/
  └── module_ext.go             - ModuleExt wrapper for IsInFlux()
```

### Moved Files
```
gompkg/retinue/go_mod_graph.go  → gompkg/gomodutils/graph.go
gompkg/retinue/repo.go          → gompkg/gomodutils/repo.go
gompkg/retinue/*_test.go        → gompkg/gomodutils/*_test.go
```

### Modified Files
```
gompkg/gomodutils/
  ├── module.go                 - Add fields, methods, SetGraph(), guards
  ├── deps.go                   - AnalyzeStatus becomes method
  ├── graph.go (moved)          - Rename GoModGraph → Graph
  ├── repo.go (moved)           - Update Module references

gompkg/retinue/
  ├── engine.go                 - Use gomodutils.Module and Graph
  ├── verdict.go                - Update imports
  └── ... (all files using Module/Graph)
```

### Deleted Files
```
gompkg/retinue/
  ├── go_module.go              - Functionality moved to gomodutils
  ├── go_mod_graph.go           - Moved to gomodutils/graph.go
  └── repo.go                   - Moved to gomodutils/repo.go
```

## Risk Mitigation

1. **Breaking changes:** Mostly internal refactoring, minimal API surface changes
2. **Test coverage:** Move existing tests to ensure no regression
3. **Incremental approach:** Each step is independently testable
4. **Rollback plan:** Work in feature branch, can revert if needed
5. **Verification:** Full test suite run before completion

## Notes

- This consolidation **must** be completed before PRE-commit work begins
- The merged Module type will be used by both existing retinue code AND new precommit analyzers
- Graph is optional - allows simple use cases without complex dependency tracking
- ModuleExt wrapper keeps Gomion-specific logic separate from generic go.mod utilities
