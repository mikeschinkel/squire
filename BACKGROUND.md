# Squire Background & Design Philosophy

This document captures the design philosophy, core concepts, and implementation details of Squire's release management workflow. It serves as context for future development sessions.

## The Publishing/Release Workflow

Squire automates a complex multi-repository Go module publishing workflow. The user's manual process (which we're automating):

### Manual Workflow Steps

1. **Find in-flux repositories**
   - Look for repos with uncommitted changes
   - Look for repos with untracked files
   - Look for repos with commits not yet tagged
   - Look for repos with tags not yet pushed to remote

2. **Find the next module to release**
   - Among all in-flux modules, find one that has NO dependencies on other in-flux modules
   - This is the "leaf" - a module that can be released without breaking anything

3. **Prepare the module for release**
   - Remove any `replace` directives from go.mod (local development overrides)
   - Run `go mod tidy` to clean up dependencies
   - Run `make vet` to check for code issues
   - Run `make lint` to check code style
   - Run `make test` to ensure all tests pass

4. **Commit the changes**
   - Run `git status` to review what changed
   - Select which files to commit
   - Run `/commit-msg` skill to generate commit message
   - Review and possibly edit the commit message
   - Run `git commit` with the message
   - Run `git pull && git push` to sync with remote

5. **Tag and release**
   - Check that GitHub Actions workflows passed
   - Run `git tag` to see existing tags
   - Make a subjective assessment: is this a breaking change?
   - Decide on the next version number based on that assessment
   - Either:
     - Add tag locally and push it, OR
     - Run GitHub Actions release workflow (manual trigger) with tag + description
   - Run `git pull` to fetch the new tag

6. **Repeat**
   - Run the tool again to find the next in-flux module
   - Continue until all modules are released (nothing in-flux)

### The Automation Goal

Squire's `process` command should:
- Find the next in-flux module to work on (the "leaf")
- Tell the user what module to focus on
- Provide verdict on whether changes are breaking (using API diff analysis)
- Eventually: automate as much of the workflow as possible (tidy, vet, lint, test, commit, tag)

## Core Concepts

### Active Development

**Definition**: A module is "actively being developed" if it exists in any of the `scan_dirs` configured in `~/.config/squire/config.json`.

**Key Points**:
- Determined by **filesystem presence**, not configuration or replace directives
- No distinction between "direct" and "indirect" dependencies
- No concept of "primary development targets" vs. secondary
- If it's in scan_dirs, we care about it
- If it's not in scan_dirs, we don't care (it comes from module cache/proxy)

**Why This Matters**:
- Simple rule: local = developing, not local = not developing
- No need for extra config files to track "active" status
- Avoids the problem of config getting out of sync with reality

### In-Flux State

**Definition**: A module/repository is "in-flux" (not ready for release) if **ANY** of the following are true:

1. **Dirty working tree**: Uncommitted changes or untracked files exist
2. **Commits not tagged**: Latest commit doesn't have a version tag
3. **Tags not pushed**: Latest tag exists locally but hasn't been pushed to remote (origin)

**Detection Algorithm**:
```
IF repo has uncommitted changes (dirty working tree, untracked files):
  → in-flux

ELSE IF latest commit != latest pushed tag:
  → in-flux (commits exist that aren't tagged/pushed)

ELSE:
  → NOT in-flux (latest commit has latest remote tag)
```

**Implementation**: Git-based checks in `gitutils` package, called from `GoModule.IsInFlux()`

### The Leaf Algorithm

**Goal**: Find which in-flux module can be released next.

**Rule**: You can only release a module if **all of its dependencies** are already released (not in-flux).

**Algorithm**:
```
1. Build dependency graph from all go.mod files in scan_dirs
2. Find all modules that are in-flux (git state check)
3. For each in-flux module:
   - Check its dependencies (from go.mod requires)
   - If ALL local dependencies are NOT in-flux: it's a candidate
   - If ANY local dependency is in-flux: skip this module (can't release yet)
4. Return the deepest candidate (in dependency order)
```

**Critical Distinction**:
- ❌ **WRONG**: "Find a module that's not in-flux"
- ✅ **RIGHT**: "Find an in-flux module whose dependencies are all clean"

**Why This Is Right**:
- During development, **most or all modules ARE in-flux** - this is the expected state!
- We're looking for which one to fix/release **next**, not which one is already done
- We can only release a module after its dependencies are released
- Bottom-up approach: release leaves first, work up the dependency tree

### Expected State: Most Things Are In-Flux

**Important Mental Model**:
- During active development, it's **normal** for most/all modules to be in-flux
- The publishing workflow systematically processes in-flux modules one at a time
- Each iteration:
  1. Find the next releasable in-flux module (the leaf)
  2. Fix it (tidy, vet, lint, test)
  3. Commit, tag, and push it
  4. Run again to find the next one
- Gradually whittle down the in-flux set until nothing remains
- When nothing is in-flux, we're done - all modules are released

**This is NOT an error condition** - it's the working state!

### Why We Scan ALL of scan_dirs

**Question**: Why scan all repos in scan_dirs, including modules we don't have replace directives for?

**Answer**: We need the complete dependency graph.

**Example**:
```
xmlui/cli → go-dt → some-lib
          ↘ go-fsfix
```

Even if `go-fsfix` is an **indirect** dependency (no replace directive in xmlui/cli's go.mod), we still need to know about it because:
- If go-fsfix is in-flux (uncommitted changes), we can't release xmlui/cli yet
- The dependency chain includes go-fsfix, even if indirectly
- We need to track its state to know when xmlui/cli is ready

**Key Insight**: Direct vs. indirect doesn't matter. If a module:
1. Exists in scan_dirs (local development), AND
2. Is in the dependency chain (even transitively)

Then we need to track it and potentially release it before we can release modules that depend on it.

### Replace Directives Are Transient State

**Important Understanding**: Replace directives in go.mod represent **CURRENT state**, not **INTENT**.

**Why They're Transient**:
- User can manually edit go.mod and add/remove replace directives
- Squire might fail partway through processing and leave replace directives in place
- Replace directives get added during development and removed before release
- They change throughout the workflow - they're working state, not configuration

**Therefore**:
- ❌ Don't use replace directives to determine "what we're developing"
- ✅ Use scan_dirs (filesystem presence) to determine "what we're developing"
- Replace directives tell us **current** local overrides, not intent

**The Right Source of Truth**:
- **Intent**: scan_dirs in config (what repos we care about)
- **Current State**: git state (dirty, commits, tags) + replace directives (current overrides)

### No Config Files for "Active Development" (For Now)

**Principle**: Derive everything from existing sources; don't duplicate.

**What We Derive From**:
1. `~/.config/squire/config.json` → scan_dirs (which repos to care about)
2. Filesystem → which go.mod files exist in scan_dirs
3. go.mod files → module paths, dependencies, replace directives (current state)
4. Git state → dirty, commits, tags, pushed/unpushed

**Why No .squire/config.json (Yet)**:
- We can figure out everything we need from the above sources
- Adding config files creates another source of truth that can get out of sync
- Keep it simple until we have stable, working functionality
- Only add config/cache later for **performance optimization** (if needed)

**Principle**: One source of truth. ("Man with one watch knows the time; man with two watches never sure.")

## Command Architecture

### `squire plan` (Original)

**Purpose**: Display the dependency graph for inspection.

**Behavior**:
- Scans repos in scan_dirs
- Finds all go.mod files
- Builds dependency graph
- Traverses in dependency order (deepest first)
- Prints repos and their modules

**Does NOT**:
- Use the ReleaseEngine
- Find leaves or suggest what to work on next
- Check in-flux state
- Compute verdicts

**Why It Exists**:
- "Known good benchmark" - original behavior before engine was added
- Useful for inspecting what repos/modules exist
- Debugging - see what the graph looks like
- Simpler - doesn't require git state checks or in-flux detection

**Backward Compatibility**:
- MUST NOT CHANGE - this is the reference implementation
- If we need engine-based behavior, create a new command
- Users rely on this for inspection/debugging

### `squire process` (Engine-Based)

**Purpose**: Find the next in-flux module to work on (for automation).

**Behavior**:
- Uses `ReleaseEngine` to find the leaf
- Runs in-flux detection (git state)
- Computes verdict (breaking change analysis)
- Returns terse, machine-parseable output: `module|verdict|reason`

**Output Format**:
```
/Users/mikeschinkel/Projects/go-pkgs/go-dt|withheld|no baseline tag found (first release?)
```

**Use Case**:
- Automation scripts
- CI/CD integration
- Publishing workflows
- Finding what to work on next

**Relationship to `plan`**:
- `plan` = inspection (graph display)
- `process` = automation (find next action)
- Both are useful for different purposes

## Historical Context & Lessons Learned

### The "Replace Directives Only" Mistake

**What We Initially Tried**:
- Only include modules that have replace directives pointing to them
- Rationale: "If it's not replaced, it's not local development"

**Why This Was Wrong**:
1. Replace directives are transient (added/removed during workflow)
2. Squire might fail partway through and leave state inconsistent
3. User might manually edit go.mod
4. We'd lose track of modules that ARE local but temporarily don't have replaces

**Correct Approach**:
- Include ALL modules in scan_dirs
- Derive "local development" from filesystem presence
- Use replace directives only for **current state**, not **intent**

### The "Find Non-In-Flux Module" Mistake

**What We Initially Tried**:
- Find a module that's NOT in-flux (clean, tagged, pushed)
- Rationale: "Find something that's ready"

**Why This Was Wrong**:
1. During development, most/all modules ARE in-flux (expected state!)
2. Looking for "not in-flux" finds modules that are already done
3. Doesn't help us know what to work on NEXT

**Correct Approach**:
- Find modules that ARE in-flux (need work)
- Among those, find ones whose dependencies are clean (can be released)
- This tells us what to work on next

### The "Primary Development Targets" Confusion

**What We Initially Thought**:
- Some modules are "primary targets" (what we're focused on)
- Others are just dependencies (less important)

**Why This Was Wrong**:
1. No such distinction in the user's workflow
2. All local modules matter equally
3. Direct vs. indirect doesn't change priority
4. If it's in scan_dirs, we're developing it

**Correct Understanding**:
- Binary distinction: local (in scan_dirs) or not local
- All local modules are equally important
- The dependency order determines **sequence**, not **importance**

## Implementation Notes

### Key Files

**Core Engine**:
- `squirepkg/retinue/engine.go` - ReleaseEngine (finds next module)
- `squirepkg/retinue/go_module.go` - GoModule with `IsInFlux()` check
- `squirepkg/retinue/go_mod_graph.go` - Dependency graph building and traversal

**Commands**:
- `squirepkg/squirecmds/plan_cmd.go` - Original graph display
- `squirepkg/squirecmds/process_cmd.go` - Engine-based next module finder

**Supporting Packages**:
- `squirepkg/gitutils/` - Git state checking (dirty, tags, push status)
- `squirepkg/modutils/` - go.mod analysis (dependencies, replace directives)
- `squirepkg/apidiffr/` - Breaking change detection (API diff)

### Deterministic Output

**Requirement**: Output must be deterministic (same order every time).

**Why**:
- Tests need to be stable
- User expects consistent behavior
- Diffs should be meaningful

**How**:
- Use `dtx.OrderedMap` instead of Go's built-in `map`
- Ensure iteration order is consistent
- Sort where necessary (by module path, directory path, etc.)

**Check These Places**:
- Graph traversal iteration
- Module directory iteration
- Printing output

### Future Enhancements

**Not Implemented Yet, But Planned**:

1. **Automated tidy/vet/lint/test**
   - Run these automatically before asking user to commit
   - Only proceed if all pass
   - Save user time

2. **Smart commit message generation**
   - Use `/commit-msg` skill automatically
   - Analyze git diff to generate message
   - Present to user for approval

3. **Automated tagging**
   - Use `go-nextver` logic to determine next version
   - Detect breaking changes automatically (via API diff)
   - Suggest tag or apply it automatically

4. **GitHub Actions integration**
   - Trigger release workflows automatically
   - Wait for CI to pass before tagging
   - Handle errors (failed tests, etc.)

5. **Multi-module release**
   - Release multiple modules in one workflow
   - Handle dependency chains automatically
   - Atomic releases (all or nothing)

6. **Workspace support**
   - Manage go.work files automatically
   - Add/remove modules from workspace as needed
   - Keep go.work in sync with replace directives

## Glossary

**In-Flux**: Not ready for release. Has uncommitted changes, commits not tagged, or tags not pushed.

**Leaf Module**: An in-flux module whose dependencies are all clean (not in-flux). Can be released next.

**scan_dirs**: Directories configured in `~/.config/squire/config.json` where we look for local development repos.

**Replace Directive**: A line in go.mod like `replace github.com/foo/bar => ../local/bar` that overrides a module with a local path.

**Verdict**: Assessment of whether changes are breaking (from API diff analysis). Values: `breaking`, `likely_breaking`, `maybe_not_breaking`, `withheld`, `unknown`.

**Active Development**: A module is actively being developed if it exists in scan_dirs (local filesystem).

**Dependency Chain**: The transitive closure of dependencies. If A depends on B and B depends on C, then A's dependency chain includes both B and C.

## Questions & Answers

**Q**: Why scan all of scan_dirs instead of just following replace directives?

**A**: Because replace directives are transient state. They get added/removed during the workflow. We need to know about all local modules, not just the ones currently replaced.

**Q**: Why does indirect dependency matter?

**A**: If we depend on it (even indirectly), and it's in-flux, we can't release until it's released. The dependency chain matters, not just direct dependencies.

**Q**: What if everything is in-flux?

**A**: That's normal! During development, most/all modules are in-flux. The algorithm finds which one to release first (the leaf), you fix/release it, then run again for the next one.

**Q**: What's the difference between `plan` and `process`?

**A**: `plan` displays the graph (inspection). `process` finds the next module to work on (automation). Different tools for different purposes.

**Q**: Why not use config files to track "active development"?

**A**: Because we can derive it from scan_dirs + filesystem. Adding config would create another source of truth that could get out of sync. Keep it simple.

---

## TODO: Future Implementation

### Branch Validation
**Status**: Design complete (ADR-2025-12-21), implementation pending

Implement branch validation as specified in `adrs/adr-2025-12-21-branch-metadata-storage.md`:
- Store branch expectations in `.git/config` (local, branch-independent)
- Mirror expectations in versioned JSON on `_squire-meta` branch (shared, auditable)
- Validate at runtime: compare expected branch vs actual checkout
- Emit actionable errors on mismatch
- Directory-anchored resolution (worktree-compatible)

**Key components to implement:**
1. Read/write branch expectations to/from `.git/config`
2. Read/write versioned JSON on `_squire-meta` branch
3. Runtime validation in ReleaseEngine
4. Keep .git/config and JSON in sync
5. Report divergence clearly

### Module-Scoped Dirty Detection
**Status**: Partially implemented

Currently `repo.IsDirty()` checks entire repository. Need to add module-scoped detection:
- `repo.IsDirtyInPath(relPath)` - check only files within module subdirectory
- Allows multi-module repos where some modules are clean
- Enables working on one module while others have uncommitted changes

**Implementation**: Add method to `gitutils.Repo` using `git status --porcelain <path>`

---

**Document Version**: 2025-12-21 (Initial)
**Last Updated**: Session with goofy-singing-jellyfish plan + human-friendly output
