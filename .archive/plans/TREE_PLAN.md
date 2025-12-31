# Session Plan - FlagSet Migration & Tree Command

## Session Objective
Implement the `requires-tree` command and migrate all commands from legacy FlagDefs to the FlagSet pattern.

---

## ✅ Completed Tasks

### 1. Tree Command Implementation
- ✅ Read PRD: `docs/gomion-cli-tree-command-prd.md`
- ✅ Added error sentinels to `gompkg/gomioncmds/errors.go`
- ✅ Created `gompkg/retinue/tree.go` with ASCII tree rendering
- ✅ Created `gompkg/gomioncmds/requires_tree_cmd.go` with FlagSet pattern
- ✅ Implemented all flags: `--show-dirs`, `--show-all`, `--embed`, `--before`, `--after`
- ✅ Tested markdown embedding - working correctly
- ✅ Created `ROADMAP.md` to document deferred `--all` flag feature

### 2. Help Output Bug Fix
**Problem**: Command-specific flags were appearing in main help output, inconsistent "Flags:" vs "OPTIONS:" headers

**Root Cause**: `/Users/mikeschinkel/Projects/go-pkgs/go-cliutil/cmd_base.go` - The `Description()` and `Usage()` methods were appending flag information

**Fix Applied**:
- ✅ Modified `cmd_base.go` lines 106-110 to remove flag rendering from `Description()`
- ✅ Modified `cmd_base.go` lines 100-104 to remove flag rendering from `Usage()`
- ✅ Removed unused `strings` import from `cmd_base.go`

### 3. Command Migration to FlagSet Pattern

Examined canonical pattern in `/Users/mikeschinkel/Projects/xmlui/cli/xmluicli/clicmds/`

**Pattern**:
```go
// Package-level opts struct with pointer fields
var myOpts = &struct {
    Flag1 *string
    Flag2 *bool
}{
    Flag1: new(string),
    Flag2: new(bool),
}

// FlagSet referencing opts
var MyFlagSet = &cliutil.FlagSet{
    Name: "mycommand",
    FlagDefs: []cliutil.FlagDef{
        {
            Name:     "flag1",
            String:   myOpts.Flag1,
            // ...
        },
    },
}

// Command with only CmdBase
type MyCmd struct {
    *cliutil.CmdBase
}

// Handle() accesses via opts
func (c *MyCmd) Handle() error {
    if *myOpts.Flag1 != "" {
        // ...
    }
}
```

**Migrated Commands**:
- ✅ `gompkg/gomioncmds/requires_tree_cmd.go` - Already used FlagSet pattern
- ✅ `gompkg/gomioncmds/scan_cmd.go` - Migrated from legacy FlagDefs
  - Changed: `c.continueOnErr` → `*scanOpts.ContinueOnErr` (lines 86, 144, 172)
  - Removed: Command struct field `continueOnErr`
- ✅ `gompkg/gomioncmds/requires_list_cmd.go` - Migrated from legacy FlagDefs
  - Changed: `c.format` → `*requiresListOpts.Format`
  - Removed: Command struct field `format`

### 4. Build Success
- ✅ Successfully built with `make build`
- ✅ All compilation errors resolved

---

## 🔄 Current Task (In Progress)

### Test All Commands After Migration

**What to test**:
```bash
# 1. Main help should show clean command list (no command-specific flags)
./bin/gomion help

# 2. Command help should consistently show "OPTIONS:" header
./bin/gomion help scan
./bin/gomion help init
./bin/gomion help requires-list
./bin/gomion help requires-tree

# 3. Test all flags work correctly
./bin/gomion scan --continue .
./bin/gomion requires-list --format=json .
./bin/gomion requires-tree --show-dirs .
./bin/gomion requires-tree --embed=/tmp/test.md --before .
```

**Expected results**:
- Main help: Only command names/descriptions + global options
- Command help: Consistent "OPTIONS:" header (not "Flags:")
- No command-specific flags appearing in main help
- All flags parse and execute correctly

---

## 📋 Pending Tasks

1. **Test Migration** (Current)
   - Run help commands to verify bug fix
   - Test each command with flags
   - Verify output consistency

2. **Verify No Regressions**
   - Ensure all existing commands still work
   - Check that flag defaults are correct
   - Verify required vs optional flags work as expected

3. **Future Feature** (Deferred to ROADMAP.md)
   - Implement `--all` flag for external dependencies
   - See `ROADMAP.md` section: "External Module Dependencies"

---

## 🔧 Key Files Modified

### In gomion repo:
- `gompkg/gomioncmds/errors.go` - Added tree error sentinels
- `gompkg/retinue/tree.go` - Tree rendering implementation
- `gompkg/gomioncmds/requires_tree_cmd.go` - New tree command
- `gompkg/gomioncmds/scan_cmd.go` - Migrated to FlagSet
- `gompkg/gomioncmds/requires_list_cmd.go` - Migrated to FlagSet
- `gompkg/gomioncmds/help_cmd.go` - Fixed for updated cliutil API
- `ROADMAP.md` - Created to track features

### In go-cliutil repo:
- `/Users/mikeschinkel/Projects/go-pkgs/go-cliutil/cmd_base.go`
  - Removed flag rendering from `Description()` method
  - Removed flag rendering from `Usage()` method
  - Removed unused `strings` import

---

## 🎯 Next Session Actions

When you restart:

1. **Run the test commands** listed in "Test All Commands After Migration"
2. **Review output** for correctness
3. **Fix any issues** discovered during testing
4. **Mark testing task as completed** when all tests pass
5. **Consider next feature** from ROADMAP.md if desired

---

## 📝 Notes

- All commands now follow canonical FlagSet pattern from xmlui
- No more legacy FlagDefs in command structs
- Help rendering now uses Go templates exclusively
- Flags are rendered by templates, not by Description()/Usage()
