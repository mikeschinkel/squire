# GRU: Split Editor - Product Requirements Document

## Verification Tracker
- Legend: `[ ]` not verified, `[~]` partially verified, `[x]` issues/blockers, `[v]` verified
- [x] Repository contents vs PRD – local `cmd/gru` only has `main.go` ("Hello, world!") and `go.mod`; none of the referenced source files, Bubble Tea UI, or testdata exist.
- [x] Dependencies/go version – `go.mod` currently targets Go 1.25.3 with only local replace directives; PRD lists different versions and packages that are not required/imported yet. User wants latest Go; needs go.mod update once target decided.
- [v] Data structures from Squire (`squirepkg/squiresvc`, `squirecfg`) – aligned PRD to actual `squirepkg` definitions (see below).
- [ ] CLI contract (flags, exit codes) – no implementation to confirm behavior.
- [ ] Input/output JSON schema – no code to validate serialization/parsing.
- [ ] UI spec (layout, key bindings, dialogs, terminal size) – not implemented; behavior unverified.
- [ ] Integration with Squire invocation/SaveStagingPlan – referenced code paths absent locally.
- [ ] Testing strategy & success criteria – no tests or tooling in repo to measure.

## Executive Summary

**Product:** Squire Split Editor - A standalone TUI application for line-level commit staging plan assignment

**Purpose:** Visual, JetBrains-style interface for assigning git hunks to staging plans

**Integration:** Standalone Go binary that communicates with Squire via temp files

**Technology:** Bubbletea (TUI framework), go-dt types, ClearPath coding style

---

## ⚠️ CRITICAL: Existing Data Structures

**DO NOT reinvent these - they already exist in Squire!**

### From `squiresvc/staging_plan.go`

```go
// StagingPlan represents a named collection of file changes to commit together
type StagingPlan struct {
    ID          dt.Identifier    `json:"id"`          // UUID
    Name        string           `json:"name"`
    Description string           `json:"description"`
    Created     time.Time        `json:"created"`
    Modified    time.Time        `json:"modified"`
    Files       []FilePatchRange `json:"files"`
    Suggested   bool             `json:"suggested"`   // AI vs user
    TakeNumber  int              `json:"take_number"` // 1-3, or 0 if user
    IsDefault   bool             `json:"is_default"`  // Auto-generated default plan
}

// FilePatchRange represents hunks from a single file assigned to a plan
type FilePatchRange struct {
    Path     dt.RelFilepath `json:"path"`
    Hunks    []HunkHeader   `json:"hunks"`
    AllLines bool           `json:"all_lines"`   // If true, entire file
}

// HunkHeader represents a single hunk with context for stable application
// Line numbers shift as edits happen - need context for stable application
type HunkHeader struct {
    Header        string   `json:"header"`          // The @@ line
    ContextBefore []string `json:"context_before"`  // Context lines before
    ContextAfter  []string `json:"context_after"`   // Context lines after
    OldStart      int      `json:"old_start"`
    OldCount      int      `json:"old_count"`
    NewStart      int      `json:"new_start"`
    NewCount      int      `json:"new_count"`
}
```

### From `squirecfg/staging_plan_takes.go`

```go
// StagingPlanTakes - AI-generated grouping suggestions (for reference, not edited)
type StagingPlanTakes struct {
    CacheKey  string            `json:"cache_key"`
    Timestamp time.Time         `json:"timestamp"`
    Takes     []StagingPlanTake `json:"takes"`
}

type StagingPlanTake struct {
    Number int         `json:"number"` // 1, 2, or 3
    Theme  string      `json:"theme"`  // "By Feature", "By Layer", "By Risk"
    Groups []TakeGroup `json:"groups"`
}

type TakeGroup struct {
    Name      string           `json:"name"`
    Rationale string           `json:"rationale"`
    Files     []dt.RelFilepath `json:"files"`
}
```

---

## Product Overview

### What It Does

The Split Editor allows users to:
1. View all changed files in a tree on the left
2. See baseline (before) and changes (after) side-by-side
3. Assign individual hunks to staging plans via checkboxes
4. Create new plans or assign to existing plans
5. Save assignments back to Squire for staging
6. Stage changes to git based on user-input

### What It Doesn't Do

- Does NOT generate commit messages (Squire's Compose mode does that)
- Does NOT run git commands (uses gitutils package)
- Does NOT directly read/write `.squire/` directory (Squire does that)

### Standalone Architecture

**Why standalone?**
- Complex UI deserves isolated development/testing
- Can be run independently for debugging
- Cleaner separation of concerns
- Easier to replace/upgrade UI without touching Squire core

**Communication:**
- Squire invokes: `gru --input=/tmp/squire-input.json --output=/tmp/squire-output.json`
- Input file contains: git diff output, existing plans, module directory
- Output file contains: updated plan assignments
- Exit code 0 = success, 1 = user cancelled, 2+ = error

---

## CLI Interface Specification

### Command

```bash
gru [flags]
```

### Required Flags

```
--input=<filepath>     Path to input JSON file (from Squire)
--output=<filepath>    Path to output JSON file (to Squire)
```

### Optional Flags

```
--module-dir=<dirpath> Module directory (for display purposes)
--help                 Show help
--version              Show version
```

### Exit Codes

```
0  - Success (user saved changes)
1  - Cancelled (user quit without saving)
2  - Invalid input file
3  - Failed to parse git diff
4  - Failed to write output file
5  - Terminal too small
6  - Unknown error
```

---

## Input/Output File Format

### Input JSON (`--input` file)

```json
{
  "module_dir": "/Users/mike/Projects/squire",
  "git_diff_output": "diff --git a/file1.go b/file1.go\nindex 1234567..abcdefg 100644\n--- a/file1.go\n+++ b/file1.go\n@@ -10,3 +10,5 @@ func Foo() {\n ...",
  "existing_plans": [
    {
      "id": "plan-uuid-1",
      "name": "Add authentication",
      "description": "User auth changes",
      "created": "2025-12-27T10:00:00Z",
      "modified": "2025-12-27T10:00:00Z",
      "files": [
        {
          "path": "auth.go",
          "hunks": [
            {
              "header": "@@ -10,3 +10,5 @@ func Login()",
              "context_before": ["func Login() {", "    // TODO"],
              "context_after": ["}"],
              "old_start": 10,
              "old_count": 3,
              "new_start": 10,
              "new_count": 5
            }
          ],
          "all_lines": false
        }
      ],
      "suggested": true,
      "take_number": 1,
      "is_default": false
    }
  ],
  "ai_takes": {
    "cache_key": "abc123def456",
    "timestamp": "2025-12-27T09:00:00Z",
    "takes": [
      {
        "number": 1,
        "theme": "By Feature",
        "groups": [
          {
            "name": "Add authentication",
            "rationale": "All auth-related changes",
            "files": ["auth.go", "middleware.go"]
          }
        ]
      }
    ]
  }
}
```

### Output JSON (`--output` file)

```json
{
  "plans": [
    {
      "id": "plan-uuid-1",
      "name": "Add authentication",
      "description": "User auth changes",
      "created": "2025-12-27T10:00:00Z",
      "modified": "2025-12-27T11:30:00Z",
      "files": [
        {
          "path": "auth.go",
          "hunks": [
            {
              "header": "@@ -10,3 +10,5 @@ func Login()",
              "context_before": ["func Login() {", "    // TODO"],
              "context_after": ["}"],
              "old_start": 10,
              "old_count": 3,
              "new_start": 10,
              "new_count": 5
            }
          ],
          "all_lines": false
        }
      ],
      "suggested": false,
      "take_number": 0,
      "is_default": false
    },
    {
      "id": "plan-uuid-2",
      "name": "Fix logging",
      "description": "Update log statements",
      "created": "2025-12-27T11:30:00Z",
      "modified": "2025-12-27T11:30:00Z",
      "files": [
        {
          "path": "logger.go",
          "hunks": [
            {
              "header": "@@ -25,2 +25,3 @@ func Log()",
              "context_before": ["func Log(msg string) {", "    fmt.Println(msg)"],
              "context_after": [],
              "old_start": 25,
              "old_count": 2,
              "new_start": 25,
              "new_count": 3
            }
          ],
          "all_lines": false
        }
      ],
      "suggested": false,
      "take_number": 0,
      "is_default": false
    }
  ]
}
```

**Key Points:**
- Output contains ALL plans (existing + new + modified)
- Plans without any hunks should be omitted
- `modified` timestamp updated for changed plans
- `suggested: false` for user-edited plans (no longer AI suggestion)
- `take_number: 0` for user-created/modified plans

---

## UI Specification

### Three-Pane Layout

```
╔════════════════════════════════════════════════════════════════════════════╗
║ Split Editor - Active Plan: "Add authentication"                           ║
╠═══════════════╦════════════════════════════╦═══════════════════════════════╣
║ FILES (25%)   ║ BASELINE (37.5%)           ║ CHANGES (37.5%)               ║
╟───────────────╫────────────────────────────╫───────────────────────────────╢
║ > auth.go [2] ║  10 func Login() {         ║  10 func Login(u *User) { [✓] ║
║   user.go [1] ║  11     // TODO            ║  11     if u == nil {     [✓] ║
║   logger.go   ║  12 }                      ║  12         return Err    [✓] ║
║               ║                            ║  13     }                 [✓] ║
║               ║                            ║  14     // Auth logic     [✓] ║
║               ║ ───────────────────────    ║ ────────────────────────────  ║
║               ║  45 func Validate() {      ║  45 func Validate(u) {    [ ] ║
║               ║  46     return true        ║  46     if u == nil {     [ ] ║
║               ║  47 }                      ║  47         return false  [ ] ║
║               ║                            ║  48     }                 [ ] ║
║               ║                            ║  49     return true       [ ] ║
╠═══════════════╩════════════════════════════╩═══════════════════════════════╣
║ Plans: [1] Add auth [2] Fix logs [3] Update tests [n] New plan             ║
║ [↑↓] Navigate  [Space] Toggle  [Enter] Assign plan  [s] Save  [q] Quit     ║
╚════════════════════════════════════════════════════════════════════════════╝
```

### Layout Details

**Left Pane (25% width):**
- Tree of changed files (grouped by directory)
- Badge showing hunk count per file: `[2]`
- `>` indicator for currently selected file
- Keyboard navigation: `↑`/`↓` or `j`/`k`

**Middle Pane (37.5% width):**
- Baseline (pre-change) code
- Line numbers match git diff output
- Context lines shown for each hunk
- Separator `─────────` between hunks

**Right Pane (37.5% width):**
- Changed code with additions/deletions
- Checkbox `[✓]` or `[ ]` per hunk
- Checked hunks assigned to active plan
- Line numbers from new file

**Bottom Bar (2 lines):**
- Line 1: Available plans with hotkeys
- Line 2: Key bindings reference

**Top Bar (1 line):**
- Current active plan name
- Mode indicator (if needed)

### Minimum Terminal Size

- **Width:** 120 columns
- **Height:** 30 rows
- If smaller: Show error message and exit with code 5

---

## Key Bindings

### Navigation

```
↑ / k       - Move selection up (within current pane context)
↓ / j       - Move selection down
← / h       - Switch to left pane
→ / l       - Switch to right pane
PgUp / Ctrl+U  - Page up
PgDn / Ctrl+D  - Page down
g           - Go to top
G           - Go to bottom
```

### Hunk Assignment

```
Space       - Toggle hunk checkbox (assign to active plan)
a           - Assign ALL hunks in current file to active plan
u           - Unassign hunk (remove from any plan)
U           - Unassign ALL hunks in current file
```

### Plan Management

```
1-9         - Switch active plan (if that many plans exist)
n           - Create new plan (prompt for name)
e           - Edit active plan name/description
d           - Delete active plan (unassigns all hunks)
```

### File Operations

```
s           - Save and exit (write output file)
q           - Quit without saving (exit code 1)
Ctrl+C      - Force quit without saving (exit code 1)
```

### Help

```
?           - Toggle help overlay
Esc         - Close help overlay / Cancel dialog
```

---

## New Data Structures (Split Editor Internal)

These are used internally by the editor, NOT saved to JSON:

```go
package main

import (
    "github.com/mikeschinkel/go-dt"
    "github.com/mikeschinkel/squire/squirepkg/squiresvc"
)

// EditorState - Main application state for Bubbletea model
type EditorState struct {
    // Input data
    ModuleDir      dt.DirPath
    GitDiffOutput  string
    ExistingPlans  []squiresvc.StagingPlan
    AITakes        *squirecfg.StagingPlanTakes

    // Parsed data
    Files          []FileWithHunks
    AllHunks       []Hunk  // Flat list of all hunks across all files

    // Current state
    ActivePlanID   string
    SelectedFileIdx   int  // Index into Files
    SelectedHunkIdx   int  // Index into Files[SelectedFileIdx].Hunks
    FocusedPane    PaneType

    // UI state
    FileListScroll   int
    BaselineScroll   int
    ChangesScroll    int
    ShowHelp         bool

    // Modified flag
    Modified       bool
}

// FileWithHunks - Represents a file and all its hunks
type FileWithHunks struct {
    Path           dt.RelFilepath
    Hunks          []Hunk
    Expanded       bool  // For future: collapsible file tree
}

// Hunk - Represents a single diff hunk with assignment state
type Hunk struct {
    // From git diff parsing
    Header         squiresvc.HunkHeader
    BaselineLines  []string  // Lines from old file
    ChangeLines    []string  // Lines from new file

    // Assignment state
    AssignedPlanID string    // Empty string = unassigned

    // Display state
    StartLine      int       // Line number in Changes pane for scrolling
}

// PaneType - Which pane has focus
type PaneType int

const (
    FileListPane PaneType = iota
    BaselinePane
    ChangesPane
)

// InputData - Deserialized from --input JSON file
type InputData struct {
    ModuleDir       string                       `json:"module_dir"`
    GitDiffOutput   string                       `json:"git_diff_output"`
    ExistingPlans   []squiresvc.StagingPlan      `json:"existing_plans"`
    AITakes         *squirecfg.StagingPlanTakes  `json:"ai_takes"`
}

// OutputData - Serialized to --output JSON file
type OutputData struct {
    Plans []squiresvc.StagingPlan `json:"plans"`
}
```

---

## Implementation Requirements

### Must Use

1. **go-dt types** for all paths (`dt.DirPath`, `dt.Filepath`, `dt.RelFilepath`)
2. **ClearPath coding style** for all production code
3. **doterr pattern** for all error handling
4. **Bubbletea** for TUI framework
5. **Lipgloss** for styling (from Charm ecosystem)

### Must NOT Use

1. **os/exec** - No git commands, use provided diff output
2. **Direct file I/O to .squire/** - Squire manages that
3. **Global state** - All state in EditorState struct
4. **Panics** - Use proper error handling

### Code Structure

```
gru/
├── main.go              # Entry point, flag parsing, file I/O
├── model.go             # Bubbletea model (EditorState + Update/View)
├── parser.go            # Git diff parsing → FileWithHunks
├── assignment.go        # Hunk assignment logic
├── plans.go             # Plan creation/editing
├── renderer.go          # UI rendering (three panes)
├── keybindings.go       # Key event handling
├── styles.go            # Lipgloss styles
└── types.go             # EditorState, FileWithHunks, Hunk, etc.
```

### Error Handling

All functions follow ClearPath pattern:

```go
func ParseGitDiff(diffOutput string) (files []FileWithHunks, err error) {
    var lines []string
    var currentFile *FileWithHunks

    lines = strings.Split(diffOutput, "\n")

    for _, line := range lines {
        // Parse logic
        if someError {
            err = NewErr(ErrInvalidDiffFormat, "line", line)
            goto end
        }
    }

end:
    return files, err
}
```

---

## Git Diff Parsing

### Input Format

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

### Parsing Requirements

1. Extract file paths from `diff --git` lines
2. Parse `@@` hunk headers (old_start, old_count, new_start, new_count)
3. Group lines by hunk
4. Separate `-` (deletions), `+` (additions), ` ` (context) lines
5. Build HunkHeader with context lines for stable application

### Context Lines

Store 2-3 lines of context before/after each hunk for stable application:
- Context helps git apply hunks even if line numbers shift
- Context stored in `HunkHeader.ContextLines`

---

## Plan Assignment Logic

### Rules

1. **One hunk, one plan:** Each hunk assigned to max 1 plan
2. **Unassigned allowed:** Hunks can be unassigned (staged later)
3. **Partial file:** File can have hunks in different plans
4. **New plans inherit:** When creating new plan, can pre-assign selected hunks

### Assignment Operations

```go
// AssignHunkToPlan assigns a hunk to a plan (unassigns from previous plan)
func AssignHunkToPlan(state *EditorState, hunkIdx int, planID string) (err error)

// UnassignHunk removes hunk from any plan
func UnassignHunk(state *EditorState, hunkIdx int) (err error)

// AssignAllHunksInFile assigns all hunks in current file to active plan
func AssignAllHunksInFile(state *EditorState) (err error)

// CreateNewPlan prompts for name, creates plan, assigns current hunk
func CreateNewPlan(state *EditorState, name, description string) (plan squiresvc.StagingPlan, err error)
```

---

## Rendering Details

### File List Pane

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

### Baseline Pane

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

### Changes Pane

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
- `[✓]` = assigned to active plan
- `[✗]` = assigned to different plan (future: show plan name on hover)
- `[ ]` = unassigned
- Scroll synchronized with Baseline pane

**Important:** Checkbox is per HUNK, not per line. All lines in a hunk share the same checkbox state.

### Bottom Bar

```
Plans: [1] Add auth (5 hunks) [2] Fix logs (2 hunks) [n] New plan
[↑↓] Navigate  [Space] Toggle  [1-9] Switch plan  [s] Save  [q] Quit
```

---

## Dialog/Prompt UI

### New Plan Dialog

```
╔══════════════════════════════════╗
║ Create New Plan                  ║
╟──────────────────────────────────╢
║ Name:                            ║
║ ┌──────────────────────────────┐ ║
║ │ Add user authentication_     │ ║
║ └──────────────────────────────┘ ║
║                                  ║
║ Description (optional):          ║
║ ┌──────────────────────────────┐ ║
║ │ User login and auth changes  │ ║
║ └──────────────────────────────┘ ║
║                                  ║
║   [Enter] Create  [Esc] Cancel   ║
╚══════════════════════════════════╝
```

### Confirmation Dialog (on quit with unsaved changes)

```
╔══════════════════════════════════╗
║ Unsaved Changes                  ║
╟──────────────────────────────────╢
║ You have unsaved changes.        ║
║ Quit without saving?             ║
║                                  ║
║   [y] Yes  [n] No  [s] Save      ║
╚══════════════════════════════════╝
```

---

## Testing Strategy

### Unit Tests

1. **Git diff parsing:** Parse various diff formats, edge cases
2. **Hunk assignment:** Verify assignment rules, conflicts
3. **Plan creation:** UUID generation, timestamps
4. **JSON serialization:** Round-trip InputData/OutputData

### Integration Tests

1. **End-to-end:** Load input JSON → assign hunks → save output JSON
2. **File I/O:** Temp file handling, error cases
3. **Exit codes:** Verify correct codes for success/cancel/errors

### Manual Testing

1. **Terminal sizes:** Test at 120x30, 160x40, 80x24 (should fail)
2. **Large diffs:** 50+ files, 200+ hunks
3. **Edge cases:** No hunks, single hunk, huge hunks
4. **Keyboard:** All key bindings work as expected

---

## Development Phases

### Phase 5.1: Foundation

- [ ] CLI flag parsing (--input, --output)
- [ ] InputData/OutputData JSON deserialization
- [ ] Git diff parser (basic)
- [ ] EditorState struct
- [ ] Bubbletea skeleton (Init/Update/View)

### Phase 5.2: Core UI

- [ ] Three-pane layout rendering
- [ ] File list pane
- [ ] Baseline pane
- [ ] Changes pane with checkboxes
- [ ] Navigation (↑↓←→)

### Phase 5.3: Assignment Logic

- [ ] Hunk assignment/unassignment
- [ ] Plan switching (1-9 keys)
- [ ] Toggle checkbox (Space)
- [ ] Active plan tracking

### Phase 5.4: Plan Management

- [ ] New plan dialog
- [ ] Edit plan dialog
- [ ] Delete plan
- [ ] Plan list in bottom bar

### Phase 5.5: Save/Load

- [ ] Save output JSON
- [ ] Unsaved changes tracking
- [ ] Quit confirmation
- [ ] Exit codes

### Phase 5.6: Polish

- [ ] Styling (Lipgloss)
- [ ] Help overlay
- [ ] Error messages
- [ ] Minimum terminal size check

---

## Non-Goals (Future Enhancements)

These are NOT in scope for Phase 5:

- Mouse support (keyboard-only for now)
- Syntax highlighting (future)
- Inline editing (future)
- Diff preview of staged changes (future)
- Undo/redo (future)
- Search/filter files (future)
- Collapsible file tree (future)
- Plan statistics/summaries (future)

---

## Integration with Squire

### Squire's Manage Mode Invocation

When user selects `[4] Split` in Manage mode:

```go
// In squiresvc/manage_mode.go
func splitAction(state *SquireModeState) (err error) {
    var inputFile dt.Filepath
    var outputFile dt.Filepath
    var inputData SplitEditorInput
    var outputData SplitEditorOutput
    var cmd *exec.Cmd
    var exitCode int

    // Create temp files
    inputFile, err = dt.TempFile("squire-split-input-*.json")
    if err != nil {
        goto end
    }
    defer inputFile.Remove()

    outputFile, err = dt.TempFile("squire-split-output-*.json")
    if err != nil {
        goto end
    }
    defer outputFile.Remove()

    // Prepare input data
    inputData.ModuleDir = string(state.ModuleDir)
    inputData.GitDiffOutput, err = getGitDiff(state.ModuleDir)
    if err != nil {
        goto end
    }
    inputData.ExistingPlans = state.ActivePlans
    // ... write inputData to inputFile as JSON ...

    // Invoke split editor
    cmd = exec.Command("gru",
        "--input", string(inputFile),
        "--output", string(outputFile))
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    err = cmd.Run()
    exitCode = cmd.ProcessState.ExitCode()

    if exitCode == 1 {
        // User cancelled, no error
        err = nil
        goto end
    }
    if exitCode != 0 {
        err = NewErr(ErrSplitEditorFailed, "exit_code", exitCode)
        goto end
    }

    // Read output data
    // ... read outputFile as JSON into outputData ...

    // Update state with new plans
    state.ActivePlans = outputData.Plans

    // Save plans to .squire/plans/
    for _, plan := range state.ActivePlans {
        err = SaveStagingPlan(state.ModuleDir, &plan)
        if err != nil {
            goto end
        }
    }

end:
    return err
}
```

### getGitDiff Helper

```go
func getGitDiff(moduleDir dt.DirPath) (diff string, err error) {
    var cmd *exec.Cmd
    var out []byte

    cmd = exec.Command("git", "diff", "--cached", "--unified=0")
    cmd.Dir = string(moduleDir)

    out, err = cmd.Output()
    if err != nil {
        goto end
    }

    diff = string(out)

end:
    return diff, err
}
```

---

## Example Workflow

1. **User in Squire Manage mode:** Presses `[4] Split`
2. **Squire creates temp files:**
   - `/tmp/squire-split-input-abc123.json`
   - `/tmp/squire-split-output-abc123.json`
3. **Squire invokes editor:**
   ```bash
   gru \
     --input=/tmp/squire-split-input-abc123.json \
     --output=/tmp/squire-split-output-abc123.json
   ```
4. **Editor loads input:** Parses JSON, parses git diff, builds UI
5. **User assigns hunks:** Uses keyboard to navigate and assign
6. **User saves:** Presses `s`, writes output JSON, exits code 0
7. **Squire reads output:** Parses JSON, updates ActivePlans
8. **Squire saves plans:** Writes to `.squire/plans/*.json`
9. **Squire cleans up:** Removes temp files

---

## Success Criteria

### Functional

- [ ] Parses git diff correctly (all formats)
- [ ] Displays three-pane UI at 120x30 or larger
- [ ] Assigns hunks to plans correctly
- [ ] Creates/edits/deletes plans
- [ ] Saves output JSON in correct format
- [ ] Returns correct exit codes

### Non-Functional

- [ ] Responsive on large diffs (50+ files, 200+ hunks)
- [ ] No crashes, panics, or data loss
- [ ] Clear error messages for invalid input
- [ ] Keyboard-only operation (no mouse required)

### Code Quality

- [ ] Follows ClearPath style
- [ ] Uses go-dt types throughout
- [ ] doterr error handling
- [ ] >80% test coverage on core logic
- [ ] No linter warnings (golangci-lint)

---

## Dependencies

### Required Packages

```go
// go.mod
module github.com/mikeschinkel/squire/gru

go 1.23

require (
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/charmbracelet/lipgloss v0.9.1
    github.com/mikeschinkel/go-dt v0.1.0
    github.com/mikeschinkel/go-doterr v0.1.0
    github.com/mikeschinkel/squire/squirepkg/squiresvc v0.1.0
    github.com/mikeschinkel/squire/squirepkg/squirecfg v0.1.0
    github.com/google/uuid v1.5.0
)
```

**Note:** Squire packages imported as dependencies for type definitions only. No code execution happens across boundary (pure data structures).

---

## Open Questions / Design Decisions

### Bubble Tea Candidate Components 
We should consider using the ones of these that meet our needs rather than build from scatch:
- https://github.com/charmbracelet/bubbles — Prefer this first party package
- https://github.com/charm-and-friends/additional-bubbles — A long list of 3rd party components
  - These are of special interest (in no particular order):
  - https://github.com/lrstanley/bubblezone
  - https://github.com/daltonsw/bubbleup
  - https://github.com/erikgeiser/promptkit
  - https://github.com/Evertras/bubble-table
  - https://github.com/KevM/bubbleo
  - https://github.com/mistakenelf/teacup — Especially the Code and Markdown viewers
  - https://github.com/mritd/bubbles
  - https://github.com/rmhubbert/bubbletea-overlay
- https://github.com/charm-and-friends/charm-in-the-wild — Bubble Tea TUIs for inspiration, learning?
  - These are of special interest (in no particular order):
  - https://github.com/rubysolo/brows
  - https://github.com/a3chron/gith/
  - https://github.com/dlvhdr/gh-dash
  - https://github.com/charmbracelet/glow

    
### Communication: Temp Files vs Pipes

**Decision: Temp files for now**

Reasons:
- Easier to debug (can inspect JSON)
- No pipe buffering issues
- Works across platforms
- Can add pipes later if needed

Future: Add `--use-pipes` flag for performance-sensitive scenarios.

### Hunk Granularity: Per-line vs Per-hunk

**Decision: Per-hunk checkboxes**

Reasons:
- Git operates on hunks, not individual lines
- Simpler UI (one checkbox per hunk)
- Matches JetBrains behavior
- Sufficient for most use cases

Future: Add line-level splitting if users request it.

### Scrolling: Synchronized vs Independent

**Decision: Synchronized Baseline/Changes panes**

Reasons:
- Keeps related code aligned
- Easier to understand changes
- Matches diff tool UX expectations

File list scrolls independently (different content).

---

## File Locations

### Split Editor Repository

```
~/Projects/squire/cmd/gru
├── go.mod
├── go.sum
├── main.go
├── model.go
├── parser.go
├── assignment.go
├── plans.go
├── renderer.go
├── keybindings.go
├── styles.go
├── types.go
├── README.md
└── testdata/
    ├── sample-diff.txt
    ├── sample-input.json
    └── sample-output.json
```

### Installation

```bash
# Build
cd ~/Projects/squire/cmd/gru
go build -o gru

# Install (optional - for testing standalone)
go install

# Squire will invoke via full path or expect it in $PATH
```

---

## Appendix: Sample Files

### Sample Input JSON

See "Input JSON" section above.

### Sample Git Diff

```
diff --git a/auth.go b/auth.go
index 1234567..abcdefg 100644
--- a/auth.go
+++ b/auth.go
@@ -10,3 +10,5 @@ func Login() {
 func Login() {
-    // TODO
+    if u == nil {
+        return ErrNilUser
+    }
 }
@@ -45,3 +47,5 @@ func Validate() {
 func Validate() {
+    if u == nil {
+        return false
+    }
     return true
 }
diff --git a/logger.go b/logger.go
index 7654321..gfedcba 100644
--- a/logger.go
+++ b/logger.go
@@ -25,2 +25,3 @@ func Log(msg string) {
 func Log(msg string) {
     fmt.Println(msg)
+    // Log to file
 }
```

---

## Timeline Estimate

**Total:** ~20-30 hours for complete implementation

- Phase 5.1 (Foundation): 4-6 hours
- Phase 5.2 (Core UI): 6-8 hours
- Phase 5.3 (Assignment): 4-5 hours
- Phase 5.4 (Plan Management): 3-4 hours
- Phase 5.5 (Save/Load): 2-3 hours
- Phase 5.6 (Polish): 3-4 hours

**Recommended approach:** Build incrementally, test after each phase.

---

## Version History

- **v1.0** (2025-12-27): Initial PRD
