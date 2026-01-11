# Gomion Bugs

Active bugs and issues to fix.

## Status Key

- 🔴 **Not Fixed** — Bug confirmed, not yet addressed
- 🟡 **In Progress** — Currently being investigated/fixed
- 🟢 **Fixed** — Resolved and merged
- 🟣 **Cannot Reproduce** — Unable to reproduce, needs more info

## Priority Levels

- **Critical** — Crashes, data loss, or completely blocks usage
- **High** — Major functionality broken, significant user impact
- **Medium** — Feature partially broken, workarounds available
- **Low** — Minor issues, edge cases, cosmetic problems

---

## Active Bugs

### Git upstream required for branches without upstream
**Status:** 🔴 Not Fixed
**Priority:** High
**Component:** gitutils

**Error:**
```
Command failed: git error; git rev-parse --abbrev-ref --symbolic-full-name @{u}:
git rev-parse --abbrev-ref --symbolic-full-name @{u}: exit status 128
(fatal: no upstream configured for branch 'known-good')
meta: dir_path=~/Projects/gomion/gommod
```

**Issue:**
Code assumes all branches have upstream configured. Fails when working with local-only branches.

**Expected Behavior:**
Handle branches without upstream gracefully. Local-only branches are valid and should not cause errors.

**Reproduction:**
1. Create local branch: `git checkout -b known-good`
2. Run gomion command that checks git status
3. Error occurs when trying to get upstream branch

**Likely Location:**
`gitutils/repo.go` or related git status checking code

---

### Switching from Module to Repo mode does not reload files
**Status:** 🔴 Not Fixed
**Priority:** Medium
**Component:** gomtui (TUI file staging)

**Issue:**
When switching from Module mode to Repo mode in the file staging TUI, the file list does not appear to refresh/reload to show the repo-wide files.

**Expected Behavior:**
Switching between Module and Repo modes should reload the file list to reflect the appropriate scope (module-only files vs. entire repository files).

**Likely Location:**
`gomtui/tui.go` or mode switching logic in file disposition model

---

### Right-pane file content view breaks layout with wrapping text
**Status:** 🔴 Not Fixed
**Priority:** High
**Component:** gomtui (TUI file viewer)

**Issue:**
When the right-pane file content view contains wrapping text, the viewport grows beyond its allocated size and breaks the overall TUI layout.

**Expected Behavior:**
The file content viewport should respect its fixed size constraints and handle text wrapping without expanding beyond the allocated layout dimensions.

**Likely Location:**
`gomtui/tui.go` or file viewer viewport sizing logic

---

## Fixed Bugs

(None yet)

---

## Notes

### For Claude Code (Bug Management Workflow)

**Moving bugs to DONE.md:**
- When a bug is marked 🟢 Fixed or 🟣 Cannot Reproduce, move it from BUGS.md to DONE.md
- BUGS.md should only contain active bugs (🔴 Not Fixed, 🟡 In Progress)
- This prevents BUGS.md from growing large over time
- DONE.md serves as the archive for resolved/closed bugs
- Format in DONE.md: Keep the same structure but add resolution date

**General Notes:**
- Include error messages, reproduction steps, and expected behavior
- Link to related issues in other files if applicable
- When migrating to GitHub Issues, reference issue numbers here
