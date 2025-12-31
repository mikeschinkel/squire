# Gomion Implementation - Completed Work

This file tracks completed phases and tasks from PLAN.md.

## Phase 1: Modal Menu System (go-cliutil) - ✅ COMPLETE

**Status**: Completed in go-cliutil package

**Display Format:**
```
Main Menu: [f3] Explore [f4] Manage [f5] Compose — [f1] Help
Actions:   [1] Commit — [0] help [9] quit
Choice:
```

## Phase 2: Gomion Data Model - ✅ TASK 1 COMPLETE

### Task 1: Merge `commitmsg` into `gompkg` - ✅ COMPLETE

**Completed**: December 27, 2025

**Changes Made:**

1. **Created `gompkg/commit_message_types.go`**:
   - Renamed `Request` → `CommitMessageRequest`
   - Renamed `Result` → `CommitMessageResponse`
   - Moved sentinel errors: `ErrCommitMsg`, `ErrEmptySubject`

2. **Created `gompkg/commit_message_generator.go`**:
   - Moved `GenerateCommitMessage()` function
   - Made `BuildPrompt()` private → `buildCommitMessagePrompt()`
   - Made `ParseResponse()` private → `parseCommitMessageResult()`
   - Embedded templates via `//go:embed templates/*.tmpl`

3. **Created `gompkg/commit_message_workflow.go`**:
   - Moved `GenerateWithAnalysis()`, `GenerateMessage()`, `RegenerateMessage()`
   - **Fixed os/exec usage** - replaced with `gitutils` methods:
     - `exec.Command("git", "diff", "--cached", "--quiet")` → `repo.GetStagedFiles()` + length check
     - `exec.Command("git", "diff", "--cached")` → `repo.GetStagedDiff(ctx)`
     - `exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")` → `repo.Branch` (field already populated by `Open()`)

4. **Moved `gompkg/templates/`**:
   - `templates/default.tmpl`
   - `templates/breaking.tmpl`

5. **Updated `gomcliui/modes.go`**:
   - Changed import from `commitmsg` to `gompkg`
   - Updated all function calls: `commitmsg.RegenerateMessage` → `gompkg.RegenerateMessage`
   - Updated all function calls: `commitmsg.GenerateWithAnalysis` → `gompkg.GenerateWithAnalysis`

6. **Removed `gompkg/commitmsg/` package**:
   - Deleted entire directory after successful merge

**Verification**: ✅ Both `gompkg` and `cmd` modules build successfully with `GOEXPERIMENT=jsonv2`

**Notes**:
- No duplicate `doterr.go` removal needed - `gompkg` already had its own copy
- All API changes align with plan: types renamed, private functions made private
- No import aliasing used (following project convention)

### Task 5: Create `gomcfg/staging_plan_takes.go` - ✅ COMPLETE

**Completed**: December 27, 2025

**Status**: File already existed with most implementation complete. Added missing `ClearStagingPlanTakes()` function.

**Types Defined**:
- `StagingPlanTakes` - Container for 3 AI-generated perspectives on grouping changes
- `StagingPlanTake` - One AI perspective with a theme (e.g., "By Feature", "By Layer", "By Risk")
- `TakeGroup` - A suggested group within a take (name, rationale, files)

**Operations Implemented**:
1. ✅ `SaveStagingPlanTakes(cacheKey string, takes *StagingPlanTakes) error`
   - Saves to `~/.cache/gomion/analysis/{key}-takes.json`
   - Creates cache directory if needed
   - JSON formatted with indentation

2. ✅ `LoadStagingPlanTakes(cacheKey string) (*StagingPlanTakes, error)`
   - Loads from `~/.cache/gomion/analysis/{key}-takes.json`
   - Returns error if file doesn't exist

3. ✅ `ClearStagingPlanTakes(cacheKey string) error` - **ADDED**
   - Deletes cached takes file
   - Gracefully handles "file not found" (not an error)

**Bonus Functions** (already existed):
- `ComputeAnalysisCacheKey(files []dt.RelFilepath, analysisInput string) string` - Generates SHA256-based cache key
- `getAnalysisCacheFile(cacheKey string) (dt.Filepath, error)` - Helper to compute cache file path

**Verification**: ✅ Package builds successfully with `GOEXPERIMENT=jsonv2`

**Notes**:
- Implementation uses `TakeGroup` instead of `TakePlan` from original plan (better naming)
- Cache directory: `~/.cache/gomion/analysis/`
- Cache file format: `{cacheKey}-takes.json`

### Task 6: Add `.gomion/` directory management - ✅ COMPLETE

**Completed**: December 27, 2025

**Changes Made**:

1. **Enhanced `gompkg/gomion_init.go`**:
   - Added `InitGomionDirectoryArgs` struct with `ModuleDir` and `CommitChanges` fields
   - Modified `InitGomionDirectory()` to accept args struct and support optional git commit
   - Created `commitGomionStructure()` function to stage and commit `.gomion/` directory structure
   - Uses proper dt types throughout (`dt.RelFilepath`, `dt.PathSegment`)
   - Only converts to strings at boundary (when calling gitutils functions)

2. **Created `gitutils.StageFiles()` in `gitutils/operations.go`**:
   - Encapsulates `git add` functionality
   - Takes module directory and variadic list of paths to stage
   - Follows ClearPath pattern with single return at end

3. **Fixed logic bug in `gitutils/const.go`**:
   - **Issue**: `ContainsPathSegment()` and `ContainsFilename()` returned inverted results
   - **Root Cause**: Line 103 had `contains = !cf.containsLine(...)` (negated result)
   - **Fix**: Removed the `!` negation so functions return `true` when path IS found (not when it's NOT found)
   - **Updated Callers**: Modified `gomion_init.go` to use `if !needsArchive` instead of `if needsArchive`
   - **Impact**: Function semantics now match their names (ContainsX returns true when X is contained)

**Functionality**:
- ✅ Auto-creates `.gomion/` directory structure on first use
- ✅ Creates subdirectories: `plans/`, `candidates/`, `snapshots/`
- ✅ Creates archive subdirectories: `.archive/candidates/`, `.archive/snapshots/`
- ✅ Adds `.gitkeep` files to empty directories
- ✅ Adds `.gomion/.archive/` to `.gitignore` (if not already present)
- ✅ Adds `gomion.json` to `.git/info/exclude` (if not already present)
- ✅ Optionally commits `.gomion/` structure to git (when `CommitChanges=true`)
- ✅ Stages `.gomion/` directory and `.gitignore` if it exists
- ✅ Creates commit with message: "Initialize .gomion/ directory structure"

**Verification**: ✅ Both `gompkg` and `cmd` modules build successfully with `GOEXPERIMENT=jsonv2`

**Type Safety Notes**:
- All path operations use go-dt types (`dt.DirPath`, `dt.Filepath`, `dt.RelFilepath`, `dt.PathSegment`)
- String conversion only happens at boundaries (when calling `exec.Command`)
- Follows go-house-rules principle: use typed paths end-to-end

---

## GRU Phase 1: Foundation & Data Structures ✅ COMPLETE

**Completed:** 2025-12-30

**Goal:** CLI skeleton, basic types, cached repo setup

**Files Created:**
- `gommod/gomtui/types.go` - Core data structures (EditorState, ChangeSet, ViewMode, Pane, FileWithHunks, Hunk, HunkHeader)
- `gommod/gomtui/errors.go` - Error sentinels (ErrGit, ErrGitChangeSet, ErrGitIndex)
- `gommod/gomtui/doterr.go` - Error handling (copied from go-doterr)
- `gommod/gomtui/git_index.go` - Git index management with methods on ChangeSet type
- `gommod/gomtui/run.go` - TUI entry point with New() and Run() methods

**Files Modified:**
- `gommod/run.go` - Updated to call `gomtui.New(writer, logger).Run(args)`
- `gommod/gomion/options.go` - Commented out InputPath/OutputPath (not needed for TUI)
- `gommod/go.mod` - Added bubbletea and lipgloss dependencies
- `/Users/mikeschinkel/Projects/go-pkgs/go-dt/errors.go` - Added ErrNotImplemented sentinel

**Key Implementations:**

1. **TUI struct** - Clean OOP design with Writer and Logger properties
2. **ChangeSet methods** - Following user's guidance to use methods instead of package functions:
   - `CreateIndex(projectRoot)` - Creates Git index file in `<projectRepo>/.git/info/changesets/<id>/index`
   - `LoadIndex(projectRoot)` - Loads existing index file
   - `StageHunk()` - Placeholder for hunk staging (uses GIT_INDEX_FILE env var)
   - `GetMetaPath()` / `GetPatchPath()` - Path helpers with caching
3. **Path management** - Using dt package methods (no String() casting):
   - `dt.DirPathJoin3()`, `dt.FilepathJoin()` for path construction
   - `fp.ReadFile()`, `fp.WriteFile()`, `fp.Stat()`, `fp.Exists()` for file operations
   - `dir.MkdirAll()`, `dir.Exists()` for directory operations
4. **Error handling** - Proper doterr usage:
   - Each package has its own doterr.go copy
   - Using `NewErr()` with sentinels and key-value pairs
   - Adding context once at `end:` label with `WithErr()`
   - No fmt.Errorf or string formatting in errors
5. **Constants** - Using gitutils.InfoPath instead of hardcoding `.git/info`

**Testing Results:**
```bash
$ GOEXPERIMENT=jsonv2 ./gomion .
GRU TUI initialized successfully
Module: .
Cached repo: /Users/mikeschinkel/Library/Caches/repos/gomion-ebab9c1a4878840e
```

**Deliverables Met:**
- ✅ gomion binary builds
- ✅ Opens cached repo successfully
- ✅ Basic EditorState struct ready
- ✅ Unit test placeholders (git_index functions ready for testing)

**Design Decisions:**
- Used methods on ChangeSet type instead of package-level functions with many parameters
- Cached ChangeSet paths in map for performance
- Used const declarations for directory/file names
- TUI type with New() constructor and Run() method for clean API

---

## GRU Phase 2: Takes Generation & Loading ✅ COMPLETE

**Completed:** 2025-12-30

**Goal:** Generate Takes via AI, load into gru, establish gomcfg/gompkg architecture

**Files Created:**
- `gompkg/plan_takes.go` - Domain types with validated types (time.Time, dt.RelFilepath)
  - PlanTakes, PlanTake, ChangeSet types for runtime use
  - Parse*() functions to convert gomcfg → gompkg
  - ToCfg() methods to convert gompkg → gomcfg
  - Save/Load/Clear wrapper functions delegating to gomcfg
- `gompkg/plan_takes_generator.go` - AI generation logic
  - GeneratePlanTakes() - Calls AI to generate 3 takes
  - CreateDefaultTake() - Fallback "stage everything" take
  - Uses text/template with //go:embed for prompt
- `gompkg/plan_takes_prompt.tmpl` - AI prompt template
  - Instructs AI to generate 3 takes with different strategies
  - By Feature, By Layer, By Risk
  - Returns structured JSON response

**Files Modified:**
- `gompkg/gomcfg/plan_takes.go` - Converted to scalar types only
  - Changed Timestamp from `time.Time` to `string` (RFC3339 format)
  - Changed Files from `[]dt.RelFilepath` to `[]string`
  - Now used ONLY for JSON serialization
- `cmd/gru/grumod/grupkg/parse.go` - Fixed for scalar type changes
  - Line 200: Format time.Time to RFC3339 string
  - Line 268-271: Convert []dt.RelFilepath to []string
- `cmd/gru/grumod/grutui/run.go` - Added load/generate logic
  - loadOrGenerateTakes() - Loads from cache or generates via AI
  - Placeholder methods: getChangedFiles(), getGitDiff(), generateTakesViaAI()

**Key Architectural Decisions:**

1. **Two-Layer Type System:**
   - **gomcfg types** (e.g., gomcfg.PlanTakes): ONLY scalar built-in types, ONLY for JSON serialization
   - **gompkg types** (e.g., gompkg.PlanTakes): Validated types (dt.RelFilepath, time.Time, etc.), for runtime use
   - Parse*() functions convert gomcfg → gompkg
   - ToCfg() methods convert gompkg → gomcfg

2. **Template-Based Prompts:**
   - AI prompts use text/template with //go:embed
   - Easier to tweak than string building
   - Clean separation of prompt content from code logic

3. **Graceful Degradation:**
   - Try cache first
   - Try AI generation
   - Fall back to default take if AI fails
   - Non-fatal cache write failures (just warn)

4. **Caching Strategy:**
   - Cache key based on hash of files + diff content
   - Stored in ~/.cache/gomion/analysis/{key}-takes.json
   - Automatic cache hit detection

**Implementation Status:**

✅ **Complete - No TODOs, No Placeholders:**
- gomcfg/gompkg architecture established
- AI prompt template created (text/template + //go:embed)
- GeneratePlanTakes() function fully implemented
- CreateDefaultTake() implemented
- Parse*() and ToCfg() conversion functions
- Save/Load/Clear wrappers
- Cache key computation
- GetChangedFiles() method in gitutils
- GetWorkingDiff() method in gitutils
- loadOrGenerateTakes() with cache, AI, and default fallback
- generateTakesViaAI() using askai.Agent with Claude CLI provider
- NewAgent() defaults to Claude CLI provider
- Graceful degradation when AI unavailable (falls back to default take)

**Testing:**
```bash
$ cd /Users/mikeschinkel/Projects/gomion/cmd/gru
$ make build
Building gru...
Built to ./bin/gru

$ cd /Users/mikeschinkel/Projects/gomion/gommod
$ GOEXPERIMENT=jsonv2 go build ./...
# Success - all packages compile
```

**Design Patterns Established:**

1. **Scalar-Only Config Types:**
   - All gomcfg types use ONLY scalar built-in types
   - No time.Time, dt.DirPath, dt.RelFilepath in gomcfg
   - Enforces clean JSON serialization boundary

2. **Parse/ToCfg Conversion:**
   - Parse*() functions validate and convert config → service
   - ToCfg() methods convert service → config for saving
   - Bidirectional conversion without data loss

3. **Template Embedding:**
   - //go:embed for templates
   - text/template for rendering
   - Keeps prompt content maintainable

---

## Already Implemented (Pre-existing)

The following tasks were already implemented before this session:

### Phase 2: Gomion Data Model

- ✅ Task 2: `gompkg/staging_plan.go` (StagingPlan, FilePatchRange, HunkHeader)
- ✅ Task 3: `gompkg/commit_candidate.go` (CommitCandidate persistence)
- ✅ Task 4: `gompkg/staging_snapshot.go` (StagingSnapshot, git integration)

### Phase 3: Gomion Modes (Text-Based)

- ✅ Task 1: `gompkg/mode_state.go` (GomionModeState - shared state)
- ✅ Task 2: `gompkg/main_mode.go` (Main mode - F2)
- ✅ Task 3: `gompkg/explore_mode.go` (Explore mode - F3)
- ✅ Task 4: `gompkg/manage_mode.go` (Manage mode - F4)
- ✅ Task 5: `gompkg/compose_mode.go` (Compose mode - F5)
