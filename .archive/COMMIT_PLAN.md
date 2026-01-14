# Plan: Implement Commit Plan Persistence

**Goal**: Implement save/load of commit plan (file dispositions) to `.git/info/gomion/commit-plan.json` with auto-save debouncing and manual save.

**Status**: 📋 Ready for Review

---

## What We're Building

Add persistence for file dispositions so they survive across TUI sessions:
- **Auto-save**: Debounced - save 2-3 seconds after last disposition change
- **Manual save**: `Ctrl+S`/`Cmd+S` keystroke for explicit checkpointing
- **Load on startup**: Restore dispositions from previous session
- **Storage**: `.git/info/gomion/commit-plan.json`

---

## Performance Compliance ✅

Per PERFORMANCE.md, all file I/O will run in `tea.Cmd`, not in `Update()`:

1. **Auto-save**: Timer fires → `tea.Cmd` writes JSON → returns SaveCompletedMsg
2. **Manual save**: `Ctrl+S`/`Cmd+S` → `tea.Cmd` writes JSON → returns SaveCompletedMsg
3. **Load**: Startup → `tea.Cmd` reads JSON → returns LoadCompletedMsg
4. **`Update()`**: Only updates lightweight state (loading flags, sequence numbers)

**No blocking I/O in Update()** - all file operations are async via commands.

---

## Architecture

### Type Layers

**Config Layer** (`gommod/gomcfg/commit_plan_v1.go` - NEW):
```go
package gomcfg

// CommitPlanV1 - scalar types only for JSON serialization
type CommitPlanV1 struct {
    Version    int               `json:"version"`
    Scope      string            `json:"scope"`        // "module" or "repo"
    ModulePath string            `json:"module_path,omitempty"`
    Timestamp  string            `json:"timestamp"`    // RFC3339
    CommitPlan map[string]string `json:"commit_plan"`  // path → disposition label
}
```

**Runtime Layer** (`gommod/gompkg/commit_plan.go` - NEW):
```go
package gompkg

type CommitPlanMap map[dt.RelFilepath]gomtui.FileDisposition

type CommitPlan struct {
    Version      int
    Scope        CommitScope
    ModulePath   dt.RelDirPath
    Timestamp    time.Time
    CommitPlan   CommitPlanMap
}

type CommitScope string
const (
    ModuleScope CommitScope = "module"
    RepoScope   CommitScope = "repo"
)
```

**Parse Function** (`gommod/gompkg/commit_plan.go`):
```go
// ParseCommitPlan converts gomcfg.CommitPlanV1 → gompkg.CommitPlan
// Validates scalar → domain type conversion
func ParseCommitPlan(cfg gomcfg.CommitPlanV1) (CommitPlan, error)
```

### JSON Format

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
    "debug.log": "ignore",
    "vendor/lib1.go": "exclude"
  }
}
```

**FileDisposition Serialization**: 
- Lowercase `Labels()`:
  - `"commit"`, 
  - `"omit"`, 
  - `"ignore"`, 
  - `"exclude"`

---

## FileDisposition JSON Support

**Add to `gommod/gomtui/file_disposition.go`**:

```go
// MarshalJSON serializes as lowercase label
func (fd *FileDisposition) MarshalJSON() ([]byte, error) {
  return jsonv2.Marshal(strings.ToLower(fd.Label()))
}

// UnmarshalJSON deserializes from string
func (fd *FileDisposition) UnmarshalJSON(data []byte) (err error) {
  var s string
  var parsed FileDisposition

  err = jsonv2.Unmarshal(data, &s)
  if err != nil {
    goto end
  }
  parsed, err = ParseFileDisposition(s)
  if err != nil {
    goto end
  }
  *fd = parsed
end:
  return err
}

// ParseFileDisposition parses string → FileDisposition
// Accepts (case-insensitive):
// - Single char: "c", "o", "i", "x"
// - String() values: "Commit", ".gitignore", ".git/info/exclude"
// - Label() values: "Commit", "Omit", "Ignore", "Exclude"
func ParseFileDisposition(s string) (fd FileDisposition, err error) {
   s = strings.TrimSpace(s)
   switch {
   case s == "":
      fd = UnspecifiedDisposition
   
   case len(s) == 1:
      switch s[0] {
      case 'c', 'C':
        fd = CommitDisposition
      case 'o', 'O':
        fd = OmitDisposition
      case 'i', 'I':
        fd = GitIgnoreDisposition
      case 'x', 'X':
        fd = GitExcludeDisposition
      }
   
   default:
      // Label() or String() matching
      switch strings.ToLower(s) {
      case "commit":
        fd = CommitDisposition
      case "omit":
        fd = OmitDisposition
      case "ignore", ".gitignore":
        fd = GitIgnoreDisposition
      case "exclude", ".git/info/exclude":
        fd = GitExcludeDisposition
      case "unspecified":
        fd = UnspecifiedDisposition
      default:
        err = NewErr(ErrInvalidFileDisposition, ErrorKV("value", s))
      }
   }
   return fd,err
}
```

---

## Save/Load Implementation (Using InfoStore)

**Save** (`gommod/gompkg/commit_plan.go`):
```go
// Save persists commit plan using InfoStore
func (cp *CommitPlan) Save(repoRoot dt.DirPath) (err error) {
  store := gitutils.NewInfoStore(repoRoot, gomion.CommitPlanFile)
  err = store.SaveJSON(cp.ToConfigV1())
  if err != nil {
    err= WithErr(err,ErrFailedToSaveCommitPlan,store.ErrKV())
  }
return err
}

// ToConfigV1 converts runtime → config type
func (cp *CommitPlan) ToConfigV1() gomcfg.CommitPlanV1 {
    cfg := gomcfg.CommitPlanV1{
        Version:    cp.Version,
        Scope:      string(cp.Scope),
        ModulePath: string(cp.ModulePath),
        Timestamp:  cp.Timestamp.Format(time.RFC3339),
        CommitPlan: make(map[string]string, len(cp.CommitPlan)),
    }
    for path, disp := range cp.CommitPlan {
        cfg.CommitPlan[string(path)] = strings.ToLower(disp.Label())
    }
    return cfg
}
```

**Load** (`gommod/gompkg/commit_plan.go`):
```go
// LoadCommitPlan loads from .git/info/gomion/commit-plan.json
func LoadCommitPlan(repoRoot dt.DirPath) (plan *CommitPlan, err error) {
  store := gitutils.NewInfoStore(repoRoot, gomion.CommitPlanFile)

  var cfg gomcfg.CommitPlanV1
  err = store.LoadJSON(&cfg)
  if errors.Is(err, dt.ErrFileNotExist) {
    err = nil  // No saved plan - not an error
    goto end
  }
  if err != nil {
    goto end
  }

  plan, err = ParseCommitPlan(cfg)
  if err != nil {
    err = NewErr(ErrInvalidCommitPlan, err)
  }
end:
  if err != nil {
    err = WithErr(err, ErrFailedToLoadCommitPlan, store.ErrKV())
  }
  return plan, err
}
```

---

## Async Save/Load via tea.Cmd (Performance Compliant)

### Message Types

```go
type BubbleTeaMsgType int 
const(
  UnspecifiedMsgType BubbleTeaMsgType = iota 	
  LoadCompleteMsgType  
  SaveCompleteMsgType  	
  LoadMsgType
  SaveMsgType	
)
// Messages
type CommitPlanMsg struct {
  msgType BubbleTeaMsgType
  seq int  // For debounce/staleness protection
  plan *gompkg.CommitPlan
  err error
}
```

### Commands

```go
type CommitPlanCmd struct {
  RepoRoot dt.DirPath 
  CommitPlan CommitPlanMap
}
// SaveCmd runs save in background (async I/O)
func (cmd CommitPlanCmd) SaveCmd(seq int) tea.Cmd {
  return func() tea.Msg {
    plan := &gompkg.CommitPlan{
      Version:      1,
      Scope:        gompkg.ModuleScope,  // TODO: Get from config
      ModulePath:   "",  // TODO: Get from EditorState
      Timestamp:    time.Now(),
      CommitPlan:   cmd.CommitPlan,
    }     
    return CommitPlanMsg{
      msgType: SaveCompleteMsgType,		
      seq:     seq, 
      err:     plan.Save(cmd.RepoRoot),
    }
  }
}

// LoadCmd runs load in background (async I/O)
func (cmd CommitPlanCmd) LoadCmd() tea.Cmd {
  return func() tea.Msg {
    plan, err := gompkg.LoadCommitPlan(cmd.RepoRoot)
    return CommitPlanMsg{
      msgType: LoadCompleteMsgType,
      plan:    plan,
      err:     err,
    }
  }
}
```

### Helper Methods

```go
// CommitPlanCmd returns a fresh CommitPlanCmd with current state
func (es EditorState) CommitPlanCmd() CommitPlanCmd {
    return CommitPlanCmd{
        RepoRoot:   es.UserRepo.Root,
        CommitPlan: es.CommitPlan,
    }
}

// LoadCommitPlanData populates dispositions from loaded plan
func (es EditorState) LoadCommitPlanData(plan *gompkg.CommitPlan) EditorState {
    if plan == nil {
        return es
    }
    for path, disp := range plan.CommitPlan {
        (*es.CommitPlan)[path] = disp
    }
    return es
}
```

### EditorState Fields

```go
type EditorState struct {
    // ... existing fields ...

    // Auto-save state
    saveTimer       *time.Timer
    saveSeq         int
    activeSaveSeq   int
    saveDebounce    time.Duration  // Default: 3 seconds
    saving          bool            // UI indicator
}
```

### Update() Logic (Non-Blocking)

```go
func (es EditorState) Update(msg tea.Msg) (EditorState, tea.Cmd) {
    var cmd tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() != "ctrl+s" {
          return es, nil
        }
        // Manual save - immediate, not debounced
        es.saving = true
        es.saveSeq++
        es.activeSaveSeq = es.saveSeq
        return es, es.CommitPlanCmd().SaveCmd(es.activeSaveSeq)

    case DispositionChangedMsg:
        // Update disposition
        (*es.CommitPlan)[msg.Path] = msg.Disposition

        // Schedule debounced save
        es = es.scheduleDebouncedSave()

    case CommitPlanMsg:
		switch msg.msgType {
        case LoadMsgType:
            // Note: Load is triggered from Init(), but could add reload via keystroke (e.g., 'r' key)

		case SaveMsgType:
            // Debounce timer fired - ignore if stale
            if msg.seq != es.saveSeq {
                goto end
            }

            // Execute save (non-blocking)
            es.saving = true
            es.activeSaveSeq = es.saveSeq
            cmd = es.CommitPlanCmd().SaveCmd(es.activeSaveSeq)

		case SaveCompleteMsgType:
            // Ignore stale results
            if msg.seq != es.activeSaveSeq {
                goto end
            }

            es.saving = false
            if msg.err != nil {
                es.Err = msg.err  // Show in UI
                es.Logger.Error("Failed to save commit plan", "error", msg.err)
            }

		case LoadCompleteMsgType:
            if msg.err != nil {
                es.Err = msg.err  // Show in UI
                es.Logger.Error("Failed to load commit plan", "error", msg.err)
                goto end
            }

            // Populate dispositions from loaded plan
            es = es.LoadCommitPlanData(msg.plan)
      }
    }

end:
    return es, cmd
}
```

### Debounce Helper

```go
// scheduleDebouncedSave schedules save after debounce period (non-blocking)
func (es EditorState) scheduleDebouncedSave() EditorState {
    // Cancel existing timer
    if es.saveTimer != nil {
        es.saveTimer.Stop()
    }

    // Bump sequence
    es.saveSeq++
    seq := es.saveSeq

    // Schedule new timer
    es.saveTimer = time.AfterFunc(es.saveDebounce, func() {
        // This runs in goroutine - send message to Update()
        es.Program.Send(CommitPlanMsg{
            msgType: SaveMsgType,
            seq:     seq,
        })
    })

    return es
}
```

**NOTE**: Need to store `*tea.Program` in EditorState to call `Program.Send()` from timer goroutine.

---

## Implementation Steps

### Step 1: Create gomcfg.CommitPlanV1 type
**File**: `gommod/gomcfg/commit_plan_v1.go` (NEW)
- Define CommitPlanV1 struct with scalar types
- Add godoc comments

### Step 2: Add FileDisposition JSON support
**File**: `gommod/gomtui/file_disposition.go` (MODIFY)
- Add `MarshalJSON()` method
- Add `UnmarshalJSON()` method
- Add `ParseFileDisposition()` function

### Step 3: Create gompkg.CommitPlan type
**File**: `gommod/gompkg/commit_plan.go` (NEW)
- Define CommitPlan struct with domain types
- Define CommitScope enum
- Implement `ParseCommitPlan()` function
- Implement `ToConfigV1()` method
- Implement `Save()` method
- Implement `LoadCommitPlan()` function

### Step 4: Add error sentinels
**File**: `gommod/gompkg/doterr.go` (MODIFY)
- Add `ErrInvalidCommitPlan`
- Add `ErrFailedToSaveCommitPlan`
- Add `ErrFailedToLoadCommitPlan`

**File**: `gommod/gomtui/doterr.go` (MODIFY)
- Add `ErrInvalidFileDisposition`

### Step 5: Add async save/load support to EditorState
**File**: `gommod/gomtui/editor_state.go` (MODIFY)
- Add type and related const: `BubbleTeaMsgType`
- Add message type: `CommitPlanMsg`
- Add command type: `CommitPlanCmd`
- Add commands: `CommitPlanCmd.SaveCmd()`, `CommitPlanCmd.LoadCmd()`
- Add helper methods: `CommitPlanCmd()`, `LoadCommitPlanData()`
- Add fields: `saveTimer`, `saveSeq`, `activeSaveSeq`, `saveDebounce`, `saving`, `Program`
- Implement `scheduleDebouncedSave()` method
- Add Ctrl+S/Cmd+S handler for manual save
- Add handling for save/load messages in `Update()`
- Hook `scheduleDebouncedSave()` into `DispositionChangedMsg` handling
- Set `es.Err` for save/load errors (in addition to logging)

### Step 6: Add load on TUI startup
**File**: `gommod/gomtui/tui.go` (MODIFY)
- Return `CommitPlanCmd.LoadCmd()` from `Init()`
- Store `*tea.Program` in EditorState after `tea.NewProgram()`

### Step 7: Initialize save debounce
**File**: `gommod/gomtui/editor_state.go` (MODIFY)
- Set `es.saveDebounce = 3 * time.Second` in initialization

---

## Files to Create/Modify

### New Files (3)
1. `gommod/gomcfg/commit_plan_v1.go` - Config layer type
2. `gommod/gompkg/commit_plan.go` - Runtime layer type, Save/Load, Parse
3. `gommod/gompkg/commit_plan_test.go` - Tests

### Modified Files (4)
1. `gommod/gomtui/file_disposition.go` - Add MarshalJSON, UnmarshalJSON, ParseFileDisposition
2. `gommod/gomtui/editor_state.go` - Add async save/load, debouncing, message handling
3. `gommod/gomtui/tui.go` - Load commit plan during initialization
4. `gommod/gompkg/doterr.go` + `gommod/gomtui/doterr.go` - Error sentinels

---

## Testing Checklist

1. ✓ Create commit plan, set dispositions, save manually (Ctrl+S)
2. ✓ Verify JSON written to `.git/info/gomion/commit-plan.json`
3. ✓ Exit TUI, restart - dispositions should be loaded
4. ✓ Change disposition - auto-save should trigger after 3 seconds
5. ✓ Change multiple dispositions rapidly - only one save should occur
6. ✓ Add new file to repo - should get UnspecifiedDisposition
7. ✓ Test ParseFileDisposition with all formats: "c", "Commit", "commit", "COMMIT"
8. ✓ Verify UI stays responsive during save/load (no blocking)

---

## Performance Notes

✅ **All file I/O is async via tea.Cmd**
- Save operations run in background goroutines
- Load operations run in background goroutines
- Update() only updates lightweight state (flags, sequence numbers)
- No blocking on filesystem in Update()

✅ **Debouncing prevents excessive saves**
- Multiple rapid disposition changes → single save
- Uses sequence numbers to ignore stale timers

✅ **Request IDs prevent stale results**
- `saveSeq` / `activeSaveSeq` pattern
- Ignore SaveCompletedMsg if seq doesn't match

---

## Summary of Improvements

**Questions Answered & Implemented**:

1. ✅ **CommitPlanCmd construction** - Extracted to `CommitPlanCmd()` helper method
2. ✅ **Load via message** - Load is triggered from `Init()`; could add reload keystroke (e.g., 'r' key) if needed
3. ✅ **Error handling** - Save/load errors now set `es.Err` (visible in UI) in addition to logging
4. ✅ **LoadCommitPlanData** - Extracted to helper method for cleaner code
5. ✅ **Bug fixes**:
   - Fixed `UnmarshalJSON` return value (was `nil`, now `err`)
   - Fixed `ToConfigV1` to use `cp.CommitPlan` (not `cp.Dispositions`)
   - Fixed typo: `plam` → `plan`
   - Fixed `LoadCommitPlan` return signature and error handling

**Open Questions**:

1. **Get ModulePath and Scope** - Need to determine where these come from in EditorState context
2. **Store `*tea.Program`** - Need to store in EditorState to call `Program.Send()` from timer goroutine
3. **Validate on load** (Future Phase 2):
   - Check for deleted files (remove from plan)
   - Check for new files (add as UnspecifiedDisposition)

