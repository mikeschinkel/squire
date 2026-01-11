# Gomion Commit Plan Persistence & Execution

**Date**: 2026-01-04
**Status**: Foundation Complete - Ready for Persistence Implementation

# Gomion Implementation Plan

## ⚠️ CRITICAL INSTRUCTIONS FOR CLAUDE - READ FIRST

**BEFORE starting ANY work on this project:**

1. **Read this entire file** - This shows what's LEFT to do, not what's already done
2. **Be aware of DONE.md** - This shows what's already completed (don't re-implement!) Only read if you need to.
3. **Review ALL configured skills** - Ensure you're familiar with go-house-rules, error-handling-author, go-dt-filepath-refactorer, etc.
4. **Only work on what's in this file** - Don't waste tokens reviewing/implementing completed work
5. **When you complete a task:**
    - MOVE the implementation details from this file to DONE.md
    - DELETE the completed content from this file.
    - Goal: Whittle PLAN.md down to nothing as work progresses

**Why this matters:**
- This file = What needs to be done (gets smaller)
- DONE.md = What was completed (gets larger)
- Don't rely on conversation summaries - always check this file first
- This prevents wasting tokens on already-completed work

---

## Critical Constraints (MUST FOLLOW)

1. **All git operations in gitutils** - Never call git commands directly
2. **doterr error handling** - Each package has own doterr.go, use NewErr() with sentinels
3. **dt package for paths** - No .String() casting, use dt methods directly
4. **ClearPath style** - Single return, goto end, no else
5. **Scalar types in gomcfg** - NO time.Time, dt.DirPath in config types (JSON only)
6. **Parse functions** - gomcfg → gompkg conversion with validation
7. **GOEXPERIMENT=jsonv2** - Required for build


## Overview

This document outlines the plan for implementing persistence and execution of file commit plan in the gomion TUI, including AI-generated commit strategies ("takes") and the workflow for applying commit plan to git ignore files and creating commits.

## Current State

**Completed:**
- ✅ TUI for file selection with tree and table views
- ✅ File disposition assignment (Commit, Omit, GitIgnore, GitExclude)
- ✅ Pane-aware disposition handling
- ✅ File metadata loading and git status integration
- ✅ Module-scoped vs repo-scoped file filtering
- ✅ **Disposition storage refactoring** (2026-01-04)
  - Separated commit plan from File struct
  - Moved to `EditorState.Dispositions *map[dt.RelFilepath]FileDisposition`
  - Pointer-to-map for efficient in-place updates (no tree rebuilding)
  - Real-time UI updates when commit plan change
  - Foundation ready for persistence

**Next Steps:**
1. **Implement persistence** - Save/load commit plan to `.git/info/gomion/commit-plan.json`
2. Apply commit plan (update .gitignore/.git/info/exclude)
3. AI-generated commit takes
4. New UI for reviewing and executing takes

---

## 1. Disposition Persistence

### Storage Format

**Important:** Dispositions are stored as explicit file paths only. No wildcards are used.

```json
{
  "version": 1,
  "scope": "module",
  "module_path": "gommod/gomtui",
  "timestamp": "2026-01-04T12:50:37Z",
  "commit_plan": {
    "gommod/gomtui/file.go": "commit",
    "gommod/gomtui/editor_state.go": "commit",
    "temp/old_code.go": "omit",
    "debug.log": "gitignore",
    "vendor/lib1.go": "gitexclude",
    "vendor/lib2.go": "gitexclude"
  }
}
```

**Note:** When user sets disposition for a directory (e.g., `/vendor/`), all children are recursively set. The saved plan contains explicit entries for each file. This ensures new files added later require explicit user review.

### File Locations

```
.git/info/gomion/
├── commit-plan.json                          # Current/active plan
├── commit-plan-history/
│   ├── 2026-01-04-125037.json                # Archived plans
│   └── 2026-01-03-093012.json
└── takes/
    └── <take-id>.json                         # Generated takes (cached)
```

### Design Decisions

**✅ Default disposition: UnspecifiedDisposition**
- All files start as `UnspecifiedDisposition` (requires user action)
- When loading saved plan, new files get `UnspecifiedDisposition`
- User must explicitly mark all files before proceeding to commit
- Prevents accidental commits of unreviewed files

**✅ No wildcards - explicit storage only**
- All file/directory commit plan are stored explicitly in the map
- When setting disposition on a directory, recursively set for all children
- Lookup is O(1) (critical for rendering performance)
- **Rationale:** Wildcard parent-walking lookup is O(depth) and called many times during rendering, causing performance issues
- Memory overhead is minimal (pointers + short paths)
- Simplicity wins over premature optimization

**✅ Two-phase loading with pattern suggestions**
- Phase 1: Load saved commit plan, apply to matching files
- Phase 2: Detect new files, suggest patterns based on parent directories
- Modal popup shows: "5 new files in `/vendor/` (previously excluded). Apply pattern?"
- User chooses: Apply suggested / Review individually / Cancel
- **Provides safety with convenience**

**✅ Paths must be repo-relative**
- Consistency with git commands
- Direct git status lookup
- No coordinate system translation needed
- Display strips prefix when module-scoped

**✅ Auto-save on every change**
- Like IDE autosave
- No "Save" button needed
  - HOWEVER, user may WANT to checkpoint
  - So we need to discuss both restoring autosaves and checkpoints
- Prevents loss of work
- Should be done in goroutine to avoid keyboard latency

**✅ Archive on apply**
- Move to history when plan is executed
- Allows undo/review
- Start fresh after commits

**⚠️ Staleness handling**
- Files might be added/deleted between sessions
- Validate on load:
  - Remove commit plan for deleted files
  - Flag new files as `UnspecifiedDisposition`
  - Check if git status changed
  - Show pattern suggestion modal for new files

**⚠️ Merge conflict potential**
- `.git/info/` is local (not tracked)
- Won't conflict across branches
- But might want branch-specific plans (future)

### Implementation

```go
// gommod/gompkg/commit_plan.go

type CommitPlan struct {
    Version      int                                 `json:"version"`
    Scope        CommitScope                         `json:"scope"`
    ModulePath   dt.RelDirPath                       `json:"module_path,omitempty"`
    Timestamp    time.Time                           `json:"timestamp"`
    Dispositions map[dt.RelFilepath]FileDisposition `json:"commit_plan"`
}

type CommitScope string

const (
    ModuleScope CommitScope = "module"
    RepoScope   CommitScope = "repo"
)

// Save persists the plan to .git/info/gomion/commit plan.json
func (cp *CommitPlan) Save(repoRoot dt.DirPath) error

// Load reads the current plan
func LoadCommitPlan(repoRoot dt.DirPath) (*CommitPlan, error)

// Archive moves current plan to history
func (cp *CommitPlan) Archive(repoRoot dt.DirPath) error

// Validate checks for stale/invalid commit plan
func (cp *CommitPlan) Validate(repoRoot dt.DirPath) (*ValidationResult, error)

type ValidationResult struct {
    Warnings   []string
    Errors     []string
    CanProceed bool
}
```

---

## 2. Applying Dispositions

### GitIgnore Disposition

**Challenges:**
1. `.gitignore` uses patterns, not literal paths
2. Need to avoid duplicates
3. File might already match existing pattern
4. Where to add: root `.gitignore` or module `.gitignore`?
5. What if file is already staged?

**Strategy Options:**

```go
type GitIgnoreStrategy int

const (
    // Add literal path: "gommod/gomtui/debug.go"
    LiteralPath GitIgnoreStrategy = iota

    // Add pattern by basename: "debug.go" (matches anywhere)
    PatternBasename

    // Add directory pattern: "gommod/gomtui/" (all files in dir)
    PatternDirectory
)
```

**Recommended Approach:**

1. **Check if already ignored**
   ```go
   // Use git check-ignore to see if file already ignored
   git check-ignore -v <filepath>
   ```

2. **Add with section marker**
   ```gitignore
   # Added by gomion on 2026-01-04
   gommod/gomtui/debug.go
   temp/scratch.txt
   ```

3. **Track additions for undo**
   ```go
   type GitIgnoreAddition struct {
       FilePath    dt.Filepath
       LineNumbers []int
       AddedLines  []string
       Timestamp   time.Time
   }
   ```

4. **Prompt user for strategy** (Phase 2)
   - Default: Literal path
   - Advanced: Let user choose pattern type

**Implementation:**

```go
type GitIgnoreManager struct {
    repoRoot   dt.DirPath
    gitignore  dt.Filepath  // Path to .gitignore
}

// AddFiles appends files to .gitignore with deduplication
func (m *GitIgnoreManager) AddFiles(files []dt.RelFilepath) (*GitIgnoreAddition, error)

// CheckIfIgnored returns true if file matches .gitignore pattern
func (m *GitIgnoreManager) CheckIfIgnored(file dt.RelFilepath) (bool, string, error)

// RemoveAddition undoes a previous addition (for rollback)
func (m *GitIgnoreManager) RemoveAddition(addition *GitIgnoreAddition) error
```

### GitExclude Disposition

**Same challenges as GitIgnore, but simpler:**
- `.git/info/exclude` is local (not tracked)
- Won't conflict across branches
- Same format as `.gitignore`
- Use case: Personal files (IDE configs, scratch files)

**Implementation:** Same as `GitIgnoreManager` but targeting `.git/info/exclude`

```go
type GitExcludeManager struct {
    repoRoot   dt.DirPath
    excludeFile dt.Filepath  // .git/info/exclude
}
```

### Omit Disposition

**Clarify semantics:**

| Disposition | Meaning | Persistence | Git Impact |
|-------------|---------|-------------|------------|
| **Commit** | Include in commits | No (cleared after apply) | Will be committed |
| **Omit** | Skip for this session | No (not saved) | No change |
| **GitIgnore** | Ignore forever (shared) | Yes (in .gitignore) | Added to .gitignore |
| **GitExclude** | Ignore forever (personal) | Yes (in .git/info/exclude) | Added to exclude |

**Recommendation:**
- **Omit** = Temporary skip (not persisted to disposition plan)
- Files with Omit disposition are excluded from commit workflow but not ignored by git
- Useful for "not ready yet" files

### Validation Before Apply

```go
type DispositionValidator struct {
    plan     *CommitPlan
    repoRoot dt.DirPath
}

func (v *DispositionValidator) Validate() (*ValidationResult, error)

// Checks:
// 1. **BLOCKER:** Are there any files with UnspecifiedDisposition or UnknownDisposition?
//    - Cannot proceed if any files are unmarked
//    - User must explicitly mark all files
// 2. Are there any files with Commit disposition?
// 3. Are files already in .gitignore getting GitIgnore disposition?
// 4. Do any files have conflicting commit plan?
// 5. Are untracked files being committed without explicit add?
// 6. Is .gitignore malformed?
// 7. Have files been modified since commit plan were saved?
```

---

## 3. AI Integration - Commit Takes

### What is a "Take"?

A **take** is an alternative commit strategy proposed by AI. Given a set of changed files, the AI suggests 2-3 different ways to organize them into logical, atomic commits.

**Example:**

```
Take 1: Feature-Focused (3 changesets)
  Rationale: Group by user-facing features

  Changeset 1: Add file metadata loading
    Files: file.go, file_metadata.go
    Message: "Add file metadata loading for directory tables"

  Changeset 2: Integrate table with metadata display
    Files: files_table_model.go, editor_state.go
    Message: "Display file metadata in directory table view"

  Changeset 3: Fix disposition handling for focused panes
    Files: editor_state.go
    Message: "Fix disposition keys to target focused pane"

Take 2: Layer-Focused (2 changesets)
  Rationale: Group by architectural layer

  Changeset 1: Data model improvements
    Files: file.go, file_metadata.go
    Message: "Improve file data model with metadata support"

  Changeset 2: UI layer updates
    Files: files_table_model.go, editor_state.go
    Message: "Update UI to display file metadata and handle commit plan"
```

### AI Prompt Structure

```markdown
# Context
Repository: gomion
Module: gommod/gomtui (optional: full repo)
Changed files: <count>

## File Changes

### gommod/gomtui/file.go (Modified, +50 -10)
```diff
<relevant diff>
```

### gommod/gomtui/editor_state.go (Modified, +120 -30)
```diff
<relevant diff>
```

# Task
Analyze these changes and propose 2-3 alternative commit strategies ("takes").

For each take:
1. Give it a descriptive name and brief rationale
2. Break changes into logical changesets (commits)
3. For each changeset provide:
   - Name/summary
   - Which files to include
   - Suggested commit message (conventional commits style)
   - Rationale (why these changes belong together)

# Constraints
- Each changeset must be atomic (buildable and testable on its own)
- Prefer smaller, focused commits over large ones
- Consider functional boundaries and dependencies
- Follow conventional commits format (feat/fix/refactor/docs/etc)

# Output Format
Return JSON in this structure:
{
  "takes": [
    {
      "name": "...",
      "rationale": "...",
      "changesets": [
        {
          "name": "...",
          "message": "...",
          "files": ["...", "..."],
          "rationale": "..."
        }
      ]
    }
  ]
}
```

### AI Provider Interface

```go
// gommod/gompkg/ai/provider.go

type TakesGenerator interface {
    GenerateTakes(ctx context.Context, req *TakesRequest) (*TakesResponse, error)
}

type TakesRequest struct {
    Files      []FileChange
    RepoName   string
    ModulePath dt.RelDirPath
    MaxTakes   int
    Style      CommitStyle  // conventional, semantic, custom
}

type FileChange struct {
    Path   dt.RelFilepath
    Status string  // Modified, Added, Deleted
    Diff   string
}

type TakesResponse struct {
    Takes []Take
}

type Take struct {
    ID         string
    Name       string
    Rationale  string
    ChangeSets []ChangeSet
}

type ChangeSet struct {
    ID        string
    Name      string
    Message   string
    Files     []dt.RelFilepath
    Rationale string
}

type CommitStyle string

const (
    ConventionalCommits CommitStyle = "conventional"
    SemanticCommits     CommitStyle = "semantic"
    CustomStyle         CommitStyle = "custom"
)
```

### AI Provider Implementations

```go
// Anthropic Claude
type ClaudeTakesGenerator struct {
    client      *anthropic.Client
    model       string
    temperature float64
}

func (g *ClaudeTakesGenerator) GenerateTakes(ctx context.Context, req *TakesRequest) (*TakesResponse, error)

// Mock for development/testing
type MockTakesGenerator struct {
    takes []Take  // Predefined responses
}

func (g *MockTakesGenerator) GenerateTakes(ctx context.Context, req *TakesRequest) (*TakesResponse, error)

// Manual (no AI)
type ManualTakesGenerator struct{}

func (g *ManualTakesGenerator) GenerateTakes(ctx context.Context, req *TakesRequest) (*TakesResponse, error) {
    // Returns single take: one changeset with all files
}
```

### Critical AI Challenges

**🚨 Context Size Limits**
- Large diffs might exceed token limits
- Need to summarize or chunk changes
- Possible solution: Summarize files, include full diff only for small changes

**🚨 Cost**
- API calls can be expensive
- Cache takes based on file set + git commit hash
- Provide budget controls

**🚨 Offline Development**
- Must work without API access
- Mock mode with predefined responses
- Local model support (future)

**⚠️ Latency**
- API calls take 10-30 seconds
- Show progress/spinner
- Support streaming responses
- Allow cancellation

**Implementation:**

```go
// Takes caching
type TakesCache struct {
    cacheDir dt.DirPath  // .git/info/gomion/takes/
}

func (c *TakesCache) Get(key CacheKey) (*TakesResponse, bool)
func (c *TakesCache) Set(key CacheKey, response *TakesResponse) error

type CacheKey struct {
    CommitHash string                             // Current HEAD
    Files      []dt.RelFilepath                   // Sorted
    Checksum   string                             // Hash of file contents
}
```

---

## 4. State Machine & UI Flow

### Editor Modes

```go
type EditorMode int

const (
    FileSelectionMode EditorMode = iota   // Current UI - assign commit plan
    PatternSuggestionMode                 // Suggest patterns for new files after load
    DispositionReviewMode                 // Review plan before applying
    ApplyingDispositionsMode              // Show progress of .gitignore updates
    TakesGenerationMode                   // "Generating commit strategies..."
    TakesSelectionMode                    // Choose/edit AI-generated takes
    ChangeSetRefinementMode               // Tweak selected take's changesets
    CommitExecutionMode                   // Creating commits (show progress)
    CompletionMode                        // Success/failure summary
)
```

### State Transitions

```
FileSelectionMode
    ↓ (On startup with saved plan + new files)
PatternSuggestionMode
    ↓ (User applies/rejects patterns, or starts fresh session)
FileSelectionMode
    ↓ (Press Enter - "Continue")
DispositionReviewMode
    ↓ (Confirm - "Apply")
ApplyingDispositionsMode
    ↓ (Complete)
TakesGenerationMode
    ↓ (AI responds or mock completes)
TakesSelectionMode
    ↓ (Select take, press Enter)
ChangeSetRefinementMode (optional)
    ↓ (Confirm - "Create Commits")
CommitExecutionMode
    ↓ (Complete)
CompletionMode
    ↓ (Press 'q' or 'n' for new session)
FileSelectionMode (fresh start)
```

**Loading Behavior:**
- On startup: Load saved plan if exists
- If new files detected: Show `PatternSuggestionMode` modal
- Modal shows filenames and suggested patterns
- User chooses: Apply / Review individually / Cancel
- Then proceed to `FileSelectionMode`

**Back navigation:**
- Any mode → Press 'b' or ESC → Previous mode
- Exception: Can't go back after commits created (irreversible)

### Mode UIs

#### PatternSuggestionMode

```
╔════════════════════════════════════════════════════════════════╗
║ New Files Detected                                             ║
╠════════════════════════════════════════════════════════════════╣
║                                                                 ║
║ The following files were added since the last session:        ║
║                                                                 ║
║ ┌────────────────────────────────────────────────────────────┐ ║
║ │ 5 files in vendor/ (previously: GitExclude)                │ ║
║ │   vendor/new-lib/file1.go                                  │ ║
║ │   vendor/new-lib/file2.go                                  │ ║
║ │   vendor/new-lib/file3.go                                  │ ║
║ │   vendor/another/lib.go                                    │ ║
║ │   vendor/another/helper.go                                 │ ║
║ │                                                             │ ║
║ │ Suggested: Apply GitExclude pattern                        │ ║
║ └────────────────────────────────────────────────────────────┘ ║
║                                                                 ║
║ ┌────────────────────────────────────────────────────────────┐ ║
║ │ 2 files in temp/ (previously: GitIgnore)                   │ ║
║ │   temp/new-scratch.txt                                     │ ║
║ │   temp/debug-output.log                                    │ ║
║ │                                                             │ ║
║ │ Suggested: Apply GitIgnore pattern                         │ ║
║ └────────────────────────────────────────────────────────────┘ ║
║                                                                 ║
║ ┌────────────────────────────────────────────────────────────┐ ║
║ │ 1 file with no pattern match                               │ ║
║ │   gommod/gomtui/new-feature.go                             │ ║
║ │                                                             │ ║
║ │ Requires manual review: UnspecifiedDisposition             │ ║
║ └────────────────────────────────────────────────────────────┘ ║
║                                                                 ║
╠════════════════════════════════════════════════════════════════╣
║ a: Apply All Patterns | r: Review Individually | ESC: Cancel  ║
╚════════════════════════════════════════════════════════════════╝
```

**Behavior:**
- Detect new files by comparing current working tree to saved plan
- Group by parent directory and analyze commit plan
- If all files in directory had same disposition, suggest pattern
- Files without patterns remain `UnspecifiedDisposition`
- User can apply all suggestions, review one by one, or cancel and mark manually

#### DispositionReviewMode

```
╔════════════════════════════════════════════════════════════════╗
║ Review Disposition Plan                                        ║
╠════════════════════════════════════════════════════════════════╣
║                                                                 ║
║ Files to Commit (3):                                           ║
║   gommod/gomtui/file.go                                        ║
║   gommod/gomtui/editor_state.go                                ║
║   gommod/gomtui/file_metadata.go                               ║
║                                                                 ║
║ Files to Add to .gitignore (2):                                ║
║   debug.log                                                    ║
║   temp/scratch.txt                                             ║
║                                                                 ║
║ Files to Add to .git/info/exclude (1):                         ║
║   .idea/workspace.xml                                          ║
║                                                                 ║
║ Files to Omit (1):                                             ║
║   work-in-progress.go                                          ║
║                                                                 ║
╠════════════════════════════════════════════════════════════════╣
║ Changes to .gitignore:                                         ║
║   + # Added by gomion on 2026-01-04                            ║
║   + debug.log                                                  ║
║   + temp/scratch.txt                                           ║
╠════════════════════════════════════════════════════════════════╣
║ Enter: Apply | b: Back to Edit | q: Quit                      ║
╚════════════════════════════════════════════════════════════════╝
```

#### TakesSelectionMode

```
╔════════════════════════════════════════════════════════════════╗
║ Select Commit Strategy                                         ║
╠════════════════════════════════════════════════════════════════╣
║                                                                 ║
║ ▶ Take 1: Feature-Focused (3 commits)                          ║
║     Rationale: Group changes by user-facing features           ║
║                                                                 ║
║     1. Add file metadata loading                               ║
║        └─ file.go, file_metadata.go                            ║
║                                                                 ║
║     2. Integrate table with metadata display                   ║
║        └─ files_table_model.go, editor_state.go                ║
║                                                                 ║
║     3. Fix disposition handling for focused panes              ║
║        └─ editor_state.go                                      ║
║                                                                 ║
║   Take 2: Layer-Focused (2 commits)                            ║
║     Rationale: Group by architectural layer                    ║
║                                                                 ║
║     1. Data model improvements                                 ║
║        └─ file.go, file_metadata.go                            ║
║                                                                 ║
║     2. UI layer updates                                        ║
║        └─ files_table_model.go, editor_state.go                ║
║                                                                 ║
╠════════════════════════════════════════════════════════════════╣
║ ↑/↓: Navigate | Enter: Select | e: Edit | r: Regenerate       ║
║ m: Manual Take | b: Back | q: Quit                             ║
╚════════════════════════════════════════════════════════════════╝
```

#### CommitExecutionMode

```
╔════════════════════════════════════════════════════════════════╗
║ Creating Commits                                                ║
╠════════════════════════════════════════════════════════════════╣
║                                                                 ║
║ ✓ Commit 1/3: Add file metadata loading                       ║
║   Files: file.go, file_metadata.go                             ║
║   Hash: a1b2c3d                                                ║
║                                                                 ║
║ ⏳ Commit 2/3: Integrate table with metadata display           ║
║   Staging files...                                             ║
║                                                                 ║
║ ⏹ Commit 3/3: Fix disposition handling for focused panes      ║
║   Pending...                                                   ║
║                                                                 ║
╠════════════════════════════════════════════════════════════════╣
║ Please wait...                                                 ║
╚════════════════════════════════════════════════════════════════╝
```

---

## 5. What's Missing - Critical Gaps

### 5.1 Hunk-Level Granularity

**Current:** File-level disposition
**Ideal:** Hunk-level assignment to changesets

Your `File` type has `Hunks []Hunk` but this is not used yet.

**Challenge:** Assigning individual hunks to different commits is complex:
- Need UI to select hunks (like `git add -p`)
- Need to generate partial diffs
- Need to apply hunks selectively

**Recommendation:** **Defer to Phase 4 / v2**
File-level granularity is good enough for MVP.

### 5.2 Dry-Run Mode

**Critical:** Show what WILL happen before doing it

```go
type DryRunResult struct {
    GitIgnoreChanges  []string  // Lines to be added
    GitExcludeChanges []string
    CommitsToCreate   []CommitPreview
}

type CommitPreview struct {
    Message string
    Files   []dt.RelFilepath
    Diff    string  // Full diff for this commit
}

func (cp *CommitPlan) DryRun(repoRoot dt.DirPath, take *Take) (*DryRunResult, error)
```

### 5.3 Undo Mechanism

**Problem:** What if user wants to undo .gitignore changes?

```go
type UndoableAction interface {
    Do() error
    Undo() error
}

type GitIgnoreModification struct {
    FilePath      dt.Filepath
    AddedLines    []string
    LineNumbers   []int
    OriginalLines []string  // For restoration
}

func (m *GitIgnoreModification) Do() error
func (m *GitIgnoreModification) Undo() error
```

**Track in:**
```json
// .git/info/gomion/undo/2026-01-04-125037.json
{
  "timestamp": "2026-01-04T12:50:37Z",
  "actions": [
    {
      "type": "gitignore_modification",
      "file": ".gitignore",
      "added_lines": ["debug.log", "temp/"],
      "line_numbers": [42, 43]
    }
  ]
}
```

### 5.4 Conflict Handling

**Scenarios:**

1. **File modified after commit plan saved**
   - Disposition plan is stale
   - Solution: Validate on load, warn user

2. **File deleted**
   - Remove from disposition plan
   - Solution: Auto-clean on load

3. **.gitignore modified externally**
   - Another tool/person modified it
   - Solution: Re-check for duplicates before adding

4. **Git index modified**
   - File staged/unstaged outside gomion
   - Solution: Refresh git status before commit

### 5.5 Progress Indication

**AI calls take time (10-30 seconds)**

```go
type ProgressUpdate struct {
    Stage   string   // "Analyzing files", "Generating takes", "Complete"
    Percent int      // 0-100
    Message string
}

// Stream progress updates via channel
func GenerateTakesWithProgress(ctx context.Context, req *TakesRequest) (<-chan ProgressUpdate, <-chan *TakesResponse, <-chan error)
```

**UI:**
- Spinner animation
- Progress bar
- Current stage message
- Allow cancellation (Ctrl+C)

### 5.6 Configuration

**AI settings:**

```json
// .git/info/gomion/config.json
{
  "ai": {
    "provider": "anthropic",
    "model": "claude-sonnet-4-5",
    "max_takes": 3,
    "style": "conventional-commits",
    "temperature": 0.7,
    "max_tokens": 4096,
    "mock_mode": false
  },
  "commit": {
    "auto_push": false,
    "gpg_sign": true,
    "verify_build": false
  }
}
```

---

## 6. Recommended Implementation Phases

### Phase 1: Basic Persistence (No AI) - MVP

**Goal:** Save/load commit plan, apply to .gitignore, manual commits

**Tasks:**
1. ✅ Implement `CommitPlan` save/load
2. ✅ Implement `GitIgnoreManager` and `GitExcludeManager`
3. ✅ Add `DispositionReviewMode` UI
4. ✅ Apply commit plan (update .gitignore/.git/info/exclude)
5. ✅ Manual commit flow (user writes message)
6. ✅ Archive plan after execution

**Deliverable:** Users can manage commit plan and create commits manually (no AI)

**Estimated Effort:** 2-3 days

### Phase 2: Simple AI Integration

**Goal:** Single AI-generated commit message

**Tasks:**
1. ✅ Implement `ClaudeTakesGenerator` basic version
2. ✅ Single API call: All files → One commit message
3. ✅ User can accept or edit message
4. ✅ Create single commit with AI message

**Deliverable:** AI helps write commit message for single commit

**Estimated Effort:** 1-2 days

### Phase 3: Takes - Full Vision

**Goal:** Multi-take, multi-commit workflow

**Tasks:**
1. ✅ Implement full AI prompt with multiple takes
2. ✅ Implement `TakesSelectionMode` UI
3. ✅ Implement `ChangeSetRefinementMode` UI (optional)
4. ✅ Multi-commit execution
5. ✅ Takes caching

**Deliverable:** Complete AI-driven commit strategy workflow

**Estimated Effort:** 3-5 days

### Phase 4: Advanced Features

**Goal:** Hunk-level, undo, advanced editing

**Tasks:**
1. ⏹ Hunk-level disposition assignment
2. ⏹ Interactive hunk selection UI
3. ⏹ Undo mechanism for .gitignore changes
4. ⏹ Advanced take editing (split/merge changesets)
5. ⏹ Disposition templates

**Deliverable:** Power-user features

**Estimated Effort:** 5-7 days

---

## 7. Suggested Architecture

### Package Structure

```
gommod/gompkg/
├── commit/
│   ├── plan.go              # CommitPlan save/load/validate
│   ├── gitignore.go         # GitIgnoreManager
│   ├── gitexclude.go        # GitExcludeManager
│   └── executor.go          # Execute commits from take
├── ai/
│   ├── provider.go          # TakesGenerator interface
│   ├── claude.go            # ClaudeTakesGenerator
│   ├── mock.go              # MockTakesGenerator
│   └── cache.go             # TakesCache
└── gomtui/
    ├── file_selection_view.go       # Current view
    ├── disposition_review_view.go   # Review before apply
    ├── takes_selection_view.go      # Select take
    ├── changeset_refinement_view.go # Edit take (optional)
    └── commit_execution_view.go     # Progress during commits
```

### Key Types

```go
// commit/plan.go
type CommitPlan struct {
    Version      int
    Scope        CommitScope
    ModulePath   dt.RelDirPath
    Timestamp    time.Time
    Dispositions map[dt.RelFilepath]FileDisposition
}

// commit/executor.go
type CommitExecutor struct {
    repo     *gitutils.Repo
    repoRoot dt.DirPath
}

func (e *CommitExecutor) ExecuteTake(ctx context.Context, take *Take) error

// ai/provider.go
type TakesGenerator interface {
    GenerateTakes(ctx context.Context, req *TakesRequest) (*TakesResponse, error)
}

type TakesRequest struct {
    Files      []FileChange
    RepoName   string
    ModulePath dt.RelDirPath
    MaxTakes   int
    Style      CommitStyle
}

type TakesResponse struct {
    Takes []Take
}

type Take struct {
    ID         string
    Name       string
    Rationale  string
    ChangeSets []ChangeSet
}

type ChangeSet struct {
    ID        string
    Name      string
    Message   string
    Files     []dt.RelFilepath
    Rationale string
}
```

---

## 8. Open Questions

1. **Commit message style:**
   - Enforce conventional commits? Allow custom?
   - Configuration option?

2. **Multi-repo support:**
   - What if workspace has multiple repos?
   - Dispositions per repo?

3. **Branch awareness:**
   - Different disposition plans per branch?
   - Or always shared (.git/info is local)?

4. **GPG signing:**
   - Auto-detect from git config?
   - Allow override?

5. **Pre-commit hooks:**
   - Run them during commit execution?
   - Show hook output to user?

6. **Build verification:**
   - Optional "verify build passes" between commits?
   - Configure test command?

7. **Remote push:**
   - Auto-push after commits?
   - Or leave local?

---

## 9. Success Criteria

**Phase 1 (MVP) is successful when:**
- ✅ User can assign commit plan to files
- ✅ Dispositions are persisted across sessions
- ✅ Files are added to .gitignore/.git/info/exclude correctly
- ✅ User can create manual commits with edited files
- ✅ Plan is archived after execution

**Phase 3 (Full vision) is successful when:**
- ✅ AI generates multiple commit strategies
- ✅ User can select/edit takes
- ✅ Multiple atomic commits are created from take
- ✅ Workflow feels fast and intuitive
- ✅ No manual git commands needed

---

## 10. Next Steps

**Immediate (Start with Phase 1):**

1. Create `gommod/gompkg/commit/` package
2. Implement `CommitPlan` type with save/load
3. Implement `GitIgnoreManager`
4. Add persistence to file selection view (auto-save)
5. Create `DispositionReviewMode` view
6. Implement apply logic

**After Phase 1 MVP works:**

1. Set up Anthropic API client
2. Implement `MockTakesGenerator` for development
3. Design AI prompt and test with real API
4. Implement `TakesSelectionMode` UI
5. Integrate AI generation

---

## Appendix A: File Disposition Semantics

| Disposition | Short | Description | Persisted | Git Action |
|-------------|-------|-------------|-----------|------------|
| **Commit** | `c` | Include in commit | No (cleared after) | Staged and committed |
| **Omit** | `o` | Skip this session | No | No change |
| **GitIgnore** | `g` | Ignore (shared) | Yes (in .gitignore) | Added to .gitignore |
| **GitExclude** | `e` | Ignore (personal) | Yes (in .git/info/exclude) | Added to exclude |
| **Unspecified** | `-` | No decision yet | No | No change |

## Appendix B: Example Take JSON

```json
{
  "takes": [
    {
      "id": "take-1-feature",
      "name": "Feature-Focused",
      "rationale": "Groups changes by user-facing features for clearer git history",
      "changesets": [
        {
          "id": "cs-1",
          "name": "Add file metadata loading",
          "message": "feat(gomtui): add file metadata loading for directory tables\n\nImplements LoadMeta() method on File type to load filesystem\nmetadata (size, mtime, permissions) and SetGitStatus() to enrich\nwith git status information.\n\nEnables directory table view to display file metadata alongside\ndisposition and change status.",
          "files": [
            "gommod/gomtui/file.go",
            "gommod/gomtui/file_metadata.go"
          ],
          "rationale": "These changes implement the data model for file metadata, which is a complete feature on its own"
        },
        {
          "id": "cs-2",
          "name": "Integrate table with metadata display",
          "message": "feat(gomtui): display file metadata in directory table view\n\nUpdates FilesTableModel to show size, modification time,\npermissions, and git status when a directory is selected in\nthe tree view.\n\nIntegrates metadata loading pipeline with table rendering.",
          "files": [
            "gommod/gomtui/files_table_model.go",
            "gommod/gomtui/editor_state.go"
          ],
          "rationale": "These changes integrate metadata into the UI, completing the feature from user perspective"
        },
        {
          "id": "cs-3",
          "name": "Fix disposition handling for focused panes",
          "message": "fix(gomtui): disposition keys now target focused pane\n\nPreviously, disposition keys (c/o/g/e) always applied to tree\nnode regardless of which pane was focused. Now disposition keys\napply to the selected table row when table is focused.\n\nMoves disposition handling from global to pane-specific.",
          "files": [
            "gommod/gomtui/editor_state.go"
          ],
          "rationale": "Bug fix that makes the UI work correctly - separate concern from the features above"
        }
      ]
    }
  ]
}
```

---

**End of Document**
