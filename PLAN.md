# Modal Menu System Implementation Plan

## Overview

Transform go-cliutil's hierarchical menu system into a **modal system** where users switch between modes using Ctrl+number keys. Integrate this into Squire's commit workflow with staging plan-based workflow, line-level control, and AI-assisted commit message generation.

**Key Difference:** Current system is hierarchical (nested function calls). New system is modal (flat mode registry with Ctrl+N switching between ANY mode at ANY time).

## Architecture Summary

### Modal Menu System (go-cliutil)

**Core Components:**
1. **KeyPress detection** - Extend `terminal.go` to detect Ctrl+number sequences
2. **ModeManager** - Registry and coordinator for all modes
3. **Extended MenuMode interface** - Add modal methods to existing MenuMode interface
4. **ShowMultiModeMenu()** - Main loop for modal navigation

**Display Format:**
```
Mode Toggle: [^1] Explore [^2] Manage [^3] Compose — [^0] Main
Actions:     [1] Action1 [2] Action2 — [0] Help [9] Quit
Choice:
```

### Squire Integration

**Four Modes:**
- **Main (^0):** [1] Commit code — starting mode
- **Explore (^1):** [1] Status [2] Breaking [3] Other Changes [4] Tests — read-only exploration
- **Manage (^2):** [1] Stage [2] Unstage [3] Plan [4] Split — staging plan-based workflow
- **Compose (^3):** [1] Staged [2] Generate [3] List [4] Merge [5] Edit — commit message workflow

**Data Model:**
- **Staging Plans** - Categorized changes with line-level hunk info (`.squire/plans/`)
- **Commit Candidates** - AI-generated messages with staging hash (`.squire/candidates/`)
- **Staging Snapshots** - Safety net for undo (`.squire/snapshots/`, auto-archive 30 days)
- **AI Plan Takes** - 3 different perspectives on staging plans (`~/.cache/squire/analysis/`)

**Key Workflow:**
1. Explore changes → 2. Manage: AI plans → Split (assign lines) → Stage plan → 3. Compose: Generate message → 4. Main: Commit

## Implementation Phases

### Phase 1: cliutil Foundation

**Goal:** Functional modal menu with mode switching

**Tasks:**
1. Extend `~/Projects/go-pkgs/go-cliutil/terminal.go`
   - Add `KeyPress` type (Rune, Ctrl, Alt flags)
   - Implement `ReadKeyPress()` to detect Ctrl+0-9
   - Ctrl+1 = byte 0x01, plain '1' = byte 0x31
   - Unit tests for Ctrl detection

2. Create `~/Projects/go-pkgs/go-cliutil/mode_manager.go`
   ```go
   type ModeManager struct {
       modes       map[int]MenuMode     // 0-9
       currentMode int
       state       ModeState            // User-defined shared state
       exitLoop    bool
   }

   Methods:
   - NewModeManager(state ModeState) *ModeManager
   - RegisterMode(id int, mode MenuMode) error
   - SwitchMode(id int) error  // Calls OnExit/OnEnter
   - CurrentMode() MenuMode
   - RequestExit()
   - ShouldExit() bool
   ```

3. Extend `~/Projects/go-pkgs/go-cliutil/menu.go` (existing MenuMode interface)
   ```go
   // Extend existing MenuMode interface with modal methods
   type MenuMode interface {
       // Existing methods
       MenuOptions() []MenuOption
       Handle(args *OptionHandlerArgs) error
       ShouldExit() bool
       RequestExit()

       // New modal methods
       ModeID() int                        // 0-9 for mode registry
       ModeName() string                   // "Main", "Explore", etc.
       OnEnter(state ModeState) error      // Called when switching TO this mode
       OnExit(state ModeState) error       // Called when switching FROM this mode
   }

   // ModeState is user-defined interface for shared state
   type ModeState interface {
       // Application defines this (e.g., SquireModeState in squiresvc)
   }
   ```

4. Create `~/Projects/go-pkgs/go-cliutil/modal_menu.go`
   ```go
   type MultiModeMenuArgs struct {
       Manager *ModeManager
       Writer  io.Writer
   }

   func ShowMultiModeMenu(args MultiModeMenuArgs) error
   ```

5. Main loop in `ShowMultiModeMenu()`:
   - Display mode toggle bar + current mode's actions
   - `ReadKeyPress()`
   - If Ctrl+N → `SwitchMode(N)`
   - If plain '9' → exit
   - If plain '0' → help
   - If plain 1-8 → `Handle(index)`
   - Loop until exit

**Deliverable:** Working demo in `go-cliutil/examples/modal_demo/`

**Risk:** Ctrl detection may fail on some terminals
**Mitigation:** Test on macOS Terminal, iTerm2, Linux early; document supported terminals

### Phase 2: Squire Data Model

**Goal:** Persistence layer for staging plans, candidates, snapshots

**Note:** Merging `commitmsg` package into `squiresvc` - single location for service layer logic.

**Tasks:**
1. Merge `commitmsg` into `squiresvc`
   - Move all files from `squirepkg/commitmsg/` to `squirepkg/squiresvc/`
   - Remove duplicate `doterr.go` (keep one copy)
   - Fix `os/exec` usage in workflow.go (use `gitutils` instead)
   - Update import in `squirescliui/modes.go`
   - **API Reconsideration:** Current GenerateWithAnalysis → GenerateMessage → GenerateCommitMessage chain needs redesign for modal context
   - **Naming Updates:** Request/Result → CommitMessageRequest/CommitMessageResponse
   - **Privacy:** BuildPrompt/ParseResponse → private buildCommitMessagePrompt/parseCommitMessageResult
   - Note: Final API design deferred until modal workflow is clearer

2. Add to `~/Projects/squire/squirepkg/squiresvc/staging_plan.go` (new file)
   ```go
   type StagingPlan struct {
       ID          string              `json:"id"`          // UUID
       Name        string              `json:"name"`
       Description string              `json:"description"`
       Created     time.Time           `json:"created"`
       Modified    time.Time           `json:"modified"`
       Files       []FilePatchRange    `json:"files"`
       Suggested   bool                `json:"suggested"`   // AI vs user
       TakeNumber  int                 `json:"take_number"` // 1-3, or 0 if user
   }

   type FilePatchRange struct {
       Path         dt.RelFilepath     `json:"path"`
       HunkHeaders  []HunkHeader       `json:"hunk_headers"`
       AllLines     bool               `json:"all_lines"`   // If true, entire file
   }

   // Store hunk headers + context (not just line numbers)
   // Line numbers shift as edits happen - need context for stable application
   type HunkHeader struct {
       OldStart     int                `json:"old_start"`
       OldCount     int                `json:"old_count"`
       NewStart     int                `json:"new_start"`
       NewCount     int                `json:"new_count"`
       HeaderLine   string             `json:"header_line"`  // The @@ line
       ContextLines []string           `json:"context_lines"` // Surrounding context
   }

   Operations:
   - SaveStagingPlan(moduleDir dt.DirPath, plan *StagingPlan) error
       → .squire/plans/{id}.json
   - LoadStagingPlan(moduleDir dt.DirPath, id string) (*StagingPlan, error)
   - ListStagingPlans(moduleDir dt.DirPath) ([]*StagingPlan, error)
   - DeleteStagingPlan(moduleDir dt.DirPath, id string) error
   ```

3. Add to `~/Projects/squire/squirepkg/squiresvc/commit_candidate.go` (new file)
   ```go
   type CommitCandidate struct {
       ID             string         `json:"id"`
       Message        string         `json:"message"`
       StagingHash    string         `json:"staging_hash"`    // SHA256 of staged files/hunks
       AnalysisHash   string         `json:"analysis_hash"`
       Created        time.Time      `json:"created"`
       Modified       time.Time      `json:"modified"`
       AIProvider     string         `json:"ai_provider"`
       AIModel        string         `json:"ai_model"`
       PlanID         string         `json:"plan_id"`
       Archived       bool           `json:"archived"`
   }

   Operations:
   - SaveCommitCandidate(moduleDir dt.DirPath, candidate *CommitCandidate) error
       → .squire/candidates/{id}.json
   - LoadCommitCandidate(moduleDir dt.DirPath, id string) (*CommitCandidate, error)
   - ListActiveCandidates(moduleDir dt.DirPath) ([]*CommitCandidate, error)
   - ArchiveCandidate(moduleDir dt.DirPath, id string) error
       → .squire/.archive/candidates/{id}.json
   ```

4. Add to `~/Projects/squire/squirepkg/squiresvc/staging_snapshot.go` (new file)
   ```go
   type StagingSnapshot struct {
       ID         string              `json:"id"`
       Timestamp  time.Time           `json:"timestamp"`
       Label      string              `json:"label"`
       Files      []SnapshotFile      `json:"files"`
       Hash       string              `json:"hash"`
   }

   type SnapshotFile struct {
       Path         dt.RelFilepath     `json:"path"`
       StagedAll    bool               `json:"staged_all"`
       HunkHeaders  []HunkHeader       `json:"hunk_headers"`
   }

   Operations:
   - CreateSnapshot(moduleDir dt.DirPath, label string) (*StagingSnapshot, error)
       - Calls `git diff --cached --unified=0` for hunks
       - Parses @@ headers: @@ -oldStart,oldCount +newStart,newCount @@
       - Stores context lines for stable application
       → .squire/snapshots/{id}.json
   - LoadSnapshot(moduleDir dt.DirPath, id string) (*StagingSnapshot, error)
   - ListActiveSnapshots(moduleDir dt.DirPath) ([]*StagingSnapshot, error)
   - RestoreSnapshot(moduleDir dt.DirPath, id string) error
       - Applies snapshot to staging via git
   - ArchiveOldSnapshots(moduleDir dt.DirPath, daysOld int) error
   ```

5. Add to `~/Projects/squire/squirepkg/squirecfg/staging_plan_takes.go` (new file)
   ```go
   // AI-generated staging plan alternatives
   type StagingPlanTakes struct {
       CacheKey   string              `json:"cache_key"`
       Timestamp  time.Time           `json:"timestamp"`
       Takes      []StagingPlanTake   `json:"takes"`      // 3 different perspectives
   }

   type StagingPlanTake struct {
       Number     int                 `json:"number"`     // 1, 2, or 3
       Theme      string              `json:"theme"`      // "By Feature", "By Layer", "By Risk"
       Plans      []TakePlan          `json:"plans"`
   }

   type TakePlan struct {
       Name       string              `json:"name"`
       Rationale  string              `json:"rationale"`
       Files      []dt.RelFilepath    `json:"files"`      // File-level (user refines to hunks)
   }

   Operations:
   - SaveStagingPlanTakes(cacheKey string, takes *StagingPlanTakes) error
       → ~/.cache/squire/analysis/{key}-takes.json
   - LoadStagingPlanTakes(cacheKey string) (*StagingPlanTakes, error)
   - ClearStagingPlanTakes(cacheKey string) error
   ```

6. Add `.squire/` directory management
   - Auto-create on first use
   - Add to `.git/info/exclude`: `squire.json`
   - Add `.squire/.archive/` to `.gitignore`
   - Commit `.squire/` structure (empty dirs) to git

**Deliverable:** Persistence layer with unit tests for JSON round-trips

### Phase 3: Squire Modes (Text-Based)

**Goal:** Four modes working without complex UIs

**Tasks:**
1. Add to `~/Projects/squire/squirepkg/squiresvc/mode_state.go` (new file)
   ```go
   // ModeState implementation for squire - shared state across all modes
   type SquireModeState struct {
       ModuleDir        dt.DirPath
       Writer           cliutil.Writer
       Logger           *slog.Logger

       // Cached state (refreshed by OnEnter)
       StagedFiles      []dt.RelFilepath
       UnstagedFiles    []dt.RelFilepath
       UntrackedFiles   []dt.RelFilepath
       AnalysisResults  *precommit.Results
       AnalysisCacheKey string

       // Working state
       ActivePlanID     string
       ActivePlans      []*StagingPlan
       ActiveCandidates []*CommitCandidate

       // AI agent
       AIAgent          *askai.Agent
   }
   ```

   **Why ModeState (not ModeContext)?**
   - Current system captures state in closures (see modes.go line 157: `message := args.Message`)
   - Problem: State scattered in closures, hard to test, can't share between modes
   - Solution: All state lives in SquireModeState struct
   - Benefits:
     - **Testable:** Can create mock SquireModeState in tests
     - **Discoverable:** All state in one place, easy to see what exists
     - **Sharable:** All modes access same state object, no duplicate data

2. Add to `~/Projects/squire/squirepkg/squiresvc/main_mode.go` (new file)
   - ModeID: 0, ModeName: "Main"
   - Actions: [1] Commit code — [0] Help [9] Quit
   - OnEnter: Display welcome, staging summary, suggest next mode
   - [1] Commit: Check staging not empty, check candidate exists, show confirmation, execute `git commit`

3. Add to `~/Projects/squire/squirepkg/squiresvc/explore_mode.go` (new file)
   - ModeID: 1, ModeName: "Explore"
   - Actions: [1] Status [2] Breaking [3] Other Changes [4] Tests
   - OnEnter: Refresh staging area, refresh analysis
   - All actions read-only (display info using existing gitutils, precommit packages)

4. Add to `~/Projects/squire/squirepkg/squiresvc/manage_mode.go` (new file)
   - ModeID: 2, ModeName: "Manage"
   - Actions: [1] Stage [2] Unstage [3] Plan [4] Split (future)
   - OnEnter: Load active plans from `.squire/plans/`
   - [1] Stage: Exclusive staging (unstage all → stage plan's files/hunks) → create snapshot
   - [2] Unstage: `git reset` → create snapshot
   - [3] Plan: AI staging plan takes → text-based selection (no MM UI yet) → save plans
   - [4] Split: Placeholder (Phase 5)

5. Add to `~/Projects/squire/squirepkg/squiresvc/compose_mode.go` (new file)
   - ModeID: 3, ModeName: "Compose"
   - Actions: [1] Staged [2] Generate [3] List [4] Merge [5] Edit
   - OnEnter: Load active candidates, refresh staging hash
   - [1] Staged: Display `git diff --cached --stat`
   - [2] Generate: Call commit message generation → create CommitCandidate → save
   - [3] List: Show all active candidates, mark stale (staging hash mismatch)
   - [4] Merge: Placeholder (future)
   - [5] Edit: Open candidate in $EDITOR, update Modified timestamp

6. Update `~/Projects/squire/squirepkg/squirecmds/next_cmd.go`
   - Create SquireModeState (populate from engine.Result)
   - Create ModeManager(state)
   - Register 4 modes (Main, Explore, Manage, Compose)
   - Set initial mode to Main (0)
   - Replace `cliutil.ShowMenu()` with `cliutil.ShowMultiModeMenu()`

**Deliverable:** Functional modal workflow (text-based plan selection, text confirmations)

### Phase 4: Staging Plan Selector (MM UI)

**Goal:** Visual take selection and manual editing

**Tasks:**
1. Add bubbletea dependency to squire
2. Create `~/Projects/squire/squirepkg/squireminiui/plan_selector.go`
   ```go
   func ShowPlanSelector(takes *squirecfg.StagingPlanTakes, writer io.Writer) ([]*squiresvc.StagingPlan, error)
   ```
   - **Two-pane UI:**
     - Left pane: Selectable list of staging plans (Take 1, Take 2, Take 3, Custom)
     - Right pane: View of files/changes in the selected plan
   - Key bindings: [↑↓] navigate, [Enter] select, [e] edit manually, [q] cancel
   - Manual edit mode: Text editor (bubbletea) with takes concatenated
   - Parse edited text → create StagingPlans → return

3. Integrate into Manage mode [3] Plan action
   - Replace text selection with `ShowPlanSelector()`
   - Returns selected/edited plans
   - Save to `.squire/plans/`

**Deliverable:** Visual staging plan selection UI

**UI Layout:**
```
╔════════════════════════════════════════════════════════════╗
║ STAGING PLAN SELECTOR                                      ║
╠══════════════════════╦═════════════════════════════════════╣
║ PLANS                ║ SELECTED PLAN DETAILS               ║
╟──────────────────────╫─────────────────────────────────────╢
║ > Take 1: By Feature ║ Plan: Add user authentication       ║
║   Take 2: By Layer   ║ Rationale: Cohesive auth changes    ║
║   Take 3: By Risk    ║                                     ║
║   Custom...          ║ Files (3):                          ║
║                      ║   • auth.go                         ║
║                      ║   • user.go                         ║
║                      ║   • middleware.go                   ║
║                      ║                                     ║
║                      ║ Plan: Update logging                ║
║                      ║ Files (2):                          ║
║                      ║   • logger.go                       ║
║                      ║   • config.go                       ║
╠══════════════════════╩═════════════════════════════════════╣
║ [↑↓] Navigate  [Enter] Select  [e] Edit  [q] Cancel        ║
╚════════════════════════════════════════════════════════════╝
```

### Phase 5: Split UI (MM UI - Future)

**Goal:** JetBrains-style hunk assignment

**Tasks:**
1. Create `~/Projects/squire/squirepkg/squireminiui/split_editor.go`
   - Three-pane layout: Files (left) | Baseline (middle) | Changes (right)
   - Hunk parsing: `git diff --cached --unified=0` → parse @@ headers
   - Checkbox per hunk for plan assignment
   - Key bindings: [↑↓] navigate, [Space] toggle, [g] change plan, [s] save

2. Integrate into Manage mode [4] Split action
   - Launch Split UI
   - User assigns hunks to plans
   - Save plans to `.squire/plans/`

**Deliverable:** Visual line-level staging plan assignment

**UI Layout:**
```
╔════════════════════════════════════════════════════════╗
║ SPLIT EDITOR             Active Plan: Feature A        ║
╠══════════╦═══════════════╦══════════════════════════════╣
║ FILES    ║ BASELINE      ║ CHANGES                      ║
╟──────────╫───────────────╫──────────────────────────────╢
║ > auth.go║ func Login()  ║ func Login(user *User) [✓]   ║
║   user.go║   // TODO     ║   if user == nil {      [✓]  ║
║          ║ }             ║     return ErrNilUser   [✓]  ║
╚══════════╩═══════════════╩══════════════════════════════╝
```

**Risk:** Complex UI, time-consuming
**Mitigation:** Defer to Phase 5 (optional)

### Phase 6: Polish

**Goal:** Refinements and nice-to-haves

**Tasks:**
- Commit Confirmation MM UI (replace text Y/N)
- Merge candidates feature (Compose [4])
- Snapshot restore feature (undo staging changes)
- Archive cleanup (auto-archive old snapshots/candidates after 30 days)
- Mode hints ("Try ^2 Manage next")
- Color/highlighting in toggle bar (highlight current mode)

## Critical Files

**go-cliutil changes:**
- `~/Projects/go-pkgs/go-cliutil/terminal.go` - Add KeyPress type, ReadKeyPress()
- `~/Projects/go-pkgs/go-cliutil/menu.go` - Extend MenuMode interface with modal methods
- `~/Projects/go-pkgs/go-cliutil/modal_menu.go` (new) - ShowMultiModeMenu() implementation
- `~/Projects/go-pkgs/go-cliutil/mode_manager.go` (new) - ModeManager, registration, switching

**Squire data model (squiresvc):**
- `~/Projects/squire/squirepkg/squiresvc/staging_plan.go` (new) - StagingPlan, FilePatchRange, HunkHeader
- `~/Projects/squire/squirepkg/squiresvc/commit_candidate.go` (new) - CommitCandidate, persistence
- `~/Projects/squire/squirepkg/squiresvc/staging_snapshot.go` (new) - StagingSnapshot, git integration
- `~/Projects/squire/squirepkg/squiresvc/workflow.go` - Merged from commitmsg, API redesign needed
- `~/Projects/squire/squirepkg/squiresvc/generator.go` - Merged from commitmsg
- `~/Projects/squire/squirepkg/squiresvc/types.go` - Merged from commitmsg (rename Request/Result)

**Squire config (squirecfg):**
- `~/Projects/squire/squirepkg/squirecfg/staging_plan_takes.go` (new) - StagingPlanTakes, AI integration

**Squire modes (squiresvc):**
- `~/Projects/squire/squirepkg/squiresvc/mode_state.go` (new) - SquireModeState (shared state)
- `~/Projects/squire/squirepkg/squiresvc/main_mode.go` (new) - Main mode (^0)
- `~/Projects/squire/squirepkg/squiresvc/explore_mode.go` (new) - Explore mode (^1)
- `~/Projects/squire/squirepkg/squiresvc/manage_mode.go` (new) - Manage mode (^2)
- `~/Projects/squire/squirepkg/squiresvc/compose_mode.go` (new) - Compose mode (^3)

**Integration:**
- `~/Projects/squire/squirepkg/squirecmds/next_cmd.go` - Create ModeManager, register modes
- `~/Projects/squire/squirepkg/squirescliui/modes.go` - Update import (commitmsg → squiresvc)

**Squire MM UIs:**
- `~/Projects/squire/squirepkg/squireminiui/plan_selector.go` (new, Phase 4) - Two-pane plan selection
- `~/Projects/squire/squirepkg/squireminiui/split_editor.go` (new, Phase 5) - Hunk assignment UI

## Key Design Decisions

**Why modal vs hierarchical?**
- Modal: Switch between ANY mode at ANY time (flat registry)
- Hierarchical: Nested function calls (current system)
- Modal supports complex workflows without deep nesting

**Why Ctrl+number?**
- Distinguishes mode switching (Ctrl+N) from actions (plain N)
- Works in raw terminal mode (ASCII control codes)
- Intuitive modifier concept

**Why store hunk headers + context (not just line numbers)?**
- Line numbers shift as edits happen
- Git hunks use @@ -oldStart,oldCount +newStart,newCount @@ format
- Context lines allow stable application even after upstream changes
- Follows git's unified diff format

**Why .squire/ instead of ~/.cache/?**
- Staging plans/candidates/snapshots are repo-specific
- Should be version controlled (tracked in git)
- Closer to git staging semantics

**Why ModeState (not ModeContext)?**
- Avoids confusion with Go's `context.Context` (cancellation/deadlines/values)
- Current system uses closure-captured state (anti-pattern)
- ModeState centralizes all shared state in one discoverable struct
- Enables testing (mock state), sharing (modes access same object), discovery (all state visible)

**Why merge commitmsg into squiresvc?**
- No import cycles (verified)
- No naming conflicts (except duplicate doterr.go - remove one)
- Only 1 file needs import update (modes.go)
- Reduces package proliferation
- Service layer functions belong together

## Testing Strategy

**Unit Tests:**
- cliutil: Ctrl detection, mode switching, exit handling
- squiresvc: JSON round-trips, file I/O, cache keys, mode initialization
- Modes: Action handlers (mock git/AI calls)

**Integration Tests:**
- Full workflow: Main → Explore → Manage → Compose → Main
- Staging plan lifecycle: AI takes → selection → split → stage
- Candidate lifecycle: Generate → edit → commit → archive

**Manual Tests:**
- Terminal compatibility (macOS Terminal, iTerm2, Linux)
- Real commit scenarios with hunk-level staging
- Multi-commit workflows (multiple plans)

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Ctrl detection fails on some terminals | Test early on macOS/Linux, document supported terminals, consider F-key fallback |
| Bubble Tea complexity delays MM UIs | Phases 1-3 work without UIs (text-based), defer complex UIs to Phases 4-5 |
| Data schema changes require migration | Version all schemas (StagingPlanV1), write ADR for changes, provide migration commands |
| Git hunk parsing complexity | Store hunk headers + context (not just line numbers), use git apply for precision |
| ModeState out of sync with git | Refresh on OnEnter, create snapshots before/after changes, show drift warnings |
| commitmsg API unclear for modal workflow | Defer final API design until modal workflow is implemented, mark for reconsideration |

## Notes for Implementation

**commitmsg → squiresvc merge:**
- Remove duplicate doterr.go
- Fix os/exec usage (use gitutils instead)
- Request/Result → CommitMessageRequest/CommitMessageResponse
- BuildPrompt/ParseResponse → private buildCommitMessagePrompt/parseCommitMessageResult
- GenerateWithAnalysis/GenerateMessage/GenerateCommitMessage chain needs redesign
- Final API deferred until modal workflow context is clearer

## Next Steps

1. Begin Phase 1: cliutil foundation
2. Implement KeyPress detection in terminal.go
3. Create ModeManager and extend MenuMode interface
4. Build modal_demo example to validate design
5. Iterate based on demo feedback before proceeding to Phase 2
