# Gomion Implementation Plan

## ⚠️ CRITICAL INSTRUCTIONS FOR CLAUDE - READ FIRST

**BEFORE starting ANY work on this project:**

1. **Read this entire PLAN.md file** - This shows what's LEFT to do, not what's already done
2. **Be aware of DONE.md** - This shows what's already completed (don't re-implement!) Only read if you need to.
3. **Review ALL configured skills** - Ensure you're familiar with go-house-rules, error-handling-author, go-dt-filepath-refactorer, etc.
4. **Only work on what's in PLAN.md** - Don't waste tokens reviewing/implementing completed work
5. **When you complete a task:**
   - MOVE the implementation details from PLAN.md to DONE.md
   - DELETE the completed content from PLAN.md
   - Goal: Whittle PLAN.md down to nothing as work progresses

**Why this matters:**
- PLAN.md = What needs to be done (gets smaller)
- DONE.md = What was completed (gets larger)
- Don't rely on conversation summaries - always check PLAN.md first
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

---

## Critical Workflow Decision

**WORKFLOW FLAW IDENTIFIED** in original plan:

1. **Wrong scope**: Getting ALL repo files, need MODULE-scoped files
2. **No file filtering**: No way to exclude files developer doesn't want to commit
3. **Missing human-in-the-loop**: Need File Selection View BEFORE generating takes

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

### Architecture Decisions

#### Filter Functions for Module Scope

**Problem**: Need to filter changed files to module scope, but gitutils shouldn't know about Go modules.

**Solution**:
- gitutils accepts a **filter func** or **iterator** (from dt.WalkDir)
- Module-specific logic lives in **gompkg**
- Keeps gitutils language-agnostic and reusable
- Future-proof for other languages

#### DO-NOT-INCLUDE as Special ChangeSet

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

---

## IMMEDIATE PRIORITY: Phase 2.5 - File Selection View

**Goal:** Module-scoped file filtering with disposition assignment BEFORE generating takes

**Why this comes first:**
- Addresses workflow flaw (wrong scope, no filtering, missing human-in-the-loop)
- Developer decides what NOT to include before AI generates takes
- Module-scoped by default with toggle to full-repo

### Step 1: Implement gitutils Filter Support

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

**Deliverable:**
- [ ] FileFilter type defined in gitutils
- [ ] GetChangedFilesFiltered method implemented

### Step 2: Implement Module-Scoped Filtering

Add to `gompkg`:

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

**Deliverable:**
- [ ] CreateModuleFileFilter function implemented
- [ ] AutoDetectModule function implemented
- [ ] Module detection tested with real go.mod files

### Step 3: Update File Disposition Types

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

**Deliverable:**
- [ ] FileDisposition type and constants defined
- [ ] FileWithDisposition struct created

### Step 4: Build File Selection View UI

Create `gomtui/file_selection_view.go`:

**Layout**: Two/three-pane
- Left: Tree view (BubbleTree)
- Right: File content display
- Top/indicator: File disposition (INCLUDE/DO-NOT-INCLUDE/GITIGNORE/GITEXCLUDE)

**Features**:
- Module-scoped file list (default) with toggle to full-repo
- Auto-detect module from go.mod location
- Filter changed files to module
- Mark files with dispositions
- Persist to `.git/info/commit-files.json` (later - in-memory for now)
- Can switch back to this view anytime

**Integration**:
- Implement BubbleTree
- Two-pane layout with BubbleTea
- File content display
- Disposition toggling with keyboard shortcuts
- Module/repo toggle

**Deliverables:**
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

---

## Phase 3: Basic UI - Takes View

**Goal:** Two-pane UI for browsing Takes and ChangeSets

**Status**: Pending Phase 2.5 completion

**Tasks:**
1. Create `model.go` with bubbletea Init()
2. Create `update.go` with key handlers (t, tab, up/down, enter, q)
3. Create `takes_view.go` for rendering left pane
4. Create `view.go` with layout
5. Create `styles.go` with lipgloss styles

**Deliverables:**
- [ ] Takes list renders in left pane
- [ ] ChangeSets list renders in middle pane
- [ ] Navigation works (up/down, select Take)
- [ ] Selecting a Take updates ChangeSets pane

**Important**: Generate takes ONLY on INCLUDE files from Phase 2.5

---

## Phase 4: Files View - Hunk Refinement

**Goal:** Three-pane diff view for assigning hunks to ChangeSets

**Tasks:**
1. Create `files_view.go` - File tree rendering
2. Create `diff_view.go` - Baseline/Changes panes
3. Parse git diff into hunks
4. Hunk assignment logic with GIT_INDEX_FILE
5. Update View() for FilesView mode
6. Key bindings: Space, 1-9, a, u

**Deliverables:**
- [ ] File tree renders with hunk counts
- [ ] Baseline/Changes panes show code side-by-side
- [ ] Hunks have checkboxes indicating assignment
- [ ] Space toggles hunk assignment
- [ ] GIT_INDEX_FILE correctly set per ChangeSet

---

## Phase 5: ChangeSet Operations

**Goal:** Create, edit, delete, merge ChangeSets

**Tasks:**
1. Create `changeset_manager.go` with CRUD operations
2. Add key bindings: n, e, d, m
3. Use promptkit for dialogs
4. ChangeSet metadata persistence

**Deliverables:**
- [ ] Create new ChangeSet from UI
- [ ] Edit ChangeSet name/rationale
- [ ] Delete ChangeSet with confirmation
- [ ] Merge two ChangeSets into one
- [ ] Metadata saved to `<projectRepo>/.git/info/changesets/<id>/meta.json`

---

## Phase 6: Commit & Persistence

**Goal:** Commit ChangeSets to user repo, handle persistence

**Tasks:**
1. Commit workflow implementation
2. Add key binding: c
3. Save/Load session state
4. Handle exit gracefully
5. Delete ChangeSet after successful commit

**Deliverables:**
- [ ] Commit ChangeSet to user repo successfully
- [ ] Conflicts abort cleanly with error message
- [ ] Session state persists across invocations
- [ ] Committed ChangeSets are deleted (or archived)

---

## Phase 7: Polish & Testing

**Goal:** Error handling, help system, testing

**Tasks:**
1. Help overlay (? key)
2. Error dialogs
3. Terminal size check
4. Unit tests (git_index, changeset_manager, parser)
5. Integration tests
6. Manual testing checklist

**Deliverables:**
- [ ] Help overlay accessible with '?'
- [ ] Error messages clear and actionable
- [ ] Terminal size validated on startup
- [ ] Unit tests pass
- [ ] Integration tests pass

---

## Phase 8: UI Enhancements (Future)

**Goal:** Improve visualization based on real-world usage

**Potential Enhancements:**
- Additional view toggles
- UI/UX improvements from testing
- Performance optimizations for large diffs

---

## Key Files to Create/Modify

**Phase 2.5 (File Selection View):**
- `gommod/gitutils/working.go` - Add FileFilter support
- `gommod/gompkg/module.go` - Module detection and filtering (NEW)
- `gommod/gomtui/types.go` - Add FileDisposition types
- `gommod/gomtui/file_selection_view.go` - File selection UI (NEW)

**Phase 3+ (GRU TUI):**
- `gommod/gomtui/model.go`
- `gommod/gomtui/update.go`
- `gommod/gomtui/view.go`
- `gommod/gomtui/takes_view.go`
- `gommod/gomtui/files_view.go`
- `gommod/gomtui/diff_view.go`
- `gommod/gomtui/changeset_manager.go`
- `gommod/gomtui/styles.go`

---

## Data Flow

```
1. Launch gru
   ↓
2. Open user repo + cached repo
   ↓
3. [Phase 2.5] File Selection View - module-scoped filtering
   ↓
4. Load or generate Takes (AI) - ONLY on INCLUDE files
   ↓
5. User selects Take → Creates ChangeSets
   ↓
6. User toggles to Files view
   ↓
7. User assigns hunks to ChangeSets (via GIT_INDEX_FILE)
   ↓
8. User commits ChangeSet
   ↓
9. Apply patch to user repo → git commit
   ↓
10. Mark ChangeSet as committed, delete index
```

---

## Key Design Decisions

1. **Filter Functions for Module Scope** - Keep gitutils language-agnostic, module logic in gompkg
2. **DO-NOT-INCLUDE as ChangeSet** - Reuse existing data structures, no special cases
3. **File Selection First** - User decides exclusions before AI generates takes
4. **Delete After Commit** - No clutter, committed work is done
5. **Per-ChangeSet Git Index** - Isolated staging via GIT_INDEX_FILE
6. **Cached Worktree** - User repo untouched until commit

---

## Build & Test Commands

```bash
cd ~/Projects/gomion

# Build
make build

# Build specific modules
cd gommod && GOEXPERIMENT=jsonv2 go build ./...

# Run
./bin/gomion

# Test
cd gommod/gompkg && GOEXPERIMENT=jsonv2 go test -v ./...
cd gommod/gitutils && GOEXPERIMENT=jsonv2 go test -v ./...
```

---

## Next Steps

**IMMEDIATE**: Phase 2.5 - File Selection View
1. Step 1: Implement gitutils FileFilter support
2. Step 2: Implement module-scoped filtering in gompkg
3. Step 3: Update file disposition types in gomtui
4. Step 4: Build File Selection View UI

**THEN**: Phase 3 - Takes View (generate takes ONLY on INCLUDE files)

**THEN**: Phase 4 - Hunk Assignment View

---

## Questions to Resolve

1. ~~Module filter in gompkg or gomcfg?~~ → **gompkg (business logic)**
2. File disposition persistence? → **`.git/info/commit-files.json`** (later, in-memory for now)
3. Tree view alternatives? → TBD during implementation

---

## Reference Documentation

**See DONE.md for:**
- Phase 1: Foundation & Data Structures (completed)
- Phase 2: Takes Generation & Loading (completed)
- Modal Menu System implementation (completed)

**See CLAUDE.md for:**
- Coding conventions
- Required packages
- Project purpose and goals
