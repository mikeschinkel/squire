# Recent Completed Work

This file tracks recently completed work (last few days only). Older work is archived.

---

## Commit Plan Persistence (Jan 4-5, 2026)

**Status:** ✅ Complete

Implemented save/load of file dispositions to `.git/info/gomion/commit-plan.json` with auto-save and manual save.

### Files Created

1. **`gommod/gomcfg/commit_plan_v1.go`** — Config layer (scalar types for JSON)
   - `CommitPlanV1` struct with version, scope, module_path, timestamp, commit_plan

2. **`gommod/gompkg/commit_plan.go`** — Runtime layer (domain types)
   - `CommitPlan` struct with `time.Time`, `dt.RelDirPath`, `CommitPlanMap`
   - `ParseCommitPlan()` — Convert gomcfg → gompkg
   - `ToConfigV1()` — Convert gompkg → gomcfg
   - `Save()` — Persist to `.git/info/gomion/commit-plan.json`
   - `LoadCommitPlan()` — Load from disk

3. **`gommod/gomion/const.go`** — Constants package
   - `CommitPlanFile = "commit-plan.json"`

### Files Modified

1. **`gommod/gomtui/file_disposition.go`**
   - Added `ParseFileDisposition()` function (case-insensitive parsing)
   - Accepts single char ("c"), String() values ("Commit"), Label() values ("commit")

2. **`gommod/gomtui/editor_state.go`**
   - Added auto-save fields: `saveSeq`, `activeSaveSeq`, `saveDebounce`, `saving`
   - Added `CommitPlanMsg` and `CommitPlanCmd` types
   - Added `Init()` — Load plan on startup (parallel with file loading)
   - Added `CommitPlanCmd()` — Create command with current state
   - Added `scheduleDebouncedSave()` — Schedule save after 3-second debounce
   - Added `handleCommitPlanMsg()` — Handle all message types (load, save, error)
   - Fixed race condition in `loadFiles()` — Only init dispositions if not already loaded

3. **`gommod/gompkg/errors.go`**
   - Added `ErrInvalidCommitPlan`, `ErrFailedToSaveCommitPlan`, `ErrFailedToLoadCommitPlan`

4. **`gommod/gomtui/errors.go`**
   - Added `ErrInvalidFileDisposition`

### Key Features

- **Two-layer type system:** gomcfg (scalars) → gompkg (domain types)
- **Auto-save:** Debounced 3 seconds after last change
- **Manual save:** Cmd-S/Ctrl-S for explicit checkpointing
- **Async I/O:** All operations via `tea.Cmd` (never blocks `Update()`)
- **Staleness protection:** Sequence numbers prevent outdated saves

### JSON Format

```json
{
  "version": 1,
  "scope": "module",
  "module_path": "gommod",
  "timestamp": "2026-01-05T03:11:03-05:00",
  "commit_plan": {
    "gommod/gomtui/file.go": "commit",
    "gommod/STATUS.md": "exclude",
    "gommod/FEATURES_TO_ADD.md": "omit",
    "gommod/PERFORMANCE.md": "ignore"
  }
}
```

### Bugs Fixed

1. **InfoStore bug** — Was using full paths with DirFS instead of relative paths
2. **Race condition** — File loading overwrote loaded dispositions (fixed with existence check)

---

## Phase 2.5: File Selection View (Dec 2025)

**Status:** ✅ Complete

Module-scoped file filtering with disposition assignment for commit workflow.

### Features Implemented

**Step 1: gitutils Filter Support**
- ✅ `FileFilter` type defined in `gitutils/working.go`
- ✅ `GetChangedFilesFiltered()` method implemented

**Step 2: Module-Scoped Filtering**
- ✅ `AutoDetectModule()` in `gompkg/module.go`
- ✅ Module detection from go.mod location

**Step 3: FileDisposition Types**
- ✅ `FileDisposition` byte enum in `gomtui/file_disposition.go`
- ✅ Constants: `CommitDisposition`, `OmitDisposition`, `GitIgnoreDisposition`, `GitExcludeDisposition`
- ✅ `ParseFileDisposition()` for case-insensitive parsing

**Step 4: File Selection View UI**
- ✅ `FileSelectionView` mode in `view_mode.go`
- ✅ `initFileSelectionView()` in `editor_state.go`
- ✅ `updateFileSelectionView()` in `editor_state.go`
- ✅ `renderFileSelectionView()` in `editor_state.go`
- ✅ Tree view with BubbleTree
- ✅ File content display pane
- ✅ Disposition toggling with keyboard shortcuts
- ✅ Table view for directory contents

### Key Design Decisions

- **Disposition naming:** Used "Commit" instead of "Include", "Omit" instead of "DoNotInclude"
- **Module-scoped by default:** Auto-detects module from go.mod
- **Four dispositions:** Commit, Omit, Ignore (gitignore), Exclude (git/info/exclude)
- **Tree + Table views:** Flexible navigation of file hierarchy

---

_For older completed work, see `.archive/DONE-archive.md`_
