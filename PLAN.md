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

**File Dispositions** (Terminology Update):
- `COMMIT` (formerly INCLUDE) - Will be committed (generate takes for these)
- `OMIT` (formerly DO-NOT-INCLUDE) - Skip for now, will appear next session
- `IGNORE` (formerly GITIGNORE) - Add to `.gitignore` (project-level, deferred)
- `EXCLUDE` (formerly GITEXCLUDE) - Add to `.git/info/exclude` (personal, immediate)

---

## Phase 2.1: Files Table Improvements

**Goal**: Fix table layout, sizing, scrolling, and visual selection

**Priority**: High (before exclude/ignore work)

### Issues to Address

1. **Dynamic filename column width**
   - Currently: Fixed 40 columns, flush left
   - Needed: Auto-size to longest filename + 2 chars padding (1 left, 1 right)

2. **Table auto-sizing to fill screen**
   - Currently: Table doesn't resize to viewport
   - Needed: Table fills available space like file content viewport
   - **Critical**: Ensures alerts always appear in same location (top right corner)

3. **Horizontal scrolling**
   - Currently: Table clips if too wide for screen
   - Needed: Left-right scrolling for long paths/many columns
   - **Keybinding**: TBD (arrow keys when in table? Shift+arrow?)

4. **Selection indicator without obscuring cell colors**
   - Currently: Row-wide selection background obscures disposition colors
   - Needed: Clear selection that preserves Plan/Change column colors

5. **Remove staging status, keep change status**
   - Currently: Shows "Status" column (staging status)
   - Needed: Remove staging status (irrelevant - Gomion manages staging)
   - **Keep**: "Change" column (M/A/D/U - git change type)

### Column Order (Updated)

**Current**: `# Plan Filename Status Change Size Modified Perms Mode Flags`

**New**: `# Plan Filename Change Size Modified Perms Mode Flags`

- Removed: "Status" (staging status - not relevant)
- Kept: "Change" (M/A/D/U - important context)

### Selection Visual Design

**Approach**: Use `▶` character for visual consistency with tree view

**Components**:
1. **Triangle indicator** (`▶`) in leftmost position before row number
2. **Bold text** for entire selected row
3. **Reverse video** on row number column

**Example**:
```
Selected:   ▶ [3] COMMIT main.go      M  1.2KB  2026-01-07 10:23  ...
Unselected:   [4] OMIT   test.go     A  0.5KB  2026-01-07 09:15  ...
```
(where [3] is reverse video, and the selected row is bold)

**Rationale**:
- **Visual consistency**: `▶` is already used in tree view for expansion/selection
- **Familiar**: Users already associate `▶` with "current/active" from tree navigation
- **No extra lines**: Single character indicator, clean and simple
- **Cell colors preserved**: Plan and Change columns show their semantic colors

### Implementation Tasks

**Phase 2.1a: Column Changes** ✅ COMPLETED
- [x] Remove "Status" column from table model
- [x] Reorder columns: `# Plan Filename Change Size Modified Perms Mode Flags`
- [x] Update column headers
- [x] Update row rendering logic

**Phase 2.1b: Dynamic Filename Column** ✅ COMPLETED
- [x] Calculate longest filename in current file list
- [x] Set filename column width = max(len(longest_filename) + 2, min_width)
- [x] Update on file list changes (handled in SetSize)
- [x] Add 1 char padding left, 1 char padding right

**Phase 2.1c: Table Auto-Sizing** ✅ COMPLETED
- [x] Make table component respect viewport bounds (already implemented with WithTargetWidth/WithMinimumHeight)
- [x] Calculate available height (handled by caller)
- [x] Resize table to fill available space
- [x] Update on terminal resize events (SetSize method)
- [x] Ensure alerts stay in top-right corner (table sizing handles this)
- [x] Fix width calculations to reach right edge of screen (RightPaneInnerWidth adjusted from -10 to -6)

**Phase 2.1d: Horizontal Scrolling** ✅ COMPLETED
- [x] Detect when table width exceeds viewport width (handled by bubble-table)
- [x] Implement horizontal scroll offset (handled by bubble-table with WithMaxTotalWidth)
- [x] Add keybindings for left-right scroll (left/right arrows with custom keymap)
- [x] Freeze first column (#) when scrolling (WithHorizontalFreezeColumnCount)
- [x] Implement cell-level cursor tracking (currentColumn field)
- [x] Left/right arrows move cell cursor, not just scroll table
- [ ] Show scroll indicators (e.g., "«" and "»" when scrolled) - Deferred (bubble-table provides "<" and ">" indicators)
- [ ] Auto-scroll when cell cursor navigates off-screen - Deferred (cell stays on-screen for now)
- [ ] Smooth scrolling behavior - Deferred (bubble-table may handle)

**Phase 2.1e: Selection Visual Indicator** ✅ COMPLETED
- [x] Add triangle indicator column (1 char wide: `▶`)
- [x] Render `▶` for selected row, space for others
- [x] Adjust # column width: left-pad single digits, no padding for double digits
  - Single digit: `│ ▶ 1 │` (space before digit)
  - Double digit: `│ ▶10 │` (no space before digits)
- [x] Update indicator when selection changes (on navigation)
- [x] Preserve cell background colors (Plan column, Change column)
- [x] Apply bold text style to entire selected row (with brighter colors)
- [x] Apply reverse video to current cell (cell-level highlighting)
- [x] Implement cell-level selection tracking (currentColumn field in FilesTableModel)
- [x] Left/right arrows move cell cursor within selected row

**Implementation Notes**:
- Row-level styling: Bold + brighter colors for entire selected row
- Cell-level styling: Reverse video applied to current cell only
- Two-level highlighting: Selected row (bold+bright) + current cell (reverse video)
- Cell cursor movement handled independently from bubble-table's row selection

---

## Exclude/Ignore Workflow Implementation

### Git Behavior Facts

**Both `.gitignore` and `.git/info/exclude` take effect immediately** (even uncommitted):
- **`.gitignore`**: Project-level, committed, shared with team
- **`.git/info/exclude`**: Personal, never committed, only affects you
- **Global config**: Personal, affects all repos, **out of scope for modification**

### Phase 2.5: Core Exclude/Ignore Functionality

**Goal**: Implement basic exclude/ignore with proper timing and visibility

**Tasks**:
- [ ] Update file disposition enum: COMMIT, OMIT, IGNORE, EXCLUDE
- [ ] Implement `.git/info/exclude` writer in gitutils
  - Append pattern (don't duplicate if already exists)
  - Atomic write with temp file
  - Error handling for permission issues
- [ ] Implement `.gitignore` modifier in gitutils
  - Append pattern to `.gitignore`
  - Create `.gitignore` if doesn't exist
  - Atomic write with temp file
- [ ] Apply EXCLUDE immediately when disposition set
- [ ] Apply IGNORE immediately (modify `.gitignore` file)
- [ ] Ensure excluded files stay visible during session
- [ ] Update commit plan persistence to handle new dispositions
- [ ] Test manual "undo" (change disposition back to COMMIT/OMIT)

**Application Workflow**:
- **EXCLUDE**: Write to `.git/info/exclude` immediately, keep visible
- **IGNORE**: Modify `.gitignore`, treat as uncommitted change, commit with code
- **OMIT**: Don't apply anything, file stays in working tree

### Phase 2.6: Show Excluded/Ignored Files

**Goal**: Display files affected by ignore/exclude rules

**Tasks**:
- [ ] Implement `git ls-files --others --ignored --exclude-standard -z` in gitutils
- [ ] Implement `git check-ignore -v -z --stdin` batch checker in gitutils
- [ ] Parse check-ignore output (source file, line, pattern)
- [ ] Add toggle keybinding for "show excluded/ignored" (e.g., `Ctrl-e`)
- [ ] Add indicator when excluded/ignored files exist (status line or header)
- [ ] Display excluded/ignored files in file list with disposition markers
- [ ] Allow un-exclude operation (remove from `.git/info/exclude`)
- [ ] Allow un-ignore operation (remove from `.gitignore`)
- [ ] Detect global ignore matches, show info message

**Reference**: See ChatGPT algorithm in conversation history for performant implementation

### Phase 2.7: File Info Popup/Overlay

**Goal**: Simple overlay showing file-specific metadata and context

**Motivation**: UI has limited space for verbose metadata. Popup provides details on demand.

**Tasks**:
- [ ] Create file info popup component (1/2 to 2/3 viewport, centered)
- [ ] Add hotkey to show popup (e.g., `i` for "info")
- [ ] ESC to clear/close popup
- [ ] Display in popup:
  - File path (full and relative)
  - Git status (modified, staged, untracked)
  - Ignore/exclude attribution (if applicable):
    - `excluded: .git/info/exclude:12 "*.log"`
    - `ignored: .gitignore:8 "dist/"`
    - `ignored: global:3 ".DS_Store"` (with note: must edit manually)
  - File metadata (size, modified time, permissions)
  - Current disposition (COMMIT/OMIT/IGNORE/EXCLUDE)
- [ ] Style with lipgloss (border, padding, title)
- [ ] Test with various file states
- [ ] Handle case where file has no special metadata (just show basics)

**Future Enhancement**: See ROADMAP.md "Enhanced File Viewer" for full-featured version

### Phase 2.8: General Undo/Redo

**Goal**: Session-level undo for all disposition changes

**Note**: Not on critical path for exclude/ignore. Manual "undo" works via disposition change.

**Tasks**:
- [ ] Define `Operation` interface (Do/Undo/Description)
- [ ] Implement `UndoStack` struct
- [ ] Implement `DispositionChangeOp` with Do/Undo
- [ ] Add undo keybinding (`Ctrl-z`)
- [ ] Add redo keybinding (`Shift-Ctrl-z`)
- [ ] Add visual feedback (toast/status message)
- [ ] Clear stack on TUI exit
- [ ] Clear stack after successful commit
- [ ] Test undo/redo across all disposition types

**Scope**: Session-level only (not cross-session)

---

## Phase 3: Basic UI - Takes View

**Goal:** Two-pane UI for browsing Takes and ChangeSets

**Status**: Ready to implement (Phase 2.5 complete)

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
