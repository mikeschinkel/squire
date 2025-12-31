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

## Overview

Gomion's modal commit workflow with staging plan-based workflow, line-level control, and AI-assisted commit message generation.

**Modal System:** Flat mode registry with F-key switching between ANY mode at ANY time.

## Architecture Summary

### Modal Menu System (go-cliutil) - ✅ COMPLETE

**Display Format:**
```
Main Menu: [f3] Explore [f4] Manage [f5] Compose — [f1] Help
Actions:   [1] Commit — [0] help [9] quit
Choice:
```

### Gomion Integration

**Four Modes:**
- **Main (F2):** [1] Commit code — starting mode
- **Explore (F3):** [1] Status [2] Breaking [3] Other Changes [4] Tests — read-only exploration
- **Manage (F4):** [1] Stage [2] Unstage [3] Plan [4] Split — staging plan-based workflow
- **Compose (F5):** [1] Staged [2] Generate [3] List [4] Merge [5] Edit — commit message workflow

**Data Model:**
- **Staging Plans** - Categorized changes with line-level hunk info (`.gomion/plans/`)
- **Commit Candidates** - AI-generated messages with staging hash (`.gomion/candidates/`)
- **Staging Snapshots** - Safety net for undo (`.gomion/snapshots/`, auto-archive 30 days)
- **AI Plan Takes** - 3 different perspectives on staging plans (`~/.cache/gomion/analysis/`)

**Key Workflow:**
1. Explore changes → 2. Manage: AI plans → Split (assign lines) → Stage plan → 3. Compose: Generate message → 4. Main: Commit

---

## GRU: Standalone TUI Staging Editor

### Executive Summary

Build **gru** - a standalone TUI staging editor in `cmd/gru` that provides visual take selection and hunk assignment. This is the interactive UI component for Gomion's staging workflow.

**Core Innovation:** Dynamic UI that switches between Take exploration (conceptual groupings) and hunk refinement (concrete staging) based on user focus.

### Architecture Overview

#### Standalone App with Cached Repo

- **Location**: `cmd/gru/` - Standalone binary
- **Integration**: gru runs standalone, reads/writes `.gomion/` directly (may add Gomion integration later)
- **Cached Repo**: Uses existing `gitutils.CachedWorktree` implementation
    - Cache: `~/.cache/repos/<repo-key>/`
    - Isolated workspace, user repo untouched until commit
- **Per-ChangeSet Index**: Separate Git index file per ChangeSet (via `GIT_INDEX_FILE`)
    - Location: `<projectRepo>/.git/info/changesets/<id>/index`
    - Persisted in user repo's `.git/info/` directory (survives cache clearing)
    - User repo uses `.gomion/` for other persistence

#### Data Model

```
PlanTake (AI strategy)
  ├─ ChangeSet 1 (conceptual group)
  │   └─ files → (user refines) → hunks
  ├─ ChangeSet 2
  │   └─ files → (user refines) → hunks
  └─ ...

Final output: StagingPlan per ChangeSet with specific hunks
```

**Key types:**
- `PlanTake` (was `StagingPlanTake`) - One AI strategy (e.g., "By Feature")
- `ChangeSet` (was `TakeGroup`) - Logical group within a take
- `StagingPlan` - Final output with specific file/hunk assignments ready to commit
    - **Note:** May not be needed - can generate diff patch directly from Git index

#### UI Modes (Dynamic)

**Mode 1: Takes Exploration** (focus on left pane, Takes list visible)
```
╔═══════════════╦══════════════════╦═════════════════╗
║ TAKES (focus) ║ CHANGESETS       ║ SOURCE/SUMMARY  ║
║ > Take 1: ... ║ • ChangeSet A    ║ Code preview    ║
║   Take 2: ... ║ • ChangeSet B    ║ or summary      ║
║═══════════════║                  ║                 ║
║ FILES         ║                  ║                 ║
║   auth.go     ║                  ║                 ║
╚═══════════════╩══════════════════╩═════════════════╝
```

**Mode 2: Hunk Refinement** (focus on files, Takes list hidden/collapsed)
```
╔═══════════════╦══════════════════╦══════════════════╗
║ SELECTED TAKE ║ BASELINE         ║ CHANGES          ║
║ Take 1        ║ 10 func Login()  ║ 10 Login(u) [✓]  ║
║═══════════════║ 11   // TODO     ║ 11 if u ... [✓]  ║
║ FILES (focus) ║ 12 }             ║ 12 return... [✓] ║
║ > auth.go [3] ║                  ║                  ║
║   user.go [1] ║ ───────────      ║ ───────────────  ║
╚═══════════════╩══════════════════╩══════════════════╝
```

**Key:** Left pane splits/toggles with hotkey (e.g., 't') to show/hide Takes list.

**Right Pane Display:** The right pane should show a unified diff view similar to JetBrains IDEs - showing the changes with line numbers and checkboxes per hunk for assignment to ChangeSets.

### GRU Implementation Phases

#### Phase 3: Basic UI - Takes View (8 hours)

**Goal:** Two-pane UI for browsing Takes and ChangeSets

**Tasks:**
1. Create `model.go` with bubbletea Init():
   ```go
   func (m EditorState) Init() tea.Cmd {
       return nil
   }
   ```
2. Create `update.go` with key handlers:
   ```go
   func (m EditorState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
       switch msg := msg.(type) {
       case tea.KeyMsg:
           switch msg.String() {
           case "t":
               return m.toggleTakesView()
           case "tab":
               return m.switchPane()
           case "up", "k":
               return m.navigateUp()
           case "down", "j":
               return m.navigateDown()
           case "enter":
               return m.selectItem()
           case "q":
               return m, tea.Quit
           }
       }
       return m, nil
   }
   ```
3. Create `takes_view.go` for rendering left pane (Takes mode):
   ```go
   func renderTakesList(takes *PlanTakes, selected int, width, height int) string
   func renderChangeSetsList(take *PlanTake, selected int, width, height int) string
   ```
4. Create `view.go` with layout:
   ```go
   func (m EditorState) View() string {
       if m.ViewMode == TakesView {
           leftPane := renderTakesPane(m)   // Takes + Files
           middlePane := renderChangeSetsPane(m)
           rightPane := renderSourcePane(m)
           return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, middlePane, rightPane)
       }
       // FilesView handled in Phase 4
   }
   ```
5. Create `styles.go` with lipgloss styles:
   ```go
   var (
       titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
       activePaneStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
       // ... more styles
   )
   ```

**Deliverables:**
- [ ] Takes list renders in left pane
- [ ] ChangeSets list renders in middle pane
- [ ] Navigation works (up/down, select Take)
- [ ] Selecting a Take updates ChangeSets pane

#### Phase 4: Files View - Hunk Refinement (10 hours)

**Goal:** Three-pane diff view for assigning hunks to ChangeSets

**Tasks:**
1. Create `files_view.go`:
   ```go
   func renderFileTree(files []FileWithHunks, selected int, width, height int) string
   ```
2. Create `diff_view.go`:
   ```go
   func renderBaselinePane(file FileWithHunks, scroll int, width, height int) string
   func renderChangesPane(file FileWithHunks, activeCS *ChangeSet, scroll int) string
   ```
3. Parse git diff into hunks:
   ```go
   type FileWithHunks struct {
       Path  dt.RelFilepath
       Hunks []Hunk
   }

   type Hunk struct {
       Header        HunkHeader
       BaselineLines []string
       ChangeLines   []string
       AssignedToCS  string  // ChangeSet ID
   }

   func ParseGitDiff(diffOutput string) ([]FileWithHunks, error)
   ```
4. Hunk assignment logic:
   ```go
   func (m *EditorState) assignHunkToChangeSet(fileIdx, hunkIdx int) error {
       hunk := m.Files[fileIdx].Hunks[hunkIdx]
       cs := m.ChangeSets[m.ActiveCS]

       // Use GIT_INDEX_FILE for this ChangeSet
       os.Setenv("GIT_INDEX_FILE", cs.IndexFile.String())
       defer os.Unsetenv("GIT_INDEX_FILE")

       // Stage this hunk
       err := stageHunk(m.CachedRepo.Dir, hunk)
       if err != nil {
           return err
       }

       hunk.AssignedToCS = cs.ID
       return nil
   }
   ```
5. Update View() for FilesView mode:
   ```go
   if m.ViewMode == FilesView {
       leftPane := renderSelectedTakeInfo(m) + renderFileTree(m)
       middlePane := renderBaselinePane(m)
       rightPane := renderChangesPane(m)  // With [✓] checkboxes per hunk
       return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, middlePane, rightPane)
   }
   ```
6. Key bindings for hunk assignment:
    - `Space`: Toggle hunk assignment to active ChangeSet
    - `1-9`: Switch active ChangeSet
    - `a`: Assign all hunks in file to active ChangeSet
    - `u`: Unassign hunk

**Deliverables:**
- [ ] File tree renders with hunk counts
- [ ] Baseline/Changes panes show code side-by-side
- [ ] Hunks have checkboxes indicating assignment
- [ ] Space toggles hunk assignment
- [ ] GIT_INDEX_FILE correctly set per ChangeSet

**Future Consideration:**
- Explore using treeview's Comparison Display format as an additional view option for ChangeSets (discuss implementation approach when we reach this phase)

#### Phase 5: ChangeSet Operations (6 hours)

**Goal:** Create, edit, delete, merge ChangeSets

**Tasks:**
1. Create `changeset_manager.go`:
   ```go
   func CreateChangeSet(cacheRepo dt.DirPath, name, rationale string) (*ChangeSet, error)
   func DeleteChangeSet(cs *ChangeSet) error
   func MergeChangeSets(cs1, cs2 *ChangeSet, cacheRepo dt.DirPath) (*ChangeSet, error)
   func EditChangeSet(cs *ChangeSet, name, rationale string) error
   ```
2. Add key bindings:
    - `n`: Create new ChangeSet (prompt for name)
    - `e`: Edit ChangeSet name/rationale
    - `d`: Delete ChangeSet (confirmation dialog)
    - `m`: Merge two ChangeSets
3. Use `github.com/erikgeiser/promptkit` for dialogs:
   ```go
   import "github.com/erikgeiser/promptkit/textinput"

   func promptChangeSetName() (string, error) {
       input := textinput.New("ChangeSet name:")
       return input.Run()
   }
   ```
4. ChangeSet metadata persistence:
   ```go
   type ChangeSetMeta struct {
       ID        string
       Name      string
       Rationale string
       TakeNumber int
       Created   time.Time
       Modified  time.Time
   }

   func SaveChangeSetMeta(cacheRepo dt.DirPath, csID string, meta ChangeSetMeta) error
   ```

**Deliverables:**
- [ ] Create new ChangeSet from UI
- [ ] Edit ChangeSet name/rationale
- [ ] Delete ChangeSet with confirmation
- [ ] Merge two ChangeSets into one
- [ ] Metadata saved to `<projectRepo>/.git/info/changesets/<id>/meta.json`

#### Phase 6: Commit & Persistence (6 hours)

**Goal:** Commit ChangeSets to user repo, handle persistence

**Tasks:**
1. Commit workflow:
   ```go
   func CommitChangeSet(userRepo *gitutils.Repo, cs *ChangeSet, cacheRepo dt.DirPath) error {
       // 1. Get staged content from cs.IndexFile
       os.Setenv("GIT_INDEX_FILE", cs.IndexFile.String())
       patch, err := generatePatch(cacheRepo)

       // 2. Apply to user repo
       err = applyPatch(userRepo.Root, patch)
       if err != nil {
           return err  // Abort on conflict
       }

       // 3. Stage in user repo
       err = stageAppliedChanges(userRepo.Root)

       // 4. Commit
       message := generateCommitMessage(cs)  // Or user-provided
       err = createCommit(userRepo.Root, message)

       // 5. Mark ChangeSet as committed
       cs.Committed = true
       SaveChangeSetMeta(cacheRepo, cs.ID, cs.ToMeta())

       return nil
   }
   ```
2. Add key binding:
    - `c`: Commit active ChangeSet (shows message editor first)
3. Save/Load session state:
   ```go
   type SessionState struct {
       ActiveTake int
       ActiveCS   int
       ViewMode   ViewMode
       // ... other UI state
   }

   func SaveSession(cacheRepo dt.DirPath, state SessionState) error
   func LoadSession(cacheRepo dt.DirPath) (*SessionState, error)
   ```
4. Handle exit:
   ```go
   func (m EditorState) handleExit() error {
       // Save session for next invocation
       SaveSession(m.CachedRepo.Dir, m.toSessionState())

       // Release cached worktree
       m.CachedRepo.Close()

       return nil
   }
   ```
5. **Persistence decision** (user's question: "After commit, do we EVEN NEED to retain staging plans?"):
    - **Recommendation**: No, delete ChangeSet after successful commit
    - Rationale: Once committed, the work is done. Keeping it adds clutter.
    - Exception: If user wants to track commit history, save to `.gomion/committed/` (optional feature)

**Deliverables:**
- [ ] Commit ChangeSet to user repo successfully
- [ ] Conflicts abort cleanly with error message
- [ ] Session state persists across invocations
- [ ] Committed ChangeSets are deleted (or archived)

#### Phase 7: Polish & Testing (8 hours)

**Goal:** Error handling, help system, testing

**Tasks:**
1. Help overlay:
   ```go
   func renderHelpOverlay() string {
       return `
       gru - Staging Editor

       TAKES VIEW:
       t       - Toggle Takes list
       ↑/↓     - Navigate
       Enter   - Select Take/ChangeSet

       FILES VIEW:
       Space   - Toggle hunk assignment
       1-9     - Switch active ChangeSet
       a       - Assign all hunks in file

       OPERATIONS:
       n       - New ChangeSet
       e       - Edit ChangeSet
       d       - Delete ChangeSet
       m       - Merge ChangeSets
       c       - Commit active ChangeSet

       OTHER:
       Tab     - Switch panes
       ?       - Toggle help
       q       - Quit
       `
   }
   ```
2. Error dialogs:
   ```go
   func showErrorDialog(err error) tea.Cmd
   func showConfirmDialog(message string) (bool, error)
   ```
3. Terminal size check:
   ```go
   func checkMinimumSize(width, height int) error {
       if width < 120 || height < 30 {
           return errors.New("terminal too small (need 120x30 minimum)")
       }
       return nil
   }
   ```
4. Unit tests:
    - `git_index_test.go`: Test GIT_INDEX_FILE operations
    - `changeset_manager_test.go`: Test CRUD operations
    - `parser_test.go`: Test git diff parsing
5. Integration tests:
    - End-to-end: Generate takes → select → assign hunks → commit
    - Test with real git repo in temp directory
6. Manual testing checklist (see Appendix)

**Deliverables:**
- [ ] Help overlay accessible with '?'
- [ ] Error messages clear and actionable
- [ ] Terminal size validated on startup
- [ ] Unit tests pass
- [ ] Integration tests pass

#### Phase 8: UI Enhancements (Future)

**Goal:** Improve visualization and user experience based on real-world usage

**Potential Enhancements:**
- Changes Overview mode using treeview's Comparison Display format
- Additional view toggles for different perspectives on changes
- UI/UX improvements identified during Phases 1-7
- Performance optimizations for large diffs

**Note:** Keep this phase deliberately open-ended. Once we have the working UI from Phases 1-7, the best enhancements will become apparent through actual use.

### GRU Critical Files & Components

#### New Files to Create

```
cmd/gru/grumod/grutui
├── main.go                    # Entry point, CLI flags
├── model.go                   # Bubbletea model (EditorState)
├── update.go                  # Update function, key handlers
├── view.go                    # View function, rendering
├── takes_view.go              # Take exploration pane rendering
├── files_view.go              # Files tree pane
├── diff_view.go               # Baseline/Changes panes
├── changeset_manager.go       # ChangeSet CRUD operations
├── git_index.go               # GIT_INDEX_FILE management
├── styles.go                  # Lipgloss styles
├── types.go                   # EditorState, ChangeSet, etc.
└── README.md                  # Usage documentation
```

#### File layout for keep track of project info
```
<projectRepo>/.git/info/   # In cached repo (gitignored)
└── changesets/
    └── <id>/
    ├── index              # Git index file (authoritative)
    ├── meta.json          # ChangeSet metadata
    └── staged.patch       # (optional, informational)
```

#### Existing Files to Leverage

- `gompkg/gitutils/repo.go` - `Repo.OpenCachedWorktree()` for cached repo
- `gompkg/gomcfg/plan_takes.go` - for different takes on staging plans
    - `PlanTakes`
    - `PlanTake`
    - `ChangeSet`
- `gompkg/gompkg/staging_plan.go` - `StagingPlan` (final output format)
- `gompkg/askai/` - AI integration for generating Takes

#### Gomion Integration Files (Optional, Later)

- `gompkg/gompkg/manage_mode.go` - Could invoke gru with `exec.Command("gru")`
- Input/output via `.gomion/` directory (not temp files initially)

### GRU Data Flow

```
1. Launch gru
   ↓
2. Open user repo + cached repo
   ↓
3. Load or generate Takes (AI)
   ↓
4. User selects Take → Creates ChangeSets
   ↓
5. User toggles to Files view
   ↓
6. User assigns hunks to ChangeSets (via GIT_INDEX_FILE)
   ↓
7. User commits ChangeSet
   ↓
8. Apply patch to user repo → git commit
   ↓
9. Mark ChangeSet as committed, delete index
   ↓
10. Repeat for more ChangeSets or quit
```

### GRU Key Design Decisions

#### 1. Standalone vs. Gomion Integration
**Decision:** Start standalone, add Gomion integration later

**Rationale:**
- Simpler development and testing
- Can be used independently of Gomion
- Easy to add exec.Command integration later

#### 2. Directory Structure
**Decision:** `.gomion/` in user repo, `.gru/` in cached repo

**Rationale:**
- Clear separation of persistent data (.gomion/) vs. working state (.gru/)
- Aligns with existing Gomion patterns
- gru-specific internals in .gru/ don't pollute .gomion/

#### 3. Takes Generation
**Decision:** Generate AI takes on startup (with caching)

**Rationale:**
- Fresh suggestions for current changes
- User can request more takes if needed
- Cache prevents redundant API calls (persists indefinitely until regenerated)

#### 4. Post-Commit Cleanup
**Decision:** Delete ChangeSet after successful commit

**Rationale:**
- User asked "do we EVEN NEED to retain staging plans?"
- Answer: No, committed work is done
- Reduces clutter, clearer state
- Can add archive feature later if needed

#### 5. ViewMode Toggle
**Decision:** Single hotkey ('t') toggles between views

**Rationale:**
- Simple, memorable
- Cycles: Takes visible → Takes hidden → Takes visible
- User's focus determines pane behavior

### GRU Dependencies

```bash
cd cmd/gru
go mod init github.com/mikeschinkel/gomion/cmd/gru

go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/erikgeiser/promptkit@latest
go get github.com/Digital-Shane/treeview@latest

# Gomion packages (via replace directive)
# Already available: gitutils, gomcfg, gompkg, askai
```

---

## Implementation Phases

### Phase 4: Staging Plan Selector (MM UI)

**Goal:** Visual take selection and manual editing

**Tasks:**
1. Add bubbletea dependency to gomion
2. Create `~/Projects/gomion/gompkg/gomioncliui/plan_selector.go`
   ```go
   func ShowPlanSelector(takes *gomcfg.StagingPlanTakes, writer io.Writer) ([]*gompkg.StagingPlan, error)
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
   - Save to `.gomion/plans/`

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
1. Create `~/Projects/gomion/gompkg/gomioncliui/split_editor.go`
   - Three-pane layout: Files (left) | Baseline (middle) | Changes (right)
   - Hunk parsing: `git diff --cached --unified=0` → parse @@ headers
   - Checkbox per hunk for plan assignment
   - Key bindings: [↑↓] navigate, [Space] toggle, [g] change plan, [s] save

2. Integrate into Manage mode [4] Split action
   - Launch Split UI
   - User assigns hunks to plans
   - Save plans to `.gomion/plans/`

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
- Mode hints ("Try F4 Manage next")
- Color/highlighting in toggle bar (highlight current mode)

## Critical Files

**Gomion MM UIs (Phase 4+):**
- `~/Projects/gomion/gompkg/gomioncliui/plan_selector.go` (new, Phase 4) - Two-pane plan selection
- `~/Projects/gomion/gompkg/gomioncliui/split_editor.go` (new, Phase 5) - Hunk assignment UI

## Key Design Decisions

**Why modal vs hierarchical?**
- Modal: Switch between ANY mode at ANY time (flat registry)
- Hierarchical: Nested function calls (current system)
- Modal supports complex workflows without deep nesting

**Why F-keys for mode switching?**
- F2-F12 provide 11 mode slots (F2=mode0, F3=mode1, etc.)
- More reliable than Ctrl+digit detection across terminals
- F1 reserved for help
- Distinguishes mode switching from menu actions (0-9)

**Why store hunk headers + context (not just line numbers)?**
- Line numbers shift as edits happen
- Git hunks use @@ -oldStart,oldCount +newStart,newCount @@ format
- Context lines allow stable application even after upstream changes
- Follows git's unified diff format

**Why .gomion/ instead of ~/.cache/?**
- Staging plans/candidates/snapshots are repo-specific
- Should be version controlled (tracked in git)
- Closer to git staging semantics

**Why ModeState (not ModeContext)?**
- Avoids confusion with Go's `context.Context` (cancellation/deadlines/values)
- Current system uses closure-captured state (anti-pattern)
- ModeState centralizes all shared state in one discoverable struct
- Enables testing (mock state), sharing (modes access same object), discovery (all state visible)

## Testing Strategy

**Unit Tests:**
- cliutil: Ctrl detection, mode switching, exit handling
- gompkg: JSON round-trips, file I/O, cache keys, mode initialization
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
| Bubble Tea complexity delays MM UIs | Phases 2-3 work without UIs (text-based), defer complex UIs to Phases 4-5 |
| Data schema changes require migration | Version all schemas (StagingPlanV1), write ADR for changes, provide migration commands |
| Git hunk parsing complexity | Store hunk headers + context (not just line numbers), use git apply for precision |
| ModeState out of sync with git | Refresh on OnEnter, create snapshots before/after changes, show drift warnings |
| commitmsg API unclear for modal workflow | Defer final API design until modal workflow is implemented, mark for reconsideration |


## Next Steps

1. **GRU Phase 3:** Implement basic Takes View UI
2. **Phase 4 Task 1:** Add bubbletea dependency to gomion
3. **Phase 4 Task 2:** Create `gomioncliui/plan_selector.go` - Visual plan selector UI
4. **Phase 4 Task 3:** Integrate plan selector into Manage mode

---

## Appendix: GRU Implementation Reference

### Manual Testing Checklist

**Takes View:**
- [ ] Generate 3 takes via AI
- [ ] Default "All Changes" take shows first
- [ ] Navigate Takes list with ↑/↓
- [ ] Select Take updates ChangeSets pane
- [ ] Navigate ChangeSets with ↑/↓
- [ ] Toggle Takes list with 't' key

**Files View:**
- [ ] File tree shows all changed files
- [ ] File badges show hunk count
- [ ] Select file updates Baseline/Changes panes
- [ ] Baseline shows old code
- [ ] Changes shows new code with checkboxes
- [ ] Hunks synchronized (scroll together)
- [ ] Consider using github.com/Digital-Shane/treeview (https://pkg.go.dev/github.com/Digital-Shane/treeview)

**Hunk Assignment:**
- [ ] Space toggles hunk assignment
- [ ] Checkbox updates: [ ] → [✓]
- [ ] Switch active ChangeSet with 1-9
- [ ] Assigned hunk shows correct ChangeSet marker
- [ ] 'a' assigns all hunks in file
- [ ] 'u' unassigns hunk

**ChangeSet Operations:**
- [ ] 'n' creates new ChangeSet with prompt
- [ ] 'e' edits ChangeSet name/rationale
- [ ] 'd' deletes ChangeSet with confirmation
- [ ] 'm' merges two ChangeSets

**Commit:**
- [ ] 'c' prompts for commit message
- [ ] Commit applies changes to user repo
- [ ] Commit succeeds with clean message
- [ ] Conflict aborts cleanly
- [ ] ChangeSet deleted after commit

**Session:**
- [ ] Exit with 'q' saves session
- [ ] Relaunch restores active Take/ChangeSet
- [ ] ViewMode preserved

**Edge Cases:**
- [ ] No changes (empty diff)
- [ ] Large diff (>100 files)
- [ ] Binary files in diff
- [ ] Merge conflicts during commit
- [ ] Terminal resize
- [ ] Ctrl+C handles gracefully

### Git Diff Parsing

#### Input Format

The editor receives output from:
```bash
git diff --cached --unified=0
```

Example:
```
diff --git a/auth.go b/auth.go
index 1234567..abcdefg 100644
--- a/auth.go
+++ b/auth.go
@@ -10,3 +10,5 @@ func Login() {
-    // TODO
+    if u == nil {
+        return ErrNilUser
+    }
 }
@@ -45,3 +47,5 @@ func Validate() {
     return true
+    if u == nil {
+        return false
+    }
 }
```

#### Parsing Requirements

1. Extract file paths from `diff --git` lines
2. Parse `@@` hunk headers (old_start, old_count, new_start, new_count)
3. Group lines by hunk
4. Separate `-` (deletions), `+` (additions), ` ` (context) lines
5. Build HunkHeader with context lines for stable application

#### Context Lines

Store 2-3 lines of context before/after each hunk for stable application:
- Context helps git apply hunks even if line numbers shift
- Context stored in `HunkHeader.ContextBefore` and `HunkHeader.ContextAfter`

### Rendering Format Examples

#### File List Pane

```
FILES
auth.go [3]     ← 3 hunks in this file
user.go [1]
> logger.go [2] ← Currently selected
config.go
```

- Show relative file paths
- Badge `[N]` shows hunk count
- `>` indicates selection
- Scroll when list exceeds pane height

#### Baseline Pane

```
BASELINE
 10 func Login() {
 11     // TODO
 12 }
────────────────
 45 func Validate() {
 46     return true
 47 }
```

- Line numbers from old file
- Separator between hunks
- Scroll synchronized with Changes pane

#### Changes Pane

```
CHANGES
 10 func Login(u *User) {     [✓]
 11     if u == nil {          [✓]
 12         return ErrNilUser  [✓]
 13     }                      [✓]
────────────────────────────────────
 45 func Validate(u) {        [ ]
 46     if u == nil {          [ ]
 47         return false       [ ]
 48     }                      [ ]
 49     return true            [ ]
```

- Line numbers from new file
- Checkbox per hunk (one checkbox represents entire hunk, not per line)
- `[✓]` = assigned to active ChangeSet
- `[✗]` = assigned to different ChangeSet (show ChangeSet name on hover)
- `[ ]` = unassigned
- Scroll synchronized with Baseline pane

**Important:** Checkbox is per HUNK, not per line. All lines in a hunk share the same checkbox state.
