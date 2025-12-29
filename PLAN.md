# Squire Implementation Plan

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

Squire's modal commit workflow with staging plan-based workflow, line-level control, and AI-assisted commit message generation.

**Modal System:** Flat mode registry with F-key switching between ANY mode at ANY time.

## Architecture Summary

### Modal Menu System (go-cliutil) - ✅ COMPLETE

**Display Format:**
```
Main Menu: [f3] Explore [f4] Manage [f5] Compose — [f1] Help
Actions:   [1] Commit — [0] help [9] quit
Choice:
```

### Squire Integration

**Four Modes:**
- **Main (F2):** [1] Commit code — starting mode
- **Explore (F3):** [1] Status [2] Breaking [3] Other Changes [4] Tests — read-only exploration
- **Manage (F4):** [1] Stage [2] Unstage [3] Plan [4] Split — staging plan-based workflow
- **Compose (F5):** [1] Staged [2] Generate [3] List [4] Merge [5] Edit — commit message workflow

**Data Model:**
- **Staging Plans** - Categorized changes with line-level hunk info (`.squire/plans/`)
- **Commit Candidates** - AI-generated messages with staging hash (`.squire/candidates/`)
- **Staging Snapshots** - Safety net for undo (`.squire/snapshots/`, auto-archive 30 days)
- **AI Plan Takes** - 3 different perspectives on staging plans (`~/.cache/squire/analysis/`)

**Key Workflow:**
1. Explore changes → 2. Manage: AI plans → Split (assign lines) → Stage plan → 3. Compose: Generate message → 4. Main: Commit

## Implementation Phases

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
- Mode hints ("Try F4 Manage next")
- Color/highlighting in toggle bar (highlight current mode)

## Critical Files

**Squire MM UIs (Phase 4+):**
- `~/Projects/squire/squirepkg/squireminiui/plan_selector.go` (new, Phase 4) - Two-pane plan selection
- `~/Projects/squire/squirepkg/squireminiui/split_editor.go` (new, Phase 5) - Hunk assignment UI

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

**Why .squire/ instead of ~/.cache/?**
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
| Bubble Tea complexity delays MM UIs | Phases 2-3 work without UIs (text-based), defer complex UIs to Phases 4-5 |
| Data schema changes require migration | Version all schemas (StagingPlanV1), write ADR for changes, provide migration commands |
| Git hunk parsing complexity | Store hunk headers + context (not just line numbers), use git apply for precision |
| ModeState out of sync with git | Refresh on OnEnter, create snapshots before/after changes, show drift warnings |
| commitmsg API unclear for modal workflow | Defer final API design until modal workflow is implemented, mark for reconsideration |


## Next Steps

1. **Phase 4 Task 1:** Add bubbletea dependency to squire
2. **Phase 4 Task 2:** Create `squireminiui/plan_selector.go` - Visual plan selector UI
3. **Phase 4 Task 3:** Integrate plan selector into Manage mode
